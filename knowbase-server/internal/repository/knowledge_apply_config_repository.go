// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package repository

import (
	"time"

	"github.com/beego/beego/v2/client/orm"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/domain"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/infrastructure/database"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/errors"
)

// ListApplyConfigQuery 应用配置查询参数
type ListApplyConfigQuery struct {
	KnowledgebaseUUID string
	AgentCode         string
	Scene             string
	TargetType        string
	TargetID          string
	Status            string
	Operator          string
	Offset            int
	Limit             int
}

// KnowledgeApplyConfigRepository 知识应用配置仓储接口
type KnowledgeApplyConfigRepository interface {
	// Create 创建配置
	Create(ctx *database.TxContext, config *domain.KnowledgeApplyConfig) error

	// Update 更新配置
	Update(ctx *database.TxContext, config *domain.KnowledgeApplyConfig) error

	// FindByUUID 根据UUID查询
	FindByUUID(ctx *database.TxContext, uuid string) (*domain.KnowledgeApplyConfig, error)

	// FindByTarget 根据目标查询（知识本+Agent+场景+目标类型+目标ID）
	FindByTarget(ctx *database.TxContext, kbUUID, agentCode, scene, targetType, targetID string) (*domain.KnowledgeApplyConfig, error)

	// ListByConditions 条件查询
	ListByConditions(ctx *database.TxContext, query *ListApplyConfigQuery) ([]*domain.KnowledgeApplyConfig, int64, error)

	// UpdateStatus 更新状态
	UpdateStatus(ctx *database.TxContext, uuid, status string) error

	// UpdateVersion 更新版本号
	UpdateVersion(ctx *database.TxContext, uuid, version, operator string) error

	// Delete 删除配置
	Delete(ctx *database.TxContext, uuid string) error
}

// knowledgeApplyConfigRepositoryImpl 知识应用配置仓储实现
type knowledgeApplyConfigRepositoryImpl struct {
	ormer orm.Ormer
}

// NewKnowledgeApplyConfigRepository 创建知识应用配置仓储实例
func NewKnowledgeApplyConfigRepository(ormer orm.Ormer) KnowledgeApplyConfigRepository {
	return &knowledgeApplyConfigRepositoryImpl{ormer: ormer}
}

// getOrmer 获取实际的 ormer
func (r *knowledgeApplyConfigRepositoryImpl) getOrmer(ctx *database.TxContext) orm.QueryExecutor {
	if ctx != nil && ctx.Ormer() != nil {
		return ctx.Ormer()
	}
	return r.ormer
}

// Create 创建配置
func (r *knowledgeApplyConfigRepositoryImpl) Create(ctx *database.TxContext, config *domain.KnowledgeApplyConfig) error {
	// TargetExtra 是 JSON 列，空字符串不是合法 JSON，需要确保为合法 JSON 或空
	if config.TargetExtra == "" {
		config.TargetExtra = "{}"
	}
	_, err := r.getOrmer(ctx).Insert(config)
	return err
}

// Update 更新配置（只更新非空字段）
func (r *knowledgeApplyConfigRepositoryImpl) Update(ctx *database.TxContext, config *domain.KnowledgeApplyConfig) error {
	params := orm.Params{}
	if config.AgentName != "" {
		params["agent_name"] = config.AgentName
	}
	if config.TargetName != "" {
		params["target_name"] = config.TargetName
	}
	// TargetExtra 是 JSON 列，空字符串不是合法 JSON，需要确保为合法 JSON 或空
	if config.TargetExtra == "" {
		config.TargetExtra = "{}"
	}
	if config.TargetExtra != "" {
		params["target_extra"] = config.TargetExtra
	}
	if config.Version != "" {
		params["version"] = config.Version
	}
	if config.Status != "" {
		params["status"] = config.Status
	}
	if config.Operator != "" {
		params["operator"] = config.Operator
	}
	if config.Remark != "" {
		params["remark"] = config.Remark
	}
	if len(params) == 0 {
		return nil
	}
	_, err := r.getOrmer(ctx).QueryTable(new(domain.KnowledgeApplyConfig)).
		Filter("uuid", config.UUID).
		Update(params)
	return err
}

