// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package handler

import (
	"strconv"

	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/api/middleware"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/api/request"
	kbResp "icode.baidu.com/baidu/devops-ai/knowbase-server/internal/api/response"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/infrastructure/database"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/infrastructure/infoflow"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/infrastructure/uuap"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/repository"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/service"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/errors"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/log"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/response"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/types/constants"

	"github.com/gin-gonic/gin"
)

// KnowledgebaseHandler 知识本处理器
type KnowledgebaseHandler struct {
	service service.KnowledgebaseService
}

// NewKnowledgebaseHandler 创建知识本处理器实例
func NewKnowledgebaseHandler() *KnowledgebaseHandler {
	ormer := database.GetOrm()

	kbRepo := repository.NewKnowledgebaseRepository(ormer)
	memberRepo := repository.NewKnowledgebaseMemberRepository(ormer)
	tagRepo := repository.NewKnowledgebaseTagRepository(ormer)
	knowRepo := repository.NewKnowledgeRepository(ormer)
	operationRepo := repository.NewKnowledgebaseOperationLogRepository(ormer)
	dirRepo := repository.NewKnowledgebaseDirectoryRepository(ormer)

	uuapClient, _ := uuap.GetClient()
	infoFlowClient := infoflow.GetClient()
	svc := service.NewKnowledgebaseService(kbRepo, memberRepo, tagRepo, knowRepo, operationRepo, dirRepo, uuapClient, infoFlowClient)

	return &KnowledgebaseHandler{
		service: svc,
	}
}

// CreateKnowledgebase 创建知识本
func (h *KnowledgebaseHandler) CreateKnowledgebase(c *gin.Context) {
	ctx := middleware.GetContext(c)

	var req request.CreateKnowledgebaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Logger.ErrorfCtx(ctx, "解析请求体失败: %v", err)
		response.Error(c, errors.ErrInvalidRequest)
		return
	}

	// 获取当前用户
	username, exists := c.Get(constants.AuthUsername)
	if !exists {
		log.Logger.ErrorfCtx(ctx, "获取用户名失败")
		response.Error(c, errors.ErrInvalidToken)
		return
	}

	// 安全的类型断言
	usernameStr, ok := username.(string)
	if !ok {
		log.Logger.ErrorfCtx(ctx, "用户名类型错误: type=%T, value=%v", username, username)
		response.Error(c, errors.ErrInvalidToken)
		return
	}

	log.Logger.InfofCtx(ctx, "用户 %s 请求创建知识本: %s", usernameStr, req.Name)

	// 调用服务层
	kb, memberCount, err := h.service.CreateKnowledgebase(ctx, &req, usernameStr)
	if err != nil {
		log.Logger.ErrorfCtx(ctx, "创建知识本失败: %v", err)
		response.Error(c, err)
		return
	}

	// 创建知识本后，标签数量为 0，知识数量为 0
	response.Success(c, kbResp.NewKnowledgebaseResponse(kb, memberCount, 0, 0))
}

// GetKnowledgebase 获取知识本详情
func (h *KnowledgebaseHandler) GetKnowledgebase(c *gin.Context) {
	ctx := middleware.GetContext(c)

	uuid := c.Param("kb_uuid")
	if uuid == "" {
		response.Error(c, errors.ErrInvalidParam)
		return
	}

	// 获取当前用户
	username, exists := c.Get(constants.AuthUsername)
	if !exists {
		log.Logger.ErrorfCtx(ctx, "获取用户名失败")
		response.Error(c, errors.ErrInvalidToken)
		return
	}

	// 安全的类型断言
	usernameStr, ok := username.(string)
	if !ok {
		log.Logger.ErrorfCtx(ctx, "用户名类型错误: type=%T, value=%v", username, username)
		response.Error(c, errors.ErrInvalidToken)
		return
	}

	kb, memberCount, tagCount, knowledgeCount, err := h.service.GetKnowledgebase(ctx, uuid, usernameStr)
	if err != nil {
		log.Logger.ErrorfCtx(ctx, "获取知识本失败: %v", err)
		response.Error(c, err)
		return
	}

	response.Success(c, kbResp.NewKnowledgebaseResponse(kb, memberCount, tagCount, knowledgeCount))
}

// GetKnowledgebaseByName 根据名称获取知识本
func (h *KnowledgebaseHandler) GetKnowledgebaseByName(c *gin.Context) {
	ctx := middleware.GetContext(c)

	name := c.Param("name")
	if name == "" {
		response.Error(c, errors.ErrInvalidParam)
		return
	}

	// 获取当前用户
	username, exists := c.Get(constants.AuthUsername)
	if !exists {
		log.Logger.ErrorfCtx(ctx, "获取用户名失败")
		response.Error(c, errors.ErrInvalidToken)
		return
	}

	// 安全的类型断言
	usernameStr, ok := username.(string)
	if !ok {
		log.Logger.ErrorfCtx(ctx, "用户名类型错误: type=%T, value=%v", username, username)
		response.Error(c, errors.ErrInvalidToken)
		return
	}

	kb, memberCount, tagCount, knowledgeCount, err := h.service.GetKnowledgebaseByName(ctx, name, usernameStr)
	if err != nil {
		log.Logger.ErrorfCtx(ctx, "获取知识本失败: %v", err)
		response.Error(c, err)
		return
	}

	response.Success(c, kbResp.NewKnowledgebaseResponse(kb, memberCount, tagCount, knowledgeCount))
}

