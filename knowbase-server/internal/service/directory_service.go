// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package service

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/google/uuid"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/domain"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/infrastructure/database"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/repository"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/errors"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/log"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/types/constants"
)

// DirectoryWithCount 带知识数量的目录
type DirectoryWithCount struct {
	Directory      *domain.KnowledgebaseDirectory
	KnowledgeCount int64
}

// DirectoryTreeNode 服务层树节点
type DirectoryTreeNode struct {
	Directory      *domain.KnowledgebaseDirectory
	KnowledgeCount int64
	Children       []*DirectoryTreeNode
}

// DirectoryService 目录服务接口
type DirectoryService interface {
	// CreateDirectory 创建目录
	CreateDirectory(ctx context.Context, kbUUID, name string, parentUUID *string, operator string, isAdmin bool) (*domain.KnowledgebaseDirectory, error)

	// GetDirectory 查询目录详情
	GetDirectory(ctx context.Context, kbUUID, dirUUID, operator string, isAdmin bool) (*DirectoryWithCount, error)

	// ListDirectories 查询目录列表
	ListDirectories(ctx context.Context, kbUUID, operator string, isAdmin bool) ([]*DirectoryWithCount, int64, error)

	// UpdateDirectory 更新目录
	UpdateDirectory(ctx context.Context, kbUUID, dirUUID, name, operator string, isAdmin bool) (*DirectoryWithCount, error)

	// DeleteDirectory 删除目录
	DeleteDirectory(ctx context.Context, kbUUID, dirUUID string, operator string, isAdmin bool) error

	// GetDirectoryTree 获取目录树（应用层组装，不依赖数据库递归）
	GetDirectoryTree(ctx context.Context, kbUUID, operator string, isAdmin bool) ([]*DirectoryTreeNode, int64, error)

	// MoveDirectory 移动目录到新的父目录
	MoveDirectory(ctx context.Context, kbUUID, dirUUID string, targetParentUUID *string, operator string, isAdmin bool) (*domain.KnowledgebaseDirectory, error)
}

// directoryServiceImpl 目录服务实现
type directoryServiceImpl struct {
	kbRepo            repository.KnowledgebaseRepository
	dirRepo           repository.KnowledgebaseDirectoryRepository
	knowRepo          repository.KnowledgeRepository
	logRepo           repository.KnowledgebaseOperationLogRepository
	permissionService PermissionService
	txManager         *database.TransactionManager
}

// NewDirectoryService 创建目录服务实例
func NewDirectoryService(
	kbRepo repository.KnowledgebaseRepository,
	dirRepo repository.KnowledgebaseDirectoryRepository,
	knowRepo repository.KnowledgeRepository,
	logRepo repository.KnowledgebaseOperationLogRepository,
	permissionService PermissionService,
) DirectoryService {
	return &directoryServiceImpl{
		kbRepo:            kbRepo,
		dirRepo:           dirRepo,
		knowRepo:          knowRepo,
		logRepo:           logRepo,
		permissionService: permissionService,
		txManager:         database.NewTransactionManager(),
	}
}

