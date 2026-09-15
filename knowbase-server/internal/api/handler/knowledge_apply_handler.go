// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package handler

import (
	"encoding/json"

	"github.com/gin-gonic/gin"

	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/api/middleware"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/api/request"
	kbResp "icode.baidu.com/baidu/devops-ai/knowbase-server/internal/api/response"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/config"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/infrastructure/database"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/repository"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/service"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/errors"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/log"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/response"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/types/constants"
)

// KnowledgeApplyHandler 知识应用处理器
type KnowledgeApplyHandler struct {
	applyService service.KnowledgeApplyService
}

// NewKnowledgeApplyHandler 创建知识应用处理器实例
func NewKnowledgeApplyHandler() *KnowledgeApplyHandler {
	ormer := database.GetOrm()

	kbRepo := repository.NewKnowledgebaseRepository(ormer)
	memberRepo := repository.NewKnowledgebaseMemberRepository(ormer)
	knowRepo := repository.NewKnowledgeRepository(ormer)
	versionRepo := repository.NewKnowledgeVersionRepository(ormer)
	configRepo := repository.NewKnowledgeApplyConfigRepository(ormer)
	recordRepo := repository.NewKnowledgeApplyRecordRepository(ormer)
	dirRepo := repository.NewKnowledgebaseDirectoryRepository(ormer)

	permissionService := service.NewPermissionService(memberRepo)

	// 创建 AgentNotifier
	var agentNotifier service.AgentNotifier
	cfg := config.GetConfig()
	if cfg != nil && len(cfg.AgentWebhooks) > 0 {
		agentNotifier = service.NewWebhookAgentNotifier(cfg.AgentWebhooks)
		log.Logger.Infof("已配置 %d 个 Agent Webhook", len(cfg.AgentWebhooks))
	} else {
		agentNotifier = service.NewNoopAgentNotifier()
		log.Logger.Info("未配置 Agent Webhook，使用空通知器")
	}

	svc := service.NewKnowledgeApplyService(
		kbRepo, knowRepo, versionRepo, configRepo, recordRepo, dirRepo,
		permissionService,
		agentNotifier,
	)

	return &KnowledgeApplyHandler{
		applyService: svc,
	}
}

// CreateOrUpdateConfig 创建或更新应用配置
// POST /api/v1/knowledge/apply/configs
func (h *KnowledgeApplyHandler) CreateOrUpdateConfig(c *gin.Context) {
	ctx := middleware.GetContext(c)

	var req request.CreateOrUpdateApplyConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Logger.ErrorfCtx(ctx, "解析请求体失败: %v", err)
		response.Error(c, errors.New(errors.ErrCodeInvalidParam, err.Error()))
		return
	}

	username, exists := c.Get(constants.AuthUsername)
	if !exists {
		log.Logger.ErrorfCtx(ctx, "获取用户名失败")
		response.Error(c, errors.ErrInvalidToken)
		return
	}

	isAdmin := middleware.GetClientIsAdmin(c)
	result, err := h.applyService.CreateOrUpdateConfig(ctx, &req, username.(string), isAdmin)
	if err != nil {
		log.Logger.ErrorfCtx(ctx, "创建或更新配置失败: %v", err)
		response.Error(c, err)
		return
	}

	response.Success(c, &kbResp.ApplyResultResponse{
		ConfigUUID:    result.ConfigUUID,
		ConfigVersion: result.ConfigVersion,
		TotalCount:    result.TotalCount,
		SuccessCount:  result.SuccessCount,
		FailedCount:   result.FailedCount,
	})
}

// GetConfig 获取应用配置详情
// GET /api/v1/knowledge/apply/configs/:uuid
func (h *KnowledgeApplyHandler) GetConfig(c *gin.Context) {
	ctx := middleware.GetContext(c)

	configUUID := c.Param("uuid")
	if configUUID == "" {
		response.Error(c, errors.ErrInvalidParam)
		return
	}

	username, exists := c.Get(constants.AuthUsername)
	if !exists {
		log.Logger.ErrorfCtx(ctx, "获取用户名失败")
		response.Error(c, errors.ErrInvalidToken)
		return
	}

	isAdmin := middleware.GetClientIsAdmin(c)
	result, err := h.applyService.GetConfig(ctx, configUUID, username.(string), isAdmin)
	if err != nil {
		log.Logger.ErrorfCtx(ctx, "获取配置详情失败: %v", err)
		response.Error(c, err)
		return
	}

	response.Success(c, convertApplyConfigToResponse(result))
}

