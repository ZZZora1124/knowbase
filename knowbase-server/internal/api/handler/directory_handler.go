// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package handler

import (
	"github.com/gin-gonic/gin"

	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/api/middleware"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/api/request"
	kbResp "icode.baidu.com/baidu/devops-ai/knowbase-server/internal/api/response"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/infrastructure/database"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/repository"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/service"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/errors"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/log"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/response"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/types/constants"
)

// DirectoryHandler 目录处理器
type DirectoryHandler struct {
	service service.DirectoryService
}

// NewDirectoryHandler 创建目录处理器实例
func NewDirectoryHandler() *DirectoryHandler {
	ormer := database.GetOrm()

	kbRepo := repository.NewKnowledgebaseRepository(ormer)
	memberRepo := repository.NewKnowledgebaseMemberRepository(ormer)
	dirRepo := repository.NewKnowledgebaseDirectoryRepository(ormer)
	knowRepo := repository.NewKnowledgeRepository(ormer)
	operationRepo := repository.NewKnowledgebaseOperationLogRepository(ormer)

	// 创建权限服务
	permissionService := service.NewPermissionService(memberRepo)

	svc := service.NewDirectoryService(kbRepo, dirRepo, knowRepo, operationRepo, permissionService)

	return &DirectoryHandler{
		service: svc,
	}
}

// CreateDirectory 创建目录
// POST /api/v1/knowledgebases/:kb_uuid/directories
func (h *DirectoryHandler) CreateDirectory(c *gin.Context) {
	ctx := middleware.GetContext(c)

	kbUUID := c.Param("kb_uuid")
	if kbUUID == "" {
		response.Error(c, errors.ErrInvalidParam)
		return
	}

	var req request.CreateDirectoryRequest
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

	// 获取是否为Admin客户端
	isAdmin := middleware.GetClientIsAdmin(c)

	log.Logger.InfofCtx(ctx, "用户 %s 请求创建目录: %s", username, req.Name)

	// 调用服务层（传入 parentUUID）
	dir, err := h.service.CreateDirectory(ctx, kbUUID, req.Name, req.ParentUUID, username.(string), isAdmin)
	if err != nil {
		log.Logger.ErrorfCtx(ctx, "创建目录失败: %v", err)
		response.Error(c, err)
		return
	}

	response.Success(c, kbResp.NewDirectoryResponse(dir, 0))
}

