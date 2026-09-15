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
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/types/constants"
)

// KnowledgebaseRepository 知识本仓储接口
type KnowledgebaseRepository interface {
	// Create 创建知识本
	// ctx: 事务上下文，nil 表示非事务操作
	Create(ctx *database.TxContext, kb *domain.Knowledgebase) error

	// Update 更新知识本
	Update(ctx *database.TxContext, kb *domain.Knowledgebase) error

	// SoftDelete 软删除知识本
	SoftDelete(ctx *database.TxContext, uuid string) error

	// FindByUUID 根据UUID查询知识本
	FindByUUID(ctx *database.TxContext, uuid string) (*domain.Knowledgebase, error)

	// FindByName 根据名称查询知识本
	FindByName(ctx *database.TxContext, name string) (*domain.Knowledgebase, error)

	// ListByConditions 根据条件查询知识本列表
	// keyword: 模糊搜索关键词，匹配名称、描述、创建者
	// currentUsername: 当前用户名，用于过滤非本人的个人知识本（为空则不过滤）
	ListByConditions(
		ctx *database.TxContext,
		keyword, visibility, owner string,
		currentUsername string,
		offset, limit int,
	) ([]*domain.Knowledgebase, int64, error)

	// FindByUUIDsWithFilters 根据UUID列表批量查询知识本（支持过滤和分页）
	// kbUUIDs: 权限已过滤的知识本UUID列表（最大1000个）
	// keyword: 模糊搜索关键词，匹配名称、描述
	// visibility: 可见性过滤
	// offset, limit: 分页参数
	FindByUUIDsWithFilters(
		ctx *database.TxContext,
		kbUUIDs []string,
		keyword, visibility string,
		offset, limit int,
	) ([]*domain.Knowledgebase, int64, error)

	// CheckNameExists 检查名称是否存在
	// excludeUUID: 排除的知识本UUID（更新时排除自身）
	CheckNameExists(ctx *database.TxContext, name string, excludeUUID string) (bool, error)

	// FindUUIDsByVisibility 根据可见性查询知识本UUID列表
	// visibility: 可见性(public/private/personal)
	// 返回: UUID列表（仅查询UUID字段，不加载完整对象）
	FindUUIDsByVisibility(ctx *database.TxContext, visibility string) ([]string, error)

	// FindUUIDsByOwner 根据owner查询知识本UUID列表
	// owner: 知识本所有者
	// visibility: 可见性过滤（可选，空字符串表示不过滤）
	// 返回: UUID列表（仅查询UUID字段，不加载完整对象）
	FindUUIDsByOwner(ctx *database.TxContext, owner string, visibility string) ([]string, error)

	// GetOrmer 获取 Ormer 实例（用于向后兼容）
	GetOrmer() orm.Ormer

	// InvalidateKBCache 删除知识本缓存
	// 在知识本更新时调用
	InvalidateKBCache(uuid, name string)
}

// knowledgebaseRepositoryImpl 知识本仓储实现
type knowledgebaseRepositoryImpl struct {
	ormer orm.Ormer
}

// NewKnowledgebaseRepository 创建知识本仓储实例
func NewKnowledgebaseRepository(ormer orm.Ormer) KnowledgebaseRepository {
	return &knowledgebaseRepositoryImpl{
		ormer: ormer,
	}
}

// getOrmer 获取实际的 ormer，优先使用事务上下文中的 ormer
func (r *knowledgebaseRepositoryImpl) getOrmer(ctx *database.TxContext) orm.QueryExecutor {
	if ctx != nil && ctx.Ormer() != nil {
		return ctx.Ormer()
	}
	return r.ormer
}

// Create 创建知识本
func (r *knowledgebaseRepositoryImpl) Create(ctx *database.TxContext, kb *domain.Knowledgebase) error {
	_, err := r.getOrmer(ctx).Insert(kb)
	if err != nil {
		// 检测唯一约束冲突错误
		if strings.Contains(err.Error(), "Duplicate entry") && strings.Contains(err.Error(), "uk_name") {
			return errors.ErrKnowledgebaseAlreadyExists
		}
	}
	return err
}

