// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package service

import (
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/api/request"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/domain"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/repository"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/errors"
)

// KnowledgeDownloadMetricService 知识下载记录服务接口
type KnowledgeDownloadMetricService interface {
	ReportDownload(req *request.ReportDownloadMetricRequest) error
}

type knowledgeDownloadMetricServiceImpl struct {
	repo repository.KnowledgeDownloadMetricRepository
}

func NewKnowledgeDownloadMetricService(repo repository.KnowledgeDownloadMetricRepository) KnowledgeDownloadMetricService {
	return &knowledgeDownloadMetricServiceImpl{
		repo: repo,
	}
}

func (s *knowledgeDownloadMetricServiceImpl) ReportDownload(req *request.ReportDownloadMetricRequest) error {
	// 构建批量领域模型
	metrics := make([]*domain.KnowledgeDownloadMetric, 0, len(req.Metrics))
	for _, item := range req.Metrics {
		metric := &domain.KnowledgeDownloadMetric{
			Platform:     req.Platform,
			Username:     req.Username,
			SourceType:   item.SourceType,
			Count:        item.Count,
			DurationMs:   item.DurationMs,
			Success:      item.Success,
			ErrorMessage: item.ErrorMessage,
		}
		metrics = append(metrics, metric)
	}

	// 批量插入
	return s.repo.BatchCreate(metrics)
}

func (s *knowledgeDownloadMetricServiceImpl) validateRequest(req *request.ReportDownloadMetricRequest) error {
	// 验证每条记录的业务规则
	for _, item := range req.Metrics {
		if !item.Success && item.ErrorMessage == "" {
			return errors.New(errors.ErrCodeInvalidParam, "下载失败时必须提供错误信息")
		}
	}

	return nil
}
