// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package repository

import (
	"fmt"
	"strings"

	"github.com/beego/beego/v2/client/orm"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/domain"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/infrastructure/database"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/errors"
)

// KnowledgeVersionQuery 版本查询参数
type KnowledgeVersionQuery struct {
	KnowledgeUUID string
	Version       string // 为空时查询最新版本
}

// KnowledgeVersionRepository 知识版本仓储接口
type KnowledgeVersionRepository interface {
	// Create 创建版本
	Create(ctx *database.TxContext, version *domain.KnowledgeVersion) error

	// FindByUUID 根据UUID查询版本
	FindByUUID(ctx *database.TxContext, uuid string) (*domain.KnowledgeVersion, error)

	// FindByKnowledgeUUIDAndVersion 根据知识UUID和版本号查询
	FindByKnowledgeUUIDAndVersion(ctx *database.TxContext, knowledgeUUID, version string) (*domain.KnowledgeVersion, error)

	// ListByKnowledgeUUID 根据知识UUID查询所有版本（按创建时间倒序）
	ListByKnowledgeUUID(ctx *database.TxContext, knowledgeUUID string) ([]*domain.KnowledgeVersion, error)

	// FindLatestByKnowledgeUUID 获取知识的最新版本
	FindLatestByKnowledgeUUID(ctx *database.TxContext, knowledgeUUID string) (*domain.KnowledgeVersion, error)

	// BatchFindByQueries 批量查询（支持指定版本或最新版本）
	BatchFindByQueries(ctx *database.TxContext, queries []KnowledgeVersionQuery) ([]*domain.KnowledgeVersion, error)

	// BatchListByKnowledgeUUIDs 批量获取多个知识的版本列表
	BatchListByKnowledgeUUIDs(ctx *database.TxContext, knowledgeUUIDs []string) (map[string][]*domain.KnowledgeVersion, error)

	// ExistsByKnowledgeUUIDAndVersion 检查版本是否存在
	ExistsByKnowledgeUUIDAndVersion(ctx *database.TxContext, knowledgeUUID, version string) (bool, error)

	// CountByKnowledgeUUID 统计知识的版本数量
	CountByKnowledgeUUID(ctx *database.TxContext, knowledgeUUID string) (int64, error)

	// DeleteByKnowledgeUUID 删除知识的所有版本（级联删除时使用）
	DeleteByKnowledgeUUID(ctx *database.TxContext, knowledgeUUID string) error
}

// knowledgeVersionRepositoryImpl 知识版本仓储实现
type knowledgeVersionRepositoryImpl struct {
	ormer orm.Ormer
}

// NewKnowledgeVersionRepository 创建知识版本仓储实例
func NewKnowledgeVersionRepository(ormer orm.Ormer) KnowledgeVersionRepository {
	return &knowledgeVersionRepositoryImpl{ormer: ormer}
}

// getOrmer 获取实际的 ormer
func (r *knowledgeVersionRepositoryImpl) getOrmer(ctx *database.TxContext) orm.QueryExecutor {
	if ctx != nil && ctx.Ormer() != nil {
		return ctx.Ormer()
	}
	return r.ormer
}

// Create 创建版本
func (r *knowledgeVersionRepositoryImpl) Create(ctx *database.TxContext, version *domain.KnowledgeVersion) error {
	_, err := r.getOrmer(ctx).Insert(version)
	return err
}

// FindByUUID 根据UUID查询版本
func (r *knowledgeVersionRepositoryImpl) FindByUUID(ctx *database.TxContext, uuid string) (*domain.KnowledgeVersion, error) {
	version := &domain.KnowledgeVersion{}
	err := r.getOrmer(ctx).QueryTable(new(domain.KnowledgeVersion)).
		Filter("uuid", uuid).
		One(version)
	if err == orm.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return version, nil
}

// FindByKnowledgeUUIDAndVersion 根据知识UUID和版本号查询
func (r *knowledgeVersionRepositoryImpl) FindByKnowledgeUUIDAndVersion(ctx *database.TxContext, knowledgeUUID, version string) (*domain.KnowledgeVersion, error) {
	kv := &domain.KnowledgeVersion{}
	err := r.getOrmer(ctx).QueryTable(new(domain.KnowledgeVersion)).
		Filter("knowledge_uuid", knowledgeUUID).
		Filter("version", version).
		One(kv)
	if err == orm.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return kv, nil
}

// ListByKnowledgeUUID 根据知识UUID查询所有版本（按创建时间倒序）
func (r *knowledgeVersionRepositoryImpl) ListByKnowledgeUUID(ctx *database.TxContext, knowledgeUUID string) ([]*domain.KnowledgeVersion, error) {
	var list []*domain.KnowledgeVersion
	_, err := r.getOrmer(ctx).QueryTable(new(domain.KnowledgeVersion)).
		Filter("knowledge_uuid", knowledgeUUID).
		OrderBy("-created_at").
		All(&list)
	if err != nil {
		return nil, err
	}
	return list, nil
}

// FindLatestByKnowledgeUUID 获取知识的最新版本
func (r *knowledgeVersionRepositoryImpl) FindLatestByKnowledgeUUID(ctx *database.TxContext, knowledgeUUID string) (*domain.KnowledgeVersion, error) {
	version := &domain.KnowledgeVersion{}
	err := r.getOrmer(ctx).QueryTable(new(domain.KnowledgeVersion)).
		Filter("knowledge_uuid", knowledgeUUID).
		OrderBy("-created_at").
		Limit(1).
		One(version)
	if err == orm.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return version, nil
}

