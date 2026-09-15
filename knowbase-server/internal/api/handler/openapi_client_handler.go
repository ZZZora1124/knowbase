// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/api/middleware"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/api/request"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/infrastructure/database"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/repository"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/service"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/errors"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/log"
	apiResponse "icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/response"
)

// OpenAPIClientHandler OpenAPI客户端管理处理器
type OpenAPIClientHandler struct {
	service service.OpenAPIClientService
}

// tokenShort 返回 token 前8位用于日志（脱敏）
func tokenShort(token string) string {
	if len(token) <= 8 {
		return token
	}
	return token[:8] + "..."
}

// NewOpenAPIClientHandler 创建OpenAPI客户端管理处理器实例
func NewOpenAPIClientHandler() *OpenAPIClientHandler {
	ormer := database.GetOrm()
	repo := repository.NewOpenAPIClientRepository(ormer)
	txManager := database.NewTransactionManager()
	svc := service.NewOpenAPIClientService(repo, txManager)

	return &OpenAPIClientHandler{
		service: svc,
	}
}

// CreateClient 创建OpenAPI客户端
// @Summary 创建OpenAPI客户端
// @Description 创建新的OpenAPI客户端，返回客户端Token（仅此一次）
// @Tags OpenAPI客户端管理
// @Accept json
// @Produce json
// @Param body body request.CreateOpenAPIClientRequest true "创建请求"
// @Success 200 {object} response.OpenAPIClientCreatedResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/v1/openapi-clients [post]
func (h *OpenAPIClientHandler) CreateClient(c *gin.Context) {
	// 权限校验：只有isAdmin客户端才能操作
	if !middleware.GetClientIsAdmin(c) {
		apiResponse.Error(c, errors.ErrNoPermission)
		return
	}

	var req request.CreateOpenAPIClientRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Logger.ErrorfCtx(c, "解析请求体失败: %v", err)
		apiResponse.Error(c, errors.ErrInvalidRequest)
		return
	}

	log.Logger.InfofCtx(c, "创建OpenAPI客户端: platformName=%s, applicant=%s", req.PlatformName, req.Applicant)

	result, err := h.service.CreateClient(c.Request.Context(), &req)
	if err != nil {
		log.Logger.ErrorfCtx(c, "创建OpenAPI客户端失败: %v", err)
		apiResponse.Error(c, err)
		return
	}

	apiResponse.Success(c, result)
}

// GetClient 查询单个OpenAPI客户端
// @Summary 查询OpenAPI客户端详情
// @Description 根据Token查询OpenAPI客户端详情
// @Tags OpenAPI客户端管理
// @Produce json
// @Param token path string true "客户端Token"
// @Success 200 {object} response.OpenAPIClientResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Router /open-api/v1/openapi-clients/:token [get]
func (h *OpenAPIClientHandler) GetClient(c *gin.Context) {
	// 权限校验：只有isAdmin客户端才能操作
	if !middleware.GetClientIsAdmin(c) {
		apiResponse.Error(c, errors.ErrNoPermission)
		return
	}

	token := c.Param("token")
	if token == "" {
		apiResponse.Error(c, errors.ErrInvalidParam)
		return
	}

	result, err := h.service.GetClient(c.Request.Context(), token)
	if err != nil {
		log.Logger.ErrorfCtx(c, "查询OpenAPI客户端失败: %v", err)
		apiResponse.Error(c, err)
		return
	}

	// 查询时不返回Token
	result.ClientToken = ""
	apiResponse.Success(c, result)
}

