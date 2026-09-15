// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package handler

import (
	"github.com/gin-gonic/gin"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/api/middleware"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/api/request"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/infrastructure/database"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/repository"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/service"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/errors"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/log"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/response"
)

// OperationLogHandler 操作日志处理器
type OperationLogHandler struct {
	service service.OperationLogService
}

// NewOperationLogHandler 创建操作日志处理器实例
func NewOperationLogHandler() *OperationLogHandler {
	logRepo := repository.NewKnowledgebaseOperationLogRepository(database.GetOrm())
	logService := service.NewOperationLogService(logRepo)
	return &OperationLogHandler{
		service: logService,
	}
}

// ListOperationLogs 查询操作日志列表
// @Summary 查询操作日志列表
// @Description 根据条件查询知识本操作日志，至少需要提供一个查询条件
// @Tags 操作日志
// @Accept json
// @Produce json
// @Param knowledgebaseUuid query string false "知识本UUID"
// @Param operator query string false "操作者"
// @Param agent query string false "Agent标识"
// @Param resourceType query string false "资源类型"
// @Param resourceUuid query string false "资源UUID"
// @Param page query int false "页码"
// @Param pageSize query int false "每页数量"
// @Success 200 {object} response.ListOperationLogsResponse
// @Router /api/v1/operation-logs [get]
func (h *OperationLogHandler) ListOperationLogs(c *gin.Context) {
	ctx := middleware.GetContext(c)

	var req request.ListOperationLogRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		log.Logger.ErrorfCtx(ctx, "解析查询参数失败: %v", err)
		response.Error(c, errors.New(errors.ErrCodeInvalidParam, err.Error()))
		return
	}

	result, err := h.service.ListOperationLogs(ctx, &req)
	if err != nil {
		log.Logger.ErrorfCtx(ctx, "查询操作日志失败: %v", err)
		response.Error(c, err)
		return
	}

	response.Success(c, result)
}
