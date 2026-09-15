// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package repository

import (
	"time"

	"github.com/beego/beego/v2/client/orm"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/domain"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/infrastructure/database"
)

// KnowledgebaseDirectoryRepository 目录仓储接口
type KnowledgebaseDirectoryRepository interface {
	// Create 创建目录
	Create(ctx *database.TxContext, dir *domain.KnowledgebaseDirectory) error

	// FindByUUID 根据UUID查询目录
	FindByUUID(ctx *database.TxContext, uuid string) (*domain.KnowledgebaseDirectory, error)

	// FindByUUIDs 根据UUID列表批量查询目录
	FindByUUIDs(ctx *database.TxContext, uuids []string) ([]*domain.KnowledgebaseDirectory, error)

	// FindByKBUUIDAndName 根据知识本UUID和名称查询目录（只返回第一条）
	FindByKBUUIDAndName(ctx *database.TxContext, kbUUID, name string) (*domain.KnowledgebaseDirectory, error)

	// FindByKBUUIDAndNames 根据知识本UUID和名称列表查询所有匹配目录（多级目录下同名目录可能存在于不同层级）
	FindByKBUUIDAndNames(ctx *database.TxContext, kbUUID string, names []string) ([]*domain.KnowledgebaseDirectory, error)

	// ListByKBUUID 查询知识本的所有目录（按名称排序）
	ListByKBUUID(ctx *database.TxContext, kbUUID string) ([]*domain.KnowledgebaseDirectory, error)

	// Update 更新目录
	Update(ctx *database.TxContext, dir *domain.KnowledgebaseDirectory) error

	// SoftDelete 软删除目录
	SoftDelete(ctx *database.TxContext, uuid string) error

	// CountByKBUUID 统计知识本的目录数量
	CountByKBUUID(ctx *database.TxContext, kbUUID string) (int64, error)

	// CountKnowledgeByDirectory 统计目录下的知识数量
	CountKnowledgeByDirectory(ctx *database.TxContext, dirUUID string) (int64, error)

	// FindByKBUUIDAndNameAndParent 同级同名唯一性检查
	// parentUUID 为 nil 表示根目录层级
	FindByKBUUIDAndNameAndParent(ctx *database.TxContext, kbUUID, name string, parentUUID *string) (*domain.KnowledgebaseDirectory, error)

	// UpdateParentUUID 更新单个目录的父节点（移动目录用）
	UpdateParentUUID(ctx *database.TxContext, uuid string, parentUUID *string) error

	// UpdateParentUUIDBatch 批量更新直接子目录的 parent_uuid（子节点提升用）
	// 将 parent_uuid == oldParentUUID 的所有未删除目录更新为 newParentUUID
	UpdateParentUUIDBatch(ctx *database.TxContext, oldParentUUID string, newParentUUID *string) error

	// SoftDeleteBatch 批量软删除目录（recursive 模式用）
	SoftDeleteBatch(ctx *database.TxContext, uuids []string) error

	// CountChildren 统计指定目录的直接子目录数量
	CountChildren(ctx *database.TxContext, dirUUID string) (int64, error)

	// FindDefaultByKBUUID 查询知识本的默认目录（is_default=true）
	FindDefaultByKBUUID(ctx *database.TxContext, kbUUID string) (*domain.KnowledgebaseDirectory, error)
}

// knowledgebaseDirectoryRepositoryImpl 目录仓储实现
type knowledgebaseDirectoryRepositoryImpl struct {
	ormer orm.Ormer
}

// NewKnowledgebaseDirectoryRepository 创建目录仓储实例
func NewKnowledgebaseDirectoryRepository(ormer orm.Ormer) KnowledgebaseDirectoryRepository {
	return &knowledgebaseDirectoryRepositoryImpl{ormer: ormer}
}

// getOrmer 获取实际的 ormer
func (r *knowledgebaseDirectoryRepositoryImpl) getOrmer(ctx *database.TxContext) orm.QueryExecutor {
	if ctx != nil && ctx.Ormer() != nil {
		return ctx.Ormer()
	}
	return r.ormer
}

// Create 创建目录
func (r *knowledgebaseDirectoryRepositoryImpl) Create(ctx *database.TxContext, dir *domain.KnowledgebaseDirectory) error {
	_, err := r.getOrmer(ctx).Insert(dir)
	return err
}