// GetDirectory 获取目录详情
// GET /api/v1/knowledgebases/:kb_uuid/directories/:uuid
func (h *DirectoryHandler) GetDirectory(c *gin.Context) {
	ctx := middleware.GetContext(c)

	kbUUID := c.Param("kb_uuid")
	uuid := c.Param("uuid")
	if kbUUID == "" || uuid == "" {
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

	// 获取是否为Admin客户端
	isAdmin := middleware.GetClientIsAdmin(c)

	result, err := h.service.GetDirectory(ctx, kbUUID, uuid, username.(string), isAdmin)
	if err != nil {
		log.Logger.ErrorfCtx(ctx, "获取目录失败: %v", err)
		response.Error(c, err)
		return
	}

	response.Success(c, kbResp.NewDirectoryResponse(result.Directory, result.KnowledgeCount))
}

// ListDirectories 查询目录列表
// GET /api/v1/knowledgebases/:kb_uuid/directories
func (h *DirectoryHandler) ListDirectories(c *gin.Context) {
	ctx := middleware.GetContext(c)

	kbUUID := c.Param("kb_uuid")
	if kbUUID == "" {
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

	// 获取是否为Admin客户端
	isAdmin := middleware.GetClientIsAdmin(c)

	log.Logger.InfofCtx(ctx, "查询目录列表: kbUUID=%s", kbUUID)

	items, total, err := h.service.ListDirectories(ctx, kbUUID, username.(string), isAdmin)
	if err != nil {
		log.Logger.ErrorfCtx(ctx, "查询目录列表失败: %v", err)
		response.Error(c, err)
		return
	}

	// 转换为响应结构
	list := make([]*kbResp.DirectoryResponse, 0, len(items))
	for _, item := range items {
		list = append(list, kbResp.NewDirectoryResponse(item.Directory, item.KnowledgeCount))
	}

	resp := &kbResp.ListDirectoriesResponse{
		Items: list,
		Total: total,
	}
	response.Success(c, resp)
}

// UpdateDirectory 更新目录
// PUT /api/v1/knowledgebases/:kb_uuid/directories/:uuid
func (h *DirectoryHandler) UpdateDirectory(c *gin.Context) {
	ctx := middleware.GetContext(c)

	kbUUID := c.Param("kb_uuid")
	uuid := c.Param("uuid")
	if kbUUID == "" || uuid == "" {
		response.Error(c, errors.ErrInvalidParam)
		return
	}

	var req request.UpdateDirectoryRequest
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

	// 获取是否为Admin客户端
	isAdmin := middleware.GetClientIsAdmin(c)

	log.Logger.InfofCtx(ctx, "用户 %s 请求更新目录: %s", username, uuid)

	result, err := h.service.UpdateDirectory(ctx, kbUUID, uuid, req.Name, username.(string), isAdmin)
	if err != nil {
		log.Logger.ErrorfCtx(ctx, "更新目录失败: %v", err)
		response.Error(c, err)
		return
	}

	response.Success(c, kbResp.NewDirectoryResponse(result.Directory, result.KnowledgeCount))
}

// DeleteDirectory 删除目录
// DELETE /api/v1/knowledgebases/:kb_uuid/directories/:uuid
func (h *DirectoryHandler) DeleteDirectory(c *gin.Context) {
	ctx := middleware.GetContext(c)

	kbUUID := c.Param("kb_uuid")
	uuid := c.Param("uuid")
	if kbUUID == "" || uuid == "" {
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

	// 获取是否为Admin客户端
	isAdmin := middleware.GetClientIsAdmin(c)

	log.Logger.InfofCtx(ctx, "用户 %s 请求删除目录: %s", username, uuid)

	err := h.service.DeleteDirectory(ctx, kbUUID, uuid, username.(string), isAdmin)
	if err != nil {
		log.Logger.ErrorfCtx(ctx, "删除目录失败: %v", err)
		response.Error(c, err)
		return
	}

	response.Success(c, nil)
}

// GetDirectoryTree 获取目录树
// GET /api/v1/knowledgebases/:kb_uuid/directories/tree
func (h *DirectoryHandler) GetDirectoryTree(c *gin.Context) {
	ctx := middleware.GetContext(c)

	kbUUID := c.Param("kb_uuid")
	if kbUUID == "" {
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

	// 获取是否为Admin客户端
	isAdmin := middleware.GetClientIsAdmin(c)

	log.Logger.InfofCtx(ctx, "查询目录树: kbUUID=%s", kbUUID)

	roots, total, err := h.service.GetDirectoryTree(ctx, kbUUID, username.(string), isAdmin)
	if err != nil {
		log.Logger.ErrorfCtx(ctx, "查询目录树失败: %v", err)
		response.Error(c, err)
		return
	}

	// 递归转换为响应结构
	respItems := convertTreeNodes(roots)
	resp := &kbResp.DirectoryTreeResponse{
		Items: respItems,
		Total: total,
	}
	response.Success(c, resp)
}

// convertTreeNodes 递归将服务层树节点转为响应树节点
func convertTreeNodes(nodes []*service.DirectoryTreeNode) []*kbResp.DirectoryTreeNode {
	result := make([]*kbResp.DirectoryTreeNode, 0, len(nodes))
	for _, n := range nodes {
		node := kbResp.NewDirectoryTreeNode(n.Directory, n.KnowledgeCount)
		if len(n.Children) > 0 {
			node.Children = convertTreeNodes(n.Children)
		}
		result = append(result, node)
	}
	return result
}

// MoveDirectory 移动目录
// POST /api/v1/knowledgebases/:kb_uuid/directories/:uuid/move
func (h *DirectoryHandler) MoveDirectory(c *gin.Context) {
	ctx := middleware.GetContext(c)

	kbUUID := c.Param("kb_uuid")
	uuid := c.Param("uuid")
	if kbUUID == "" || uuid == "" {
		response.Error(c, errors.ErrInvalidParam)
		return
	}

	var req request.MoveDirectoryRequest
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

	// 获取是否为Admin客户端
	isAdmin := middleware.GetClientIsAdmin(c)

	log.Logger.InfofCtx(ctx, "用户 %s 请求移动目录: %s", username, uuid)

	dir, err := h.service.MoveDirectory(ctx, kbUUID, uuid, req.ParentUUID, username.(string), isAdmin)
	if err != nil {
		log.Logger.ErrorfCtx(ctx, "移动目录失败: %v", err)
		response.Error(c, err)
		return
	}

	response.Success(c, kbResp.NewDirectoryResponse(dir, 0))
}
