// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package service

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/domain"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/infrastructure/database"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/repository"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/errors"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/log"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/types/constants"
)

// TagService 标签管理服务接口
type TagService interface {
	// CreateTag 创建标签
	CreateTag(ctx context.Context, kbUUID, name, description, operator string, isAdmin bool) (*domain.KnowledgebaseTag, error)

	// GetTag 查询单个标签
	GetTag(ctx context.Context, kbUUID, tagUUID, operator string) (*domain.KnowledgebaseTag, int64, error)

	// ListTags 查询标签列表，返回标签列表、知识数量Map和总数
	ListTags(ctx context.Context, kbUUID, operator string) ([]*domain.KnowledgebaseTag, map[string]int64, int64, error)

	// UpdateTag 更新标签
	UpdateTag(ctx context.Context, kbUUID, tagUUID, name, description, operator string, isAdmin bool) (*domain.KnowledgebaseTag, error)

	// DeleteTag 删除标签
	DeleteTag(ctx context.Context, kbUUID, tagUUID, operator string, isAdmin bool) error
}

// tagServiceImpl 标签管理服务实现
type tagServiceImpl struct {
	kbRepo            repository.KnowledgebaseRepository
	memberRepo        repository.KnowledgebaseMemberRepository
	tagRepo           repository.KnowledgebaseTagRepository
	tagRelRepo        repository.KnowledgeTagRelationRepository
	logRepo           repository.KnowledgebaseOperationLogRepository
	permissionService PermissionService
	txManager         *database.TransactionManager
}

// NewTagService 创建标签管理服务实例
func NewTagService(
	kbRepo repository.KnowledgebaseRepository,
	memberRepo repository.KnowledgebaseMemberRepository,
	tagRepo repository.KnowledgebaseTagRepository,
	tagRelRepo repository.KnowledgeTagRelationRepository,
	logRepo repository.KnowledgebaseOperationLogRepository,
	txManager *database.TransactionManager,
) TagService {
	return &tagServiceImpl{
		kbRepo:            kbRepo,
		memberRepo:        memberRepo,
		tagRepo:           tagRepo,
		tagRelRepo:        tagRelRepo,
		logRepo:           logRepo,
		permissionService: NewPermissionService(memberRepo),
		txManager:         txManager,
	}
}

// CreateTag 创建标签
func (s *tagServiceImpl) CreateTag(ctx context.Context, kbUUID, name, description, operator string, isAdmin bool) (*domain.KnowledgebaseTag, error) {
	// 1. 检查知识本是否存在
	kb, err := s.kbRepo.FindByUUID(nil, kbUUID)
	if err != nil {
		return nil, err
	}
	if kb == nil {
		return nil, errors.ErrKnowledgebaseNotFound
	}

	// 2. 检查操作者权限（只有admin可以创建标签）
	if err := s.permissionService.CheckManageTagPermission(kb, kbUUID, operator, isAdmin); err != nil {
		return nil, err
	}

	// 3. 验证标签名称格式
	if !domain.ValidateTagName(name) {
		return nil, errors.ErrInvalidTagName
	}

	// 4. 检查标签名称是否已存在（知识本内唯一）
	existingTag, err := s.tagRepo.FindByKBUUIDAndName(nil, kbUUID, name)
	if err != nil {
		return nil, err
	}
	if existingTag != nil {
		return nil, errors.ErrTagAlreadyExists
	}

	// 5. 创建标签对象
	tag := &domain.KnowledgebaseTag{
		UUID:              uuid.New().String(),
		KnowledgebaseUUID: kbUUID,
		Name:              name,
		Description:       description,
		CreatedBy:         operator,
	}

	// 6. 在事务中执行创建标签和记录操作日志
	err = s.txManager.ExecuteInTransactionWithContext(ctx, func(txCtx *database.TxContext) error {
		// 先物理删除已软删除的同名标签，避免唯一键冲突
		if err := s.tagRepo.HardDeleteByKBUUIDAndName(txCtx, kbUUID, name); err != nil {
			return err
		}

		// 创建标签记录
		if err := s.tagRepo.Create(txCtx, tag); err != nil {
			return err
		}

		// 记录操作日志
		detail := map[string]interface{}{
			"name":        name,
			"description": description,
		}
		detailJSON, err := json.Marshal(detail)
		if err != nil {
			return errors.Wrap(errors.ErrCodeInternalError, "序列化操作日志详情失败", err)
		}
		opLog := &domain.KnowledgebaseOperationLog{
			KnowledgebaseUUID: kbUUID,
			Operator:          operator,
			Operation:         constants.OperationCreateTag,
			ResourceType:      constants.ResourceTypeTag,
			ResourceUUID:      tag.UUID,
			Detail:            string(detailJSON),
		}
		return s.logRepo.Create(txCtx, opLog)
	})

	if err != nil {
		return nil, err
	}

	// 7. 清除标签列表缓存
	s.tagRepo.InvalidateTagListCache(kbUUID)
	s.tagRepo.InvalidateTagCountCache(kbUUID)

	log.Logger.InfofCtx(ctx, "创建标签成功: uuid=%s, name=%s, kbUUID=%s", tag.UUID, tag.Name, kbUUID)

	return tag, nil
}

