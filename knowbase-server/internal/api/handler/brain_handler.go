// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package handler

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/api/middleware"
	brainResp "icode.baidu.com/baidu/devops-ai/knowbase-server/internal/api/response"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/infrastructure/contextmap"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/infrastructure/database"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/infrastructure/infoflow"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/repository"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/service"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/errors"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/log"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/response"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/types/constants"
)

// BrainHandler brain 模块处理器
type BrainHandler struct {
	brainService   service.BrainService
	infoflowClient *infoflow.Client
}

// NewBrainHandler 创建 brain 处理器实例
func NewBrainHandler() *BrainHandler {
	ormer := database.GetOrm()

	kbRepo := repository.NewKnowledgebaseRepository(ormer)
	memberRepo := repository.NewKnowledgebaseMemberRepository(ormer)
	dirRepo := repository.NewKnowledgebaseDirectoryRepository(ormer)
	knowRepo := repository.NewKnowledgeRepository(ormer)
	versionRepo := repository.NewKnowledgeVersionRepository(ormer)
	logRepo := repository.NewKnowledgebaseOperationLogRepository(ormer)
	permissionService := service.NewPermissionService(memberRepo)
	cmClient := contextmap.GetClient()
	txManager := database.NewTransactionManager()

	svc := service.NewBrainService(kbRepo, dirRepo, knowRepo, permissionService, cmClient, txManager, versionRepo, logRepo)
	return &BrainHandler{brainService: svc, infoflowClient: infoflow.GetClient()}
}

// getOperatorInfo 从 gin 上下文提取操作人和 admin 标识
func getOperatorInfo(c *gin.Context) (string, bool, bool) {
	usernameRaw, exists := c.Get(constants.AuthUsername)
	if !exists {
		return "", false, false
	}
	username, ok := usernameRaw.(string)
	if !ok || username == "" {
		return "", false, false
	}
	isAdmin := middleware.GetClientIsAdmin(c)
	return username, isAdmin, true
}

// getSlug 从 gin 通配符参数中提取 slug（去掉前导 /）
// gin 的 *param 通配符会包含前导 /，如 /people/alice
func getSlug(c *gin.Context) string {
	return strings.TrimPrefix(c.Param("slug"), "/")
}

// getKbUUID 从请求头获取 x-kb-uuid
func getKbUUID(c *gin.Context) string {
	return c.GetHeader(contextmap.HeaderKbUUID)
}

// checkKbUUIDPermission 校验 operator 是否有权限访问 kbUUID
// kbUUID 可为逗号分隔列表，每个 item 全部通过才算通过
// item 为纯数字则视为群ID，校验 operator 是否在群中；否则视为 uuap 前缀，校验与 operator 是否相等
func (h *BrainHandler) checkKbUUIDPermission(ctx context.Context, kbUUID, operator string) error {
	return checkPermission(ctx, h.infoflowClient, kbUUID, operator)
}

// infoflowGroupChecker 抽象如流群成员校验，便于测试注入
type infoflowGroupChecker interface {
	IsUserInGroup(ctx context.Context, groupID int64, userID string) (bool, error)
}

// checkPermission 核心校验逻辑，client 可为 nil（群ID类型 item 将被跳过）
func checkPermission(ctx context.Context, client infoflowGroupChecker, kbUUID, operator string) error {
	items := strings.Split(kbUUID, ",")
	for _, raw := range items {
		item := strings.TrimSpace(raw)
		if item == "" {
			continue
		}
		if groupID, err := strconv.ParseInt(item, 10, 64); err == nil {
			if client == nil {
				log.Logger.WarnfCtx(ctx, "infoflow client 未初始化，跳过群 %d 校验", groupID)
				continue
			}
			inGroup, err := client.IsUserInGroup(ctx, groupID, operator)
			if err != nil {
				log.Logger.WarnfCtx(ctx, "校验用户是否在群 %d 中失败: %v", groupID, err)
				return errors.ErrNoPermission
			}
			if !inGroup {
				return errors.ErrNoPermission
			}
		} else {
			if item != operator {
				return errors.ErrNoPermission
			}
		}
	}
	return nil
}