// ListKnowledgebases 查询知识本列表
func (h *KnowledgebaseHandler) ListKnowledgebases(c *gin.Context) {
	ctx := middleware.GetContext(c)

	var req request.ListKnowledgebasesRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		log.Logger.ErrorfCtx(ctx, "解析查询参数失败: %v", err)
		response.Error(c, errors.ErrInvalidRequest)
		return
	}

	// 获取当前用户
	username, exists := c.Get(constants.AuthUsername)
	if !exists {
		log.Logger.ErrorfCtx(ctx, "获取用户名失败")
		response.Error(c, errors.ErrInvalidToken)
		return
	}

	// 安全的类型断言
	usernameStr, ok := username.(string)
	if !ok {
		log.Logger.ErrorfCtx(ctx, "用户名类型错误: type=%T, value=%v", username, username)
		response.Error(c, errors.ErrInvalidToken)
		return
	}

	// 获取认证类型
	authType, _ := c.Get(constants.AuthType)

	// 如果是 OpenAPI 调用，强制只返回 owner 为该 OpenAPI 管理员的知识本
	if authType == constants.AuthTypeOpenAPI {
		req.Owner = usernameStr
	}

	log.Logger.InfofCtx(ctx, "查询知识本列表: keyword=%s, visibility=%s, owner=%s, page=%d, pageSize=%d",
		req.Keyword, req.Visibility, req.Owner, req.Page, req.PageSize)

	kbsWithRole, total, err := h.service.ListKnowledgebases(ctx, &req, usernameStr)
	if err != nil {
		log.Logger.ErrorfCtx(ctx, "查询知识本列表失败: %v", err)
		response.Error(c, err)
		return
	}

	// 转换为响应结构
	list := make([]*kbResp.KnowledgebaseListItemResponse, 0, len(kbsWithRole))
	for _, item := range kbsWithRole {
		list = append(list, kbResp.NewKnowledgebaseListItemResponse(item.KB, item.Role, item.MemberCount, item.TagCount, item.KnowledgeCount))
	}

	resp := &kbResp.ListKnowledgebasesResponse{
		List:     list,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}
	response.Success(c, resp)
}

// ListMyKnowledgebases 查询用户有权限的知识本列表
func (h *KnowledgebaseHandler) ListMyKnowledgebases(c *gin.Context) {
	ctx := middleware.GetContext(c)

	var req request.ListMyKnowledgebasesRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		log.Logger.ErrorfCtx(ctx, "解析查询参数失败: %v", err)
		response.Error(c, errors.ErrInvalidRequest)
		return
	}

	// 设置默认值
	if req.Page == 0 {
		req.Page = 1
	}
	if req.PageSize == 0 {
		req.PageSize = 20 // 默认20条
	}

	// 获取当前用户
	username, exists := c.Get(constants.AuthUsername)
	if !exists {
		log.Logger.ErrorfCtx(ctx, "获取用户名失败")
		response.Error(c, errors.ErrInvalidToken)
		return
	}

	// 安全的类型断言
	usernameStr, ok := username.(string)
	if !ok {
		log.Logger.ErrorfCtx(ctx, "用户名类型错误: type=%T, value=%v", username, username)
		response.Error(c, errors.ErrInvalidToken)
		return
	}

	// 获取认证类型
	authType, _ := c.Get(constants.AuthType)
	isOpenAPI := authType == constants.AuthTypeOpenAPI

	log.Logger.InfofCtx(ctx, "查询用户有权限的知识本列表: username=%s, authType=%v, isOpenAPI=%v, keyword=%s, visibility=%s, role=%s, page=%d, pageSize=%d",
		usernameStr, authType, isOpenAPI, req.Keyword, req.Visibility, req.Role, req.Page, req.PageSize)

	kbsWithRole, total, err := h.service.ListMyKnowledgebases(ctx, &req, usernameStr, isOpenAPI)
	if err != nil {
		log.Logger.ErrorfCtx(ctx, "查询用户有权限的知识本列表失败: %v", err)
		response.Error(c, err)
		return
	}

	// 转换为响应结构
	list := make([]*kbResp.KnowledgebaseListItemResponse, 0, len(kbsWithRole))
	for _, item := range kbsWithRole {
		list = append(list, kbResp.NewKnowledgebaseListItemResponse(item.KB, item.Role, item.MemberCount, item.TagCount, item.KnowledgeCount))
	}

	resp := &kbResp.ListKnowledgebasesResponse{
		List:     list,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}
	response.Success(c, resp)
}