// ListConfigs 查询应用配置列表
// GET /api/v1/knowledge/apply/configs
func (h *KnowledgeApplyHandler) ListConfigs(c *gin.Context) {
	ctx := middleware.GetContext(c)

	var req request.ListApplyConfigRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		log.Logger.ErrorfCtx(ctx, "解析查询参数失败: %v", err)
		response.Error(c, errors.New(errors.ErrCodeInvalidParam, err.Error()))
		return
	}

	// 校验：knowledgebaseUuid、agentCode、targetId 必须有一个非空
	if req.KnowledgebaseUUID == "" && req.AgentCode == "" && req.TargetID == "" {
		response.Error(c, errors.New(errors.ErrCodeInvalidParam, "knowledgebaseUuid、agentCode、targetId 必须有一个非空"))
		return
	}

	// 设置默认分页
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}

	username, exists := c.Get(constants.AuthUsername)
	if !exists {
		log.Logger.ErrorfCtx(ctx, "获取用户名失败")
		response.Error(c, errors.ErrInvalidToken)
		return
	}

	configs, total, err := h.applyService.ListConfigs(ctx, &req, username.(string))
	if err != nil {
		log.Logger.ErrorfCtx(ctx, "查询配置列表失败: %v", err)
		response.Error(c, err)
		return
	}

	// 转换响应
	list := make([]*kbResp.ApplyConfigResponse, len(configs))
	for i, config := range configs {
		list[i] = convertApplyConfigToResponse(config)
	}

	response.Success(c, &kbResp.ApplyConfigListResponse{
		List:  list,
		Total: total,
	})
}

// GetConfigHistory 获取配置历史版本
// GET /api/v1/knowledge/apply/configs/:uuid/history
func (h *KnowledgeApplyHandler) GetConfigHistory(c *gin.Context) {
	ctx := middleware.GetContext(c)

	configUUID := c.Param("uuid")
	if configUUID == "" {
		response.Error(c, errors.ErrInvalidParam)
		return
	}

	username, exists := c.Get(constants.AuthUsername)
	if !exists {
		log.Logger.ErrorfCtx(ctx, "获取用户名失败")
		response.Error(c, errors.ErrInvalidToken)
		return
	}

	history, err := h.applyService.GetConfigHistory(ctx, configUUID, username.(string))
	if err != nil {
		log.Logger.ErrorfCtx(ctx, "获取配置历史失败: %v", err)
		response.Error(c, err)
		return
	}

	// 转换响应
	items := make([]*kbResp.ApplyConfigHistoryItem, len(history))
	for i, h := range history {
		configuredEntries := make([]*kbResp.ApplyConfigEntryItem, len(h.ConfiguredEntries))
		for j, e := range h.ConfiguredEntries {
			configuredEntries[j] = &kbResp.ApplyConfigEntryItem{
				EntryType:        e.EntryType,
				KnowledgeUUID:    e.KnowledgeUUID,
				KnowledgeVersion: e.KnowledgeVersion,
				KnowledgeTitle:   e.KnowledgeTitle,
				DirectoryUUID:    e.DirectoryUUID,
				DirectoryName:    e.DirectoryName,
				Operator:         e.Operator,
				CreatedAt:        e.CreatedAt,
			}
		}
		knowledgeList := make([]*kbResp.AppliedKnowledgeItem, len(h.KnowledgeList))
		for j, k := range h.KnowledgeList {
			knowledgeList[j] = &kbResp.AppliedKnowledgeItem{
				KnowledgeUUID:       k.KnowledgeUUID,
				KnowledgeVersion:    k.KnowledgeVersion,
				KnowledgeTitle:      k.KnowledgeTitle,
				KnowledgeContent:    k.KnowledgeContent,
				KnowledgeType:       k.KnowledgeType,
				TemplateName:        k.TemplateName,
				PushStatus:          k.PushStatus,
				Operator:            k.Operator,
				SourceEntryType:     k.SourceEntryType,
				SourceDirectoryUUID: k.SourceDirectoryUUID,
				SourceDirectoryName: k.SourceDirectoryName,
				CreatedAt:           k.CreatedAt,
			}
		}
		items[i] = &kbResp.ApplyConfigHistoryItem{
			Version:           h.Version,
			Operator:          h.Operator,
			CreatedAt:         h.CreatedAt,
			ConfiguredEntries: configuredEntries,
			KnowledgeList:     knowledgeList,
		}
	}

	response.Success(c, &kbResp.ApplyConfigHistoryResponse{History: items})
}