// FindByUUID 根据UUID查询目录
func (r *knowledgebaseDirectoryRepositoryImpl) FindByUUID(ctx *database.TxContext, uuid string) (*domain.KnowledgebaseDirectory, error) {
	dir := &domain.KnowledgebaseDirectory{}
	err := r.getOrmer(ctx).QueryTable(new(domain.KnowledgebaseDirectory)).
		Filter("uuid", uuid).
		Filter("deleted_at__isnull", true).
		One(dir)
	if err == orm.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return dir, nil
}

// FindByUUIDs 根据UUID列表批量查询目录
func (r *knowledgebaseDirectoryRepositoryImpl) FindByUUIDs(ctx *database.TxContext, uuids []string) ([]*domain.KnowledgebaseDirectory, error) {
	if len(uuids) == 0 {
		return []*domain.KnowledgebaseDirectory{}, nil
	}
	dirs := make([]*domain.KnowledgebaseDirectory, 0)
	_, err := r.getOrmer(ctx).QueryTable(new(domain.KnowledgebaseDirectory)).
		Filter("uuid__in", uuids).
		Filter("deleted_at__isnull", true).
		All(&dirs)
	if err != nil {
		return nil, err
	}
	return dirs, nil
}

// FindByKBUUIDAndName 根据知识本UUID和名称查询目录
func (r *knowledgebaseDirectoryRepositoryImpl) FindByKBUUIDAndName(ctx *database.TxContext, kbUUID, name string) (*domain.KnowledgebaseDirectory, error) {
	dir := &domain.KnowledgebaseDirectory{}
	err := r.getOrmer(ctx).QueryTable(new(domain.KnowledgebaseDirectory)).
		Filter("knowledgebase_uuid", kbUUID).
		Filter("name", name).
		Filter("deleted_at__isnull", true).
		One(dir)
	if err == orm.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return dir, nil
}

// FindByKBUUIDAndNames 根据知识本UUID和名称列表查询所有匹配目录
func (r *knowledgebaseDirectoryRepositoryImpl) FindByKBUUIDAndNames(ctx *database.TxContext, kbUUID string, names []string) ([]*domain.KnowledgebaseDirectory, error) {
	if len(names) == 0 {
		return []*domain.KnowledgebaseDirectory{}, nil
	}
	dirs := make([]*domain.KnowledgebaseDirectory, 0)
	_, err := r.getOrmer(ctx).QueryTable(new(domain.KnowledgebaseDirectory)).
		Filter("knowledgebase_uuid", kbUUID).
		Filter("name__in", names).
		Filter("deleted_at__isnull", true).
		All(&dirs)
	if err != nil {
		return nil, err
	}
	return dirs, nil
}

// ListByKBUUID 查询知识本的所有目录（按名称排序）
func (r *knowledgebaseDirectoryRepositoryImpl) ListByKBUUID(ctx *database.TxContext, kbUUID string) ([]*domain.KnowledgebaseDirectory, error) {
	dirs := make([]*domain.KnowledgebaseDirectory, 0)
	_, err := r.getOrmer(ctx).QueryTable(new(domain.KnowledgebaseDirectory)).
		Filter("knowledgebase_uuid", kbUUID).
		Filter("deleted_at__isnull", true).
		OrderBy("name").
		All(&dirs)
	return dirs, err
}

// Update 更新目录
func (r *knowledgebaseDirectoryRepositoryImpl) Update(ctx *database.TxContext, dir *domain.KnowledgebaseDirectory) error {
	_, err := r.getOrmer(ctx).QueryTable(new(domain.KnowledgebaseDirectory)).
		Filter("uuid", dir.UUID).
		Update(orm.Params{
			"name": dir.Name,
		})
	return err
}

// SoftDelete 软删除目录
func (r *knowledgebaseDirectoryRepositoryImpl) SoftDelete(ctx *database.TxContext, uuid string) error {
	_, err := r.getOrmer(ctx).QueryTable(new(domain.KnowledgebaseDirectory)).
		Filter("uuid", uuid).
		Update(orm.Params{
			"deleted_at": time.Now(),
		})
	return err
}