// Update 更新知识本
// 使用 uuid 作为更新条件，避免依赖主键 id
func (r *knowledgebaseRepositoryImpl) Update(ctx *database.TxContext, kb *domain.Knowledgebase) error {
	_, err := r.getOrmer(ctx).QueryTable(new(domain.Knowledgebase)).
		Filter("uuid", kb.UUID).
		Update(orm.Params{
			"name":        kb.Name,
			"description": kb.Description,
			"visibility":  kb.Visibility,
			"status":      kb.Status,
		})
	return err
}

// SoftDelete 软删除知识本
func (r *knowledgebaseRepositoryImpl) SoftDelete(ctx *database.TxContext, uuid string) error {
	_, err := r.getOrmer(ctx).QueryTable(new(domain.Knowledgebase)).
		Filter("uuid", uuid).
		Update(orm.Params{
			"deleted_at": time.Now(),
			"status":     constants.KnowledgebaseStatusDisabled,
		})
	return err
}

// FindByUUID 根据UUID查询知识本
// 事务内不使用缓存，保证数据一致性
func (r *knowledgebaseRepositoryImpl) FindByUUID(ctx *database.TxContext, uuid string) (*domain.Knowledgebase, error) {
	// 事务内不使用缓存
	if ctx != nil {
		return r.findByUUIDFromDB(ctx, uuid)
	}

	// 1. 先查缓存
	cacher := cache.GetCache()
	cacheKey := fmt.Sprintf("%s%s", constants.CacheKeyKBUUID, uuid)
	cachedData, err := cacher.Get(context.Background(), cacheKey)
	if err == nil && cachedData != "" {
		var kb domain.Knowledgebase
		if err := json.Unmarshal([]byte(cachedData), &kb); err == nil {
			// 检查是否已软删除
			if kb.DeletedAt != nil {
				return nil, nil
			}
			return &kb, nil
		}
	}

	// 2. 缓存未命中，查询数据库
	kb, err := r.findByUUIDFromDB(nil, uuid)
	if err != nil || kb == nil {
		return kb, err
	}

	// 3. 写入缓存（过期时间10分钟）
	data, _ := json.Marshal(kb)
	cacher.Set(context.Background(), cacheKey, string(data), 10*time.Minute)

	return kb, nil
}

// findByUUIDFromDB 从数据库根据UUID查询知识本
func (r *knowledgebaseRepositoryImpl) findByUUIDFromDB(ctx *database.TxContext, uuid string) (*domain.Knowledgebase, error) {
	kb := &domain.Knowledgebase{UUID: uuid}
	err := r.getOrmer(ctx).Read(kb, "uuid")
	if err == orm.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	// 检查是否已软删除
	if kb.DeletedAt != nil {
		return nil, nil
	}
	return kb, nil
}

// FindByName 根据名称查询知识本
// 事务内不使用缓存，保证数据一致性
func (r *knowledgebaseRepositoryImpl) FindByName(ctx *database.TxContext, name string) (*domain.Knowledgebase, error) {
	// 事务内不使用缓存
	if ctx != nil {
		return r.findByNameFromDB(ctx, name)
	}

	// 1. 先查缓存
	cacher := cache.GetCache()
	cacheKey := fmt.Sprintf("%s%s", constants.CacheKeyKBName, name)
	cachedData, err := cacher.Get(context.Background(), cacheKey)
	if err == nil {
		// 缓存为空
		if cachedData == "null" {
			return nil, nil
		}
		// 缓存有效
		if cachedData != "" {
			var kb domain.Knowledgebase
			if err := json.Unmarshal([]byte(cachedData), &kb); err == nil {
				return &kb, nil
			}

			// 反序列化失败，删除异常缓存
			_ = cacher.Delete(context.Background(), cacheKey)
		}
	}

	// 2. 缓存未命中，查询数据库
	kb, err := r.findByNameFromDB(nil, name)
	if err != nil {
		return nil, err
	}

	// 3. 无论是否找到都写入缓存
	if kb == nil {
		// 缓存空结果，使用特殊标记，过期时间较短
		cacher.Set(context.Background(), cacheKey, "null", 2*time.Minute)
	} else {
		data, _ := json.Marshal(kb)
		cacher.Set(context.Background(), cacheKey, string(data), 10*time.Minute)
	}

	return kb, nil
}

