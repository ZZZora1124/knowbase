// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package repository

import (
	"github.com/beego/beego/v2/client/orm"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/domain"
)

// KnowledgeConsolidationMetricRepository 知识沉淀性能指标仓储接口
type KnowledgeConsolidationMetricRepository interface {
	BatchCreate(metrics []*domain.KnowledgeConsolidationMetric) error
}

type knowledgeConsolidationMetricRepositoryImpl struct {
	ormer orm.Ormer
}

func NewKnowledgeConsolidationMetricRepository(ormer orm.Ormer) KnowledgeConsolidationMetricRepository {
	return &knowledgeConsolidationMetricRepositoryImpl{ormer: ormer}
}

func (r *knowledgeConsolidationMetricRepositoryImpl) BatchCreate(metrics []*domain.KnowledgeConsolidationMetric) error {
	if len(metrics) == 0 {
		return nil
	}
	_, err := r.ormer.InsertMulti(len(metrics), metrics)
	return err
}