// CountByKBUUID 统计知识本的目录数量
func (r *knowledgebaseDirectoryRepositoryImpl) CountByKBUUID(ctx *database.TxContext, kbUUID string) (int64, error) {
	return r.getOrmer(ctx).QueryTable(new(domain.KnowledgebaseDirectory)).
		Filter("knowledgebase_uuid", kbUUID).
		Filter("deleted_at__isnull", true).
		Count()
}

// CountKnowledgeByDirectory 统计目录下的知识数量
func (r *knowledgebaseDirectoryRepositoryImpl) CountKnowledgeByDirectory(ctx *database.TxContext, dirUUID string) (int64, error) {
	return r.getOrmer(ctx).QueryTable(new(domain.Knowledge)).
		Filter("directory_uuid", dirUUID).
		Filter("deleted_at__isnull", true).
		Count()
}

// FindByKBUUIDAndNameAndParent 同级同名唯一性检查
func (r *knowledgebaseDirectoryRepositoryImpl) FindByKBUUIDAndNameAndParent(ctx *database.TxContext, kbUUID, name string, parentUUID *string) (*domain.KnowledgebaseDirectory, error) {
	dir := &domain.KnowledgebaseDirectory{}
	qs := r.getOrmer(ctx).QueryTable(new(domain.KnowledgebaseDirectory)).
		Filter("knowledgebase_uuid", kbUUID).
		Filter("name", name).
		Filter("deleted_at__isnull", true)
	if parentUUID == nil {
		qs = qs.Filter("parent_uuid__isnull", true)
	} else {
		qs = qs.Filter("parent_uuid", *parentUUID)
	}
	err := qs.One(dir)
	if err == orm.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return dir, nil
}

// UpdateParentUUID 更新单个目录的父节点
func (r *knowledgebaseDirectoryRepositoryImpl) UpdateParentUUID(ctx *database.TxContext, uuid string, parentUUID *string) error {
	_, err := r.getOrmer(ctx).QueryTable(new(domain.KnowledgebaseDirectory)).
		Filter("uuid", uuid).
		Update(orm.Params{"parent_uuid": parentUUID})
	return err
}

// UpdateParentUUIDBatch 批量更新直接子目录的 parent_uuid（子节点提升）
func (r *knowledgebaseDirectoryRepositoryImpl) UpdateParentUUIDBatch(ctx *database.TxContext, oldParentUUID string, newParentUUID *string) error {
	_, err := r.getOrmer(ctx).QueryTable(new(domain.KnowledgebaseDirectory)).
		Filter("parent_uuid", oldParentUUID).
		Filter("deleted_at__isnull", true).
		Update(orm.Params{"parent_uuid": newParentUUID})
	return err
}

// SoftDeleteBatch 批量软删除目录
func (r *knowledgebaseDirectoryRepositoryImpl) SoftDeleteBatch(ctx *database.TxContext, uuids []string) error {
	if len(uuids) == 0 {
		return nil
	}
	_, err := r.getOrmer(ctx).QueryTable(new(domain.KnowledgebaseDirectory)).
		Filter("uuid__in", uuids).
		Update(orm.Params{"deleted_at": time.Now()})
	return err
}

// CountChildren 统计指定目录的直接子目录数量
func (r *knowledgebaseDirectoryRepositoryImpl) CountChildren(ctx *database.TxContext, dirUUID string) (int64, error) {
	count, err := r.getOrmer(ctx).QueryTable(new(domain.KnowledgebaseDirectory)).
		Filter("parent_uuid", dirUUID).
		Filter("deleted_at__isnull", true).
		Count()
	return count, err
}

// FindDefaultByKBUUID 查询知识本的默认目录（is_default=true）
func (r *knowledgebaseDirectoryRepositoryImpl) FindDefaultByKBUUID(ctx *database.TxContext, kbUUID string) (*domain.KnowledgebaseDirectory, error) {
	dir := &domain.KnowledgebaseDirectory{}
	err := r.getOrmer(ctx).QueryTable(new(domain.KnowledgebaseDirectory)).
		Filter("knowledgebase_uuid", kbUUID).
		Filter("is_default", true).
		Filter("deleted_at__isnull", true).
		One(dir)
	if err == orm.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return dir, nil
}
