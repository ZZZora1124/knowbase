// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package handler

import (
	"github.com/gin-gonic/gin"

	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/api/middleware"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/api/request"
	kbResp "icode.baidu.com/baidu/devops-ai/knowbase-server/internal/api/response"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/domain"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/infrastructure/database"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/repository"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/service"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/errors"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/log"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/response"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/types/constants"
)

// KnowledgeVersionHandler 知识版本处理器
type KnowledgeVersionHandler struct {
	versionService service.KnowledgeVersionService
}

// NewKnowledgeVersionHandler 创建知识版本处理器实例
func NewKnowledgeVersionHandler() *KnowledgeVersionHandler {
	ormer := database.GetOrm()

	kbRepo := repository.NewKnowledgebaseRepository(ormer)
	memberRepo := repository.NewKnowledgebaseMemberRepository(ormer)
	dirRepo := repository.NewKnowledgebaseDirectoryRepository(ormer)
	knowRepo := repository.NewKnowledgeRepository(ormer)
	versionRepo := repository.NewKnowledgeVersionRepository(ormer)
	tagRelRepo := repository.NewKnowledgeTagRelationRepository(ormer)

	permissionService := service.NewPermissionService(memberRepo)

	svc := service.NewKnowledgeVersionService(
		kbRepo, dirRepo, knowRepo, versionRepo, tagRelRepo,
		permissionService,
	)

	return &KnowledgeVersionHandler{
		versionService: svc,
	}
}

