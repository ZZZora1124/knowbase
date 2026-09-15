// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package repository

import (
	"fmt"
	"strings"
	"time"

	"github.com/beego/beego/v2/client/orm"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/domain"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/infrastructure/database"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/errors"
)

// KnowledgeApplyRecordRepository 知识应用记录仓储接口
type KnowledgeApplyRecordRepository interface {
	// BatchCreate 批量创建记录
	BatchCreate(ctx *database.TxContext, records []*domain.KnowledgeApplyRecord) error

	// FindByUUID 根据UUID查询
	FindByUUID(ctx *database.TxContext, uuid string) (*domain.KnowledgeApplyRecord, error)

	// ListByConfigUUIDAndVersion 根据配置UUID和版本查询
	ListByConfigUUIDAndVersion(ctx *database.TxContext, configUUID, configVersion string) ([]*domain.KnowledgeApplyRecord, error)

	// ListByConfigUUID 根据配置UUID查询所有记录（含历史）
	ListByConfigUUID(ctx *database.TxContext, configUUID string) ([]*domain.KnowledgeApplyRecord, error)

	// ListByKnowledgeUUID 根据知识UUID查询应用记录
	ListByKnowledgeUUID(ctx *database.TxContext, knowledgeUUID string) ([]*domain.KnowledgeApplyRecord, error)

	// UpdatePushStatus 更新推送状态
	UpdatePushStatus(ctx *database.TxContext, uuid, status, errorMsg string) error

	// BatchUpdatePushStatus 批量更新推送状态
	BatchUpdatePushStatus(ctx *database.TxContext, uuids []string, status, errorMsg string) error

	// ListPendingRecords 查询待推送的记录
	ListPendingRecords(ctx *database.TxContext, limit int) ([]*domain.KnowledgeApplyRecord, error)

	// ListFailedRecordsByConfigUUID 查询配置下失败的记录
	ListFailedRecordsByConfigUUID(ctx *database.TxContext, configUUID string) ([]*domain.KnowledgeApplyRecord, error)

	// DeleteByConfigUUID 删除配置下的所有记录
	DeleteByConfigUUID(ctx *database.TxContext, configUUID string) error

	// GetDistinctVersionsByConfigUUID 获取配置的所有版本号（去重）
	GetDistinctVersionsByConfigUUID(ctx *database.TxContext, configUUID string) ([]string, error)
}

// knowledgeApplyRecordRepositoryImpl 知识应用记录仓储实现
type knowledgeApplyRecordRepositoryImpl struct {
	ormer orm.Ormer
}

// NewKnowledgeApplyRecordRepository 创建知识应用记录仓储实例
func NewKnowledgeApplyRecordRepository(ormer orm.Ormer) KnowledgeApplyRecordRepository {
	return &knowledgeApplyRecordRepositoryImpl{ormer: ormer}
}

// getOrmer 获取实际的 ormer
func (r *knowledgeApplyRecordRepositoryImpl) getOrmer(ctx *database.TxContext) orm.QueryExecutor {
	if ctx != nil && ctx.Ormer() != nil {
		return ctx.Ormer()
	}
	return r.ormer
}

// BatchCreate 批量创建记录
func (r *knowledgeApplyRecordRepositoryImpl) BatchCreate(ctx *database.TxContext, records []*domain.KnowledgeApplyRecord) error {
	if len(records) == 0 {
		return nil
	}

	if len(records) > 100 {
		return errors.New(errors.ErrCodeInvalidParam, "批量创建数量超过限制(最多100个)")
	}

	// 构建批量插入SQL
	var values []string
	var args []any
	for _, rec := range records {
		values = append(values, "(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")

		// 如果未设置 CreatedAt，使用当前时间
		createdAt := rec.CreatedAt
		if createdAt.IsZero() {
			createdAt = time.Now()
		}

		// entry_type 默认为 knowledge
		entryType := rec.EntryType
		if entryType == "" {
			entryType = domain.EntryTypeKnowledge
		}

		// directory_uuid 为空时落库为 NULL，便于索引
		var directoryUUID any
		if rec.DirectoryUUID != "" {
			directoryUUID = rec.DirectoryUUID
		}

		args = append(args,
			rec.UUID, rec.ConfigUUID, rec.ConfigVersion,
			rec.KnowledgebaseUUID, rec.KnowledgeUUID, rec.KnowledgeVersion,
			entryType, directoryUUID,
			rec.Operator, rec.PushStatus, createdAt)
	}

	sql := fmt.Sprintf(`
		INSERT INTO %s (
			uuid, config_uuid, config_version,
			knowledge_base_uuid, knowledge_uuid, knowledge_version,
			entry_type, directory_uuid,
			operator, push_status, created_at
		) VALUES %s
	`, new(domain.KnowledgeApplyRecord).TableName(), strings.Join(values, ","))

	_, err := r.getOrmer(ctx).Raw(sql, args...).Exec()
	return err
}

// FindByUUID 根据UUID查询
func (r *knowledgeApplyRecordRepositoryImpl) FindByUUID(ctx *database.TxContext, uuid string) (*domain.KnowledgeApplyRecord, error) {
	record := &domain.KnowledgeApplyRecord{}
	err := r.getOrmer(ctx).QueryTable(new(domain.KnowledgeApplyRecord)).
		Filter("uuid", uuid).
		One(record)
	if err == orm.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return record, nil
}