// findByNameFromDB 从数据库根据名称查询知识本
func (r *knowledgebaseRepositoryImpl) findByNameFromDB(ctx *database.TxContext, name string) (*domain.Knowledgebase, error) {
	kb := &domain.Knowledgebase{Name: name}
	err := r.getOrmer(ctx).Read(kb, "name")
	if err == orm.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	// 检查是否已软删除
	if kb.DeletedAt != nil {
		return nil, nil
	}
	return kb, nil
}

// ListByConditions 根据条件查询知识本列表
func (r *knowledgebaseRepositoryImpl) ListByConditions(
	ctx *database.TxContext,
	keyword, visibility, owner string,
	currentUsername string,
	offset, limit int,
) ([]*domain.Knowledgebase, int64, error) {
	// 计算分页参数（用于缓存 key）
	page := offset/limit + 1

	// 【重要】如果有 currentUsername 参数，说明需要按用户过滤，不同用户结果不同，禁用缓存
	shouldUseCache := (ctx == nil && currentUsername == "")

	// 事务内不使用缓存，保证数据一致性
	if shouldUseCache {
		// 1. 生成缓存 Key
		cacheKey, shouldCache, ttl := r.generateListCacheKey(keyword, visibility, owner, page, limit)

		// 2. 尝试从缓存读取
		if shouldCache {
			cacher := cache.GetCache()
			cachedData, err := cacher.Get(context.Background(), cacheKey)
			if err == nil && cachedData != "" {
				var cached listCacheValue
				if err := json.Unmarshal([]byte(cachedData), &cached); err == nil {
					return cached.List, cached.Total, nil
				}
				// 反序列化失败,删除异常缓存
				_ = cacher.Delete(context.Background(), cacheKey)
			}
		}

		// 3. 缓存未命中，查询数据库
		kbs, total, err := r.queryListFromDB(ctx, keyword, visibility, owner, currentUsername, offset, limit)
		if err != nil {
			return nil, 0, err
		}

		// 4. 写入缓存
		if shouldCache {
			cacher := cache.GetCache()
			cacheValue := listCacheValue{
				List:  kbs,
				Total: total,
			}
			data, _ := json.Marshal(cacheValue)
			_ = cacher.Set(context.Background(), cacheKey, string(data), ttl)
		}

		return kbs, total, nil
	}

	// 事务内或有 currentUsername 时，直接查询数据库，不使用缓存
	return r.queryListFromDB(ctx, keyword, visibility, owner, currentUsername, offset, limit)
}

// queryListFromDB 从数据库查询列表（抽取出来便于复用）
func (r *knowledgebaseRepositoryImpl) queryListFromDB(
	ctx *database.TxContext,
	keyword, visibility, owner string,
	currentUsername string,
	offset, limit int,
) ([]*domain.Knowledgebase, int64, error) {
	qs := r.getOrmer(ctx).QueryTable(new(domain.Knowledgebase))

	// 【重要】使用一个统一的 Condition 对象，避免 SetCond 覆盖问题
	cond := orm.NewCondition()

	// 基础条件：未删除 + 已启用
	cond = cond.And("deleted_at__isnull", true).
		And("status", constants.KnowledgebaseStatusEnabled)

	// keyword 参数：模糊搜索名称、描述、创建者
	if keyword != "" {
		keywordCond := orm.NewCondition()
		keywordCond = keywordCond.Or("name__icontains", keyword).
			Or("description__icontains", keyword).
			Or("owner__icontains", keyword)
		cond = cond.AndCond(keywordCond)
	}

	// 可见性过滤
	if visibility != "" {
		cond = cond.And("visibility", visibility)
	}

	// 所有者过滤
	if owner != "" {
		cond = cond.And("owner", owner)
	}

	// 【核心优化】SQL层过滤：排除非当前用户的个人知识本
	// 条件：visibility != 'personal' OR owner = currentUsername
	// 即：只显示非个人知识本，或者是当前用户的个人知识本
	if currentUsername != "" {
		personalCond := orm.NewCondition()
		personalCond = personalCond.Or("visibility__ne", constants.VisibilityPersonal).
			Or("owner", currentUsername)
		cond = cond.AndCond(personalCond)
	}

	// 应用所有条件
	qs = qs.SetCond(cond)

	total, err := qs.Count()
	if err != nil {
		return nil, 0, err
	}

	kbs := make([]*domain.Knowledgebase, 0)
	_, err = qs.OrderBy("-created_at").Limit(limit, offset).All(&kbs)
	if err != nil {
		return nil, 0, err
	}

	return kbs, total, nil
}