// CreateDirectory 创建目录
func (s *directoryServiceImpl) CreateDirectory(ctx context.Context, kbUUID, name string, parentUUID *string, operator string, isAdmin bool) (*domain.KnowledgebaseDirectory, error) {
	// 1. 检查知识本是否存在
	kb, err := s.kbRepo.FindByUUID(nil, kbUUID)
	if err != nil {
		return nil, errors.Wrap(errors.ErrCodeInternalError, "查询知识本失败", err)
	}
	if kb == nil {
		return nil, errors.ErrKnowledgebaseNotFound
	}

	// 2. 权限检查（editor及以上可以创建目录）
	if err := s.permissionService.CheckCreatePermission(kb, kbUUID, operator, isAdmin); err != nil {
		return nil, err
	}

	// 3. 验证目录名称
	if !domain.ValidateDirectoryName(name) {
		return nil, errors.New(errors.ErrCodeInvalidParam, "目录名称格式不正确")
	}

	// 4. 若传入 parentUUID，校验父目录存在且属于同一知识本；同时校验层级深度
	if parentUUID != nil {
		parentDir, err := s.dirRepo.FindByUUID(nil, *parentUUID)
		if err != nil {
			return nil, errors.Wrap(errors.ErrCodeInternalError, "查询父目录失败", err)
		}
		if parentDir == nil || parentDir.KnowledgebaseUUID != kbUUID {
			return nil, errors.New(errors.ErrCodeNotFound, "父目录不存在")
		}

		// 校验目录层级深度（新目录将在父目录下，深度 = 父目录深度 + 1）
		allDirs, err := s.dirRepo.ListByKBUUID(nil, kbUUID)
		if err != nil {
			return nil, errors.Wrap(errors.ErrCodeInternalError, "查询目录列表失败", err)
		}
		parentDepth := s.calculateDepth(*parentUUID, allDirs)
		if parentDepth+1 > constants.DirectoryMaxDepth {
			return nil, errors.New(errors.ErrCodeInvalidParam,
				fmt.Sprintf("目录层级已达上限（最多%d级）", constants.DirectoryMaxDepth))
		}
	}

	// 5. 同级同名唯一性检查
	existing, err := s.dirRepo.FindByKBUUIDAndNameAndParent(nil, kbUUID, name, parentUUID)
	if err != nil {
		return nil, errors.Wrap(errors.ErrCodeInternalError, "检查目录名称失败", err)
	}
	if existing != nil {
		return nil, errors.New(errors.ErrCodeAlreadyExists, "同级目录名称已存在")
	}

	// 6. 创建目录
	dir := &domain.KnowledgebaseDirectory{
		UUID:              uuid.New().String(),
		KnowledgebaseUUID: kbUUID,
		ParentUUID:        parentUUID,
		Name:              name,
		IsDefault:         false, // 普通目录不是默认目录，防御性显式设置
	}

	// 7. 事务执行
	err = s.txManager.ExecuteInTransactionWithContext(ctx, func(txCtx *database.TxContext) error {
		if err := s.dirRepo.Create(txCtx, dir); err != nil {
			return errors.Wrap(errors.ErrCodeInternalError, "创建目录失败", err)
		}

		// 记录操作日志
		detail, _ := json.Marshal(map[string]string{"name": name})
		opLog := &domain.KnowledgebaseOperationLog{
			KnowledgebaseUUID: kbUUID,
			Operator:          operator,
			Operation:         constants.OperationCreateDir,
			ResourceType:      constants.ResourceTypeDirectory,
			ResourceUUID:      dir.UUID,
			Detail:            string(detail),
		}
		return s.logRepo.Create(txCtx, opLog)
	})

	if err != nil {
		return nil, err
	}

	log.Logger.InfofCtx(ctx, "创建目录成功: uuid=%s, name=%s, kbUUID=%s", dir.UUID, dir.Name, kbUUID)

	return dir, nil
}

// GetDirectory 查询目录详情
func (s *directoryServiceImpl) GetDirectory(ctx context.Context, kbUUID, dirUUID, operator string, isAdmin bool) (*DirectoryWithCount, error) {
	// 1. 检查知识本是否存在
	kb, err := s.kbRepo.FindByUUID(nil, kbUUID)
	if err != nil {
		return nil, errors.Wrap(errors.ErrCodeInternalError, "查询知识本失败", err)
	}
	if kb == nil {
		return nil, errors.ErrKnowledgebaseNotFound
	}

	// 2. 权限检查
	if !s.permissionService.CheckViewPermission(kb, kbUUID, operator, isAdmin) {
		return nil, errors.ErrNotKnowledgebaseMember
	}

	// 3. 查询目录
	dir, err := s.dirRepo.FindByUUID(nil, dirUUID)
	if err != nil {
		return nil, errors.Wrap(errors.ErrCodeInternalError, "查询目录失败", err)
	}
	if dir == nil || dir.KnowledgebaseUUID != kbUUID {
		return nil, errors.New(errors.ErrCodeNotFound, "目录不存在")
	}

	// 4. 查询知识数量
	count, err := s.knowRepo.CountByDirectory(nil, dirUUID)
	if err != nil {
		return nil, errors.Wrap(errors.ErrCodeInternalError, "查询知识数量失败", err)
	}

	return &DirectoryWithCount{
		Directory:      dir,
		KnowledgeCount: count,
	}, nil
}