// GetTag 查询单个标签
func (s *tagServiceImpl) GetTag(ctx context.Context, kbUUID, tagUUID, operator string) (*domain.KnowledgebaseTag, int64, error) {
	// 1. 检查知识本是否存在
	kb, err := s.kbRepo.FindByUUID(nil, kbUUID)
	if err != nil {
		return nil, 0, err
	}
	if kb == nil {
		return nil, 0, errors.ErrKnowledgebaseNotFound
	}

	// 2. 权限检查（所有成员都可以查看标签）
	if kb.Visibility == constants.VisibilityPrivate {
		role, err := s.memberRepo.GetUserRole(nil, kbUUID, operator)
		if err != nil {
			return nil, 0, err
		}
		if role == "" {
			return nil, 0, errors.ErrNotKnowledgebaseMember
		}
	}

	// 3. 查询标签
	tag, err := s.tagRepo.FindByUUID(nil, tagUUID)
	if err != nil {
		return nil, 0, err
	}
	if tag == nil {
		return nil, 0, errors.ErrTagNotFound
	}

	// 4. 检查标签是否属于该知识本
	if tag.KnowledgebaseUUID != kbUUID {
		return nil, 0, errors.ErrTagNotFound
	}

	// 5. 统计关联的知识数量
	count, err := s.tagRelRepo.CountByTagUUID(nil, tagUUID)
	if err != nil {
		return nil, 0, err
	}

	return tag, count, nil
}

// ListTags 查询标签列表
func (s *tagServiceImpl) ListTags(ctx context.Context, kbUUID, operator string) ([]*domain.KnowledgebaseTag, map[string]int64, int64, error) {
	// 1. 检查知识本是否存在
	kb, err := s.kbRepo.FindByUUID(nil, kbUUID)
	if err != nil {
		return nil, nil, 0, err
	}
	if kb == nil {
		return nil, nil, 0, errors.ErrKnowledgebaseNotFound
	}

	// 2. 权限检查（所有成员都可以查看标签）
	if kb.Visibility == constants.VisibilityPrivate {
		role, err := s.memberRepo.GetUserRole(nil, kbUUID, operator)
		if err != nil {
			return nil, nil, 0, err
		}
		if role == "" {
			return nil, nil, 0, errors.ErrNotKnowledgebaseMember
		}
	}

	// 3. 查询标签列表
	tags, err := s.tagRepo.ListByKBUUID(nil, kbUUID)
	if err != nil {
		return nil, nil, 0, err
	}

	// 4. 批量统计知识数量
	tagUUIDs := make([]string, 0, len(tags))
	for _, tag := range tags {
		tagUUIDs = append(tagUUIDs, tag.UUID)
	}
	countMap, err := s.tagRelRepo.BatchCountByTagUUIDs(nil, tagUUIDs)
	if err != nil {
		return nil, nil, 0, err
	}

	// 5. 统计总数
	total := int64(len(tags))

	return tags, countMap, total, nil
}

