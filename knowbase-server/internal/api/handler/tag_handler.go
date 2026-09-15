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

// TagHandler 标签管理处理器
type TagHandler struct {
	tagService service.TagService
}

// NewTagHandler 创建标签管理处理器实例
func NewTagHandler() *TagHandler {
	ormer := database.GetOrm()

	kbRepo := repository.NewKnowledgebaseRepository(ormer)
	memberRepo := repository.NewKnowledgebaseMemberRepository(ormer)
	tagRepo := repository.NewKnowledgebaseTagRepository(ormer)
	tagRelRepo := repository.NewKnowledgeTagRelationRepository(ormer)
	logRepo := repository.NewKnowledgebaseOperationLogRepository(ormer)
	txManager := database.NewTransactionManager()

	tagService := service.NewTagService(kbRepo, memberRepo, tagRepo, tagRelRepo, logRepo, txManager)

	return &TagHandler{
		tagService: tagService,
	}
}

// CreateTag 创建标签
// POST /api/v1/knowledgebases/:kb_uuid/tags
func (h *TagHandler) CreateTag(c *gin.Context) {
	ctx := middleware.GetContext(c)

	kbUUID := c.Param("kb_uuid")
	if kbUUID == "" {
		response.Error(c, errors.ErrInvalidParam)
		return
	}

	var req request.CreateTagRequest
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

	log.Logger.InfofCtx(ctx, "用户 %s 请求在知识本 %s 创建标签: %s", username, kbUUID, req.Name)

	// 获取是否为Admin客户端
	isAdmin := middleware.GetClientIsAdmin(c)

	// 调用服务层
	tag, err := h.tagService.CreateTag(ctx, kbUUID, req.Name, req.Description, username.(string), isAdmin)
	if err != nil {
		log.Logger.ErrorfCtx(ctx, "创建标签失败: %v", err)
		response.Error(c, err)
		return
	}

	response.Success(c, toTagResponse(tag, 0))
}

// ListTags 查询标签列表
// GET /api/v1/knowledgebases/:kb_uuid/tags
func (h *TagHandler) ListTags(c *gin.Context) {
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

	// 调用服务层
	tags, countMap, total, err := h.tagService.ListTags(ctx, kbUUID, username.(string))
	if err != nil {
		log.Logger.ErrorfCtx(ctx, "查询标签列表失败: %v", err)
		response.Error(c, err)
		return
	}

	// 转换为响应格式
	items := make([]*kbResp.TagResponse, 0, len(tags))
	for _, tag := range tags {
		items = append(items, toTagResponse(tag, countMap[tag.UUID]))
	}

	resp := &kbResp.ListTagsResponse{
		Total: total,
		Items: items,
	}

	response.Success(c, resp)
}

// UpdateTag 更新标签
// PUT /api/v1/knowledgebases/:kb_uuid/tags/:uuid
func (h *TagHandler) UpdateTag(c *gin.Context) {
	ctx := middleware.GetContext(c)

	kbUUID := c.Param("kb_uuid")
	tagUUID := c.Param("uuid")
	if kbUUID == "" || tagUUID == "" {
		response.Error(c, errors.ErrInvalidParam)
		return
	}

	var req request.UpdateTagRequest
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

	log.Logger.InfofCtx(ctx, "用户 %s 请求更新标签 %s", username, tagUUID)

	// 获取是否为Admin客户端
	isAdmin := middleware.GetClientIsAdmin(c)

	// 调用服务层
	tag, err := h.tagService.UpdateTag(ctx, kbUUID, tagUUID, req.Name, req.Description, username.(string), isAdmin)
	if err != nil {
		log.Logger.ErrorfCtx(ctx, "更新标签失败: %v", err)
		response.Error(c, err)
		return
	}

	response.Success(c, toTagResponse(tag, 0))
}

// DeleteTag 删除标签
// DELETE /api/v1/knowledgebases/:kb_uuid/tags/:uuid
func (h *TagHandler) DeleteTag(c *gin.Context) {
	ctx := middleware.GetContext(c)

	kbUUID := c.Param("kb_uuid")
	tagUUID := c.Param("uuid")
	if kbUUID == "" || tagUUID == "" {
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

	log.Logger.InfofCtx(ctx, "用户 %s 请求删除标签 %s", username, tagUUID)

	// 获取是否为Admin客户端
	isAdmin := middleware.GetClientIsAdmin(c)

	// 调用服务层
	err := h.tagService.DeleteTag(ctx, kbUUID, tagUUID, username.(string), isAdmin)
	if err != nil {
		log.Logger.ErrorfCtx(ctx, "删除标签失败: %v", err)
		response.Error(c, err)
		return
	}

	response.Success(c, nil)
}

// toTagResponse 将领域模型转换为响应模型
func toTagResponse(tag *domain.KnowledgebaseTag, knowledgeCount int64) *kbResp.TagResponse {
	if tag == nil {
		return nil
	}

	return &kbResp.TagResponse{
		UUID:              tag.UUID,
		KnowledgebaseUUID: tag.KnowledgebaseUUID,
		Name:              tag.Name,
		Description:       tag.Description,
		CreatedBy:         tag.CreatedBy,
		KnowledgeCount:    knowledgeCount,
		CreatedAt:         tag.CreatedAt,
		UpdatedAt:         tag.UpdatedAt,
	}
}