// ListDirectories 查询目录列表
func (s *directoryServiceImpl) ListDirectories(ctx context.Context, kbUUID, operator string, isAdmin bool) ([]*DirectoryWithCount, int64, error) {
	// 1. 检查知识本是否存在
	kb, err := s.kbRepo.FindByUUID(nil, kbUUID)
	if err != nil {
		return nil, 0, errors.Wrap(errors.ErrCodeInternalError, "查询知识本失败", err)
	}
	if kb == nil {
		return nil, 0, errors.ErrKnowledgebaseNotFound
	}

	// 2. 权限检查
	if !s.permissionService.CheckViewPermission(kb, kbUUID, operator, isAdmin) {
		return nil, 0, errors.ErrNotKnowledgebaseMember
	}

	// 3. 查询目录列表
	dirs, err := s.dirRepo.ListByKBUUID(nil, kbUUID)
	if err != nil {
		return nil, 0, errors.Wrap(errors.ErrCodeInternalError, "查询目录列表失败", err)
	}

	// 4. 批量查询知识数量（优化：避免N+1查询）
	result := make([]*DirectoryWithCount, 0, len(dirs))
	dirUUIDs := make([]string, 0, len(dirs))

	if len(dirs) > 0 {
		for _, dir := range dirs {
			dirUUIDs = append(dirUUIDs, dir.UUID)
		}
		countMap, err := s.knowRepo.CountByDirectories(nil, dirUUIDs)
		if err != nil {
			return nil, 0, errors.Wrap(errors.ErrCodeInternalError, "查询知识数量失败", err)
		}

		if err := s.mergeNullCount(kbUUID, countMap, dirs); err != nil {
			return nil, 0, errors.Wrap(errors.ErrCodeInternalError, "查询默认目录知识数量失败", err)
		}

		for _, dir := range dirs {
			result = append(result, &DirectoryWithCount{
				Directory:      dir,
				KnowledgeCount: countMap[dir.UUID],
			})
		}
	}

	total := int64(len(result))

	return result, total, nil
}