// AddKnowledge 增量添加知识到配置
// POST /api/v1/knowledge/apply/configs/:uuid/knowledge
func (h *KnowledgeApplyHandler) AddKnowledge(c *gin.Context) {
	ctx := middleware.GetContext(c)

	configUUID := c.Param("uuid")
	if configUUID == "" {
		response.Error(c, errors.ErrInvalidParam)
		return
	}

	var req request.AddKnowledgeToConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Logger.ErrorfCtx(ctx, "解析请求体失败: %v", err)
		response.Error(c, errors.New(errors.ErrCodeInvalidParam, err.Error()))
		return
	}
	req.ConfigUUID = configUUID

	username, exists := c.Get(constants.AuthUsername)
	if !exists {
		log.Logger.ErrorfCtx(ctx, "获取用户名失败")
		response.Error(c, errors.ErrInvalidToken)
		return
	}

	isAdmin := middleware.GetClientIsAdmin(c)
	result, err := h.applyService.AddKnowledge(ctx, &req, username.(string), isAdmin)
	if err != nil {
		log.Logger.ErrorfCtx(ctx, "添加知识到配置失败: %v", err)
		response.Error(c, err)
		return
	}

	response.Success(c, &kbResp.ApplyResultResponse{
		ConfigUUID:    result.ConfigUUID,
		ConfigVersion: result.ConfigVersion,
		TotalCount:    result.TotalCount,
		SuccessCount:  result.SuccessCount,
		FailedCount:   result.FailedCount,
	})
}

// RemoveKnowledge 从配置中移除知识
// DELETE /api/v1/knowledge/apply/configs/:uuid/knowledge
func (h *KnowledgeApplyHandler) RemoveKnowledge(c *gin.Context) {
	ctx := middleware.GetContext(c)

	configUUID := c.Param("uuid")
	if configUUID == "" {
		response.Error(c, errors.ErrInvalidParam)
		return
	}

	var req request.RemoveKnowledgeFromConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Logger.ErrorfCtx(ctx, "解析请求体失败: %v", err)
		response.Error(c, errors.New(errors.ErrCodeInvalidParam, err.Error()))
		return
	}
	req.ConfigUUID = configUUID

	username, exists := c.Get(constants.AuthUsername)
	if !exists {
		log.Logger.ErrorfCtx(ctx, "获取用户名失败")
		response.Error(c, errors.ErrInvalidToken)
		return
	}

	isAdmin := middleware.GetClientIsAdmin(c)
	err := h.applyService.RemoveKnowledge(ctx, &req, username.(string), isAdmin)
	if err != nil {
		log.Logger.ErrorfCtx(ctx, "从配置移除知识失败: %v", err)
		response.Error(c, err)
		return
	}

	response.Success(c, nil)
}

// UpdateConfigStatus 更新配置状态
// PATCH /api/v1/knowledge/apply/configs/:uuid/status
func (h *KnowledgeApplyHandler) UpdateConfigStatus(c *gin.Context) {
	ctx := middleware.GetContext(c)

	configUUID := c.Param("uuid")
	if configUUID == "" {
		response.Error(c, errors.ErrInvalidParam)
		return
	}

	var req request.UpdateConfigStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Logger.ErrorfCtx(ctx, "解析请求体失败: %v", err)
		response.Error(c, errors.New(errors.ErrCodeInvalidParam, err.Error()))
		return
	}

	username, exists := c.Get(constants.AuthUsername)
	if !exists {
		log.Logger.ErrorfCtx(ctx, "获取用户名失败")
		response.Error(c, errors.ErrInvalidToken)
		return
	}

	isAdmin := middleware.GetClientIsAdmin(c)
	if err := h.applyService.UpdateConfigStatus(ctx, configUUID, req.Status, username.(string), isAdmin); err != nil {
		log.Logger.ErrorfCtx(ctx, "更新配置状态失败: %v", err)
		response.Error(c, err)
		return
	}

	response.Success(c, nil)
}

