// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package repository

import (
	"context"
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

// KnowledgeRepository 知识仓储接口
type KnowledgeRepository interface {
	// Create 创建知识
	Create(ctx *database.TxContext, knowledge *domain.Knowledge) error

	// FindByUUID 根据UUID查询知识
	FindByUUID(ctx *database.TxContext, uuid string) (*domain.Knowledge, error)

	// FindByUUIDs 根据UUID列表批量查询知识
	FindByUUIDs(ctx *database.TxContext, uuids []string) ([]*domain.Knowledge, error)

	// Update 更新知识
	Update(ctx *database.TxContext, knowledge *domain.Knowledge) error

	// SoftDelete 软删除知识
	SoftDelete(ctx *database.TxContext, uuid string) error

	// SoftDeleteByDirectory 软删除指定目录下的所有知识
	SoftDeleteByDirectory(ctx *database.TxContext, dirUUID string) error

	// SoftDeleteByDirectories 批量软删除多个目录下的所有知识
	SoftDeleteByDirectories(ctx *database.TxContext, dirUUIDs []string) error

	// ListByConditions 根据条件查询知识列表
	ListByConditions(ctx *database.TxContext, req *ListKnowledgeQuery) ([]*domain.Knowledge, int64, error)

	// ListByTags 根据标签查询知识
	ListByTags(ctx *database.TxContext, req *ListByTagsQuery) ([]*domain.Knowledge, int64, error)

	// CountByKBUUID 统计知识本的知识数量
	CountByKBUUID(ctx *database.TxContext, kbUUID string) (int64, error)

	// CountByKBUUIDsBatch 批量统计多个知识本的知识数量
	CountByKBUUIDsBatch(ctx *database.TxContext, kbUUIDs []string) (map[string]int64, error)

	// CountByDirectory 统计目录下的知识数量
	CountByDirectory(ctx *database.TxContext, dirUUID string) (int64, error)

	// CountByDirectories 批量统计多个目录下的知识数量
	CountByDirectories(ctx *database.TxContext, dirUUIDs []string) (map[string]int64, error)

	// CountByNullDirectory 统计没有 dirUUID 的知识数量（默认目录下的知识）
	CountByNullDirectory(ctx *database.TxContext, kbUUID string) (int64, error)

	// UnlinkDirectory 解除知识-目录关联（将知识的directory_uuid设为NULL）
	UnlinkDirectory(ctx *database.TxContext, dirUUID string) error

	// UpdateDirectory 更新知识的目录
	UpdateDirectory(ctx *database.TxContext, knowledgeUUID, dirUUID string) error

	// InvalidateKnowledgeCountCache 清除知识数量缓存
	InvalidateKnowledgeCountCache(kbUUID string)

	// ExistsByDirectoryAndTitle 检查同一目录下是否存在相同标题的知识
	// directoryUUID 为 nil 表示默认目录，excludeUUID 用于排除自身（更新场景）
	ExistsByDirectoryAndTitle(ctx *database.TxContext, kbUUID string, directoryUUID *string, title string, excludeUUID ...string) (bool, error)

	// ExistsByKnowledgebaseAndType 检查知识本中是否存在指定类型的知识
	// excludeUUID 用于排除自身（更新场景）
	ExistsByKnowledgebaseAndType(ctx *database.TxContext, kbUUID string, knowledgeType string, excludeUUID ...string) (bool, error)

	// UpdateCurrentVersion 更新当前版本号
	UpdateCurrentVersion(ctx *database.TxContext, uuid, version string) error

	// UpdateDirectoryBatch 将某目录下的直接知识批量迁移到新目录
	// newDirectoryUUID 为 nil 表示迁移到默认目录（directory_uuid 置为 NULL）
	UpdateDirectoryBatch(ctx *database.TxContext, oldDirectoryUUID string, newDirectoryUUID *string) error

	// UpdateDirectoryBatchByUUIDs 将指定多个目录下的所有知识批量迁移到新目录（recursive 删除模式用）
	// newDirectoryUUID 为 nil 表示迁移到默认目录
	UpdateDirectoryBatchByUUIDs(ctx *database.TxContext, oldDirUUIDs []string, newDirectoryUUID *string) error

	// GetOrmer 获取 Ormer 实例
	GetOrmer() orm.Ormer

	// ListByKnowledgebaseAndSourceNotCustom 获取知识本下系统知识源（非自定义、非README类型），最多返回 limit 条
	ListByKnowledgebaseAndSourceNotCustom(ctx *database.TxContext, kbUUID string, limit int) ([]*domain.Knowledge, error)

	// ListByKnowledgebaseAndSourceCustomTopN 获取知识本下最新N条自定义知识（非README类型）
	ListByKnowledgebaseAndSourceCustomTopN(ctx *database.TxContext, kbUUID string, limit int) ([]*domain.Knowledge, error)

	// FindByKbUUIDAndDirUUIDAndTitle 按 (kbUUID, dirUUID, title) 查找知识
	// brain put_page upsert 专用：dirUUID 为 nil 表示无目录
	FindByKbUUIDAndDirUUIDAndTitle(ctx *database.TxContext, kbUUID string, dirUUID *string, title string) (*domain.Knowledge, error)
}

// ListKnowledgeQuery 知识列表查询条件
type ListKnowledgeQuery struct {
	KnowledgebaseUUID    string
	DirectoryUUID        string   // 单目录过滤（可选）；与 DirectoryUUIDs 互斥，DirectoryUUIDs 优先
	DirectoryUUIDs       []string // 多目录过滤（可选）；非空时忽略 DirectoryUUID
	IncludeNullDirectory bool     // 为 true 时额外匹配 directory_uuid=NULL 的历史知识（单/多目录模式均有效）
	KnowledgeSource      string   // 可选
	KnowledgeType        string   // 可选
	TemplateName         string   // 可选
	Keyword              string   // 可选，模糊匹配标题
	CreatorUsername      string   // 可选
	Offset               int
	Limit                int
}

// ListByTagsQuery 标签查询条件
type ListByTagsQuery struct {
	KnowledgebaseUUID    string
	DirectoryUUIDs       []string // 可选，目录UUID列表过滤
	IncludeNullDirectory bool     // 为 true 时同时匹配 directory_uuid IS NULL 的历史知识（is_default 目录兼容）
	TagUUIDs             []string
	MatchType            string // "any" 或 "all"
	KnowledgeSource      string // 可选
	KnowledgeType        string // 可选
	TemplateName         string // 可选
	Offset               int
	Limit                int
}

// knowledgeRepositoryImpl 知识仓储实现
type knowledgeRepositoryImpl struct {
	ormer orm.Ormer
}

// NewKnowledgeRepository 创建知识仓储实例
func NewKnowledgeRepository(ormer orm.Ormer) KnowledgeRepository {
	return &knowledgeRepositoryImpl{ormer: ormer}
}

// getOrmer 获取实际的 ormer
func (r *knowledgeRepositoryImpl) getOrmer(ctx *database.TxContext) orm.QueryExecutor {
	if ctx != nil && ctx.Ormer() != nil {
		return ctx.Ormer()
	}
	return r.ormer
}

// Create 创建知识
func (r *knowledgeRepositoryImpl) Create(ctx *database.TxContext, knowledge *domain.Knowledge) error {
	_, err := r.getOrmer(ctx).Insert(knowledge)
	return err
}

// FindByUUID 根据UUID查询知识
func (r *knowledgeRepositoryImpl) FindByUUID(ctx *database.TxContext, uuid string) (*domain.Knowledge, error) {
	knowledge := &domain.Knowledge{}
	err := r.getOrmer(ctx).QueryTable(new(domain.Knowledge)).
		Filter("uuid", uuid).
		Filter("deleted_at__isnull", true).
		One(knowledge)
	if err == orm.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return knowledge, nil
}

// FindByUUIDs 根据UUID列表批量查询知识
func (r *knowledgeRepositoryImpl) FindByUUIDs(ctx *database.TxContext, uuids []string) ([]*domain.Knowledge, error) {
	if len(uuids) == 0 {
		return []*domain.Knowledge{}, nil
	}

	// 安全校验：限制批量查询数量
	if len(uuids) > 100 {
		return nil, errors.New(errors.ErrCodeInvalidParam, "批量查询数量超过限制(最多100个)")
	}

	var list []*domain.Knowledge
	_, err := r.getOrmer(ctx).QueryTable(new(domain.Knowledge)).
		Filter("uuid__in", uuids).
		Filter("deleted_at__isnull", true).
		All(&list)
	if err != nil {
		return nil, err
	}
	return list, nil
}

// Update 更新知识
func (r *knowledgeRepositoryImpl) Update(ctx *database.TxContext, knowledge *domain.Knowledge) error {
	params := orm.Params{
		"knowledge_type":   knowledge.KnowledgeType,
		"title":            knowledge.Title,
		"content":          knowledge.Content,
		"file_path":        knowledge.FilePath,
		"file_size":        knowledge.FileSize,
		"file_format":      knowledge.FileFormat,
		"directory_uuid":   knowledge.DirectoryUUID,
		"metadata":         knowledge.Metadata,
		"current_version":  knowledge.CurrentVersion,
		"updater_username": knowledge.UpdaterUsername,
	}
	_, err := r.getOrmer(ctx).QueryTable(new(domain.Knowledge)).
		Filter("uuid", knowledge.UUID).
		Update(params)
	return err
}

// SoftDelete 软删除知识
func (r *knowledgeRepositoryImpl) SoftDelete(ctx *database.TxContext, uuid string) error {
	_, err := r.getOrmer(ctx).QueryTable(new(domain.Knowledge)).
		Filter("uuid", uuid).
		Update(orm.Params{
			"deleted_at": time.Now(),
		})
	return err
}

// SoftDeleteByDirectory 软删除指定目录下的所有知识
func (r *knowledgeRepositoryImpl) SoftDeleteByDirectory(ctx *database.TxContext, dirUUID string) error {
	_, err := r.getOrmer(ctx).QueryTable(new(domain.Knowledge)).
		Filter("directory_uuid", dirUUID).
		Filter("deleted_at__isnull", true).
		Update(orm.Params{
			"deleted_at": time.Now(),
		})
	return err
}

// SoftDeleteByDirectories 批量软删除多个目录下的所有知识
func (r *knowledgeRepositoryImpl) SoftDeleteByDirectories(ctx *database.TxContext, dirUUIDs []string) error {
	if len(dirUUIDs) == 0 {
		return nil
	}
	_, err := r.getOrmer(ctx).QueryTable(new(domain.Knowledge)).
		Filter("directory_uuid__in", dirUUIDs).
		Filter("deleted_at__isnull", true).
		Update(orm.Params{
			"deleted_at": time.Now(),
		})
	return err
}

// ListByConditions 根据条件查询知识列表
func (r *knowledgeRepositoryImpl) ListByConditions(ctx *database.TxContext, req *ListKnowledgeQuery) ([]*domain.Knowledge, int64, error) {
	qs := r.getOrmer(ctx).QueryTable(new(domain.Knowledge)).
		Filter("knowledgebase_uuid", req.KnowledgebaseUUID).
		Filter("deleted_at__isnull", true)

	// 目录过滤：DirectoryUUIDs 优先（多目录模式）
	if len(req.DirectoryUUIDs) > 0 {
		var subCond *orm.Condition
		if req.IncludeNullDirectory {
			// is_default 目录在子树中：同时包含 NULL 历史知识
			subCond = orm.NewCondition().
				And("directory_uuid__in", req.DirectoryUUIDs).
				Or("directory_uuid__isnull", true)
		} else {
			subCond = orm.NewCondition().And("directory_uuid__in", req.DirectoryUUIDs)
		}
		qs = qs.SetCond(qs.GetCond().AndCond(subCond))
	} else if req.DirectoryUUID != "" {
		if req.DirectoryUUID == constants.DirectoryRootUUID {
			// 兼容旧虚拟目录标识"/"
			qs = qs.Filter("directory_uuid__isnull", true)
		} else if req.IncludeNullDirectory {
			// is_default 目录：同时匹配真实 UUID 和历史 NULL 知识
			// 使用子条件包裹，确保 OR 条件有括号，不会脱离已有过滤条件的约束范围
			// 注意：SetCond 会完全替换已有条件，必须通过 GetCond() 保留已有条件再追加
			subCond := orm.NewCondition().
				And("directory_uuid", req.DirectoryUUID).
				Or("directory_uuid__isnull", true)
			qs = qs.SetCond(qs.GetCond().AndCond(subCond))
		} else {
			qs = qs.Filter("directory_uuid", req.DirectoryUUID)
		}
	}

	// 知识源过滤
	if req.KnowledgeSource != "" {
		qs = qs.Filter("knowledge_source", req.KnowledgeSource)
	}

	// 知识类型过滤
	if req.KnowledgeType != "" {
		qs = qs.Filter("knowledge_type", req.KnowledgeType)
	}

	// 模板名称过滤
	if req.TemplateName != "" {
		qs = qs.Filter("template_name", req.TemplateName)
	}

	// 创建者过滤
	if req.CreatorUsername != "" {
		qs = qs.Filter("creator_username", req.CreatorUsername)
	}

	// 关键词搜索
	if req.Keyword != "" {
		qs = qs.Filter("title__icontains", req.Keyword)
	}

	// 统计总数
	total, err := qs.Count()
	if err != nil {
		return nil, 0, err
	}

	// 分页查询
	var list []*domain.Knowledge
	_, err = qs.OrderBy("-created_at").Limit(req.Limit, req.Offset).All(&list)
	if err != nil {
		return nil, 0, err
	}

	return list, total, nil
}

// ListByTags 根据标签查询知识
func (r *knowledgeRepositoryImpl) ListByTags(ctx *database.TxContext, req *ListByTagsQuery) ([]*domain.Knowledge, int64, error) {
	if len(req.TagUUIDs) == 0 {
		return []*domain.Knowledge{}, 0, nil
	}

	// 安全校验：限制标签数量，防止SQL语句过长和DoS攻击
	if len(req.TagUUIDs) > 100 {
		return nil, 0, errors.New(errors.ErrCodeInvalidParam, "标签数量超过限制(最多100个)")
	}

	// 安全校验：限制目录数量
	if len(req.DirectoryUUIDs) > 100 {
		return nil, 0, errors.New(errors.ErrCodeInvalidParam, "目录数量超过限制(最多100个)")
	}

	// 构建占位符
	placeholders := strings.Repeat("?,", len(req.TagUUIDs))
	placeholders = placeholders[:len(placeholders)-1]

	// 构建目录过滤条件（支持多目录，兼容 is_default 目录的 NULL 历史知识）
	directoryCondition := ""
	if len(req.DirectoryUUIDs) > 0 {
		dirPlaceholders := strings.Repeat("?,", len(req.DirectoryUUIDs))
		dirPlaceholders = dirPlaceholders[:len(dirPlaceholders)-1]
		if req.IncludeNullDirectory {
			directoryCondition = fmt.Sprintf(" AND (k.directory_uuid IN (%s) OR k.directory_uuid IS NULL)", dirPlaceholders)
		} else {
			directoryCondition = fmt.Sprintf(" AND k.directory_uuid IN (%s)", dirPlaceholders)
		}
	}

	// 构建模板名称过滤条件
	templateCondition := ""
	if req.TemplateName != "" {
		templateCondition = " AND k.template_name = ?"
	}

	var sql string
	var args []interface{}

	if req.MatchType == "all" {
		// 必须匹配所有标签
		sql = fmt.Sprintf(`
			SELECT k.*
			FROM knowledge k
			INNER JOIN knowledge_tag_relation ktr ON k.uuid = ktr.knowledge_uuid
			WHERE k.knowledgebase_uuid = ?
			  AND ktr.tag_uuid IN (%s)
			  AND k.deleted_at IS NULL
			  %s%s
			GROUP BY k.uuid
			HAVING COUNT(DISTINCT ktr.tag_uuid) = ?
			ORDER BY k.created_at DESC
			LIMIT ? OFFSET ?
		`, placeholders, directoryCondition, templateCondition)

		args = []interface{}{req.KnowledgebaseUUID}
		for _, tagUUID := range req.TagUUIDs {
			args = append(args, tagUUID)
		}
		for _, dirUUID := range req.DirectoryUUIDs {
			args = append(args, dirUUID)
		}
		if req.TemplateName != "" {
			args = append(args, req.TemplateName)
		}
		args = append(args, len(req.TagUUIDs), req.Limit, req.Offset)
	} else {
		// 匹配任意标签
		sql = fmt.Sprintf(`
			SELECT DISTINCT k.*
			FROM knowledge k
			INNER JOIN knowledge_tag_relation ktr ON k.uuid = ktr.knowledge_uuid
			WHERE k.knowledgebase_uuid = ?
			  AND ktr.tag_uuid IN (%s)
			  AND k.deleted_at IS NULL
			  %s%s
			ORDER BY k.created_at DESC
			LIMIT ? OFFSET ?
		`, placeholders, directoryCondition, templateCondition)

		args = []interface{}{req.KnowledgebaseUUID}
		for _, tagUUID := range req.TagUUIDs {
			args = append(args, tagUUID)
		}
		for _, dirUUID := range req.DirectoryUUIDs {
			args = append(args, dirUUID)
		}
		if req.TemplateName != "" {
			args = append(args, req.TemplateName)
		}
		args = append(args, req.Limit, req.Offset)
	}

	// 执行查询
	var list []*domain.Knowledge
	_, err := r.getOrmer(ctx).Raw(sql, args...).QueryRows(&list)
	if err != nil {
		return nil, 0, err
	}

	// 查询总数
	var total int64
	if req.MatchType == "all" {
		countSQL := fmt.Sprintf(`
			SELECT COUNT(DISTINCT k.uuid)
			FROM knowledge k
			INNER JOIN knowledge_tag_relation ktr ON k.uuid = ktr.knowledge_uuid
			WHERE k.knowledgebase_uuid = ?
			  AND ktr.tag_uuid IN (%s)
			  AND k.deleted_at IS NULL
			  %s%s
			GROUP BY k.uuid
			HAVING COUNT(DISTINCT ktr.tag_uuid) = ?
		`, placeholders, directoryCondition, templateCondition)
		countArgs := []interface{}{req.KnowledgebaseUUID}
		for _, tagUUID := range req.TagUUIDs {
			countArgs = append(countArgs, tagUUID)
		}
		for _, dirUUID := range req.DirectoryUUIDs {
			countArgs = append(countArgs, dirUUID)
		}
		if req.TemplateName != "" {
			countArgs = append(countArgs, req.TemplateName)
		}
		countArgs = append(countArgs, len(req.TagUUIDs))

		var results []struct {
			Count int64
		}
		_, err = r.getOrmer(ctx).Raw(countSQL, countArgs...).QueryRows(&results)
		if err != nil {
			return nil, 0, err
		}
		total = int64(len(results))
	} else {
		countSQL := fmt.Sprintf(`
			SELECT COUNT(DISTINCT k.uuid)
			FROM knowledge k
			INNER JOIN knowledge_tag_relation ktr ON k.uuid = ktr.knowledge_uuid
			WHERE k.knowledgebase_uuid = ?
			  AND ktr.tag_uuid IN (%s)
			  AND k.deleted_at IS NULL
			  %s%s
		`, placeholders, directoryCondition, templateCondition)
		countArgs := []interface{}{req.KnowledgebaseUUID}
		for _, tagUUID := range req.TagUUIDs {
			countArgs = append(countArgs, tagUUID)
		}
		for _, dirUUID := range req.DirectoryUUIDs {
			countArgs = append(countArgs, dirUUID)
		}
		if req.TemplateName != "" {
			countArgs = append(countArgs, req.TemplateName)
		}

		err = r.getOrmer(ctx).Raw(countSQL, countArgs...).QueryRow(&total)
		if err != nil {
			return nil, 0, err
		}
	}

	return list, total, nil
}

// CountByKBUUID 统计知识本的知识数量
func (r *knowledgeRepositoryImpl) CountByKBUUID(ctx *database.TxContext, kbUUID string) (int64, error) {
	// 事务内不使用缓存
	if ctx == nil {
		// 尝试从缓存获取
		cacher := cache.GetCache()
		cacheKey := fmt.Sprintf("%s%s", constants.CacheKeyKnowledgeCount, kbUUID)
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
	count, err := r.getOrmer(ctx).QueryTable(new(domain.Knowledge)).
		Filter("knowledgebase_uuid", kbUUID).
		Filter("deleted_at__isnull", true).
		Count()
	if err != nil {
		return 0, err
	}

	// 写入缓存（仅在非事务环境）
	// 知识数量变更频率较高，缓存5分钟
	if ctx == nil {
		cacher := cache.GetCache()
		cacheKey := fmt.Sprintf("%s%s", constants.CacheKeyKnowledgeCount, kbUUID)
		if data, err := encodeToJSON(count); err == nil {
			if err := cacher.Set(context.Background(), cacheKey, data, 5*time.Minute); err != nil {
				log.Logger.Warnf("缓存知识数量失败: %v, kbUUID: %s", err, kbUUID)
			}
		}
	}

	return count, nil
}

// CountByKBUUIDsBatch 批量统计多个知识本的知识数量
func (r *knowledgeRepositoryImpl) CountByKBUUIDsBatch(ctx *database.TxContext, kbUUIDs []string) (map[string]int64, error) {
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
		cacheKey := fmt.Sprintf("%s%s", constants.CacheKeyKnowledgeCount, kbUUID)
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
		cacheKey := fmt.Sprintf("%s%s", constants.CacheKeyKnowledgeCount, kbUUID)
		if data, err := encodeToJSON(count); err == nil {
			if err := cacher.Set(context.Background(), cacheKey, data, 5*time.Minute); err != nil {
				log.Logger.Warnf("缓存知识数量失败: %v, kbUUID: %s", err, kbUUID)
			}
		}
		result[kbUUID] = count
	}

	return result, nil
}

// countByKBUUIDsBatchFromDB 从数据库批量统计知识数量
func (r *knowledgeRepositoryImpl) countByKBUUIDsBatchFromDB(ctx *database.TxContext, kbUUIDs []string) (map[string]int64, error) {
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
		FROM knowledge
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

// CountByDirectory 统计目录下的知识数量
func (r *knowledgeRepositoryImpl) CountByDirectory(ctx *database.TxContext, dirUUID string) (int64, error) {
	return r.getOrmer(ctx).QueryTable(new(domain.Knowledge)).
		Filter("directory_uuid", dirUUID).
		Filter("deleted_at__isnull", true).
		Count()
}

// CountByDirectories 批量统计多个目录下的知识数量
func (r *knowledgeRepositoryImpl) CountByDirectories(ctx *database.TxContext, dirUUIDs []string) (map[string]int64, error) {
	result := make(map[string]int64)

	if len(dirUUIDs) == 0 {
		return result, nil
	}

	// 安全校验：限制批量查询数量，防止SQL语句过长和内存消耗过大
	if len(dirUUIDs) > 1000 {
		return nil, errors.New(errors.ErrCodeInvalidParam, "批量查询目录数量超过限制(最多1000个)")
	}

	// 初始化所有目录的数量为0
	for _, uuid := range dirUUIDs {
		result[uuid] = 0
	}

	// 使用原生SQL批量查询
	type CountResult struct {
		DirectoryUUID string `orm:"column(directory_uuid)"`
		Count         int64  `orm:"column(count)"`
	}

	// 构建占位符
	placeholders := strings.Repeat("?,", len(dirUUIDs))
	placeholders = placeholders[:len(placeholders)-1]

	sql := fmt.Sprintf(`
		SELECT directory_uuid, COUNT(*) as count
		FROM knowledge
		WHERE directory_uuid IN (%s)
		  AND deleted_at IS NULL
		GROUP BY directory_uuid
	`, placeholders)

	// 构建参数
	params := make([]interface{}, len(dirUUIDs))
	for i, uuid := range dirUUIDs {
		params[i] = uuid
	}

	var results []CountResult
	_, err := r.getOrmer(ctx).Raw(sql, params...).QueryRows(&results)

	if err != nil {
		return nil, err
	}

	// 填充实际数量
	for _, res := range results {
		result[res.DirectoryUUID] = res.Count
	}

	return result, nil
}

// CountByNullDirectory 统计没有 dirUUID 的知识数量（默认目录下的知识）
func (r *knowledgeRepositoryImpl) CountByNullDirectory(ctx *database.TxContext, kbUUID string) (int64, error) {
	return r.getOrmer(ctx).QueryTable(new(domain.Knowledge)).
		Filter("knowledgebase_uuid", kbUUID).
		Filter("deleted_at__isnull", true).
		Filter("directory_uuid__isnull", true).
		Count()
}

// UnlinkDirectory 解除知识-目录关联
func (r *knowledgeRepositoryImpl) UnlinkDirectory(ctx *database.TxContext, dirUUID string) error {
	_, err := r.getOrmer(ctx).QueryTable(new(domain.Knowledge)).
		Filter("directory_uuid", dirUUID).
		Update(orm.Params{
			"directory_uuid": nil,
		})
	return err
}

// UpdateDirectory 更新知识的目录
func (r *knowledgeRepositoryImpl) UpdateDirectory(ctx *database.TxContext, knowledgeUUID, dirUUID string) error {
	params := orm.Params{}
	if dirUUID == "" {
		params["directory_uuid"] = nil
	} else {
		params["directory_uuid"] = dirUUID
	}

	_, err := r.getOrmer(ctx).QueryTable(new(domain.Knowledge)).
		Filter("uuid", knowledgeUUID).
		Update(params)
	return err
}

// GetOrmer 获取 Ormer 实例
func (r *knowledgeRepositoryImpl) GetOrmer() orm.Ormer {
	return r.ormer
}

// InvalidateKnowledgeCountCache 清除知识数量缓存
func (r *knowledgeRepositoryImpl) InvalidateKnowledgeCountCache(kbUUID string) {
	if kbUUID == "" {
		return
	}
	cacher := cache.GetCache()
	cacheKey := fmt.Sprintf("%s%s", constants.CacheKeyKnowledgeCount, kbUUID)
	if err := cacher.Delete(context.Background(), cacheKey); err != nil {
		log.Logger.Warnf("清除知识数量缓存失败: %v, kbUUID: %s", err, kbUUID)
	}
}

// ExistsByDirectoryAndTitle 检查同一目录下是否存在相同标题的知识
func (r *knowledgeRepositoryImpl) ExistsByDirectoryAndTitle(ctx *database.TxContext, kbUUID string, directoryUUID *string, title string, excludeUUID ...string) (bool, error) {
	qs := r.getOrmer(ctx).QueryTable(new(domain.Knowledge)).
		Filter("knowledgebase_uuid", kbUUID).
		Filter("deleted_at__isnull", true).
		Filter("title", title)

	// 处理目录过滤
	if directoryUUID != nil && *directoryUUID != "" {
		qs = qs.Filter("directory_uuid", *directoryUUID)
	} else {
		// 默认目录：directory_uuid 为 NULL
		qs = qs.Filter("directory_uuid__isnull", true)
	}

	// 排除自身（更新场景）
	if len(excludeUUID) > 0 && excludeUUID[0] != "" {
		qs = qs.Filter("uuid__ne", excludeUUID[0])
	}

	count, err := qs.Count()
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// ExistsByKnowledgebaseAndType 检查知识本中是否存在指定类型的知识
func (r *knowledgeRepositoryImpl) ExistsByKnowledgebaseAndType(ctx *database.TxContext, kbUUID string, knowledgeType string, excludeUUID ...string) (bool, error) {
	qs := r.getOrmer(ctx).QueryTable(new(domain.Knowledge)).
		Filter("knowledgebase_uuid", kbUUID).
		Filter("deleted_at__isnull", true).
		Filter("knowledge_type", knowledgeType)

	// 排除自身（更新场景）
	if len(excludeUUID) > 0 && excludeUUID[0] != "" {
		qs = qs.Filter("uuid__ne", excludeUUID[0])
	}

	count, err := qs.Count()
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// UpdateCurrentVersion 更新当前版本号
func (r *knowledgeRepositoryImpl) UpdateCurrentVersion(ctx *database.TxContext, uuid, version string) error {
	_, err := r.getOrmer(ctx).QueryTable(new(domain.Knowledge)).
		Filter("uuid", uuid).
		Update(orm.Params{
			"current_version": version,
		})
	return err
}

// UpdateDirectoryBatch 将某目录下的直接知识批量迁移到新目录
func (r *knowledgeRepositoryImpl) UpdateDirectoryBatch(ctx *database.TxContext, oldDirectoryUUID string, newDirectoryUUID *string) error {
	_, err := r.getOrmer(ctx).QueryTable(new(domain.Knowledge)).
		Filter("directory_uuid", oldDirectoryUUID).
		Filter("deleted_at__isnull", true).
		Update(orm.Params{"directory_uuid": newDirectoryUUID})
	return err
}

// UpdateDirectoryBatchByUUIDs 将指定多个目录下的所有知识批量迁移到新目录
func (r *knowledgeRepositoryImpl) UpdateDirectoryBatchByUUIDs(ctx *database.TxContext, oldDirUUIDs []string, newDirectoryUUID *string) error {
	if len(oldDirUUIDs) == 0 {
		return nil
	}
	_, err := r.getOrmer(ctx).QueryTable(new(domain.Knowledge)).
		Filter("directory_uuid__in", oldDirUUIDs).
		Filter("deleted_at__isnull", true).
		Update(orm.Params{"directory_uuid": newDirectoryUUID})
	return err
}

// ListByKnowledgebaseAndSourceNotCustom 获取知识本下系统知识源（非自定义、非README类型），最多返回 limit 条
func (r *knowledgeRepositoryImpl) ListByKnowledgebaseAndSourceNotCustom(ctx *database.TxContext, kbUUID string, limit int) ([]*domain.Knowledge, error) {
	var list []*domain.Knowledge
	_, err := r.getOrmer(ctx).QueryTable(new(domain.Knowledge)).
		Filter("knowledgebase_uuid", kbUUID).
		Filter("deleted_at__isnull", true).
		Filter("knowledge_source__ne", constants.SourceTypeCustom).
		Filter("knowledge_type__ne", constants.KnowledgeTypeReadme).
		OrderBy("-created_at").
		Limit(limit).
		All(&list)
	if err != nil {
		return nil, err
	}
	return list, nil
}

// ListByKnowledgebaseAndSourceCustomTopN 获取知识本下最新N条自定义知识（非README类型）
func (r *knowledgeRepositoryImpl) ListByKnowledgebaseAndSourceCustomTopN(ctx *database.TxContext, kbUUID string, limit int) ([]*domain.Knowledge, error) {
	var list []*domain.Knowledge
	_, err := r.getOrmer(ctx).QueryTable(new(domain.Knowledge)).
		Filter("knowledgebase_uuid", kbUUID).
		Filter("deleted_at__isnull", true).
		Filter("knowledge_source", constants.SourceTypeCustom).
		Filter("knowledge_type__ne", constants.KnowledgeTypeReadme).
		OrderBy("-created_at").
		Limit(limit).
		All(&list)
	if err != nil {
		return nil, err
	}
	return list, nil
}

// FindByKbUUIDAndDirUUIDAndTitle 按 (kbUUID, dirUUID, title) 查找知识
// brain put_page upsert 专用
func (r *knowledgeRepositoryImpl) FindByKbUUIDAndDirUUIDAndTitle(ctx *database.TxContext, kbUUID string, dirUUID *string, title string) (*domain.Knowledge, error) {
	q := r.getOrmer(ctx).QueryTable(new(domain.Knowledge)).
		Filter("knowledgebase_uuid", kbUUID).
		Filter("title", title).
		Filter("deleted_at__isnull", true)

	if dirUUID == nil {
		q = q.Filter("directory_uuid__isnull", true)
	} else {
		q = q.Filter("directory_uuid", *dirUUID)
	}

	knowledge := &domain.Knowledge{}
	err := q.One(knowledge)
	if err == orm.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return knowledge, nil
}