// UpdateDirectory 更新目录
func (s *directoryServiceImpl) UpdateDirectory(ctx context.Context, kbUUID, dirUUID, name, operator string, isAdmin bool) (*DirectoryWithCount, error) {
	// 1. 检查知识本和目录存在性
	kb, err := s.kbRepo.FindByUUID(nil, kbUUID)
	if err != nil {
		return nil, errors.Wrap(errors.ErrCodeInternalError, "查询知识本失败", err)
	}
	if kb == nil {
		return nil, errors.ErrKnowledgebaseNotFound
	}

	dir, err := s.dirRepo.FindByUUID(nil, dirUUID)
	if err != nil {
		return nil, errors.Wrap(errors.ErrCodeInternalError, "查询目录失败", err)
	}
	if dir == nil || dir.KnowledgebaseUUID != kbUUID {
		return nil, errors.New(errors.ErrCodeNotFound, "目录不存在")
	}

	// 2. 权限检查
	if err := s.permissionService.CheckEditPermission(kb, kbUUID, operator, isAdmin); err != nil {
		return nil, err
	}

	// 3. 验证目录名称
	if !domain.ValidateDirectoryName(name) {
		return nil, errors.New(errors.ErrCodeInvalidParam, "目录名称格式不正确")
	}

	// 4. 检查同级目录名称是否已存在（排除自己）
	existing, err := s.dirRepo.FindByKBUUIDAndNameAndParent(nil, kbUUID, name, dir.ParentUUID)
	if err != nil {
		return nil, errors.Wrap(errors.ErrCodeInternalError, "检查目录名称失败", err)
	}
	if existing != nil && existing.UUID != dirUUID {
		return nil, errors.New(errors.ErrCodeAlreadyExists, "同级目录名称已存在")
	}

	// 5. 更新目录
	oldName := dir.Name
	dir.Name = name

	err = s.txManager.ExecuteInTransactionWithContext(ctx, func(txCtx *database.TxContext) error {
		if err := s.dirRepo.Update(txCtx, dir); err != nil {
			return errors.Wrap(errors.ErrCodeInternalError, "更新目录失败", err)
		}

		// 记录操作日志
		detail, _ := json.Marshal(map[string]any{
			"oldName": oldName,
			"newName": name,
		})
		opLog := &domain.KnowledgebaseOperationLog{
			KnowledgebaseUUID: kbUUID,
			Operator:          operator,
			Operation:         constants.OperationUpdateDir,
			ResourceType:      constants.ResourceTypeDirectory,
			ResourceUUID:      dirUUID,
			Detail:            string(detail),
		}
		return s.logRepo.Create(txCtx, opLog)
	})

	if err != nil {
		return nil, err
	}

	// 6. 查询知识数量（名称更新不影响知识数量，此处单独查询避免 handler 层二次调用 GetDirectory）
	count, err := s.knowRepo.CountByDirectory(nil, dirUUID)
	if err != nil {
		return nil, errors.Wrap(errors.ErrCodeInternalError, "查询知识数量失败", err)
	}

	log.Logger.InfofCtx(ctx, "更新目录成功: uuid=%s, name=%s", dir.UUID, dir.Name)

	return &DirectoryWithCount{Directory: dir, KnowledgeCount: count}, nil
}

// DeleteDirectory 删除目录
// 级联删除：软删除目录本身及其所有后代子目录，以及这些目录下的全部知识
func (s *directoryServiceImpl) DeleteDirectory(ctx context.Context, kbUUID, dirUUID string, operator string, isAdmin bool) error {
	// 1. 检查知识本和目录存在性
	kb, err := s.kbRepo.FindByUUID(nil, kbUUID)
	if err != nil {
		return errors.Wrap(errors.ErrCodeInternalError, "查询知识本失败", err)
	}
	if kb == nil {
		return errors.ErrKnowledgebaseNotFound
	}

	dir, err := s.dirRepo.FindByUUID(nil, dirUUID)
	if err != nil {
		return errors.Wrap(errors.ErrCodeInternalError, "查询目录失败", err)
	}
	if dir == nil || dir.KnowledgebaseUUID != kbUUID {
		return errors.New(errors.ErrCodeNotFound, "目录不存在")
	}

	// 2. 权限检查
	if err := s.permissionService.CheckEditPermission(kb, kbUUID, operator, isAdmin); err != nil {
		return err
	}

	// 3. 默认目录不允许删除
	if dir.IsDefault {
		return errors.New(errors.ErrCodeInvalidParam, "默认目录不允许删除")
	}

	// 4. 收集所有后代目录 UUID（不含根目录自身）
	allDirs, err := s.dirRepo.ListByKBUUID(nil, kbUUID)
	if err != nil {
		return errors.Wrap(errors.ErrCodeInternalError, "查询目录列表失败", err)
	}
	descendantUUIDs := s.collectSubtreeUUIDs(dirUUID, allDirs)
	// 所有需要删除的目录（根目录 + 所有后代）
	allDeleteUUIDs := append([]string{dirUUID}, descendantUUIDs...)

	// 5. 事务级联删除：软删除所有目录及其下全部知识
	err = s.txManager.ExecuteInTransactionWithContext(ctx, func(txCtx *database.TxContext) error {
		// 软删除所有目录下的知识（批量）
		if err := s.knowRepo.SoftDeleteByDirectories(txCtx, allDeleteUUIDs); err != nil {
			return errors.Wrap(errors.ErrCodeInternalError, "删除目录下知识失败", err)
		}
		// 软删除所有目录（批量，含根目录）
		if err := s.dirRepo.SoftDeleteBatch(txCtx, allDeleteUUIDs); err != nil {
			return errors.Wrap(errors.ErrCodeInternalError, "删除目录失败", err)
		}

		// 记录操作日志
		detail, _ := json.Marshal(map[string]any{
			"name":           dir.Name,
			"subDirCount":    len(descendantUUIDs),
		})
		opLog := &domain.KnowledgebaseOperationLog{
			KnowledgebaseUUID: kbUUID,
			Operator:          operator,
			Operation:         constants.OperationDeleteDir,
			ResourceType:      constants.ResourceTypeDirectory,
			ResourceUUID:      dirUUID,
			Detail:            string(detail),
		}
		return s.logRepo.Create(txCtx, opLog)
	})

	if err != nil {
		return err
	}

	log.Logger.InfofCtx(ctx, "删除目录成功: uuid=%s, name=%s, subDirCount=%d", dir.UUID, dir.Name, len(descendantUUIDs))

	return nil
}