// DeleteConfig 删除应用配置
// DELETE /api/v1/knowledge/apply/configs/:uuid
func (h *KnowledgeApplyHandler) DeleteConfig(c *gin.Context) {
	ctx := middleware.GetContext(c)

	configUUID := c.Param("uuid")
	if configUUID == "" {
		response.Error(c, errors.ErrInvalidParam)
		return
	}

	username, exists := c.Get(constants.AuthUsername)
	if !exists {
		log.Logger.ErrorfCtx(ctx, "获取用户名失败")
		response.Error(c, errors.ErrInvalidToken)
		return
	}

	isAdmin := middleware.GetClientIsAdmin(c)
	if err := h.applyService.DeleteConfig(ctx, configUUID, username.(string), isAdmin); err != nil {
		log.Logger.ErrorfCtx(ctx, "删除配置失败: %v", err)
		response.Error(c, err)
		return
	}

	response.Success(c, nil)
}

// RetryPush 重试推送失败的知识
// POST /api/v1/knowledge/apply/configs/:uuid/retry
func (h *KnowledgeApplyHandler) RetryPush(c *gin.Context) {
	ctx := middleware.GetContext(c)

	configUUID := c.Param("uuid")
	if configUUID == "" {
		response.Error(c, errors.ErrInvalidParam)
		return
	}

	username, exists := c.Get(constants.AuthUsername)
	if !exists {
		log.Logger.ErrorfCtx(ctx, "获取用户名失败")
		response.Error(c, errors.ErrInvalidToken)
		return
	}

	result, err := h.applyService.RetryPush(ctx, configUUID, username.(string))
	if err != nil {
		log.Logger.ErrorfCtx(ctx, "重试推送失败: %v", err)
		response.Error(c, err)
		return
	}

	response.Success(c, &kbResp.RetryResultResponse{
		RetryCount:   result.RetryCount,
		SuccessCount: result.SuccessCount,
		FailedCount:  result.FailedCount,
	})
}

// convertApplyConfigToResponse 转换配置到响应格式
func convertApplyConfigToResponse(data *service.ApplyConfigWithKnowledge) *kbResp.ApplyConfigResponse {
	config := data.Config
	resp := &kbResp.ApplyConfigResponse{
		UUID:                    config.UUID,
		KnowledgebaseUUID:       config.KnowledgebaseUUID,
		KnowledgebaseName:       data.KnowledgebaseName,
		KnowledgebaseVisibility: data.KnowledgebaseVisibility,
		AgentCode:               config.AgentCode,
		AgentName:               config.AgentName,
		Scene:                   config.Scene,
		TargetType:              config.TargetType,
		TargetID:                config.TargetID,
		TargetName:              config.TargetName,
		Version:                 config.Version,
		Status:                  config.Status,
		Operator:                config.Operator,
		Remark:                  config.Remark,
		CreatedAt:               config.CreatedAt,
		UpdatedAt:               config.UpdatedAt,
	}

	// 反序列化 TargetExtra
	if config.TargetExtra != "" {
		var targetExtra map[string]any
		if err := json.Unmarshal([]byte(config.TargetExtra), &targetExtra); err == nil {
			resp.TargetExtra = targetExtra
		}
	}

	if data.ConfiguredEntries != nil {
		resp.ConfiguredEntries = make([]*kbResp.ApplyConfigEntryItem, len(data.ConfiguredEntries))
		for i, e := range data.ConfiguredEntries {
			resp.ConfiguredEntries[i] = &kbResp.ApplyConfigEntryItem{
				EntryType:        e.EntryType,
				KnowledgeUUID:    e.KnowledgeUUID,
				KnowledgeVersion: e.KnowledgeVersion,
				KnowledgeTitle:   e.KnowledgeTitle,
				DirectoryUUID:    e.DirectoryUUID,
				DirectoryName:    e.DirectoryName,
				Operator:         e.Operator,
				CreatedAt:        e.CreatedAt,
			}
		}
	}

	if data.KnowledgeList != nil {
		resp.KnowledgeList = make([]*kbResp.AppliedKnowledgeItem, len(data.KnowledgeList))
		for i, k := range data.KnowledgeList {
			resp.KnowledgeList[i] = &kbResp.AppliedKnowledgeItem{
				KnowledgeUUID:       k.KnowledgeUUID,
				KnowledgeVersion:    k.KnowledgeVersion,
				KnowledgeTitle:      k.KnowledgeTitle,
				KnowledgeContent:    k.KnowledgeContent,
				KnowledgeType:       k.KnowledgeType,
				TemplateName:        k.TemplateName,
				PushStatus:          k.PushStatus,
				Operator:            k.Operator,
				SourceEntryType:     k.SourceEntryType,
				SourceDirectoryUUID: k.SourceDirectoryUUID,
				SourceDirectoryName: k.SourceDirectoryName,
				CreatedAt:           k.CreatedAt,
			}
		}
	}

	return resp
}
