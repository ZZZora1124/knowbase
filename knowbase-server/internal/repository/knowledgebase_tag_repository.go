// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/beego/beego/v2/client/orm"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/domain"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/infrastructure/cache"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/infrastructure/database"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/errors"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/log"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/types/constants"
)

// KnowledgebaseTagRepository 知识本标签仓储接口
type KnowledgebaseTagRepository interface {
	// Create 创建标签
	Create(ctx *database.TxContext, tag *domain.KnowledgebaseTag) error

	// FindByUUID 根据UUID查询标签
	FindByUUID(ctx *database.TxContext, uuid string) (*domain.KnowledgebaseTag, error)

	// FindByKBUUIDAndName 根据知识本UUID和标签名称查询标签
	FindByKBUUIDAndName(ctx *database.TxContext, kbUUID, name string) (*domain.KnowledgebaseTag, error)

	// ListByKBUUID 查询知识本的所有标签
	ListByKBUUID(ctx *database.TxContext, kbUUID string) ([]*domain.KnowledgebaseTag, error)

	// Update 更新标签
	Update(ctx *database.TxContext, tag *domain.KnowledgebaseTag) error

	// SoftDelete 软删除标签
	SoftDelete(ctx *database.TxContext, uuid string) error

	// CountByKBUUID 统计知识本的标签数量
	CountByKBUUID(ctx *database.TxContext, kbUUID string) (int64, error)

	// CountByKBUUIDsBatch 批量统计多个知识本的标签数量
	CountByKBUUIDsBatch(ctx *database.TxContext, kbUUIDs []string) (map[string]int64, error)

	// FindByUUIDs 根据UUID列表批量查询标签
	FindByUUIDs(ctx *database.TxContext, uuids []string) ([]*domain.KnowledgebaseTag, error)

	// FindByNames 根据名称列表查询标签
	FindByNames(ctx *database.TxContext, kbUUID string, names []string) ([]*domain.KnowledgebaseTag, error)

	// InvalidateTagListCache 清除标签列表缓存
	InvalidateTagListCache(kbUUID string)

	// InvalidateTagCountCache 清除标签数量缓存
	InvalidateTagCountCache(kbUUID string)

	// FindDeletedByKBUUIDAndName 根据知识本UUID和标签名称查询已软删除的标签
	FindDeletedByKBUUIDAndName(ctx *database.TxContext, kbUUID, name string) (*domain.KnowledgebaseTag, error)

	// HardDeleteByKBUUIDAndName 物理删除指定知识本中指定名称的已软删除标签
	HardDeleteByKBUUIDAndName(ctx *database.TxContext, kbUUID, name string) error
}

// knowledgebaseTagRepositoryImpl 知识本标签仓储实现
type knowledgebaseTagRepositoryImpl struct {
	ormer orm.Ormer
}

// NewKnowledgebaseTagRepository 创建知识本标签仓储实例
func NewKnowledgebaseTagRepository(ormer orm.Ormer) KnowledgebaseTagRepository {
	return &knowledgebaseTagRepositoryImpl{
		ormer: ormer,
	}
}

// getOrmer 获取实际的 ormer，优先使用事务上下文中的 ormer
func (r *knowledgebaseTagRepositoryImpl) getOrmer(ctx *database.TxContext) orm.QueryExecutor {
	if ctx != nil && ctx.Ormer() != nil {
		return ctx.Ormer()
	}
	return r.ormer
}

// Create 创建标签
func (r *knowledgebaseTagRepositoryImpl) Create(ctx *database.TxContext, tag *domain.KnowledgebaseTag) error {
	_, err := r.getOrmer(ctx).Insert(tag)
	return err
}

// FindByUUID 根据UUID查询标签
func (r *knowledgebaseTagRepositoryImpl) FindByUUID(ctx *database.TxContext, uuid string) (*domain.KnowledgebaseTag, error) {
	tag := &domain.KnowledgebaseTag{}
	err := r.getOrmer(ctx).QueryTable(new(domain.KnowledgebaseTag)).
		Filter("uuid", uuid).
		Filter("deleted_at__isnull", true).
		One(tag)
	if err == orm.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return tag, nil
}