// calculateDepth 计算指定目录的层级深度（根目录为第 1 级）
// 包含循环引用检测：若数据不一致存在环路，返回 DirectoryMaxDepth+1 以触发深度超限校验
func (s *directoryServiceImpl) calculateDepth(dirUUID string, allDirs []*domain.KnowledgebaseDirectory) int {
	// 构建 uuid -> parentUUID 映射
	parentMap := make(map[string]*string, len(allDirs))
	for _, d := range allDirs {
		parentMap[d.UUID] = d.ParentUUID
	}

	depth := 1
	cur := dirUUID
	visited := make(map[string]bool)
	for {
		if visited[cur] {
			// 检测到循环引用，数据不一致，返回超限值触发校验失败
			log.Logger.Errorf("检测到目录循环引用: uuid=%s", cur)
			return constants.DirectoryMaxDepth + 1
		}
		visited[cur] = true
		parent, ok := parentMap[cur]
		if !ok || parent == nil {
			break
		}
		depth++
		cur = *parent
	}
	return depth
}

// collectSubtreeUUIDs 使用 BFS 收集以 rootUUID 为根的所有后代目录 UUID（不含 rootUUID 本身）
func (s *directoryServiceImpl) collectSubtreeUUIDs(rootUUID string, allDirs []*domain.KnowledgebaseDirectory) []string {
	return CollectSubtreeUUIDs(rootUUID, allDirs)
}

// GetDirectoryTree 获取目录树（应用层组装）
func (s *directoryServiceImpl) GetDirectoryTree(ctx context.Context, kbUUID, operator string, isAdmin bool) ([]*DirectoryTreeNode, int64, error) {
	// 1. 检查知识本是否存在
	kb, err := s.kbRepo.FindByUUID(nil, kbUUID)
	if err != nil {
		return nil, 0, errors.Wrap(errors.ErrCodeInternalError, "查询知识本失败", err)
	}
	if kb == nil {
		return nil, 0, errors.ErrKnowledgebaseNotFound
	}

	// 2. 权限检查
	if !s.permissionService.CheckViewPermission(kb, kbUUID, operator, isAdmin) {
		return nil, 0, errors.ErrNotKnowledgebaseMember
	}

	// 3. 获取所有目录
	allDirs, err := s.dirRepo.ListByKBUUID(nil, kbUUID)
	if err != nil {
		return nil, 0, errors.Wrap(errors.ErrCodeInternalError, "查询目录列表失败", err)
	}

	// 4. 批量查询知识数量
	dirUUIDs := make([]string, 0, len(allDirs))
	for _, d := range allDirs {
		dirUUIDs = append(dirUUIDs, d.UUID)
	}

	countMap := make(map[string]int64)
	if len(dirUUIDs) > 0 {
		countMap, err = s.knowRepo.CountByDirectories(nil, dirUUIDs)
		if err != nil {
			return nil, 0, errors.Wrap(errors.ErrCodeInternalError, "查询知识数量失败", err)
		}
	}

	// 5. 构建节点映射，查询 directory_uuid=NULL 的历史知识数量，累加到 is_default 目录
	if err := s.mergeNullCount(kbUUID, countMap, allDirs); err != nil {
		return nil, 0, errors.Wrap(errors.ErrCodeInternalError, "查询默认目录知识数量失败", err)
	}

	nodeMap := make(map[string]*DirectoryTreeNode, len(allDirs))
	for _, d := range allDirs {
		nodeMap[d.UUID] = &DirectoryTreeNode{
			Directory:      d,
			KnowledgeCount: countMap[d.UUID],
			Children:       []*DirectoryTreeNode{},
		}
	}

	// 6. 组装树
	roots := make([]*DirectoryTreeNode, 0)
	for _, d := range allDirs {
		node := nodeMap[d.UUID]
		if d.ParentUUID == nil {
			roots = append(roots, node)
		} else if parent, ok := nodeMap[*d.ParentUUID]; ok {
			parent.Children = append(parent.Children, node)
		} else {
			// 父节点已被删除，降级为根
			roots = append(roots, node)
		}
	}

	return roots, int64(len(roots)), nil
}