// BatchFindByQueries 批量查询（支持指定版本或最新版本）
func (r *knowledgeVersionRepositoryImpl) BatchFindByQueries(ctx *database.TxContext, queries []KnowledgeVersionQuery) ([]*domain.KnowledgeVersion, error) {
	if len(queries) == 0 {
		return []*domain.KnowledgeVersion{}, nil
	}

	if len(queries) > 100 {
		return nil, errors.New(errors.ErrCodeInvalidParam, "批量查询数量超过限制(最多100个)")
	}

	var result []*domain.KnowledgeVersion

	// 分离指定版本和最新版本的查询
	var specificVersionQueries []KnowledgeVersionQuery
	var latestVersionKnowledgeUUIDs []string

	for _, q := range queries {
		if q.Version != "" {
			specificVersionQueries = append(specificVersionQueries, q)
		} else {
			latestVersionKnowledgeUUIDs = append(latestVersionKnowledgeUUIDs, q.KnowledgeUUID)
		}
	}

	// 查询指定版本
	for _, q := range specificVersionQueries {
		version, err := r.FindByKnowledgeUUIDAndVersion(ctx, q.KnowledgeUUID, q.Version)
		if err != nil {
			return nil, err
		}
		if version != nil {
			result = append(result, version)
		}
	}

	// 查询最新版本
	for _, knowledgeUUID := range latestVersionKnowledgeUUIDs {
		version, err := r.FindLatestByKnowledgeUUID(ctx, knowledgeUUID)
		if err != nil {
			return nil, err
		}
		if version != nil {
			result = append(result, version)
		}
	}

	return result, nil
}

// BatchListByKnowledgeUUIDs 批量获取多个知识的版本列表
func (r *knowledgeVersionRepositoryImpl) BatchListByKnowledgeUUIDs(ctx *database.TxContext, knowledgeUUIDs []string) (map[string][]*domain.KnowledgeVersion, error) {
	result := make(map[string][]*domain.KnowledgeVersion)
	if len(knowledgeUUIDs) == 0 {
		return result, nil
	}

	if len(knowledgeUUIDs) > 100 {
		return nil, errors.New(errors.ErrCodeInvalidParam, "批量查询数量超过限制(最多100个)")
	}

	// 初始化空列表
	for _, uuid := range knowledgeUUIDs {
		result[uuid] = []*domain.KnowledgeVersion{}
	}

	var list []*domain.KnowledgeVersion
	_, err := r.getOrmer(ctx).QueryTable(new(domain.KnowledgeVersion)).
		Filter("knowledge_uuid__in", knowledgeUUIDs).
		OrderBy("-created_at").
		All(&list)
	if err != nil {
		return nil, err
	}

	// 按知识UUID分组
	for _, v := range list {
		result[v.KnowledgeUUID] = append(result[v.KnowledgeUUID], v)
	}

	return result, nil
}

// ExistsByKnowledgeUUIDAndVersion 检查版本是否存在
func (r *knowledgeVersionRepositoryImpl) ExistsByKnowledgeUUIDAndVersion(ctx *database.TxContext, knowledgeUUID, version string) (bool, error) {
	count, err := r.getOrmer(ctx).QueryTable(new(domain.KnowledgeVersion)).
		Filter("knowledge_uuid", knowledgeUUID).
		Filter("version", version).
		Count()
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// CountByKnowledgeUUID 统计知识的版本数量
func (r *knowledgeVersionRepositoryImpl) CountByKnowledgeUUID(ctx *database.TxContext, knowledgeUUID string) (int64, error) {
	return r.getOrmer(ctx).QueryTable(new(domain.KnowledgeVersion)).
		Filter("knowledge_uuid", knowledgeUUID).
		Count()
}

// DeleteByKnowledgeUUID 删除知识的所有版本
func (r *knowledgeVersionRepositoryImpl) DeleteByKnowledgeUUID(ctx *database.TxContext, knowledgeUUID string) error {
	// 构建占位符（支持SQL注入防护）
	placeholders := "?"
	sql := fmt.Sprintf("DELETE FROM %s WHERE knowledge_uuid IN (%s)",
		new(domain.KnowledgeVersion).TableName(), placeholders)
	_, err := r.getOrmer(ctx).Raw(sql, knowledgeUUID).Exec()
	return err
}

// 补充：支持原生SQL的批量插入方法（高性能场景）
func (r *knowledgeVersionRepositoryImpl) BatchCreate(ctx *database.TxContext, versions []*domain.KnowledgeVersion) error {
	if len(versions) == 0 {
		return nil
	}

	if len(versions) > 100 {
		return errors.New(errors.ErrCodeInvalidParam, "批量创建数量超过限制(最多100个)")
	}

	// 构建批量插入SQL
	var values []string
	var args []interface{}
	for _, v := range versions {
		values = append(values, "(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")
		args = append(args,
			v.UUID, v.KnowledgeUUID, v.Version, v.Title,
			v.Content, v.FilePath, v.FileSize, v.FileFormat,
			v.Metadata, v.Operator, v.Operation, v.Remark)
	}

	sql := fmt.Sprintf(`
		INSERT INTO %s (
			uuid, knowledge_uuid, version, title,
			content, file_path, file_size, file_format,
			metadata, operator, operation, remark
		) VALUES %s
	`, new(domain.KnowledgeVersion).TableName(), strings.Join(values, ","))

	_, err := r.getOrmer(ctx).Raw(sql, args...).Exec()
	return err
}