// PutPage 写入/更新 brain page
// PUT /open-api/v1/brain/page
func (h *BrainHandler) PutPage(c *gin.Context) {
	ctx := middleware.GetContext(c)

	kbUUID := getKbUUID(c)
	if kbUUID == "" {
		response.Error(c, errors.New(errors.ErrCodeInvalidParam, "缺少 x-kb-uuid 请求头"))
		return
	}

	operator, isAdmin, ok := getOperatorInfo(c)
	if !ok {
		log.Logger.ErrorfCtx(ctx, "获取用户名失败")
		response.Error(c, errors.ErrInvalidToken)
		return
	}

	if !isAdmin {
		if err := h.checkKbUUIDPermission(ctx, kbUUID, operator); err != nil {
			response.Error(c, err)
			return
		}
	}

	var req struct {
		Slug    string `json:"slug"`
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, errors.ErrInvalidParam)
		return
	}

	result, err := h.brainService.PutPage(ctx, kbUUID, operator, isAdmin, req.Slug, req.Content)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.SuccessSimple(c, brainResp.PutPageResponse{
		Slug:   result.Slug,
		Status: result.Status,
		Chunks: result.Chunks,
		Links: func() *brainResp.PutPageLinks {
			if result.Links == nil {
				return nil
			}
			return &brainResp.PutPageLinks{
				Created: result.Links.Created,
				Removed: result.Links.Removed,
				Errors:  result.Links.Errors,
			}
		}(),
		Timeline: func() *brainResp.PutPageTimeline {
			if result.Timeline == nil {
				return nil
			}
			return &brainResp.PutPageTimeline{Created: result.Timeline.Created}
		}(),
	})
}

// GetPage 读取单页全文
// GET /open-api/v1/brain/page/:slug
func (h *BrainHandler) GetPage(c *gin.Context) {
	ctx := middleware.GetContext(c)

	kbUUID := getKbUUID(c)
	if kbUUID == "" {
		response.Error(c, errors.New(errors.ErrCodeInvalidParam, "缺少 x-kb-uuid 请求头"))
		return
	}

	operator, isAdmin, ok := getOperatorInfo(c)
	if !ok {
		log.Logger.ErrorfCtx(ctx, "获取用户名失败")
		response.Error(c, errors.ErrInvalidToken)
		return
	}

	if !isAdmin {
		if err := h.checkKbUUIDPermission(ctx, kbUUID, operator); err != nil {
			response.Error(c, err)
			return
		}
	}

	slug := getSlug(c)
	fuzzy := c.Query("fuzzy") == "true"

	rawBytes, statusCode, err := h.brainService.GetPage(ctx, kbUUID, operator, isAdmin, slug, fuzzy)
	if err != nil {
		response.Error(c, err)
		return
	}

	c.Data(statusCode, "application/json", rawBytes)
}

// SearchPages 关键词搜索
// GET /open-api/v1/brain/search
func (h *BrainHandler) SearchPages(c *gin.Context) {
	ctx := middleware.GetContext(c)

	kbUUID := getKbUUID(c)
	if kbUUID == "" {
		response.Error(c, errors.New(errors.ErrCodeInvalidParam, "缺少 x-kb-uuid 请求头"))
		return
	}

	operator, isAdmin, ok := getOperatorInfo(c)
	if !ok {
		log.Logger.ErrorfCtx(ctx, "获取用户名失败")
		response.Error(c, errors.ErrInvalidToken)
		return
	}

	if !isAdmin {
		if err := h.checkKbUUIDPermission(ctx, kbUUID, operator); err != nil {
			response.Error(c, err)
			return
		}
	}

	q := c.Query("q")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	rawBytes, statusCode, err := h.brainService.SearchPages(ctx, kbUUID, operator, isAdmin, q, limit, offset)
	if err != nil {
		response.Error(c, err)
		return
	}

	c.Data(statusCode, "application/json", rawBytes)
}

// QueryPages 混合检索
// GET /open-api/v1/brain/query
func (h *BrainHandler) QueryPages(c *gin.Context) {
	ctx := middleware.GetContext(c)

	kbUUID := getKbUUID(c)
	if kbUUID == "" {
		response.Error(c, errors.New(errors.ErrCodeInvalidParam, "缺少 x-kb-uuid 请求头"))
		return
	}

	operator, isAdmin, ok := getOperatorInfo(c)
	if !ok {
		log.Logger.ErrorfCtx(ctx, "获取用户名失败")
		response.Error(c, errors.ErrInvalidToken)
		return
	}

	if !isAdmin {
		if err := h.checkKbUUIDPermission(ctx, kbUUID, operator); err != nil {
			response.Error(c, err)
			return
		}
	}

	q := c.Query("q")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	expand := c.Query("expand") == "true"
	detail := c.Query("detail")

	rawBytes, statusCode, err := h.brainService.QueryPages(ctx, kbUUID, operator, isAdmin, q, limit, expand, detail)
	if err != nil {
		response.Error(c, err)
		return
	}

	c.Data(statusCode, "application/json", rawBytes)
}