// FindByKBUUIDAndName 根据知识本UUID和标签名称查询标签
func (r *knowledgebaseTagRepositoryImpl) FindByKBUUIDAndName(
	ctx *database.TxContext, kbUUID, name string,
) (*domain.KnowledgebaseTag, error) {
	tag := &domain.KnowledgebaseTag{}
	err := r.getOrmer(ctx).QueryTable(new(domain.KnowledgebaseTag)).
		Filter("knowledgebase_uuid", kbUUID).
		Filter("name", name).
		Filter("deleted_at__isnull", true).
		One(tag)
	if err == orm.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return tag, nil
}

// ListByKBUUID 查询知识本的所有标签
func (r *knowledgebaseTagRepositoryImpl) ListByKBUUID(ctx *database.TxContext, kbUUID string) ([]*domain.KnowledgebaseTag, error) {
	// 事务内不使用缓存
	if ctx == nil {
		// 尝试从缓存获取
		cacher := cache.GetCache()
		cacheKey := fmt.Sprintf("%s%s", constants.CacheKeyTagList, kbUUID)
		cachedData, err := cacher.Get(context.Background(), cacheKey)
		if err == nil {
			// 缓存命中，解析数据
			tags := make([]*domain.KnowledgebaseTag, 0)
			if err := decodeFromJSON(cachedData, &tags); err == nil {
				return tags, nil
			}
		}
	}

	// 查询数据库
	tags := make([]*domain.KnowledgebaseTag, 0)
	_, err := r.getOrmer(ctx).QueryTable(new(domain.KnowledgebaseTag)).
		Filter("knowledgebase_uuid", kbUUID).
		Filter("deleted_at__isnull", true).
		OrderBy("name").
		All(&tags)
	if err != nil {
		return nil, err
	}

	// 写入缓存（仅在非事务环境）
	// 标签列表变更频率低，缓存1小时
	if ctx == nil {
		cacher := cache.GetCache()
		cacheKey := fmt.Sprintf("%s%s", constants.CacheKeyTagList, kbUUID)
		if data, err := encodeToJSON(tags); err == nil {
			if err := cacher.Set(context.Background(), cacheKey, data, time.Hour); err != nil {
				log.Logger.Warnf("缓存标签列表失败: %v, kbUUID: %s", err, kbUUID)
			}
		}
	}

	return tags, nil
}

// Update 更新标签
func (r *knowledgebaseTagRepositoryImpl) Update(ctx *database.TxContext, tag *domain.KnowledgebaseTag) error {
	_, err := r.getOrmer(ctx).Update(tag)
	return err
}

// SoftDelete 软删除标签
func (r *knowledgebaseTagRepositoryImpl) SoftDelete(ctx *database.TxContext, uuid string) error {
	now := time.Now()
	_, err := r.getOrmer(ctx).QueryTable(new(domain.KnowledgebaseTag)).
		Filter("uuid", uuid).
		Update(orm.Params{
			"deleted_at": now,
		})
	return err
}

// CountByKBUUID 统计知识本的标签数量
func (r *knowledgebaseTagRepositoryImpl) CountByKBUUID(ctx *database.TxContext, kbUUID string) (int64, error) {
	// 事务内不使用缓存
	if ctx == nil {
		// 尝试从缓存获取
		cacher := cache.GetCache()
		cacheKey := fmt.Sprintf("%s%s", constants.CacheKeyTagCount, kbUUID)
		cachedData, err := cacher.Get(context.Background(), cacheKey)
		if err == nil {
			// 缓存命中，解析数据
			var count int64
			if err := decodeFromJSON(cachedData, &count); err == nil {
				return count, nil
			}
		}
	}

	// 查询数据库
	count, err := r.getOrmer(ctx).QueryTable(new(domain.KnowledgebaseTag)).
		Filter("knowledgebase_uuid", kbUUID).
		Filter("deleted_at__isnull", true).
		Count()
	if err != nil {
		return 0, err
	}

	// 写入缓存（仅在非事务环境）
	// 标签数量变更频率低，缓存1小时
	if ctx == nil {
		cacher := cache.GetCache()
		cacheKey := fmt.Sprintf("%s%s", constants.CacheKeyTagCount, kbUUID)
		if data, err := encodeToJSON(count); err == nil {
			if err := cacher.Set(context.Background(), cacheKey, data, time.Hour); err != nil {
				log.Logger.Warnf("缓存标签数量失败: %v, kbUUID: %s", err, kbUUID)
			}
		}
	}

	return count, nil
}