// CheckNameExists 检查名称是否存在
// excludeUUID: 排除的知识本UUID（更新时排除自身）
func (r *knowledgebaseRepositoryImpl) CheckNameExists(ctx *database.TxContext, name string, excludeUUID string) (bool, error) {
	qs := r.getOrmer(ctx).QueryTable(new(domain.Knowledgebase)).
		Filter("name", name).
		Filter("deleted_at__isnull", true)

	// 排除指定的UUID（更新时排除自身）
	if excludeUUID != "" {
		qs = qs.Filter("uuid__ne", excludeUUID)
	}

	count, err := qs.Count()
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// GetOrmer 获取 Ormer 实例
func (r *knowledgebaseRepositoryImpl) GetOrmer() orm.Ormer {
	return r.ormer
}

// InvalidateKBCache 删除知识本缓存
// 在知识本更新时调用，删除UUID和Name两个缓存键
func (r *knowledgebaseRepositoryImpl) InvalidateKBCache(uuid, name string) {
	cacher := cache.GetCache()
	// 删除UUID缓存
	if uuid != "" {
		cacher.Delete(context.Background(), constants.CacheKeyKBUUID+uuid)
	}
	// 删除Name缓存
	if name != "" {
		cacher.Delete(context.Background(), constants.CacheKeyKBName+name)
	}
	// 删除所有列表缓存
	cacher.DeleteByPattern(context.Background(), constants.CacheKeyKBList+"*")
}

// listCacheValue 列表缓存值结构
type listCacheValue struct {
	List  []*domain.Knowledgebase `json:"list"`
	Total int64                   `json:"total"`
}

// generateListCacheKey 生成列表缓存 Key，返回 (cacheKey, shouldCache, ttl)
// 使用分层缓存策略，只缓存高频场景
func (r *knowledgebaseRepositoryImpl) generateListCacheKey(
	keyword, visibility, owner string,
	page, pageSize int,
) (string, bool, time.Duration) {
	// 不缓存大分页（减少缓存占用）
	if pageSize > 50 {
		return "", false, 0
	}

	// 不缓存深分页（page > 10）
	if page > 10 {
		return "", false, 0
	}

	// Tier 1: 全部列表（无任何过滤条件）- 适用于首页默认列表
	// Key: kb:list:all:{page}:{pageSize}
	// TTL: 2分钟
	if keyword == "" && visibility == "" && owner == "" {
		cacheKey := fmt.Sprintf("%sall:%d:%d", constants.CacheKeyKBList, page, pageSize)
		return cacheKey, true, 2 * time.Minute
	}

	// Tier 2: 按可见性过滤（public/private/personal）
	// Key: kb:list:visibility:{visibility}:{page}:{pageSize}
	// TTL: 2分钟
	if keyword == "" && visibility != "" && owner == "" {
		cacheKey := fmt.Sprintf("%svisibility:%s:%d:%d",
			constants.CacheKeyKBList, visibility, page, pageSize)
		return cacheKey, true, 2 * time.Minute
	}

	// Tier 2: 用户知识本列表（指定 owner）
	// Key: kb:list:owner:{owner}:{visibility}:{page}:{pageSize}
	// TTL: 1分钟
	if keyword == "" && owner != "" {
		visibilityPart := visibility
		if visibilityPart == "" {
			visibilityPart = "all"
		}
		cacheKey := fmt.Sprintf("%sowner:%s:%s:%d:%d",
			constants.CacheKeyKBList, owner, visibilityPart, page, pageSize)
		return cacheKey, true, 1 * time.Minute
	}

	// Tier 3: 关键词搜索（仅 keyword，无其他过滤条件）
	// Key: kb:list:search:keyword:{keywordHash}:{page}:{pageSize}
	// TTL: 30秒
	if keyword != "" && visibility == "" && owner == "" {
		keywordHash := fmt.Sprintf("%d", hashString(keyword))
		cacheKey := fmt.Sprintf("%ssearch:keyword:%s:%d:%d",
			constants.CacheKeyKBList, keywordHash, page, pageSize)
		return cacheKey, true, 30 * time.Second
	}

	// 复杂查询不缓存（如：keyword + visibility/owner 组合）
	return "", false, 0
}

// hashString 简单的字符串哈希函数
func hashString(s string) uint32 {
	var hash uint32
	for _, c := range s {
		hash = hash*31 + uint32(c)
	}
	return hash
}

// FindByUUIDsWithFilters 根据UUID列表批量查询知识本（支持过滤和分页）
func (r *knowledgebaseRepositoryImpl) FindByUUIDsWithFilters(
	ctx *database.TxContext,
	kbUUIDs []string,
	keyword, visibility string,
	offset, limit int,
) ([]*domain.Knowledgebase, int64, error) {
	// 1. 空列表快速返回
	if len(kbUUIDs) == 0 {
		return []*domain.Knowledgebase{}, 0, nil
	}

	// 2. 限制UUID列表最大长度为1000（防止SQL过大）
	if len(kbUUIDs) > 1000 {
		kbUUIDs = kbUUIDs[:1000]
	}

	// 3. 构建查询条件
	qs := r.getOrmer(ctx).QueryTable(new(domain.Knowledgebase))

	// 基础条件：UUID在列表中、未删除、已启用
	cond := orm.NewCondition()
	cond = cond.And("uuid__in", kbUUIDs).
		And("deleted_at__isnull", true).
		And("status", constants.KnowledgebaseStatusEnabled)

	// keyword 过滤：模糊搜索名称、描述
	if keyword != "" {
		keywordCond := orm.NewCondition()
		keywordCond = keywordCond.Or("name__icontains", keyword).
			Or("description__icontains", keyword)
		cond = cond.AndCond(keywordCond)
	}

	// visibility 过滤
	if visibility != "" {
		cond = cond.And("visibility", visibility)
	}

	qs = qs.SetCond(cond)

	// 4. 先查询总数
	total, err := qs.Count()
	if err != nil {
		return nil, 0, err
	}

	// 5. 查询数据（按创建时间倒序）
	kbs := make([]*domain.Knowledgebase, 0)
	_, err = qs.OrderBy("-created_at").Limit(limit, offset).All(&kbs)
	if err != nil {
		return nil, 0, err
	}

	return kbs, total, nil
}

// FindUUIDsByVisibility 根据可见性查询知识本UUID列表
// 仅查询UUID字段，不加载完整对象，用于性能优化
func (r *knowledgebaseRepositoryImpl) FindUUIDsByVisibility(
	ctx *database.TxContext,
	visibility string,
) ([]string, error) {
	var list orm.ParamsList
	_, err := r.getOrmer(ctx).QueryTable(new(domain.Knowledgebase)).
		Filter("visibility", visibility).
		Filter("deleted_at__isnull", true).
		Filter("status", constants.KnowledgebaseStatusEnabled).
		ValuesFlat(&list, "uuid")
	if err != nil && err != orm.ErrNoRows {
		return nil, err
	}

	// 转换 orm.ParamsList 为 []string
	uuids := make([]string, 0, len(list))
	for _, v := range list {
		if uuid, ok := v.(string); ok {
			uuids = append(uuids, uuid)
		}
	}
	return uuids, nil
}

// FindUUIDsByOwner 根据owner查询知识本UUID列表
// 仅查询UUID字段，不加载完整对象，用于性能优化
func (r *knowledgebaseRepositoryImpl) FindUUIDsByOwner(
	ctx *database.TxContext,
	owner string,
	visibility string,
) ([]string, error) {
	qs := r.getOrmer(ctx).QueryTable(new(domain.Knowledgebase)).
		Filter("owner", owner).
		Filter("deleted_at__isnull", true).
		Filter("status", constants.KnowledgebaseStatusEnabled)

	// 可选的 visibility 过滤
	if visibility != "" {
		qs = qs.Filter("visibility", visibility)
	}

	var list orm.ParamsList
	_, err := qs.ValuesFlat(&list, "uuid")
	if err != nil && err != orm.ErrNoRows {
		return nil, err
	}

	// 转换 orm.ParamsList 为 []string
	uuids := make([]string, 0, len(list))
	for _, v := range list {
		if uuid, ok := v.(string); ok {
			uuids = append(uuids, uuid)
		}
	}
	return uuids, nil
}