// MoveDirectory 移动目录到新的父目录
func (s *directoryServiceImpl) MoveDirectory(ctx context.Context, kbUUID, dirUUID string, targetParentUUID *string, operator string, isAdmin bool) (*domain.KnowledgebaseDirectory, error) {
	// 1. 检查知识本
	kb, err := s.kbRepo.FindByUUID(nil, kbUUID)
	if err != nil {
		return nil, errors.Wrap(errors.ErrCodeInternalError, "查询知识本失败", err)
	}
	if kb == nil {
		return nil, errors.ErrKnowledgebaseNotFound
	}

	// 2. 检查目录存在性
	dir, err := s.dirRepo.FindByUUID(nil, dirUUID)
	if err != nil {
		return nil, errors.Wrap(errors.ErrCodeInternalError, "查询目录失败", err)
	}
	if dir == nil || dir.KnowledgebaseUUID != kbUUID {
		return nil, errors.New(errors.ErrCodeNotFound, "目录不存在")
	}

	// 3. 权限检查
	if err := s.permissionService.CheckEditPermission(kb, kbUUID, operator, isAdmin); err != nil {
		return nil, err
	}

	// 4. 不能移动到自身
	if targetParentUUID != nil && *targetParentUUID == dirUUID {
		return nil, errors.New(errors.ErrCodeInvalidParam, "不能将目录移动到自身")
	}

	// 5. 目标父目录必须属于同一知识本（非nil时）
	if targetParentUUID != nil {
		targetDir, err := s.dirRepo.FindByUUID(nil, *targetParentUUID)
		if err != nil {
			return nil, errors.Wrap(errors.ErrCodeInternalError, "查询目标父目录失败", err)
		}
		if targetDir == nil || targetDir.KnowledgebaseUUID != kbUUID {
			return nil, errors.New(errors.ErrCodeNotFound, "目标父目录不存在")
		}

		// 6. 防止循环：目标父目录不能是被移动目录的后代
		allDirs, err := s.dirRepo.ListByKBUUID(nil, kbUUID)
		if err != nil {
			return nil, errors.Wrap(errors.ErrCodeInternalError, "查询目录列表失败", err)
		}
		descendants := s.collectSubtreeUUIDs(dirUUID, allDirs)
		if slices.Contains(descendants, *targetParentUUID) {
			return nil, errors.New(errors.ErrCodeInvalidParam, "不能将目录移动到其子目录下")
		}

		// 6.5. 校验移动后深度不超限：目标父目录深度 + 被移动子树最大相对深度 ≤ DirectoryMaxDepth
		targetDepth := s.calculateDepth(*targetParentUUID, allDirs)
		maxSubtreeDepth := s.calculateSubtreeMaxDepth(dirUUID, allDirs)
		if targetDepth+maxSubtreeDepth > constants.DirectoryMaxDepth {
			return nil, errors.New(errors.ErrCodeInvalidParam,
				fmt.Sprintf("移动后目录层级将超过上限（最多%d级）", constants.DirectoryMaxDepth))
		}
	}

	// 7. 同级同名唯一性检查
	existing, err := s.dirRepo.FindByKBUUIDAndNameAndParent(nil, kbUUID, dir.Name, targetParentUUID)
	if err != nil {
		return nil, errors.Wrap(errors.ErrCodeInternalError, "检查目录名称失败", err)
	}
	if existing != nil && existing.UUID != dirUUID {
		return nil, errors.New(errors.ErrCodeAlreadyExists, "目标位置已存在同名目录")
	}

	// 8. 事务更新
	err = s.txManager.ExecuteInTransactionWithContext(ctx, func(txCtx *database.TxContext) error {
		if err := s.dirRepo.UpdateParentUUID(txCtx, dirUUID, targetParentUUID); err != nil {
			return errors.Wrap(errors.ErrCodeInternalError, "移动目录失败", err)
		}

		detail, _ := json.Marshal(map[string]any{
			"name":             dir.Name,
			"targetParentUUID": ptrStringValue(targetParentUUID),
		})
		opLog := &domain.KnowledgebaseOperationLog{
			KnowledgebaseUUID: kbUUID,
			Operator:          operator,
			Operation:         constants.OperationMoveDir,
			ResourceType:      constants.ResourceTypeDirectory,
			ResourceUUID:      dirUUID,
			Detail:            string(detail),
		}
		return s.logRepo.Create(txCtx, opLog)
	})

	if err != nil {
		return nil, err
	}

	dir.ParentUUID = targetParentUUID
	log.Logger.InfofCtx(ctx, "移动目录成功: uuid=%s, name=%s", dir.UUID, dir.Name)

	return dir, nil
}