// ListClients 查询OpenAPI客户端列表
// @Summary 查询OpenAPI客户端列表
// @Description 分页查询OpenAPI客户端列表
// @Tags OpenAPI客户端管理
// @Produce json
// @Param page query int false "页码" default(1)
// @Param pageSize query int false "每页数量" default(20)
// @Success 200 {object} response.ListOpenAPIClientsResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/v1/openapi-clients [get]
func (h *OpenAPIClientHandler) ListClients(c *gin.Context) {
	// 权限校验：只有isAdmin客户端才能操作
	if !middleware.GetClientIsAdmin(c) {
		apiResponse.Error(c, errors.ErrNoPermission)
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	result, err := h.service.ListClients(c.Request.Context(), page, pageSize)
	if err != nil {
		log.Logger.ErrorfCtx(c, "查询OpenAPI客户端列表失败: %v", err)
		apiResponse.Error(c, err)
		return
	}

	// 列表查询时不返回Token
	for _, item := range result.Items {
		item.ClientToken = ""
	}

	apiResponse.Success(c, result)
}

// UpdateClient 更新OpenAPI客户端
// @Summary 更新OpenAPI客户端
// @Description 更新OpenAPI客户端信息
// @Tags OpenAPI客户端管理
// @Accept json
// @Produce json
// @Param token path string true "客户端Token"
// @Param body body request.UpdateOpenAPIClientRequest true "更新请求"
// @Success 200 {object} response.OpenAPIClientResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Router /open-api/v1/openapi-clients/:token [put]
func (h *OpenAPIClientHandler) UpdateClient(c *gin.Context) {
	// 权限校验：只有isAdmin客户端才能操作
	if !middleware.GetClientIsAdmin(c) {
		apiResponse.Error(c, errors.ErrNoPermission)
		return
	}

	token := c.Param("token")
	if token == "" {
		apiResponse.Error(c, errors.ErrInvalidParam)
		return
	}

	var req request.UpdateOpenAPIClientRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Logger.ErrorfCtx(c, "解析请求体失败: %v", err)
		apiResponse.Error(c, errors.ErrInvalidRequest)
		return
	}

	log.Logger.InfofCtx(c, "更新OpenAPI客户端: token=%s", tokenShort(token))

	result, err := h.service.UpdateClient(c.Request.Context(), token, &req)
	if err != nil {
		log.Logger.ErrorfCtx(c, "更新OpenAPI客户端失败: %v", err)
		apiResponse.Error(c, err)
		return
	}

	// 更新时不返回Token
	result.ClientToken = ""
	apiResponse.Success(c, result)
}

// DeleteClient 删除OpenAPI客户端
// @Summary 删除OpenAPI客户端
// @Description 软删除OpenAPI客户端
// @Tags OpenAPI客户端管理
// @Param token path string true "客户端Token"
// @Success 200 {object} response.SuccessResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Router /open-api/v1/openapi-clients/:token [delete]
func (h *OpenAPIClientHandler) DeleteClient(c *gin.Context) {
	// 权限校验：只有isAdmin客户端才能操作
	if !middleware.GetClientIsAdmin(c) {
		apiResponse.Error(c, errors.ErrNoPermission)
		return
	}

	token := c.Param("token")
	if token == "" {
		apiResponse.Error(c, errors.ErrInvalidParam)
		return
	}

	log.Logger.InfofCtx(c, "删除OpenAPI客户端: token=%s", tokenShort(token))

	err := h.service.DeleteClient(c.Request.Context(), token)
	if err != nil {
		log.Logger.ErrorfCtx(c, "删除OpenAPI客户端失败: %v", err)
		apiResponse.Error(c, err)
		return
	}

	apiResponse.Success(c, nil)
}

// DisableClient 禁用OpenAPI客户端
// @Summary 禁用OpenAPI客户端
// @Description 禁用指定OpenAPI客户端，立即使其Token失效
// @Tags OpenAPI客户端管理
// @Param token path string true "客户端Token"
// @Success 200 {object} response.SuccessResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Router /open-api/v1/openapi-clients/:token/disable [post]
func (h *OpenAPIClientHandler) DisableClient(c *gin.Context) {
	// 权限校验：只有isAdmin客户端才能操作
	if !middleware.GetClientIsAdmin(c) {
		apiResponse.Error(c, errors.ErrNoPermission)
		return
	}

	token := c.Param("token")
	if token == "" {
		apiResponse.Error(c, errors.ErrInvalidParam)
		return
	}

	log.Logger.InfofCtx(c.Request.Context(), "禁用OpenAPI客户端: token=%s", tokenShort(token))

	err := h.service.DisableClient(c.Request.Context(), token)
	if err != nil {
		log.Logger.ErrorfCtx(c, "禁用OpenAPI客户端失败: %v", err)
		apiResponse.Error(c, err)
		return
	}

	apiResponse.Success(c, nil)
}
