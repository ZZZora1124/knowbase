// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package repository

import (
	"fmt"
	"strings"

	"github.com/beego/beego/v2/client/orm"
	"github.com/google/uuid"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/domain"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/infrastructure/database"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/errors"
)

// KnowledgeTagRelationRepository 知识-标签关联仓储接口
type KnowledgeTagRelationRepository interface {
	// Create 创建关联
	Create(ctx *database.TxContext, relation *domain.KnowledgeTagRelation) error

	// BatchCreate 批量创建关联
	BatchCreate(ctx *database.TxContext, relations []*domain.KnowledgeTagRelation) error

	// DeleteByKnowledgeUUID 删除知识的所有标签关联
	DeleteByKnowledgeUUID(ctx *database.TxContext, knowledgeUUID string) error

	// DeleteByTagUUID 删除标签的所有知识关联
	DeleteByTagUUID(ctx *database.TxContext, tagUUID string) error

	// DeleteByKnowledgeAndTag 删除指定的知识-标签关联
	DeleteByKnowledgeAndTag(ctx *database.TxContext, knowledgeUUID, tagUUID string) error

	// BatchDeleteByKnowledgeAndTags 批量删除指定的知识-标签关联
	BatchDeleteByKnowledgeAndTags(ctx *database.TxContext, knowledgeUUID string, tagUUIDs []string) error

	// ListByKnowledgeUUID 查询知识的所有标签UUID
	ListByKnowledgeUUID(ctx *database.TxContext, knowledgeUUID string) ([]string, error)

	// ListByTagUUID 查询标签关联的所有知识UUID
	ListByTagUUID(ctx *database.TxContext, tagUUID string) ([]string, error)

	// CountByTagUUID 统计标签关联的知识数量
	CountByTagUUID(ctx *database.TxContext, tagUUID string) (int64, error)

	// BatchCountByTagUUIDs 批量统计多个标签关联的知识数量
	BatchCountByTagUUIDs(ctx *database.TxContext, tagUUIDs []string) (map[string]int64, error)

	// FindTagsByKnowledgeUUID 查询知识的所有标签信息
	FindTagsByKnowledgeUUID(ctx *database.TxContext, knowledgeUUID string) ([]*domain.KnowledgebaseTag, error)

	// FindTagsByKnowledgeUUIDs 批量查询多个知识的标签信息
	FindTagsByKnowledgeUUIDs(ctx *database.TxContext, knowledgeUUIDs []string) (map[string][]*domain.KnowledgebaseTag, error)
}

// knowledgeTagRelationRepositoryImpl 知识-标签关联仓储实现
type knowledgeTagRelationRepositoryImpl struct {
	ormer orm.Ormer
}

// NewKnowledgeTagRelationRepository 创建知识-标签关联仓储实例
func NewKnowledgeTagRelationRepository(ormer orm.Ormer) KnowledgeTagRelationRepository {
	return &knowledgeTagRelationRepositoryImpl{
		ormer: ormer,
	}
}

// getOrmer 获取实际的 ormer，优先使用事务上下文中的 ormer
func (r *knowledgeTagRelationRepositoryImpl) getOrmer(ctx *database.TxContext) orm.QueryExecutor {
	if ctx != nil && ctx.Ormer() != nil {
		return ctx.Ormer()
	}
	return r.ormer
}

// Create 创建关联
func (r *knowledgeTagRelationRepositoryImpl) Create(ctx *database.TxContext, relation *domain.KnowledgeTagRelation) error {
	if relation.UUID == "" {
		relation.UUID = uuid.New().String()
	}
	_, err := r.getOrmer(ctx).Insert(relation)
	return err
}

// BatchCreate 批量创建关联
func (r *knowledgeTagRelationRepositoryImpl) BatchCreate(ctx *database.TxContext, relations []*domain.KnowledgeTagRelation) error {
	if len(relations) == 0 {
		return nil
	}

	// 为每个关联生成UUID
	for _, rel := range relations {
		if rel.UUID == "" {
			rel.UUID = uuid.New().String()
		}
	}

	_, err := r.getOrmer(ctx).InsertMulti(len(relations), relations)
	return err
}

// DeleteByKnowledgeUUID 删除知识的所有标签关联
func (r *knowledgeTagRelationRepositoryImpl) DeleteByKnowledgeUUID(ctx *database.TxContext, knowledgeUUID string) error {
	_, err := r.getOrmer(ctx).QueryTable(new(domain.KnowledgeTagRelation)).
		Filter("knowledge_uuid", knowledgeUUID).
		Delete()
	return err
}