// CountByKBUUIDsBatch 批量统计多个知识本的标签数量
func (r *knowledgebaseTagRepositoryImpl) CountByKBUUIDsBatch(ctx *database.TxContext, kbUUIDs []string) (map[string]int64, error) {
	result := make(map[string]int64)
	if len(kbUUIDs) == 0 {
		return result, nil
	}

	// 事务内不使用缓存，直接查询数据库
	if ctx != nil {
		return r.countByKBUUIDsBatchFromDB(ctx, kbUUIDs)
	}

	// 1. 先从缓存中逐个获取
	cacher := cache.GetCache()
	missingKBUUIDs := make([]string, 0)

	for _, kbUUID := range kbUUIDs {
		cacheKey := fmt.Sprintf("%s%s", constants.CacheKeyTagCount, kbUUID)
		cachedData, err := cacher.Get(context.Background(), cacheKey)
		if err == nil {
			var count int64
			if err := decodeFromJSON(cachedData, &count); err == nil {
				result[kbUUID] = count
				continue
			}
		}
		// 缓存未命中，记录需要查询数据库的知识本UUID
		missingKBUUIDs = append(missingKBUUIDs, kbUUID)
	}

	// 2. 如果所有数据都在缓存中，直接返回
	if len(missingKBUUIDs) == 0 {
		return result, nil
	}

	// 3. 批量查询缓存未命中的数据
	dbResult, err := r.countByKBUUIDsBatchFromDB(nil, missingKBUUIDs)
	if err != nil {
		return nil, err
	}

	// 4. 写入缓存并合并结果
	for _, kbUUID := range missingKBUUIDs {
		count := dbResult[kbUUID]
		cacheKey := fmt.Sprintf("%s%s", constants.CacheKeyTagCount, kbUUID)
		if data, err := encodeToJSON(count); err == nil {
			if err := cacher.Set(context.Background(), cacheKey, data, time.Hour); err != nil {
				log.Logger.Warnf("缓存标签数量失败: %v, kbUUID: %s", err, kbUUID)
			}
		}
		result[kbUUID] = count
	}

	return result, nil
}

// countByKBUUIDsBatchFromDB 从数据库批量统计标签数量
func (r *knowledgebaseTagRepositoryImpl) countByKBUUIDsBatchFromDB(ctx *database.TxContext, kbUUIDs []string) (map[string]int64, error) {
	result := make(map[string]int64)
	if len(kbUUIDs) == 0 {
		return result, nil
	}

	// 安全校验：限制批量查询数量，防止SQL语句过长和内存消耗过大
	if len(kbUUIDs) > 1000 {
		return nil, errors.New(errors.ErrCodeInvalidParam, "批量查询数量超过限制(最多1000个)")
	}

	// 初始化结果，确保所有知识本都有计数
	for _, uuid := range kbUUIDs {
		result[uuid] = 0
	}

	type countResult struct {
		KnowledgebaseUUID string `orm:"column(knowledgebase_uuid)"`
		Count             int64  `orm:"column(cnt)"`
	}

	var counts []countResult

	// 使用 Raw SQL 查询，避免 Beego ORM 对聚合表达式的解析问题
	placeholders := strings.Repeat("?,", len(kbUUIDs))
	placeholders = placeholders[:len(placeholders)-1]

	sql := fmt.Sprintf(`
		SELECT knowledgebase_uuid, COUNT(id) AS cnt
		FROM knowledgebase_tag
		WHERE knowledgebase_uuid IN (%s)
		AND deleted_at IS NULL
		GROUP BY knowledgebase_uuid
	`, placeholders)

	params := make([]interface{}, len(kbUUIDs))
	for i, v := range kbUUIDs {
		params[i] = v
	}

	_, err := r.getOrmer(ctx).Raw(sql, params...).QueryRows(&counts)
	if err != nil {
		return nil, err
	}

	for _, c := range counts {
		result[c.KnowledgebaseUUID] = c.Count
	}

	return result, nil
}