// ptrStringValue 将 *string 解引用为 any，nil 指针返回 nil（用于 JSON 序列化日志）
func ptrStringValue(s *string) any {
	if s == nil {
		return nil
	}
	return *s
}

// calculateSubtreeMaxDepth 计算以 rootUUID 为根的子树的最大相对深度（root 自身为 1）
func (s *directoryServiceImpl) calculateSubtreeMaxDepth(rootUUID string, allDirs []*domain.KnowledgebaseDirectory) int {
	childMap := make(map[string][]string)
	for _, d := range allDirs {
		if d.ParentUUID != nil {
			childMap[*d.ParentUUID] = append(childMap[*d.ParentUUID], d.UUID)
		}
	}

	var dfs func(string) int
	dfs = func(u string) int {
		children := childMap[u]
		if len(children) == 0 {
			return 1
		}
		maxChild := 0
		for _, child := range children {
			if d := dfs(child); d > maxChild {
				maxChild = d
			}
		}
		return maxChild + 1
	}
	return dfs(rootUUID)
}

// mergeNullCount 查询 directory_uuid=NULL 的历史知识数量，将其累加到 countMap 中 is_default 目录对应的计数
func (s *directoryServiceImpl) mergeNullCount(kbUUID string, countMap map[string]int64, dirs []*domain.KnowledgebaseDirectory) error {
	nullCount, err := s.knowRepo.CountByNullDirectory(nil, kbUUID)
	if err != nil {
		return err
	}
	if nullCount == 0 {
		return nil
	}

	defaultCount := 0
	for _, d := range dirs {
		if d.IsDefault {
			defaultCount++
			if defaultCount == 1 {
				// 只累加到第一个默认目录，防止重复累加
				countMap[d.UUID] += nullCount
			}
		}
	}

	if defaultCount > 1 {
		log.Logger.Warnf("知识本 %s 存在多个默认目录(%d个)，数据不一致，nullCount 仅累加到第一个", kbUUID, defaultCount)
	} else if defaultCount == 0 {
		log.Logger.Warnf("知识本 %s 不存在默认目录但有 %d 条 NULL 知识，数据不一致", kbUUID, nullCount)
	}

	return nil
}