// DeleteByTagUUID 删除标签的所有知识关联
func (r *knowledgeTagRelationRepositoryImpl) DeleteByTagUUID(ctx *database.TxContext, tagUUID string) error {
	_, err := r.getOrmer(ctx).QueryTable(new(domain.KnowledgeTagRelation)).
		Filter("tag_uuid", tagUUID).
		Delete()
	return err
}

// DeleteByKnowledgeAndTag 删除指定的知识-标签关联
func (r *knowledgeTagRelationRepositoryImpl) DeleteByKnowledgeAndTag(ctx *database.TxContext, knowledgeUUID, tagUUID string) error {
	_, err := r.getOrmer(ctx).QueryTable(new(domain.KnowledgeTagRelation)).
		Filter("knowledge_uuid", knowledgeUUID).
		Filter("tag_uuid", tagUUID).
		Delete()
	return err
}

// BatchDeleteByKnowledgeAndTags 批量删除指定的知识-标签关联
func (r *knowledgeTagRelationRepositoryImpl) BatchDeleteByKnowledgeAndTags(ctx *database.TxContext, knowledgeUUID string, tagUUIDs []string) error {
	if len(tagUUIDs) == 0 {
		return nil
	}
	_, err := r.getOrmer(ctx).QueryTable(new(domain.KnowledgeTagRelation)).
		Filter("knowledge_uuid", knowledgeUUID).
		Filter("tag_uuid__in", tagUUIDs).
		Delete()
	return err
}

// ListByKnowledgeUUID 查询知识的所有标签UUID
func (r *knowledgeTagRelationRepositoryImpl) ListByKnowledgeUUID(ctx *database.TxContext, knowledgeUUID string) ([]string, error) {
	var relations []*domain.KnowledgeTagRelation
	_, err := r.getOrmer(ctx).QueryTable(new(domain.KnowledgeTagRelation)).
		Filter("knowledge_uuid", knowledgeUUID).
		All(&relations, "tag_uuid")
	if err != nil {
		return nil, err
	}

	tagUUIDs := make([]string, 0, len(relations))
	for _, rel := range relations {
		tagUUIDs = append(tagUUIDs, rel.TagUUID)
	}
	return tagUUIDs, nil
}

// ListByTagUUID 查询标签关联的所有知识UUID
func (r *knowledgeTagRelationRepositoryImpl) ListByTagUUID(ctx *database.TxContext, tagUUID string) ([]string, error) {
	var relations []*domain.KnowledgeTagRelation
	_, err := r.getOrmer(ctx).QueryTable(new(domain.KnowledgeTagRelation)).
		Filter("tag_uuid", tagUUID).
		All(&relations, "knowledge_uuid")
	if err != nil {
		return nil, err
	}

	knowledgeUUIDs := make([]string, 0, len(relations))
	for _, rel := range relations {
		knowledgeUUIDs = append(knowledgeUUIDs, rel.KnowledgeUUID)
	}
	return knowledgeUUIDs, nil
}

// CountByTagUUID 统计标签关联的知识数量
func (r *knowledgeTagRelationRepositoryImpl) CountByTagUUID(ctx *database.TxContext, tagUUID string) (int64, error) {
	count, err := r.getOrmer(ctx).QueryTable(new(domain.KnowledgeTagRelation)).
		Filter("tag_uuid", tagUUID).
		Count()
	return count, err
}

// BatchCountByTagUUIDs 批量统计多个标签关联的知识数量
func (r *knowledgeTagRelationRepositoryImpl) BatchCountByTagUUIDs(ctx *database.TxContext, tagUUIDs []string) (map[string]int64, error) {
	result := make(map[string]int64)
	if len(tagUUIDs) == 0 {
		return result, nil
	}

	// 安全校验：限制批量查询数量，防止SQL语句过长和内存消耗过大
	if len(tagUUIDs) > 1000 {
		return nil, errors.New(errors.ErrCodeInvalidParam, "批量查询标签数量超过限制(最多1000个)")
	}

	// 使用 GROUP BY 一次性查询所有标签的知识数量
	type countResult struct {
		TagUUID string `orm:"column(tag_uuid)"`
		Count   int64  `orm:"column(cnt)"`
	}

	var counts []countResult

	placeholders := strings.Repeat("?,", len(tagUUIDs))
	placeholders = placeholders[:len(placeholders)-1]

	sql := fmt.Sprintf(`
	SELECT tag_uuid, COUNT(id) AS cnt
	FROM knowledge_tag_relation
	WHERE tag_uuid IN (%s)
	GROUP BY tag_uuid
`, placeholders)

	params := make([]interface{}, len(tagUUIDs))
	for i, v := range tagUUIDs {
		params[i] = v
	}

	_, err := r.getOrmer(ctx).Raw(sql, params...).QueryRows(&counts)
	if err != nil {
		return nil, err
	}

	for _, c := range counts {
		result[c.TagUUID] = c.Count
	}

	// 对于没有关联知识的标签，设置数量为0
	for _, uuid := range tagUUIDs {
		if _, exists := result[uuid]; !exists {
			result[uuid] = 0
		}
	}

	return result, nil
}