// FindByUUID 根据UUID查询
func (r *knowledgeApplyConfigRepositoryImpl) FindByUUID(ctx *database.TxContext, uuid string) (*domain.KnowledgeApplyConfig, error) {
	config := &domain.KnowledgeApplyConfig{}
	err := r.getOrmer(ctx).QueryTable(new(domain.KnowledgeApplyConfig)).
		Filter("uuid", uuid).
		Filter("deleted_at__isnull", true).
		One(config)
	if err == orm.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return config, nil
}

// FindByTarget 根据目标查询
func (r *knowledgeApplyConfigRepositoryImpl) FindByTarget(ctx *database.TxContext, kbUUID, agentCode, scene, targetType, targetID string) (*domain.KnowledgeApplyConfig, error) {
	config := &domain.KnowledgeApplyConfig{}
	err := r.getOrmer(ctx).QueryTable(new(domain.KnowledgeApplyConfig)).
		Filter("knowledge_base_uuid", kbUUID).
		Filter("agent_code", agentCode).
		Filter("scene", scene).
		Filter("target_type", targetType).
		Filter("target_id", targetID).
		Filter("deleted_at__isnull", true).
		One(config)
	if err == orm.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return config, nil
}

// ListByConditions 条件查询
func (r *knowledgeApplyConfigRepositoryImpl) ListByConditions(ctx *database.TxContext, query *ListApplyConfigQuery) ([]*domain.KnowledgeApplyConfig, int64, error) {
	qs := r.getOrmer(ctx).QueryTable(new(domain.KnowledgeApplyConfig)).
		Filter("deleted_at__isnull", true)

	if query.KnowledgebaseUUID != "" {
		qs = qs.Filter("knowledge_base_uuid", query.KnowledgebaseUUID)
	}
	if query.AgentCode != "" {
		qs = qs.Filter("agent_code", query.AgentCode)
	}
	if query.Scene != "" {
		qs = qs.Filter("scene", query.Scene)
	}
	if query.TargetType != "" {
		qs = qs.Filter("target_type", query.TargetType)
	}
	if query.TargetID != "" {
		qs = qs.Filter("target_id", query.TargetID)
	}
	if query.Status != "" {
		qs = qs.Filter("status", query.Status)
	}
	if query.Operator != "" {
		qs = qs.Filter("operator", query.Operator)
	}

	// 统计总数
	total, err := qs.Count()
	if err != nil {
		return nil, 0, err
	}

	// 分页查询
	var list []*domain.KnowledgeApplyConfig
	_, err = qs.OrderBy("-updated_at").Limit(query.Limit, query.Offset).All(&list)
	if err != nil {
		return nil, 0, err
	}

	return list, total, nil
}

// UpdateStatus 更新状态
func (r *knowledgeApplyConfigRepositoryImpl) UpdateStatus(ctx *database.TxContext, uuid, status string) error {
	if status != domain.ApplyConfigStatusActive && status != domain.ApplyConfigStatusDisabled {
		return errors.New(errors.ErrCodeInvalidParam, "无效的状态值")
	}
	_, err := r.getOrmer(ctx).QueryTable(new(domain.KnowledgeApplyConfig)).
		Filter("uuid", uuid).
		Update(orm.Params{"status": status})
	return err
}

// UpdateVersion 更新版本号
func (r *knowledgeApplyConfigRepositoryImpl) UpdateVersion(ctx *database.TxContext, uuid, version, operator string) error {
	_, err := r.getOrmer(ctx).QueryTable(new(domain.KnowledgeApplyConfig)).
		Filter("uuid", uuid).
		Update(orm.Params{
			"version":  version,
			"operator": operator,
		})
	return err
}

// Delete 删除配置（软删除）
func (r *knowledgeApplyConfigRepositoryImpl) Delete(ctx *database.TxContext, uuid string) error {
	now := time.Now()
	_, err := r.getOrmer(ctx).QueryTable(new(domain.KnowledgeApplyConfig)).
		Filter("uuid", uuid).
		Filter("deleted_at__isnull", true).
		Update(orm.Params{"deleted_at": now})
	return err
}
