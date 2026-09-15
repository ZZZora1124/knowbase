// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package service

import (
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/api/request"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/domain"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/repository"
)

// KnowledgeConsolidationMetricService 知识沉淀性能指标服务接口
type KnowledgeConsolidationMetricService interface {
	ReportConsolidation(req *request.ReportConsolidationMetricRequest) error
}

type knowledgeConsolidationMetricServiceImpl struct {
	repo repository.KnowledgeConsolidationMetricRepository
}

func NewKnowledgeConsolidationMetricService(repo repository.KnowledgeConsolidationMetricRepository) KnowledgeConsolidationMetricService {
	return &knowledgeConsolidationMetricServiceImpl{repo: repo}
}

func (s *knowledgeConsolidationMetricServiceImpl) ReportConsolidation(req *request.ReportConsolidationMetricRequest) error {
	metrics := make([]*domain.KnowledgeConsolidationMetric, 0, len(req.Metrics))
	for _, item := range req.Metrics {
		metrics = append(metrics, &domain.KnowledgeConsolidationMetric{
			Platform:     req.Platform,
			Username:     req.Username,
			KbUUID:       item.KbUUID,
			OpType:       item.OpType,
			InputTokens:  item.InputTokens,
			OutputTokens: item.OutputTokens,
			TotalTokens:  item.TotalTokens,
			Model:        item.Model,
			DurationS:    item.DurationS,
			PageCount:    item.PageCount,
			Success:      item.Success,
			ErrorMessage: item.ErrorMessage,
		})
	}
	return s.repo.BatchCreate(metrics)
}