// ListByConfigUUIDAndVersion 根据配置UUID和版本查询
func (r *knowledgeApplyRecordRepositoryImpl) ListByConfigUUIDAndVersion(ctx *database.TxContext, configUUID, configVersion string) ([]*domain.KnowledgeApplyRecord, error) {
	var list []*domain.KnowledgeApplyRecord
	_, err := r.getOrmer(ctx).QueryTable(new(domain.KnowledgeApplyRecord)).
		Filter("config_uuid", configUUID).
		Filter("config_version", configVersion).
		OrderBy("-created_at").
		All(&list)
	if err != nil {
		return nil, err
	}
	return list, nil
}

// ListByConfigUUID 根据配置UUID查询所有记录
func (r *knowledgeApplyRecordRepositoryImpl) ListByConfigUUID(ctx *database.TxContext, configUUID string) ([]*domain.KnowledgeApplyRecord, error) {
	var list []*domain.KnowledgeApplyRecord
	_, err := r.getOrmer(ctx).QueryTable(new(domain.KnowledgeApplyRecord)).
		Filter("config_uuid", configUUID).
		OrderBy("-created_at").
		All(&list)
	if err != nil {
		return nil, err
	}
	return list, nil
}

// ListByKnowledgeUUID 根据知识UUID查询应用记录
func (r *knowledgeApplyRecordRepositoryImpl) ListByKnowledgeUUID(ctx *database.TxContext, knowledgeUUID string) ([]*domain.KnowledgeApplyRecord, error) {
	var list []*domain.KnowledgeApplyRecord
	_, err := r.getOrmer(ctx).QueryTable(new(domain.KnowledgeApplyRecord)).
		Filter("knowledge_uuid", knowledgeUUID).
		OrderBy("-created_at").
		All(&list)
	if err != nil {
		return nil, err
	}
	return list, nil
}

// UpdatePushStatus 更新推送状态
func (r *knowledgeApplyRecordRepositoryImpl) UpdatePushStatus(ctx *database.TxContext, uuid, status, errorMsg string) error {
	params := orm.Params{
		"push_status": status,
	}
	if status == domain.PushStatusSuccess || status == domain.PushStatusFailed {
		now := time.Now()
		params["pushed_at"] = &now
	}
	if errorMsg != "" {
		params["push_error_msg"] = errorMsg
	}
	_, err := r.getOrmer(ctx).QueryTable(new(domain.KnowledgeApplyRecord)).
		Filter("uuid", uuid).
		Update(params)
	return err
}

// BatchUpdatePushStatus 批量更新推送状态
func (r *knowledgeApplyRecordRepositoryImpl) BatchUpdatePushStatus(ctx *database.TxContext, uuids []string, status, errorMsg string) error {
	if len(uuids) == 0 {
		return nil
	}

	params := orm.Params{
		"push_status": status,
	}
	if status == domain.PushStatusSuccess || status == domain.PushStatusFailed {
		now := time.Now()
		params["pushed_at"] = &now
	}
	if errorMsg != "" {
		params["push_error_msg"] = errorMsg
	}

	_, err := r.getOrmer(ctx).QueryTable(new(domain.KnowledgeApplyRecord)).
		Filter("uuid__in", uuids).
		Update(params)
	return err
}

// ListPendingRecords 查询待推送的记录
func (r *knowledgeApplyRecordRepositoryImpl) ListPendingRecords(ctx *database.TxContext, limit int) ([]*domain.KnowledgeApplyRecord, error) {
	var list []*domain.KnowledgeApplyRecord
	_, err := r.getOrmer(ctx).QueryTable(new(domain.KnowledgeApplyRecord)).
		Filter("push_status", domain.PushStatusPending).
		OrderBy("created_at").
		Limit(limit).
		All(&list)
	if err != nil {
		return nil, err
	}
	return list, nil
}

// ListFailedRecordsByConfigUUID 查询配置下失败的记录
func (r *knowledgeApplyRecordRepositoryImpl) ListFailedRecordsByConfigUUID(ctx *database.TxContext, configUUID string) ([]*domain.KnowledgeApplyRecord, error) {
	var list []*domain.KnowledgeApplyRecord
	_, err := r.getOrmer(ctx).QueryTable(new(domain.KnowledgeApplyRecord)).
		Filter("config_uuid", configUUID).
		Filter("push_status", domain.PushStatusFailed).
		All(&list)
	if err != nil {
		return nil, err
	}
	return list, nil
}

// DeleteByConfigUUID 删除配置下的所有记录
func (r *knowledgeApplyRecordRepositoryImpl) DeleteByConfigUUID(ctx *database.TxContext, configUUID string) error {
	_, err := r.getOrmer(ctx).QueryTable(new(domain.KnowledgeApplyRecord)).
		Filter("config_uuid", configUUID).
		Delete()
	return err
}

// GetDistinctVersionsByConfigUUID 获取配置的所有版本号
func (r *knowledgeApplyRecordRepositoryImpl) GetDistinctVersionsByConfigUUID(ctx *database.TxContext, configUUID string) ([]string, error) {
	type VersionResult struct {
		ConfigVersion string `orm:"column(config_version)"`
	}

	var results []VersionResult
	sql := fmt.Sprintf(`
		SELECT DISTINCT config_version
		FROM %s
		WHERE config_uuid = ?
		ORDER BY created_at DESC
	`, new(domain.KnowledgeApplyRecord).TableName())

	_, err := r.getOrmer(ctx).Raw(sql, configUUID).QueryRows(&results)
	if err != nil {
		return nil, err
	}

	versions := make([]string, len(results))
	for i, v := range results {
		versions[i] = v.ConfigVersion
	}
	return versions, nil
}
