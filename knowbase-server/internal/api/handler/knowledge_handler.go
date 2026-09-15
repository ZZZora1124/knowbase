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
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/domain"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/domain/handler"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/infrastructure/database"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/infrastructure/storage"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/repository"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/service"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/errors"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/log"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/response"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/types/constants"
)

// KnowledgeHandler 知识处理器
type KnowledgeHandler struct {
	knowledgeService service.KnowledgeService
}

// isOpenAPIOrHeaderRequest 判断是否为需要追加知识列表的认证类型（OpenAPI 或 Header/actoken）
func (h *KnowledgeHandler) isOpenAPIOrHeaderRequest(c *gin.Context) bool {
	authType := middleware.GetAuthType(c)
	return authType == constants.AuthTypeOpenAPI || authType == constants.AuthTypeHeader
}

// NewKnowledgeHandler 创建知识处理器实例
func NewKnowledgeHandler(bosClient *storage.BOSClient) *KnowledgeHandler {
	ormer := database.GetOrm()

	kbRepo := repository.NewKnowledgebaseRepository(ormer)
	memberRepo := repository.NewKnowledgebaseMemberRepository(ormer)
	dirRepo := repository.NewKnowledgebaseDirectoryRepository(ormer)
	knowRepo := repository.NewKnowledgeRepository(ormer)
	tagRepo := repository.NewKnowledgebaseTagRepository(ormer)
	tagRelRepo := repository.NewKnowledgeTagRelationRepository(ormer)
	logRepo := repository.NewKnowledgebaseOperationLogRepository(ormer)
	versionRepo := repository.NewKnowledgeVersionRepository(ormer)
	handlerFactory := handler.NewHandlerFactory(bosClient)

	// 创建权限服务
	permissionService := service.NewPermissionService(memberRepo)

	svc := service.NewKnowledgeService(
		kbRepo, dirRepo, knowRepo,
		tagRepo, tagRelRepo, logRepo, versionRepo, handlerFactory,
		permissionService,
		bosClient,
	)

	return &KnowledgeHandler{
		knowledgeService: svc,
	}
}

// CreateKnowledge 创建知识
// POST /api/v1/knowledgebases/:kb_uuid/knowledge
func (h *KnowledgeHandler) CreateKnowledge(c *gin.Context) {
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

	// 构建 CreateKnowledgeRequest
	req := &request.CreateKnowledgeRequest{
		KnowledgebaseUUID: kbUUID,
		Title:             c.PostForm("title"),
		KnowledgeType:     c.PostForm("knowledgeType"),
		TemplateName:      c.PostForm("templateName"),
		KnowledgeSource:   c.PostForm("knowledgeSource"),
		Content:           c.PostForm("content"),
	}

	// 处理 directoryUuid
	if dirUUID := c.PostForm("directoryUuid"); dirUUID != "" && dirUUID != "/" {
		req.DirectoryUUID = &dirUUID
	}

	// 处理文件上传
	if file, err := c.FormFile("file"); err == nil {
		req.File = file
	}

	// 处理 metadata (JSON字符串)
	if metadataStr := c.PostForm("metadata"); metadataStr != "" {
		var metadata map[string]interface{}
		if err := json.Unmarshal([]byte(metadataStr), &metadata); err != nil {
			log.Logger.ErrorfCtx(ctx, "解析metadata失败: %v, metadata内容: %s", err, metadataStr)
			response.Error(c, errors.New(errors.ErrCodeInvalidParam, "metadata格式错误，必须是有效的JSON对象"))
			return
		}
		req.Metadata = metadata
	}

	// 根据认证类型设置 SourceMethod
	// openapi 路由调用时设置为 "api"，其他设置为 "manual"
	authType, _ := c.Get(constants.AuthType)
	if authType == constants.AuthTypeOpenAPI {
		req.SourceMethod = "api"
	} else {
		req.SourceMethod = "manual"
	}

	log.Logger.InfofCtx(ctx, "用户 %s 请求创建知识: %s", username, req.Title)

	// 获取是否为管理员
	isAdmin := middleware.GetClientIsAdmin(c)

	// 调用服务层
	knowledge, err := h.knowledgeService.CreateKnowledge(ctx, req, username.(string), isAdmin)
	if err != nil {
		log.Logger.ErrorfCtx(ctx, "创建知识失败: %v", err)
		response.Error(c, err)
		return
	}

	// 查询标签
	tags, err := h.knowledgeService.GetTags(ctx, knowledge.UUID, username.(string), isAdmin)
	if err != nil {
		log.Logger.WarnfCtx(ctx, "获取知识标签失败: %v", err)
		tags = []*domain.KnowledgebaseTag{}
	}

	response.Success(c, kbResp.NewKnowledgeResponse(knowledge, tags, "", ""))
}