// FindTagsByKnowledgeUUID 查询知识的所有标签信息
func (r *knowledgeTagRelationRepositoryImpl) FindTagsByKnowledgeUUID(ctx *database.TxContext, knowledgeUUID string) ([]*domain.KnowledgebaseTag, error) {
	var tags []*domain.KnowledgebaseTag

	// 使用JOIN查询获取标签信息
	sql := `
		SELECT t.id, t.uuid, t.knowledgebase_uuid, t.name, t.description, t.created_by, t.created_at, t.updated_at
		FROM knowledgebase_tag t
		INNER JOIN knowledge_tag_relation r ON t.uuid = r.tag_uuid
		WHERE r.knowledge_uuid = ? AND t.deleted_at IS NULL
		ORDER BY t.name
	`

	_, err := r.getOrmer(ctx).Raw(sql, knowledgeUUID).QueryRows(&tags)
	if err != nil {
		return nil, err
	}

	return tags, nil
}

// FindTagsByKnowledgeUUIDs 批量查询多个知识的标签信息
func (r *knowledgeTagRelationRepositoryImpl) FindTagsByKnowledgeUUIDs(ctx *database.TxContext, knowledgeUUIDs []string) (map[string][]*domain.KnowledgebaseTag, error) {
	result := make(map[string][]*domain.KnowledgebaseTag)

	if len(knowledgeUUIDs) == 0 {
		return result, nil
	}

	// 安全校验：限制批量查询数量，防止SQL语句过长和内存消耗过大
	if len(knowledgeUUIDs) > 1000 {
		return nil, errors.New(errors.ErrCodeInvalidParam, "批量查询知识数量超过限制(最多1000个)")
	}

	// 初始化所有知识的标签列表为空
	for _, uuid := range knowledgeUUIDs {
		result[uuid] = []*domain.KnowledgebaseTag{}
	}

	// 构建占位符
	placeholders := strings.Repeat("?,", len(knowledgeUUIDs))
	placeholders = placeholders[:len(placeholders)-1]

	// 使用JOIN批量查询标签信息
	sql := fmt.Sprintf(`
		SELECT r.knowledge_uuid, t.id, t.uuid, t.knowledgebase_uuid, t.name, t.description, t.created_by, t.created_at, t.updated_at
		FROM knowledgebase_tag t
		INNER JOIN knowledge_tag_relation r ON t.uuid = r.tag_uuid
		WHERE r.knowledge_uuid IN (%s) AND t.deleted_at IS NULL
		ORDER BY r.knowledge_uuid, t.name
	`, placeholders)

	// 构建参数
	params := make([]interface{}, len(knowledgeUUIDs))
	for i, uuid := range knowledgeUUIDs {
		params[i] = uuid
	}

	// 定义查询结果结构
	type tagWithKnowledgeUUID struct {
		KnowledgeUUID    string `orm:"column(knowledge_uuid)"`
		ID               int64  `orm:"column(id)"`
		UUID             string `orm:"column(uuid)"`
		KnowledgebaseUUID string `orm:"column(knowledgebase_uuid)"`
		Name             string `orm:"column(name)"`
		Description      string `orm:"column(description)"`
		CreatedBy        string `orm:"column(created_by)"`
		CreatedAt        string `orm:"column(created_at)"`
		UpdatedAt        string `orm:"column(updated_at)"`
	}

	var rows []tagWithKnowledgeUUID
	_, err := r.getOrmer(ctx).Raw(sql, params...).QueryRows(&rows)
	if err != nil {
		return nil, err
	}

	// 按knowledge_uuid分组
	for _, row := range rows {
		tag := &domain.KnowledgebaseTag{
			ID:               row.ID,
			UUID:             row.UUID,
			KnowledgebaseUUID: row.KnowledgebaseUUID,
			Name:             row.Name,
			Description:      row.Description,
			CreatedBy:        row.CreatedBy,
		}
		result[row.KnowledgeUUID] = append(result[row.KnowledgeUUID], tag)
	}

	return result, nil
}