// UpdateTag 更新标签
func (s *tagServiceImpl) UpdateTag(ctx context.Context, kbUUID, tagUUID, name, description, operator string, isAdmin bool) (*domain.KnowledgebaseTag, error) {
	// 1. 检查知识本是否存在
	kb, err := s.kbRepo.FindByUUID(nil, kbUUID)
	if err != nil {
		return nil, err
	}
	if kb == nil {
		return nil, errors.ErrKnowledgebaseNotFound
	}

	// 2. 检查操作者权限（只有admin可以更新标签）
	if err := s.permissionService.CheckManageTagPermission(kb, kbUUID, operator, isAdmin); err != nil {
		return nil, err
	}

	// 3. 查询标签
	tag, err := s.tagRepo.FindByUUID(nil, tagUUID)
	if err != nil {
		return nil, err
	}
	if tag == nil {
		return nil, errors.ErrTagNotFound
	}

	// 4. 检查标签是否属于该知识本
	if tag.KnowledgebaseUUID != kbUUID {
		return nil, errors.ErrTagNotFound
	}

	// 5. 记录变更前的值
	oldName := tag.Name
	oldDesc := tag.Description

	// 6. 更新标签信息
	hasChanges := false
	if name != "" && name != tag.Name {
		// 验证标签名称格式
		if !domain.ValidateTagName(name) {
			return nil, errors.ErrInvalidTagName
		}
		// 检查新名称是否已存在
		existingTag, err := s.tagRepo.FindByKBUUIDAndName(nil, kbUUID, name)
		if err != nil {
			return nil, err
		}
		if existingTag != nil && existingTag.UUID != tagUUID {
			return nil, errors.ErrTagAlreadyExists
		}
		tag.Name = name
		hasChanges = true
	}
	// 只有当 description 有值时才更新，空字符串不覆盖原有值
	if description != "" && description != tag.Description {
		tag.Description = description
		hasChanges = true
	}

	// 如果没有变更，直接返回
	if !hasChanges {
		return tag, nil
	}

	// 7. 在事务中执行更新标签和记录操作日志
	err = s.txManager.ExecuteInTransactionWithContext(ctx, func(txCtx *database.TxContext) error {
		// 更新标签记录
		if err := s.tagRepo.Update(txCtx, tag); err != nil {
			return err
		}

		// 记录操作日志 - 记录完整的 before 和 after 状态，便于审计追溯
		changedFields := make([]string, 0)
		if name != "" && name != oldName {
			changedFields = append(changedFields, "name")
		}
		if description != "" && description != oldDesc {
			changedFields = append(changedFields, "description")
		}

		detail := map[string]interface{}{
			"before": map[string]string{
				"name":        oldName,
				"description": oldDesc,
			},
			"after": map[string]string{
				"name":        tag.Name,
				"description": tag.Description,
			},
			"changedFields": changedFields,
		}
		detailJSON, err := json.Marshal(detail)
		if err != nil {
			return errors.Wrap(errors.ErrCodeInternalError, "序列化操作日志详情失败", err)
		}
		opLog := &domain.KnowledgebaseOperationLog{
			KnowledgebaseUUID: kbUUID,
			Operator:          operator,
			Operation:         constants.OperationUpdateTag,
			ResourceType:      constants.ResourceTypeTag,
			ResourceUUID:      tag.UUID,
			Detail:            string(detailJSON),
		}
		return s.logRepo.Create(txCtx, opLog)
	})

	if err != nil {
		return nil, err
	}

	// 8. 清除标签列表缓存
	s.tagRepo.InvalidateTagListCache(kbUUID)

	log.Logger.InfofCtx(ctx, "更新标签成功: uuid=%s, name=%s", tag.UUID, tag.Name)

	return tag, nil
}

// DeleteTag 删除标签
func (s *tagServiceImpl) DeleteTag(ctx context.Context, kbUUID, tagUUID, operator string, isAdmin bool) error {
	// 1. 检查知识本是否存在
	kb, err := s.kbRepo.FindByUUID(nil, kbUUID)
	if err != nil {
		return err
	}
	if kb == nil {
		return errors.ErrKnowledgebaseNotFound
	}

	// 2. 检查操作者权限（只有admin可以删除标签）
	if err := s.permissionService.CheckManageTagPermission(kb, kbUUID, operator, isAdmin); err != nil {
		return err
	}

	// 3. 查询标签
	tag, err := s.tagRepo.FindByUUID(nil, tagUUID)
	if err != nil {
		return err
	}
	if tag == nil {
		return errors.ErrTagNotFound
	}

	// 4. 检查标签是否属于该知识本
	if tag.KnowledgebaseUUID != kbUUID {
		return errors.ErrTagNotFound
	}

	// 5. 在事务中执行删除标签和关联记录
	err = s.txManager.ExecuteInTransactionWithContext(ctx, func(txCtx *database.TxContext) error {
		// 删除所有知识-标签关联记录
		if err := s.tagRelRepo.DeleteByTagUUID(txCtx, tagUUID); err != nil {
			return err
		}

		// 软删除标签
		if err := s.tagRepo.SoftDelete(txCtx, tagUUID); err != nil {
			return err
		}

		// 记录操作日志
		detail := map[string]interface{}{
			"name":        tag.Name,
			"description": tag.Description,
		}
		detailJSON, err := json.Marshal(detail)
		if err != nil {
			return errors.Wrap(errors.ErrCodeInternalError, "序列化操作日志详情失败", err)
		}
		opLog := &domain.KnowledgebaseOperationLog{
			KnowledgebaseUUID: kbUUID,
			Operator:          operator,
			Operation:         constants.OperationDeleteTag,
			ResourceType:      constants.ResourceTypeTag,
			ResourceUUID:      tag.UUID,
			Detail:            string(detailJSON),
		}
		return s.logRepo.Create(txCtx, opLog)
	})

	if err != nil {
		return err
	}

	// 6. 清除相关缓存
	s.tagRepo.InvalidateTagListCache(kbUUID)
	s.tagRepo.InvalidateTagCountCache(kbUUID)

	log.Logger.InfofCtx(ctx, "删除标签成功: uuid=%s, name=%s", tag.UUID, tag.Name)

	return nil
}
