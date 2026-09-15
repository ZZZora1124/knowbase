// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package handler

import (
	"github.com/gin-gonic/gin"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/api/request"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/infrastructure/database"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/repository"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/service"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/errors"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/log"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/response"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/types/constants"
)

type MetricHandler struct {
	service              service.KnowledgeDownloadMetricService
	consolidationService service.KnowledgeConsolidationMetricService
}

func NewMetricHandler() *MetricHandler {
	ormer := database.GetOrm()
	repo := repository.NewKnowledgeDownloadMetricRepository(ormer)
	svc := service.NewKnowledgeDownloadMetricService(repo)

	consolidationRepo := repository.NewKnowledgeConsolidationMetricRepository(ormer)
	consolidationSvc := service.NewKnowledgeConsolidationMetricService(consolidationRepo)

	return &MetricHandler{
		service:              svc,
		consolidationService: consolidationSvc,
	}
}

func (h *MetricHandler) HandleReportDownload(c *gin.Context) {
	var req request.ReportDownloadMetricRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Logger.Errorf("解析请求体失败: %v", err)
		response.Error(c, errors.ErrInvalidRequest)
		return
	}
	username, exists := c.Get(constants.AuthUsername)
	if !exists {
		log.Logger.Errorf("获取用户名失败")
		response.Error(c, errors.ErrInvalidToken)
		return
	}
	req.Username = username.(string)
	platform, exists := c.Get(constants.AuthPlatform)
	if !exists {
		log.Logger.Errorf("获取平台失败")
		response.Error(c, errors.ErrInvalidToken)
		return
	}
	req.Platform = platform.(string)

	log.Logger.Infof("收到知识下载记录批量上报请求, username: %s, platform: %s, records: %d",
		req.Username, req.Platform, len(req.Metrics))

	if err := h.service.ReportDownload(&req); err != nil {
		log.Logger.Errorf("处理知识下载记录失败: %v", err)
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr)
		} else {
			response.Error(c, errors.ErrInternalError)
		}
		return
	}

	response.Success(c, gin.H{
		"message": "下载记录上报成功",
	})
}

func (h *MetricHandler) HandleReportConsolidation(c *gin.Context) {
	var req request.ReportConsolidationMetricRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Logger.Errorf("解析请求体失败: %v", err)
		response.Error(c, errors.ErrInvalidRequest)
		return
	}
	username, exists := c.Get(constants.AuthUsername)
	if !exists {
		log.Logger.Errorf("获取用户名失败")
		response.Error(c, errors.ErrInvalidToken)
		return
	}
	req.Username = username.(string)
	platform, exists := c.Get(constants.AuthPlatform)
	if !exists {
		log.Logger.Errorf("获取平台失败")
		response.Error(c, errors.ErrInvalidToken)
		return
	}
	req.Platform = platform.(string)

	log.Logger.Infof("收到知识沉淀指标批量上报请求, username: %s, platform: %s, records: %d",
		req.Username, req.Platform, len(req.Metrics))

	if err := h.consolidationService.ReportConsolidation(&req); err != nil {
		log.Logger.Errorf("处理知识沉淀指标失败: %v", err)
		if appErr, ok := err.(*errors.AppError); ok {
			response.Error(c, appErr)
		} else {
			response.Error(c, errors.ErrInternalError)
		}
		return
	}

	response.Success(c, gin.H{
		"message": "沉淀指标上报成功",
	})
}