// ListVersions 获取知识版本历史
// GET /api/v1/knowledge/:uuid/versions
func (h *KnowledgeVersionHandler) ListVersions(c *gin.Context) {
	ctx := middleware.GetContext(c)

	knowledgeUUID := c.Param("uuid")
	if knowledgeUUID == "" {
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
	versions, err := h.versionService.ListVersions(ctx, knowledgeUUID, username.(string), isAdmin)
	if err != nil {
		log.Logger.ErrorfCtx(ctx, "获取版本列表失败: %v", err)
		response.Error(c, err)
		return
	}

	// 转换为响应格式
	items := make([]*kbResp.KnowledgeVersionItem, len(versions))
	for i, v := range versions {
		items[i] = &kbResp.KnowledgeVersionItem{
			UUID:      v.UUID,
			Version:   v.Version,
			Title:     v.Title,
			Operator:  v.Operator,
			Operation: v.Operation,
			Remark:    v.Remark,
			CreatedAt: v.CreatedAt,
		}
	}

	response.Success(c, &kbResp.KnowledgeVersionListResponse{Versions: items})
}

// BatchListVersions 批量获取知识版本列表
// POST /api/v1/knowledge/versions
func (h *KnowledgeVersionHandler) BatchListVersions(c *gin.Context) {
	ctx := middleware.GetContext(c)

	var req request.BatchListVersionsRequest
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
	versionsMap, err := h.versionService.BatchListVersions(ctx, req.KnowledgeUUIDs, username.(string), isAdmin)
	if err != nil {
		log.Logger.ErrorfCtx(ctx, "批量获取版本列表失败: %v", err)
		response.Error(c, err)
		return
	}

	// 转换为响应格式
	data := make([]*kbResp.KnowledgeWithVersions, 0, len(versionsMap))
	for knowledgeUUID, versions := range versionsMap {
		items := make([]*kbResp.KnowledgeVersionItem, len(versions))
		for i, v := range versions {
			items[i] = &kbResp.KnowledgeVersionItem{
				UUID:      v.UUID,
				Version:   v.Version,
				Title:     v.Title,
				Operator:  v.Operator,
				Operation: v.Operation,
				Remark:    v.Remark,
				CreatedAt: v.CreatedAt,
			}
		}
		data = append(data, &kbResp.KnowledgeWithVersions{
			KnowledgeUUID: knowledgeUUID,
			Versions:      items,
		})
	}

	response.Success(c, &kbResp.BatchVersionsResponse{Data: data})
}

// GetVersion 获取指定版本详情
// GET /api/v1/knowledge/:uuid/versions/:version
func (h *KnowledgeVersionHandler) GetVersion(c *gin.Context) {
	ctx := middleware.GetContext(c)

	knowledgeUUID := c.Param("uuid")
	version := c.Param("version")
	if knowledgeUUID == "" || version == "" {
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
	ver, err := h.versionService.GetVersion(ctx, knowledgeUUID, version, username.(string), isAdmin)
	if err != nil {
		log.Logger.ErrorfCtx(ctx, "获取版本详情失败: %v", err)
		response.Error(c, err)
		return
	}

	response.Success(c, convertVersionToResponse(ver))
}

// RollbackToVersion 回滚到指定版本
// POST /api/v1/knowledge/:uuid/rollback
func (h *KnowledgeVersionHandler) RollbackToVersion(c *gin.Context) {
	ctx := middleware.GetContext(c)

	knowledgeUUID := c.Param("uuid")
	if knowledgeUUID == "" {
		response.Error(c, errors.ErrInvalidParam)
		return
	}

	var req request.RollbackVersionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Logger.ErrorfCtx(ctx, "解析请求体失败: %v", err)
		response.Error(c, errors.New(errors.ErrCodeInvalidParam, err.Error()))
		return
	}
	req.KnowledgeUUID = knowledgeUUID

	username, exists := c.Get(constants.AuthUsername)
	if !exists {
		log.Logger.ErrorfCtx(ctx, "获取用户名失败")
		response.Error(c, errors.ErrInvalidToken)
		return
	}

	isAdmin := middleware.GetClientIsAdmin(c)
	knowledge, err := h.versionService.RollbackToVersion(ctx, &req, username.(string), isAdmin)
	if err != nil {
		log.Logger.ErrorfCtx(ctx, "回滚版本失败: %v", err)
		response.Error(c, err)
		return
	}

	response.Success(c, convertKnowledgeToResponse(knowledge))
}

// BatchGetWithVersion 批量查询知识（带版本号）
// POST /api/v1/knowledge/batch-with-version
func (h *KnowledgeVersionHandler) BatchGetWithVersion(c *gin.Context) {
	ctx := middleware.GetContext(c)

	var req request.BatchGetKnowledgeWithVersionRequest
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
	result, err := h.versionService.GetKnowledgeBatchWithVersion(ctx, &req, username.(string), isAdmin)
	if err != nil {
		log.Logger.ErrorfCtx(ctx, "批量获取知识版本失败: %v", err)
		response.Error(c, err)
		return
	}

	// 转换为响应格式
	items := make([]*kbResp.KnowledgeWithVersionItem, len(result))
	for i, item := range result {
		tags := make([]*kbResp.TagItem, len(item.Tags))
		for j, tag := range item.Tags {
			tags[j] = &kbResp.TagItem{
				UUID: tag.UUID,
				Name: tag.Name,
			}
		}

		items[i] = &kbResp.KnowledgeWithVersionItem{
			Knowledge: &kbResp.KnowledgeBasicInfo{
				UUID:           item.Knowledge.UUID,
				Title:          item.Knowledge.Title,
				KnowledgeType:  item.Knowledge.KnowledgeType,
				TemplateName:   item.Knowledge.TemplateName,
				CurrentVersion: item.Knowledge.CurrentVersion,
				DirectoryUUID:  item.Knowledge.DirectoryUUID,
				DirectoryName:  item.DirectoryName,
			},
			Version: convertVersionToResponse(item.Version),
			Tags:    tags,
		}
	}

	response.Success(c, &kbResp.BatchKnowledgeWithVersionResponse{List: items})
}

// convertVersionToResponse 转换版本到响应格式
func convertVersionToResponse(v *domain.KnowledgeVersion) *kbResp.KnowledgeVersionDetailResponse {
	return &kbResp.KnowledgeVersionDetailResponse{
		UUID:          v.UUID,
		KnowledgeUUID: v.KnowledgeUUID,
		Version:       v.Version,
		Title:         v.Title,
		Content:       v.Content,
		FilePath:      v.FilePath,
		FileSize:      v.FileSize,
		FileFormat:    v.FileFormat,
		Metadata:      v.Metadata,
		Operator:      v.Operator,
		Operation:     v.Operation,
		Remark:        v.Remark,
		CreatedAt:     v.CreatedAt,
	}
}

// convertKnowledgeToResponse 转换知识到响应格式
func convertKnowledgeToResponse(k *domain.Knowledge) map[string]interface{} {
	return map[string]interface{}{
		"uuid":              k.UUID,
		"knowledgebaseUuid": k.KnowledgebaseUUID,
		"title":             k.Title,
		"currentVersion":    k.CurrentVersion,
		"content":           k.Content,
		"createdAt":         k.CreatedAt,
		"updatedAt":         k.UpdatedAt,
	}
}