// ListPages 枚举页面
// GET /open-api/v1/brain/page
func (h *BrainHandler) ListPages(c *gin.Context) {
	ctx := middleware.GetContext(c)

	kbUUID := getKbUUID(c)
	if kbUUID == "" {
		response.Error(c, errors.New(errors.ErrCodeInvalidParam, "缺少 x-kb-uuid 请求头"))
		return
	}

	operator, isAdmin, ok := getOperatorInfo(c)
	if !ok {
		log.Logger.ErrorfCtx(ctx, "获取用户名失败")
		response.Error(c, errors.ErrInvalidToken)
		return
	}

	if !isAdmin {
		if err := h.checkKbUUIDPermission(ctx, kbUUID, operator); err != nil {
			response.Error(c, err)
			return
		}
	}

	pageType := c.Query("type")
	tag := c.Query("tag")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	rawBytes, statusCode, err := h.brainService.ListPages(ctx, kbUUID, operator, isAdmin, pageType, tag, limit, offset)
	if err != nil {
		response.Error(c, err)
		return
	}

	c.Data(statusCode, "application/json", rawBytes)
}

// GetBacklinks 获取反向链接
// GET /open-api/v1/brain/page/:slug/backlinks
func (h *BrainHandler) GetBacklinks(c *gin.Context) {
	ctx := middleware.GetContext(c)

	kbUUID := getKbUUID(c)
	if kbUUID == "" {
		response.Error(c, errors.New(errors.ErrCodeInvalidParam, "缺少 x-kb-uuid 请求头"))
		return
	}

	operator, isAdmin, ok := getOperatorInfo(c)
	if !ok {
		log.Logger.ErrorfCtx(ctx, "获取用户名失败")
		response.Error(c, errors.ErrInvalidToken)
		return
	}

	if !isAdmin {
		if err := h.checkKbUUIDPermission(ctx, kbUUID, operator); err != nil {
			response.Error(c, err)
			return
		}
	}

	slug := getSlug(c)

	rawBytes, statusCode, err := h.brainService.GetBacklinks(ctx, kbUUID, operator, isAdmin, slug)
	if err != nil {
		response.Error(c, err)
		return
	}

	c.Data(statusCode, "application/json", rawBytes)
}

// TraverseGraph 图谱遍历
// GET /open-api/v1/brain/page/:slug/graph
func (h *BrainHandler) TraverseGraph(c *gin.Context) {
	ctx := middleware.GetContext(c)

	kbUUID := getKbUUID(c)
	if kbUUID == "" {
		response.Error(c, errors.New(errors.ErrCodeInvalidParam, "缺少 x-kb-uuid 请求头"))
		return
	}

	operator, isAdmin, ok := getOperatorInfo(c)
	if !ok {
		log.Logger.ErrorfCtx(ctx, "获取用户名失败")
		response.Error(c, errors.ErrInvalidToken)
		return
	}

	if !isAdmin {
		if err := h.checkKbUUIDPermission(ctx, kbUUID, operator); err != nil {
			response.Error(c, err)
			return
		}
	}

	slug := getSlug(c)
	linkType := c.Query("type")
	direction := c.DefaultQuery("direction", "both")
	depth, _ := strconv.Atoi(c.DefaultQuery("depth", "2"))

	rawBytes, statusCode, err := h.brainService.TraverseGraph(ctx, kbUUID, operator, isAdmin, slug, linkType, direction, depth)
	if err != nil {
		response.Error(c, err)
		return
	}

	c.Data(statusCode, "application/json", rawBytes)
}

// GetTimeline 读取时间线
// GET /open-api/v1/brain/page/:slug/timeline
func (h *BrainHandler) GetTimeline(c *gin.Context) {
	ctx := middleware.GetContext(c)

	kbUUID := getKbUUID(c)
	if kbUUID == "" {
		response.Error(c, errors.New(errors.ErrCodeInvalidParam, "缺少 x-kb-uuid 请求头"))
		return
	}

	operator, isAdmin, ok := getOperatorInfo(c)
	if !ok {
		log.Logger.ErrorfCtx(ctx, "获取用户名失败")
		response.Error(c, errors.ErrInvalidToken)
		return
	}

	if !isAdmin {
		if err := h.checkKbUUIDPermission(ctx, kbUUID, operator); err != nil {
			response.Error(c, err)
			return
		}
	}

	slug := getSlug(c)
	since := c.Query("since")
	until := c.Query("until")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	rawBytes, statusCode, err := h.brainService.GetTimeline(ctx, kbUUID, operator, isAdmin, slug, since, until, limit)
	if err != nil {
		response.Error(c, err)
		return
	}

	c.Data(statusCode, "application/json", rawBytes)
}