// ListKnowledge 查询知识列表
// GET /api/v1/knowledgebases/:kb_uuid/knowledge
func (h *KnowledgeHandler) ListKnowledge(c *gin.Context) {
	ctx := middleware.GetContext(c)
	kbUUID := c.Param("kb_uuid")

	var req request.ListKnowledgeRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Error(c, errors.ErrInvalidRequest)
		return
	}

	req.KnowledgebaseUUID = kbUUID

	username, _ := c.Get(constants.AuthUsername)
	isAdmin := middleware.GetClientIsAdmin(c)

	log.Logger.InfofCtx(ctx, "查询知识列表: kbUUID=%s，用户=%s", kbUUID, username)

	var list []*service.KnowledgeWithTags
	var total int64
	var err error

	if h.isOpenAPIOrHeaderRequest(c) {
		list, total, err = h.knowledgeService.ListKnowledgeForOpenAPI(ctx, &req, username.(string), isAdmin)
	} else {
		list, total, err = h.knowledgeService.ListKnowledge(ctx, &req, username.(string), isAdmin)
	}

	if err != nil {
		response.Error(c, err)
		return
	}

	// 转换响应
	items := make([]*kbResp.KnowledgeListItemResponse, len(list))
	for i, item := range list {
		items[i] = kbResp.NewKnowledgeListItemResponse(item.Knowledge, item.Tags, item.DirPath)
	}

	response.Success(c, &kbResp.ListKnowledgeResponse{
		List:     items,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	})
}

// ListKnowledgeByTags 根据标签查询知识
// GET /api/v1/knowledgebases/:kb_uuid/knowledge/by-tags
func (h *KnowledgeHandler) ListKnowledgeByTags(c *gin.Context) {
	ctx := middleware.GetContext(c)
	kbUUID := c.Param("kb_uuid")

	var req request.ListKnowledgeByTagsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, errors.ErrInvalidRequest)
		return
	}

	req.KnowledgebaseUUID = kbUUID
	if len(req.TagUUIDs) == 0 && len(req.TagNames) == 0 {
		response.Error(c, errors.New(errors.ErrCodeInvalidParam, "标签参数不能为空"))
		return
	}

	username, _ := c.Get(constants.AuthUsername)
	isAdmin := middleware.GetClientIsAdmin(c)

	list, total, err := h.knowledgeService.ListKnowledgeByTags(ctx, &req, username.(string), isAdmin)
	if err != nil {
		response.Error(c, err)
		return
	}

	// 转换响应（包含 matchedTags）
	items := make([]*kbResp.KnowledgeListItemResponse, len(list))
	for i, item := range list {
		items[i] = kbResp.NewKnowledgeListItemResponseWithMatchedTags(item.Knowledge, item.Tags, item.MatchedTags, item.DirPath)
	}

	response.Success(c, &kbResp.ListKnowledgeResponse{
		List:     items,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	})
}

// GetKnowledge 查询知识详情
// GET /api/v1/knowledge/:uuid
func (h *KnowledgeHandler) GetKnowledge(c *gin.Context) {
	ctx := middleware.GetContext(c)
	uuid := c.Param("uuid")

	username, _ := c.Get(constants.AuthUsername)
	isAdmin := middleware.GetClientIsAdmin(c)

	var knowledge *domain.Knowledge
	var tags []*domain.KnowledgebaseTag
	var directoryName string
	var dirPath string
	var err error

	if h.isOpenAPIOrHeaderRequest(c) {
		knowledge, tags, directoryName, dirPath, err = h.knowledgeService.GetKnowledgeForOpenAPI(ctx, uuid, username.(string), isAdmin)
	} else {
		knowledge, tags, directoryName, dirPath, err = h.knowledgeService.GetKnowledge(ctx, uuid, username.(string), isAdmin)
	}

	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, kbResp.NewKnowledgeResponse(knowledge, tags, directoryName, dirPath))
}