// UpdateKnowledgebase 更新知识本
func (h *KnowledgebaseHandler) UpdateKnowledgebase(c *gin.Context) {
	ctx := middleware.GetContext(c)

	uuid := c.Param("kb_uuid")
	if uuid == "" {
		response.Error(c, errors.ErrInvalidParam)
		return
	}

	var req request.UpdateKnowledgebaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Logger.ErrorfCtx(ctx, "解析请求体失败: %v", err)
		response.Error(c, errors.ErrInvalidRequest)
		return
	}

	// 获取当前用户
	username, exists := c.Get(constants.AuthUsername)
	if !exists {
		log.Logger.ErrorfCtx(ctx, "获取用户名失败")
		response.Error(c, errors.ErrInvalidToken)
		return
	}

	// 安全的类型断言
	usernameStr, ok := username.(string)
	if !ok {
		log.Logger.ErrorfCtx(ctx, "用户名类型错误: type=%T, value=%v", username, username)
		response.Error(c, errors.ErrInvalidToken)
		return
	}

	log.Logger.InfofCtx(ctx, "用户 %s 请求更新知识本: %s", usernameStr, uuid)

	isAdmin := middleware.GetClientIsAdmin(c)
	kb, memberCount, tagCount, knowledgeCount, err := h.service.UpdateKnowledgebase(ctx, uuid, &req, usernameStr, isAdmin)
	if err != nil {
		log.Logger.ErrorfCtx(ctx, "更新知识本失败: %v", err)
		response.Error(c, err)
		return
	}

	response.Success(c, kbResp.NewKnowledgebaseResponse(kb, memberCount, tagCount, knowledgeCount))
}

// DeleteKnowledgebase 删除知识本
func (h *KnowledgebaseHandler) DeleteKnowledgebase(c *gin.Context) {
	ctx := middleware.GetContext(c)

	uuid := c.Param("kb_uuid")
	if uuid == "" {
		response.Error(c, errors.ErrInvalidParam)
		return
	}

	// 获取当前用户
	username, exists := c.Get(constants.AuthUsername)
	if !exists {
		log.Logger.ErrorfCtx(ctx, "获取用户名失败")
		response.Error(c, errors.ErrInvalidToken)
		return
	}

	// 安全的类型断言
	usernameStr, ok := username.(string)
	if !ok {
		log.Logger.ErrorfCtx(ctx, "用户名类型错误: type=%T, value=%v", username, username)
		response.Error(c, errors.ErrInvalidToken)
		return
	}

	log.Logger.InfofCtx(ctx, "用户 %s 请求删除知识本: %s", usernameStr, uuid)

	// 只能用户自己删除
	if err := h.service.DeleteKnowledgebase(ctx, uuid, usernameStr, false); err != nil {
		log.Logger.ErrorfCtx(ctx, "删除知识本失败: %v", err)
		response.Error(c, err)
		return
	}

	response.Success(c, nil)
}

// GetGroupKnowledgebases 查询群聊有权限的知识本列表
func (h *KnowledgebaseHandler) GetGroupKnowledgebases(c *gin.Context) {
	var req request.GetGroupKnowledgebasesRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		log.Logger.Errorf("解析请求参数失败: %v", err)
		response.Error(c, errors.ErrInvalidRequest)
		return
	}

	// 设置分页默认值
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}
	if req.PageSize > 100 {
		req.PageSize = 100
	}

	groupIDStr := strconv.FormatInt(req.GroupID, 10)
	ctx := middleware.GetContext(c)
	log.Logger.InfofCtx(ctx, "查询群聊有权限的知识本列表: groupId=%s, keyword=%s, role=%s, page=%d, pageSize=%d",
		groupIDStr, req.Keyword, req.Role, req.Page, req.PageSize)

	kbsWithRole, total, err := h.service.ListGroupKnowledgebases(ctx, groupIDStr, req.Keyword, req.Role, req.Page, req.PageSize)
	if err != nil {
		log.Logger.ErrorfCtx(ctx, "查询群聊有权限的知识本列表失败: %v", err)
		response.Error(c, err)
		return
	}

	// 转换为响应结构
	list := make([]*kbResp.KnowledgebaseListItemResponse, 0, len(kbsWithRole))
	for _, item := range kbsWithRole {
		list = append(list, kbResp.NewKnowledgebaseListItemResponse(item.KB, item.Role, item.MemberCount, item.TagCount, item.KnowledgeCount))
	}

	resp := &kbResp.ListKnowledgebasesResponse{
		List:     list,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}
	response.Success(c, resp)
}