// DeletePage 删除单页
// DELETE /open-api/v1/brain/page/*slug
func (h *BrainHandler) DeletePage(c *gin.Context) {
	ctx := middleware.GetContext(c)

	kbUUID := getKbUUID(c)
	if kbUUID == "" {
		response.Error(c, errors.New(errors.ErrCodeInvalidParam, "缺少 x-kb-uuid 请求头"))
		return
	}

	operator, isAdmin, ok := getOperatorInfo(c)
	if !ok {
		log.Logger.ErrorfCtx(ctx, "获取用户名失败")
		response.Error(c, errors.ErrInvalidToken)
		return
	}

	if !isAdmin {
		if err := h.checkKbUUIDPermission(ctx, kbUUID, operator); err != nil {
			response.Error(c, err)
			return
		}
	}

	slug := getSlug(c)
	if slug == "" {
		response.Error(c, errors.New(errors.ErrCodeInvalidParam, "slug 不能为空"))
		return
	}

	rawBytes, statusCode, err := h.brainService.DeletePage(ctx, kbUUID, operator, isAdmin, slug)
	if err != nil {
		response.Error(c, err)
		return
	}

	c.Data(statusCode, "application/json", rawBytes)
}

// AddTimelineEntry 追加单条时间线事件
// POST /open-api/v1/brain/timeline/*slug
func (h *BrainHandler) AddTimelineEntry(c *gin.Context) {
	ctx := middleware.GetContext(c)

	kbUUID := getKbUUID(c)
	if kbUUID == "" {
		response.Error(c, errors.New(errors.ErrCodeInvalidParam, "缺少 x-kb-uuid 请求头"))
		return
	}

	operator, isAdmin, ok := getOperatorInfo(c)
	if !ok {
		log.Logger.ErrorfCtx(ctx, "获取用户名失败")
		response.Error(c, errors.ErrInvalidToken)
		return
	}

	if !isAdmin {
		if err := h.checkKbUUIDPermission(ctx, kbUUID, operator); err != nil {
			response.Error(c, err)
			return
		}
	}

	slug := getSlug(c)
	if slug == "" {
		response.Error(c, errors.New(errors.ErrCodeInvalidParam, "slug 不能为空"))
		return
	}

	var req struct {
		Date    string `json:"date"`
		Summary string `json:"summary"`
		Detail  string `json:"detail"`
		Source  string `json:"source"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, errors.ErrInvalidParam)
		return
	}

	// 客户端日期校验：格式 YYYY-MM-DD 且为合法日历日期
	if _, err := time.Parse("2006-01-02", req.Date); err != nil {
		response.Error(c, errors.New(errors.ErrCodeInvalidParam, "date 须为合法日历日期，格式 YYYY-MM-DD"))
		return
	}

	if strings.TrimSpace(req.Summary) == "" {
		response.Error(c, errors.New(errors.ErrCodeInvalidParam, "summary 不能为空"))
		return
	}

	if err := h.brainService.AddTimelineEntry(ctx, kbUUID, operator, isAdmin, slug, req.Date, req.Summary, req.Detail, req.Source); err != nil {
		response.Error(c, err)
		return
	}

	response.SuccessSimple(c, brainResp.AddTimelineEntryResponse{Status: "ok"})
}

// GetDefaultModel 获取 brain 模块全局默认模型名
// GET /open-api/v1/brain/config/default-model
func (h *BrainHandler) GetDefaultModel(c *gin.Context) {
	ctx := middleware.GetContext(c)
	model := h.brainService.GetDefaultModel(ctx)
	response.SuccessSimple(c, brainResp.DefaultModelResponse{Model: model})
}

// SetDefaultModel 设置 brain 模块全局默认模型名（仅管理员）
// PUT /open-api/v1/brain/config/default-model
// Body: {"model": "GLM-5.1"}
func (h *BrainHandler) SetDefaultModel(c *gin.Context) {
	ctx := middleware.GetContext(c)

	// 权限校验：仅 admin 客户端可写全局配置
	if !middleware.GetClientIsAdmin(c) {
		response.Error(c, errors.ErrNoPermission)
		return
	}

	var req struct {
		Model string `json:"model"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, errors.ErrInvalidParam)
		return
	}

	model, err := h.brainService.SetDefaultModel(ctx, req.Model)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.SuccessSimple(c, brainResp.DefaultModelResponse{Model: model})
}