// GetKnowledgeBatch 批量查询知识详情
// POST /api/v1/knowledge/batch
func (h *KnowledgeHandler) GetKnowledgeBatch(c *gin.Context) {
	ctx := middleware.GetContext(c)
	var req request.GetKnowledgeBatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, errors.ErrInvalidRequest)
		return
	}

	username, _ := c.Get(constants.AuthUsername)
	isAdmin := middleware.GetClientIsAdmin(c)

	log.Logger.InfofCtx(ctx, "用户 %s 批量查询知识详情: %d 个", username, len(req.UUIDs))

	list, err := h.knowledgeService.GetKnowledgeBatch(ctx, req.UUIDs, username.(string), isAdmin)
	if err != nil {
		response.Error(c, err)
		return
	}

	// 转换响应
	items := make([]*kbResp.KnowledgeResponse, len(list))
	for i, item := range list {
		items[i] = kbResp.NewKnowledgeResponse(item.Knowledge, item.Tags, item.DirectoryName, item.DirPath)
	}

	response.Success(c, &kbResp.GetKnowledgeBatchResponse{
		List: items,
	})
}

// UpdateKnowledge 更新知识
// PUT /api/v1/knowledge/:uuid
func (h *KnowledgeHandler) UpdateKnowledge(c *gin.Context) {
	ctx := middleware.GetContext(c)
	uuid := c.Param("uuid")

	var req request.UpdateKnowledgeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, errors.ErrInvalidRequest)
		return
	}

	username, _ := c.Get(constants.AuthUsername)
	isAdmin := middleware.GetClientIsAdmin(c)

	log.Logger.InfofCtx(ctx, "用户 %s 请求更新知识: %s", username, uuid)

	knowledge, err := h.knowledgeService.UpdateKnowledge(ctx, uuid, &req, username.(string), isAdmin)
	if err != nil {
		response.Error(c, err)
		return
	}

	// 查询标签
	tags, err := h.knowledgeService.GetTags(ctx, uuid, username.(string), isAdmin)
	if err != nil {
		log.Logger.WarnfCtx(ctx, "获取知识标签失败: %v", err)
		tags = []*domain.KnowledgebaseTag{}
	}

	response.Success(c, kbResp.NewKnowledgeResponse(knowledge, tags, "", ""))
}

// UpdateKnowledgeFile 更新知识文件
// POST /api/v1/knowledge/:uuid/file
func (h *KnowledgeHandler) UpdateKnowledgeFile(c *gin.Context) {
	ctx := middleware.GetContext(c)
	uuid := c.Param("uuid")
	if uuid == "" {
		response.Error(c, errors.ErrInvalidParam)
		return
	}

	username, _ := c.Get(constants.AuthUsername)
	isAdmin := middleware.GetClientIsAdmin(c)

	// 构建 UpdateKnowledgeFileRequest
	req := &request.UpdateKnowledgeRequest{
		Title: c.PostForm("title"),
	}

	// 处理 directoryUuid
	if dirUUID := c.PostForm("directoryUuid"); dirUUID != "" && dirUUID != "/" {
		req.DirectoryUUID = &dirUUID
	}

	// 处理文件上传
	if file, err := c.FormFile("file"); err == nil {
		req.File = file
	}

	if req.File == nil {
		response.Error(c, errors.ErrInvalidRequest)
		log.Logger.ErrorfCtx(ctx, "没有上传文件")
		return
	}

	// 处理 metadata (JSON字符串)
	if metadataStr := c.PostForm("metadata"); metadataStr != "" {
		var metadata map[string]interface{}
		if err := json.Unmarshal([]byte(metadataStr), &metadata); err != nil {
			log.Logger.ErrorfCtx(ctx, "解析metadata失败: %v, metadata内容: %s", err, metadataStr)
			response.Error(c, errors.New(errors.ErrCodeInvalidParam, "metadata格式错误，必须是有效的JSON对象"))
			return
		}
		req.Metadata = metadata
	}

	log.Logger.InfofCtx(ctx, "用户 %s 请求更新知识文件: %s", username, uuid)

	knowledge, err := h.knowledgeService.UpdateKnowledge(ctx, uuid, req, username.(string), isAdmin)
	if err != nil {
		response.Error(c, err)
		return
	}

	// 查询标签
	tags, err := h.knowledgeService.GetTags(ctx, uuid, username.(string), isAdmin)
	if err != nil {
		log.Logger.WarnfCtx(ctx, "获取知识标签失败: %v", err)
		tags = []*domain.KnowledgebaseTag{}
	}

	response.Success(c, kbResp.NewKnowledgeResponse(knowledge, tags, "", ""))
}