// FindByUUIDs 根据UUID列表批量查询标签
func (r *knowledgebaseTagRepositoryImpl) FindByUUIDs(ctx *database.TxContext, uuids []string) ([]*domain.KnowledgebaseTag, error) {
	if len(uuids) == 0 {
		return []*domain.KnowledgebaseTag{}, nil
	}

	tags := make([]*domain.KnowledgebaseTag, 0)
	_, err := r.getOrmer(ctx).QueryTable(new(domain.KnowledgebaseTag)).
		Filter("uuid__in", uuids).
		Filter("deleted_at__isnull", true).
		All(&tags)
	if err != nil {
		return nil, err
	}
	return tags, nil
}

// FindByNames 根据名称列表查询标签
func (r *knowledgebaseTagRepositoryImpl) FindByNames(ctx *database.TxContext, kbUUID string, names []string) ([]*domain.KnowledgebaseTag, error) {
	if len(names) == 0 {
		return []*domain.KnowledgebaseTag{}, nil
	}

	tags := make([]*domain.KnowledgebaseTag, 0)
	_, err := r.getOrmer(ctx).QueryTable(new(domain.KnowledgebaseTag)).
		Filter("knowledgebase_uuid", kbUUID).
		Filter("name__in", names).
		Filter("deleted_at__isnull", true).
		All(&tags)
	if err != nil {
		return nil, err
	}
	return tags, nil
}

// InvalidateTagListCache 清除标签列表缓存
func (r *knowledgebaseTagRepositoryImpl) InvalidateTagListCache(kbUUID string) {
	cacher := cache.GetCache()
	cacheKey := fmt.Sprintf("%s%s", constants.CacheKeyTagList, kbUUID)
	cacher.Delete(context.Background(), cacheKey)
}

// InvalidateTagCountCache 清除标签数量缓存
func (r *knowledgebaseTagRepositoryImpl) InvalidateTagCountCache(kbUUID string) {
	cacher := cache.GetCache()
	cacheKey := fmt.Sprintf("%s%s", constants.CacheKeyTagCount, kbUUID)
	cacher.Delete(context.Background(), cacheKey)
}

// encodeToJSON 将对象编码为JSON字符串
func encodeToJSON(v interface{}) (string, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// decodeFromJSON 从JSON字符串解码对象
func decodeFromJSON(data string, v interface{}) error {
	return json.Unmarshal([]byte(data), v)
}

// FindDeletedByKBUUIDAndName 根据知识本UUID和标签名称查询已软删除的标签
func (r *knowledgebaseTagRepositoryImpl) FindDeletedByKBUUIDAndName(
	ctx *database.TxContext, kbUUID, name string,
) (*domain.KnowledgebaseTag, error) {
	tag := &domain.KnowledgebaseTag{}
	err := r.getOrmer(ctx).QueryTable(new(domain.KnowledgebaseTag)).
		Filter("knowledgebase_uuid", kbUUID).
		Filter("name", name).
		Filter("deleted_at__isnull", false).
		One(tag)
	if err == orm.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return tag, nil
}

// HardDeleteByKBUUIDAndName 物理删除指定知识本中指定名称的已软删除标签
func (r *knowledgebaseTagRepositoryImpl) HardDeleteByKBUUIDAndName(
	ctx *database.TxContext, kbUUID, name string,
) error {
	_, err := r.getOrmer(ctx).QueryTable(new(domain.KnowledgebaseTag)).
		Filter("knowledgebase_uuid", kbUUID).
		Filter("name", name).
		Filter("deleted_at__isnull", false).
		Delete()
	return err
}