// DeleteKnowledge 删除知识
// DELETE /api/v1/knowledge/:uuid
func (h *KnowledgeHandler) DeleteKnowledge(c *gin.Context) {
	ctx := middleware.GetContext(c)
	uuid := c.Param("uuid")

	username, _ := c.Get(constants.AuthUsername)
	isAdmin := middleware.GetClientIsAdmin(c)

	log.Logger.InfofCtx(ctx, "用户 %s 请求删除知识: %s", username, uuid)

	err := h.knowledgeService.DeleteKnowledge(ctx, uuid, username.(string), isAdmin)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, nil)
}

// MoveKnowledge 移动知识到其他目录
// POST /api/v1/knowledge/:uuid/move
func (h *KnowledgeHandler) MoveKnowledge(c *gin.Context) {
	ctx := middleware.GetContext(c)
	uuid := c.Param("uuid")

	var req request.MoveKnowledgeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, errors.ErrInvalidRequest)
		return
	}

	username, _ := c.Get(constants.AuthUsername)
	isAdmin := middleware.GetClientIsAdmin(c)

	log.Logger.InfofCtx(ctx, "用户 %s 请求移动知识: %s", username, uuid)

	var dirUUID string
	if req.DirectoryUUID != nil && *req.DirectoryUUID != "/" {
		dirUUID = *req.DirectoryUUID
	}

	err := h.knowledgeService.MoveKnowledge(ctx, uuid, dirUUID, username.(string), isAdmin)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, nil)
}

// BindTags 绑定标签
// POST /api/v1/knowledge/:uuid/tags
func (h *KnowledgeHandler) BindTags(c *gin.Context) {
	ctx := middleware.GetContext(c)
	uuid := c.Param("uuid")

	var req request.BindTagsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, errors.ErrInvalidRequest)
		return
	}

	username, _ := c.Get(constants.AuthUsername)
	isAdmin := middleware.GetClientIsAdmin(c)

	log.Logger.InfofCtx(ctx, "用户 %s 请求绑定标签: %s", username, uuid)

	err := h.knowledgeService.BindTags(ctx, uuid, req.TagUUIDs, username.(string), isAdmin)
	if err != nil {
		response.Error(c, err)
		return
	}

	// 返回绑定后的标签
	tags, err := h.knowledgeService.GetTags(ctx, uuid, username.(string), isAdmin)
	if err != nil {
		log.Logger.WarnfCtx(ctx, "获取知识标签失败: %v", err)
		tags = []*domain.KnowledgebaseTag{}
	}
	response.Success(c, tags)
}

// UnbindTags 解绑标签
// DELETE /api/v1/knowledge/:uuid/tags
func (h *KnowledgeHandler) UnbindTags(c *gin.Context) {
	ctx := middleware.GetContext(c)
	uuid := c.Param("uuid")

	var req request.BindTagsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, errors.ErrInvalidRequest)
		return
	}

	username, _ := c.Get(constants.AuthUsername)
	isAdmin := middleware.GetClientIsAdmin(c)

	log.Logger.InfofCtx(ctx, "用户 %s 请求解绑标签: %s", username, uuid)

	tags, err := h.knowledgeService.UnbindTags(ctx, uuid, req.TagUUIDs, username.(string), isAdmin)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, tags)
}

// GetKnowledgeTags 查询知识的标签
// GET /api/v1/knowledge/:uuid/tags
func (h *KnowledgeHandler) GetKnowledgeTags(c *gin.Context) {
	ctx := middleware.GetContext(c)
	uuid := c.Param("uuid")

	username, _ := c.Get(constants.AuthUsername)
	isAdmin := middleware.GetClientIsAdmin(c)

	tags, err := h.knowledgeService.GetTags(ctx, uuid, username.(string), isAdmin)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, tags)
}
