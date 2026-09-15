// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/api/request"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/domain"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/infrastructure/cache"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/infrastructure/database"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/infrastructure/infoflow"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/infrastructure/uuap"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/repository"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/errors"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/log"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/types/constants"
)

// userGroupsCacheTTL 返回 [10min, 20min) 的随机 TTL，防止缓存集中过期
func userGroupsCacheTTL() time.Duration {
	const min, max = 10 * time.Minute, 20 * time.Minute
	return min + time.Duration(rand.Int63n(int64(max-min)))
}

// KnowledgebaseWithRole 带权限信息的知识本
type KnowledgebaseWithRole struct {
	KB             *domain.Knowledgebase
	Role           string // admin | editor | viewer | ""
	MemberCount    int64  // 成员数量
	TagCount       int64  // 标签数量
	KnowledgeCount int64  // 知识数量
}

// KnowledgebaseService 知识本服务接口
type KnowledgebaseService interface {
	// CreateKnowledgebase 创建知识本
	// 返回: 知识本、成员数量、错误
	CreateKnowledgebase(ctx context.Context, req *request.CreateKnowledgebaseRequest, username string) (*domain.Knowledgebase, int64, error)

	// GetKnowledgebase 获取知识本详情
	// 返回: 知识本、成员数量、标签数量、知识数量、错误
	GetKnowledgebase(ctx context.Context, uuid, username string) (*domain.Knowledgebase, int64, int64, int64, error)

	// GetKnowledgebaseByName 根据名称获取知识本
	// 返回: 知识本、成员数量、标签数量、知识数量、错误
	GetKnowledgebaseByName(ctx context.Context, name, username string) (*domain.Knowledgebase, int64, int64, int64, error)

	// ListKnowledgebases 查询知识本列表
	ListKnowledgebases(ctx context.Context, req *request.ListKnowledgebasesRequest, username string) ([]*KnowledgebaseWithRole, int64, error)

	// ListMyKnowledgebases 查询用户有权限的知识本列表
	// isOpenAPI: 是否为OpenAPI调用,如果是则只返回owner为当前用户的知识本
	ListMyKnowledgebases(ctx context.Context, req *request.ListMyKnowledgebasesRequest, username string, isOpenAPI bool) ([]*KnowledgebaseWithRole, int64, error)

	// UpdateKnowledgebase 更新知识本
	// 返回: 知识本、成员数量、标签数量、知识数量、错误
	UpdateKnowledgebase(ctx context.Context, uuid string, req *request.UpdateKnowledgebaseRequest, username string, isAdmin bool) (*domain.Knowledgebase, int64, int64, int64, error)

	// DeleteKnowledgebase 删除知识本
	DeleteKnowledgebase(ctx context.Context, uuid, username string, isAdmin bool) error

	// ListGroupKnowledgebases 查询群聊有权限的知识本列表
	// 返回以 groupID 作为 chat_group 成员注册的知识本列表及该群的角色
	// role: 可选的角色过滤条件（admin/editor/viewer），为空时不过滤
	ListGroupKnowledgebases(ctx context.Context, groupID string, keyword string, role string, page, pageSize int) ([]*KnowledgebaseWithRole, int64, error)
}

// knowledgebaseServiceImpl 知识本服务实现
type knowledgebaseServiceImpl struct {
	kbRepo            repository.KnowledgebaseRepository
	memberRepo        repository.KnowledgebaseMemberRepository
	tagRepo           repository.KnowledgebaseTagRepository
	knowRepo          repository.KnowledgeRepository
	operationRepo     repository.KnowledgebaseOperationLogRepository
	dirRepo           repository.KnowledgebaseDirectoryRepository
	permissionService PermissionService
	txManager         *database.TransactionManager
	uuapClient        *uuap.UUAPClient
	infoFlowClient    *infoflow.Client
}

// NewKnowledgebaseService 创建知识本服务实例
func NewKnowledgebaseService(
	kbRepo repository.KnowledgebaseRepository,
	memberRepo repository.KnowledgebaseMemberRepository,
	tagRepo repository.KnowledgebaseTagRepository,
	knowRepo repository.KnowledgeRepository,
	operationRepo repository.KnowledgebaseOperationLogRepository,
	dirRepo repository.KnowledgebaseDirectoryRepository,
	uuapClient *uuap.UUAPClient,
	infoFlowClient *infoflow.Client,
) KnowledgebaseService {
	return &knowledgebaseServiceImpl{
		kbRepo:            kbRepo,
		memberRepo:        memberRepo,
		tagRepo:           tagRepo,
		knowRepo:          knowRepo,
		operationRepo:     operationRepo,
		dirRepo:           dirRepo,
		permissionService: NewPermissionService(memberRepo),
		txManager:         database.NewTransactionManager(),
		uuapClient:        uuapClient,
		infoFlowClient:    infoFlowClient,
	}
}

func (s *knowledgebaseServiceImpl) CreateKnowledgebase(
	ctx context.Context,
	req *request.CreateKnowledgebaseRequest,
	username string,
) (*domain.Knowledgebase, int64, error) {
	// 验证名称格式
	if !domain.ValidateName(req.Name) {
		return nil, 0, errors.ErrInvalidKnowledgebaseName
	}

	// 检查名称是否已存在（创建时不需要排除自身）
	exists, err := s.kbRepo.CheckNameExists(nil, req.Name, "")
	if err != nil {
		return nil, 0, errors.Wrap(errors.ErrCodeInternalError, "检查名称失败", err)
	}
	if exists {
		return nil, 0, errors.ErrKnowledgebaseAlreadyExists
	}

	// 校验附带的成员列表
	if len(req.Members) > 0 {
		// personal 类型不允许添加成员
		if req.Visibility == constants.VisibilityPersonal {
			return nil, 0, errors.ErrCannotManagePersonalKB
		}
		// 校验成员去重
		if err := validateCreateMembers(req.Members, username); err != nil {
			return nil, 0, err
		}
		// 校验成员合法性（用户/邮件组/部门是否真实存在）
		if err := s.validateMembersExist(req.Members, username); err != nil {
			return nil, 0, err
		}
	}

	// 过滤掉创建者自身（创建者已默认为 admin）
	filteredMembers := filterOutCreator(req.Members, username)

	// 创建知识本
	kb := &domain.Knowledgebase{
		UUID:        uuid.New().String(),
		Name:        req.Name,
		Description: req.Description,
		Visibility:  req.Visibility,
		Owner:       username,
		Status:      constants.KnowledgebaseStatusEnabled,
	}

	// 在事务中执行创建操作
	err = s.txManager.ExecuteInTransactionWithContext(ctx, func(txCtx *database.TxContext) error {
		// 创建知识本
		if err := s.kbRepo.Create(txCtx, kb); err != nil {
			return errors.Wrap(errors.ErrCodeInternalError, "创建知识本失败", err)
		}

		// 创建默认目录（parent_uuid = NULL，真实入库）
		defaultDir := &domain.KnowledgebaseDirectory{
			UUID:              uuid.New().String(),
			KnowledgebaseUUID: kb.UUID,
			ParentUUID:        nil,
			Name:              constants.DirectoryRootName,
			IsDefault:         true,
		}
		if err := s.dirRepo.Create(txCtx, defaultDir); err != nil {
			return errors.Wrap(errors.ErrCodeInternalError, "创建默认目录失败", err)
		}

		// 添加创建者为管理员成员
		member := &domain.KnowledgebaseMember{
			UUID:              uuid.New().String(),
			KnowledgebaseUUID: kb.UUID,
			MemberType:        constants.KnowledgebaseMemberTypeUsername,
			MemberValue:       username,
			Role:              constants.KnowledgebaseMemberRoleAdmin,
		}
		if err := s.memberRepo.Create(txCtx, member); err != nil {
			return errors.Wrap(errors.ErrCodeInternalError, "创建成员失败", err)
		}

		// 批量插入附带的成员
		for _, m := range filteredMembers {
			newMember := &domain.KnowledgebaseMember{
				UUID:              uuid.New().String(),
				KnowledgebaseUUID: kb.UUID,
				MemberType:        m.MemberType,
				MemberValue:       m.MemberValue,
				Role:              m.Role,
			}
			if err := s.memberRepo.Create(txCtx, newMember); err != nil {
				return errors.Wrap(errors.ErrCodeInternalError,
					fmt.Sprintf("创建成员 %s 失败", m.MemberValue), err)
			}

			// 记录添加成员操作日志
			memberDetail, err := json.Marshal(map[string]interface{}{
				"memberType":  m.MemberType,
				"memberValue": m.MemberValue,
				"role":        m.Role,
			})
			if err != nil {
				return errors.Wrap(errors.ErrCodeInternalError, "序列化成员操作日志详情失败", err)
			}
			memberLog := &domain.KnowledgebaseOperationLog{
				KnowledgebaseUUID: kb.UUID,
				Operator:          username,
				Operation:         constants.OperationAddMember,
				ResourceType:      constants.ResourceTypeMember,
				ResourceUUID:      newMember.UUID,
				Detail:            string(memberDetail),
			}
			if err := s.operationRepo.Create(txCtx, memberLog); err != nil {
				return errors.Wrap(errors.ErrCodeInternalError, "记录成员操作日志失败", err)
			}
		}

		// 记录创建知识本操作日志
		detail, err := json.Marshal(map[string]interface{}{
			"name":        kb.Name,
			"description": kb.Description,
			"visibility":  kb.Visibility,
		})
		if err != nil {
			return errors.Wrap(errors.ErrCodeInternalError, "序列化操作日志详情失败", err)
		}

		opLog := &domain.KnowledgebaseOperationLog{
			KnowledgebaseUUID: kb.UUID,
			Operator:          username,
			Operation:         constants.OperationCreateKB,
			ResourceType:      constants.ResourceTypeKnowledgebase,
			ResourceUUID:      kb.UUID,
			Detail:            string(detail),
		}
		if err := s.operationRepo.Create(txCtx, opLog); err != nil {
			return errors.Wrap(errors.ErrCodeInternalError, "记录操作日志失败", err)
		}

		return nil
	})

	if err != nil {
		return nil, 0, err
	}

	// 创建成功后，删除列表缓存
	s.kbRepo.InvalidateKBCache("", "")

	// 实际成员数量 = 创建者(1) + 附带成员数
	memberCount := int64(1 + len(filteredMembers))

	log.Logger.InfofCtx(ctx, "创建知识本成功: uuid=%s, name=%s, owner=%s", kb.UUID, kb.Name, username)

	return kb, memberCount, nil
}

// validateCreateMembers 校验创建知识本时附带的成员列表
// 检查成员内部是否有重复
func validateCreateMembers(members []request.CreateKnowledgebaseMember, creator string) error {
	seen := make(map[string]struct{})
	for _, m := range members {
		key := m.MemberType + ":" + m.MemberValue
		// 跳过创建者自身的去重检查（后续会过滤掉）
		if m.MemberType == constants.KnowledgebaseMemberTypeUsername && m.MemberValue == creator {
			continue
		}
		if _, exists := seen[key]; exists {
			return errors.ErrDuplicateMemberInRequest
		}
		seen[key] = struct{}{}
	}
	return nil
}

// filterOutCreator 过滤掉创建者自身（创建者已默认为 admin）
func filterOutCreator(members []request.CreateKnowledgebaseMember, creator string) []request.CreateKnowledgebaseMember {
	if len(members) == 0 {
		return nil
	}
	filtered := make([]request.CreateKnowledgebaseMember, 0, len(members))
	for _, m := range members {
		if m.MemberType == constants.KnowledgebaseMemberTypeUsername && m.MemberValue == creator {
			continue
		}
		filtered = append(filtered, m)
	}
	return filtered
}

// checkViewPermission 检查查看权限
// 检查逻辑:
// 1. 公开知识本，所有用户都有权限
// 2. 个人知识本，只有owner本人有权限
// 3. 私密知识本，admin/editor/viewer 成员有权限
// 返回: true 表示有权限，false 表示无权限
func (s *knowledgebaseServiceImpl) checkViewPermission(kb *domain.Knowledgebase, username string) bool {
	// 公开知识本允许访问
	if kb.Visibility == constants.VisibilityPublic {
		return true
	}

	// 个人知识本只允许owner访问
	if kb.Visibility == constants.VisibilityPersonal {
		return kb.Owner == username
	}

	// 检查用户是否为成员
	role, err := s.memberRepo.GetUserRole(nil, kb.UUID, username)
	if err != nil {
		return false
	}

	// 空字符串表示非成员，不允许访问
	return role != ""
}

// checkEditPermission 检查编辑权限
// 检查逻辑:
// 1. 个人知识本，只有owner本人有权限
// 2. 私密/公开知识本，只有 admin 成员有权限
// 返回: true 表示有权限，false 表示无权限
func (s *knowledgebaseServiceImpl) checkEditPermission(kb *domain.Knowledgebase, username string) bool {
	// 个人知识本只允许owner编辑
	if kb.Visibility == constants.VisibilityPersonal && kb.Owner != username {
		return false
	}

	role, err := s.memberRepo.GetUserRole(nil, kb.UUID, username)
	if err != nil {
		return false
	}

	// 只有admin可以编辑知识本信息
	return role == constants.KnowledgebaseMemberRoleAdmin
}

func (s *knowledgebaseServiceImpl) GetKnowledgebase(ctx context.Context, uuid, username string) (*domain.Knowledgebase, int64, int64, int64, error) {
	kb, err := s.kbRepo.FindByUUID(nil, uuid)
	if err != nil {
		return nil, 0, 0, 0, errors.Wrap(errors.ErrCodeInternalError, "查询知识本失败", err)
	}
	if kb == nil {
		return nil, 0, 0, 0, errors.ErrKnowledgebaseNotFound
	}

	// 检查查看权限
	if !s.checkViewPermission(kb, username) {
		return nil, 0, 0, 0, errors.ErrNotKnowledgebaseMember
	}

	// 查询成员数量
	memberCount, err := s.memberRepo.CountByKBUUID(nil, kb.UUID)
	if err != nil {
		return nil, 0, 0, 0, errors.Wrap(errors.ErrCodeInternalError, "查询成员数量失败", err)
	}

	// 查询标签数量
	tagCount, err := s.tagRepo.CountByKBUUID(nil, kb.UUID)
	if err != nil {
		return nil, 0, 0, 0, errors.Wrap(errors.ErrCodeInternalError, "查询标签数量失败", err)
	}

	// 查询知识数量
	knowledgeCount, err := s.knowRepo.CountByKBUUID(nil, kb.UUID)
	if err != nil {
		return nil, 0, 0, 0, errors.Wrap(errors.ErrCodeInternalError, "查询知识数量失败", err)
	}

	log.Logger.InfofCtx(ctx, "获取知识本详情: uuid=%s, name=%s", kb.UUID, kb.Name)

	return kb, memberCount, tagCount, knowledgeCount, nil
}

func (s *knowledgebaseServiceImpl) GetKnowledgebaseByName(ctx context.Context, name, username string) (*domain.Knowledgebase, int64, int64, int64, error) {
	kb, err := s.kbRepo.FindByName(nil, name)
	if err != nil {
		return nil, 0, 0, 0, errors.Wrap(errors.ErrCodeInternalError, "查询知识本失败", err)
	}
	if kb == nil {
		return nil, 0, 0, 0, errors.ErrKnowledgebaseNotFound
	}

	// 检查查看权限
	if !s.checkViewPermission(kb, username) {
		return nil, 0, 0, 0, errors.ErrNotKnowledgebaseMember
	}

	// 查询成员数量
	memberCount, err := s.memberRepo.CountByKBUUID(nil, kb.UUID)
	if err != nil {
		return nil, 0, 0, 0, errors.Wrap(errors.ErrCodeInternalError, "查询成员数量失败", err)
	}

	// 查询标签数量
	tagCount, err := s.tagRepo.CountByKBUUID(nil, kb.UUID)
	if err != nil {
		return nil, 0, 0, 0, errors.Wrap(errors.ErrCodeInternalError, "查询标签数量失败", err)
	}

	// 查询知识数量
	knowledgeCount, err := s.knowRepo.CountByKBUUID(nil, kb.UUID)
	if err != nil {
		return nil, 0, 0, 0, errors.Wrap(errors.ErrCodeInternalError, "查询知识数量失败", err)
	}

	log.Logger.InfofCtx(ctx, "根据名称获取知识本: name=%s, uuid=%s", name, kb.UUID)

	return kb, memberCount, tagCount, knowledgeCount, nil
}

func (s *knowledgebaseServiceImpl) ListKnowledgebases(
	ctx context.Context,
	req *request.ListKnowledgebasesRequest,
	username string,
) ([]*KnowledgebaseWithRole, int64, error) {
	// 处理分页参数
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}
	if req.PageSize > 100 {
		req.PageSize = 100
	}

	offset := (req.Page - 1) * req.PageSize

	// 如果查询个人可见性，强制设置owner为当前用户
	if req.Visibility == constants.VisibilityPersonal {
		req.Owner = username
	}

	// 【核心变更】传入 username 参数，让 Repository 在 SQL 层过滤非本人的个人知识本
	kbs, total, err := s.kbRepo.ListByConditions(
		nil, // 非事务操作
		req.Keyword,
		req.Visibility,
		req.Owner,
		username, // 当前用户名，用于过滤
		offset,
		req.PageSize,
	)
	if err != nil {
		return nil, 0, errors.Wrap(errors.ErrCodeInternalError, "查询知识本列表失败", err)
	}

	// 如果没有知识本，直接返回空列表
	if len(kbs) == 0 {
		return []*KnowledgebaseWithRole{}, total, nil
	}

	// 批量查询用户在这些知识本中的角色
	kbUUIDs := make([]string, len(kbs))
	for i, kb := range kbs {
		kbUUIDs[i] = kb.UUID
	}

	roleMap, err := s.memberRepo.GetUserRolesBatch(nil, kbUUIDs, username)
	if err != nil {
		return nil, 0, errors.Wrap(errors.ErrCodeInternalError, "批量查询用户角色失败", err)
	}

	// 批量查询成员数量
	memberCountMap, err := s.memberRepo.CountByKBUUIDsBatch(nil, kbUUIDs)
	if err != nil {
		return nil, 0, errors.Wrap(errors.ErrCodeInternalError, "批量查询成员数量失败", err)
	}

	// 批量查询标签数量
	tagCountMap, err := s.tagRepo.CountByKBUUIDsBatch(nil, kbUUIDs)
	if err != nil {
		return nil, 0, errors.Wrap(errors.ErrCodeInternalError, "批量查询标签数量失败", err)
	}

	// 批量查询知识数量
	knowledgeCountMap, err := s.knowRepo.CountByKBUUIDsBatch(nil, kbUUIDs)
	if err != nil {
		return nil, 0, errors.Wrap(errors.ErrCodeInternalError, "批量查询知识数量失败", err)
	}

	// 组装结果
	result := make([]*KnowledgebaseWithRole, 0, len(kbs))
	for _, kb := range kbs {
		role := roleMap[kb.UUID]

		// 公开知识本，如果未查到成员信息，默认为 viewer
		if kb.Visibility == constants.VisibilityPublic && role == "" {
			role = constants.KnowledgebaseMemberRoleViewer
		}

		result = append(result, &KnowledgebaseWithRole{
			KB:             kb,
			Role:           role,
			MemberCount:    memberCountMap[kb.UUID],
			TagCount:       tagCountMap[kb.UUID],
			KnowledgeCount: knowledgeCountMap[kb.UUID],
		})
	}

	return result, total, nil
}

func (s *knowledgebaseServiceImpl) UpdateKnowledgebase(
	ctx context.Context,
	uuid string,
	req *request.UpdateKnowledgebaseRequest,
	username string,
	isAdmin bool,
) (*domain.Knowledgebase, int64, int64, int64, error) {
	// 获取知识本（非事务操作）
	kb, err := s.kbRepo.FindByUUID(nil, uuid)
	if err != nil {
		return nil, 0, 0, 0, errors.Wrap(errors.ErrCodeInternalError, "查询知识本失败", err)
	}
	if kb == nil {
		return nil, 0, 0, 0, errors.ErrKnowledgebaseNotFound
	}

	// 检查编辑权限
	if err := s.permissionService.CheckUpdateKnowledgebasePermission(kb, kb.UUID, username, isAdmin); err != nil {
		return nil, 0, 0, 0, err
	}

	// 如果修改名称，验证格式和唯一性
	if req.Name != "" && req.Name != kb.Name {
		if !domain.ValidateName(req.Name) {
			return nil, 0, 0, 0, errors.ErrInvalidKnowledgebaseName
		}

		// 检查名称是否已存在，排除当前知识本
		exists, err := s.kbRepo.CheckNameExists(nil, req.Name, kb.UUID)
		if err != nil {
			return nil, 0, 0, 0, errors.Wrap(errors.ErrCodeInternalError, "检查名称失败", err)
		}
		if exists {
			return nil, 0, 0, 0, errors.ErrKnowledgebaseAlreadyExists
		}
	}

	// 记录变更前数据
	oldData := map[string]interface{}{
		"name":        kb.Name,
		"description": kb.Description,
		"visibility":  kb.Visibility,
		"status":      kb.Status,
	}

	// 更新字段
	if req.Name != "" {
		kb.Name = req.Name
	}
	if req.Description != "" {
		kb.Description = req.Description
	}
	if req.Visibility != "" {
		// 校验：不允许将已有成员的知识本变更为个人可见性
		if kb.Visibility != constants.VisibilityPersonal && req.Visibility == constants.VisibilityPersonal {
			// 检查是否有成员（除了owner之外的成员）
			members, err := s.memberRepo.ListByKBUUID(nil, kb.UUID)
			if err != nil {
				return nil, 0, 0, 0, errors.Wrap(errors.ErrCodeInternalError, "查询成员失败", err)
			}
			// 只有创建者（owner）且无其他成员时才允许变更为个人可见性
			if len(members) > 1 || (len(members) == 1 && members[0].MemberValue != username) {
				return nil, 0, 0, 0, errors.ErrInvalidVisibility
			}
		}
		kb.Visibility = req.Visibility
	}

	// 在事务中执行更新操作
	err = s.txManager.ExecuteInTransactionWithContext(ctx, func(txCtx *database.TxContext) error {
		// 更新知识本
		if err := s.kbRepo.Update(txCtx, kb); err != nil {
			return errors.Wrap(errors.ErrCodeInternalError, "更新知识本失败", err)
		}

		// 记录操作日志
		newData := map[string]interface{}{
			"name":        kb.Name,
			"description": kb.Description,
			"visibility":  kb.Visibility,
			"status":      kb.Status,
		}

		detail, err := json.Marshal(map[string]interface{}{
			"old": oldData,
			"new": newData,
		})
		if err != nil {
			return errors.Wrap(errors.ErrCodeInternalError, "序列化操作日志详情失败", err)
		}

		opLog := &domain.KnowledgebaseOperationLog{
			KnowledgebaseUUID: kb.UUID,
			Operator:          username,
			Operation:         constants.OperationUpdateKB,
			ResourceType:      constants.ResourceTypeKnowledgebase,
			ResourceUUID:      kb.UUID,
			Detail:            string(detail),
		}
		if err := s.operationRepo.Create(txCtx, opLog); err != nil {
			return errors.Wrap(errors.ErrCodeInternalError, "记录操作日志失败", err)
		}

		return nil
	})

	if err != nil {
		return nil, 0, 0, 0, err
	}

	// 事务成功后，删除相关缓存
	s.invalidateKBCache(kb.UUID, kb.Name)

	// 重新查询获取最新的 UpdatedAt 时间
	updatedKB, err := s.kbRepo.FindByUUID(nil, kb.UUID)
	if err != nil {
		return nil, 0, 0, 0, errors.Wrap(errors.ErrCodeInternalError, "查询更新后的知识本失败", err)
	}
	if updatedKB == nil {
		return nil, 0, 0, 0, errors.ErrKnowledgebaseNotFound
	}

	// 查询成员数量
	memberCount, err := s.memberRepo.CountByKBUUID(nil, kb.UUID)
	if err != nil {
		return nil, 0, 0, 0, errors.Wrap(errors.ErrCodeInternalError, "查询成员数量失败", err)
	}

	// 查询标签数量
	tagCount, err := s.tagRepo.CountByKBUUID(nil, kb.UUID)
	if err != nil {
		return nil, 0, 0, 0, errors.Wrap(errors.ErrCodeInternalError, "查询标签数量失败", err)
	}

	// 查询知识数量
	knowledgeCount, err := s.knowRepo.CountByKBUUID(nil, kb.UUID)
	if err != nil {
		return nil, 0, 0, 0, errors.Wrap(errors.ErrCodeInternalError, "查询知识数量失败", err)
	}

	log.Logger.InfofCtx(ctx, "更新知识本成功: uuid=%s, name=%s", updatedKB.UUID, updatedKB.Name)

	return updatedKB, memberCount, tagCount, knowledgeCount, nil
}

// ListMyKnowledgebases 查询用户有权限的知识本列表
// isOpenAPI: 是否为OpenAPI调用,如果是则只返回owner为当前用户的知识本
func (s *knowledgebaseServiceImpl) ListMyKnowledgebases(
	ctx context.Context,
	req *request.ListMyKnowledgebasesRequest,
	username string,
	isOpenAPI bool,
) ([]*KnowledgebaseWithRole, int64, error) {
	// 处理分页参数
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}
	if req.PageSize > 100 {
		req.PageSize = 100
	}

	offset := (req.Page - 1) * req.PageSize

	// OpenAPI 调用时,直接查询 owner 为当前用户的所有知识本
	if isOpenAPI {
		kbs, total, err := s.kbRepo.ListByConditions(
			nil,
			req.Keyword,
			req.Visibility,
			username, // owner 过滤
			"",       // OpenAPI场景不需要personal过滤（已通过owner过滤）
			offset,
			req.PageSize,
		)
		if err != nil {
			return nil, 0, errors.Wrap(errors.ErrCodeInternalError, "查询知识本列表失败", err)
		}

		// 如果没有知识本，直接返回空列表
		if len(kbs) == 0 {
			return []*KnowledgebaseWithRole{}, total, nil
		}

		// 批量查询用户在这些知识本中的角色
		kbUUIDs := make([]string, len(kbs))
		for i, kb := range kbs {
			kbUUIDs[i] = kb.UUID
		}

		roleMap, err := s.memberRepo.GetUserRolesBatch(nil, kbUUIDs, username)
		if err != nil {
			return nil, 0, errors.Wrap(errors.ErrCodeInternalError, "批量查询用户角色失败", err)
		}

		// 批量查询统计信息
		memberCountMap, err := s.memberRepo.CountByKBUUIDsBatch(nil, kbUUIDs)
		if err != nil {
			return nil, 0, errors.Wrap(errors.ErrCodeInternalError, "批量查询成员数量失败", err)
		}

		tagCountMap, err := s.tagRepo.CountByKBUUIDsBatch(nil, kbUUIDs)
		if err != nil {
			return nil, 0, errors.Wrap(errors.ErrCodeInternalError, "批量查询标签数量失败", err)
		}

		knowledgeCountMap, err := s.knowRepo.CountByKBUUIDsBatch(nil, kbUUIDs)
		if err != nil {
			return nil, 0, errors.Wrap(errors.ErrCodeInternalError, "批量查询知识数量失败", err)
		}

		// 组装结果，若有 role 过滤则跳过不匹配的条目
		result := make([]*KnowledgebaseWithRole, 0, len(kbs))
		for _, kb := range kbs {
			role := roleMap[kb.UUID]
			// owner 的知识本默认为 admin
			if role == "" {
				role = constants.KnowledgebaseMemberRoleAdmin
			}

			// role 过滤
			if req.Role != "" && role != req.Role {
				continue
			}

			result = append(result, &KnowledgebaseWithRole{
				KB:             kb,
				Role:           role,
				MemberCount:    memberCountMap[kb.UUID],
				TagCount:       tagCountMap[kb.UUID],
				KnowledgeCount: knowledgeCountMap[kb.UUID],
			})
		}

		return result, total, nil
	}

	// 非 OpenAPI 调用,查询所有有权限的知识本
	// Step 1: 查询直接成员关系（username）
	directKBUUIDs, err := s.memberRepo.FindKBUUIDsByMemberValues(
		nil,
		constants.KnowledgebaseMemberTypeUsername,
		[]string{username},
	)
	if err != nil {
		return nil, 0, errors.Wrap(errors.ErrCodeInternalError, "查询直接成员关系失败", err)
	}

	// Step 2: 查询邮件组成员关系（带降级）
	emailGroupKBUUIDs := []string{}
	if s.uuapClient != nil {
		emailGroups, err := s.uuapClient.GetUserEmailGroupAddresses(nil, username)
		if err != nil {
			// 降级处理：记录日志但不阻断流程
			log.Logger.WarnfCtx(ctx, "查询用户邮件组失败(降级处理): username=%s, error=%v", username, err)
		} else if len(emailGroups) > 0 {
			log.Logger.InfofCtx(ctx, "查询到用户邮件组: username=%s, count=%d, groups=%v",
				username, len(emailGroups), emailGroups)
			emailGroupKBUUIDs, err = s.memberRepo.FindKBUUIDsByMemberValues(
				nil,
				constants.KnowledgebaseMemberTypeEmailGroup,
				emailGroups,
			)
			if err != nil {
				return nil, 0, errors.Wrap(errors.ErrCodeInternalError, "查询邮件组成员关系失败", err)
			}
		}
	}

	// // Step 3: 查询部门成员关系（带降级） - 产品上未实现
	// deptKBUUIDs := []string{}
	// if s.uuapClient != nil {
	// 	dept, err := s.uuapClient.GetUserBusinessDeptOrg(nil, username)
	// 	if err != nil {
	// 		// 降级处理：记录日志但不阻断流程
	// 		log.Logger.Warnf("查询用户部门失败(降级处理): username=%s, error=%v", username, err)
	// 	} else if dept != nil {
	// 		deptKeys := dept.GetAllDepartmentKeys()
	// 		if len(deptKeys) > 0 {
	// 			log.Logger.Infof("查询到用户部门: username=%s, count=%d, depts=%v",
	// 				username, len(deptKeys), deptKeys)
	// 			deptKBUUIDs, err = s.memberRepo.FindKBUUIDsByMemberValues(
	// 				nil,
	// 				constants.KnowledgebaseMemberTypeDepartment,
	// 				deptKeys,
	// 			)
	// 			if err != nil {
	// 				return nil, 0, errors.Wrap(errors.ErrCodeInternalError, "查询部门成员关系失败", err)
	// 			}
	// 		}
	// 	}
	// }

	// Step 3: 查询群成员关系（使用如流API）
	// 策略：先查 user_matched_groups 缓存；未命中则调 IsUserInGroups（内部已缓存群成员列表），并写缓存
	var matchedGroupIDs []string
	chatGroupKBUUIDs := []string{}
	if s.infoFlowClient != nil {
		allGroupValues, err := s.memberRepo.FindDistinctMemberValues(nil, constants.KnowledgebaseMemberTypeChatGroup)
		if err != nil {
			log.Logger.Warnf("查询群成员配置失败(降级处理): username=%s, error=%v", username, err)
		} else if len(allGroupValues) > 0 {
			cacheHit := false
			cacher := cache.GetCache()
			userGroupsCacheKey := fmt.Sprintf("%s%s", constants.CacheKeyUserMatchedGroups, username)
			if cacher != nil {
				if cached, cerr := cacher.Get(context.Background(), userGroupsCacheKey); cerr == nil && cached != "" {
					var ids []string
					if jerr := json.Unmarshal([]byte(cached), &ids); jerr == nil {
						matchedGroupIDs = ids
						cacheHit = true
						log.Logger.Debugf("缓存命中用户群列表: username=%s, count=%d", username, len(matchedGroupIDs))
					}
				}
			}
			if !cacheHit {
				matchedGroupIDs, err = s.infoFlowClient.IsUserInGroups(context.Background(), allGroupValues, username)
				if err != nil {
					log.Logger.Warnf("查询用户群失败(降级处理): username=%s, error=%v", username, err)
				}
				if cacher != nil && err == nil {
					if data, merr := json.Marshal(matchedGroupIDs); merr == nil {
						if serr := cacher.Set(context.Background(), userGroupsCacheKey, string(data), userGroupsCacheTTL()); serr != nil {
							log.Logger.Warnf("写入用户群缓存失败: %v", serr)
						}
					}
				}
			}
			if len(matchedGroupIDs) > 0 {
				log.Logger.Infof("查询到用户群: username=%s, count=%d, groups=%v",
					username, len(matchedGroupIDs), matchedGroupIDs)
				chatGroupKBUUIDs, err = s.memberRepo.FindKBUUIDsByMemberValues(
					nil,
					constants.KnowledgebaseMemberTypeChatGroup,
					matchedGroupIDs,
				)
				if err != nil {
					return nil, 0, errors.Wrap(errors.ErrCodeInternalError, "查询群成员关系失败", err)
				}
			}
		}
	}

	// Step 4: 查询个人知识本UUID（优化：只查询UUID）
	personalUUIDs, err := s.kbRepo.FindUUIDsByOwner(nil, username, constants.VisibilityPersonal)
	if err != nil {
		return nil, 0, errors.Wrap(errors.ErrCodeInternalError, "查询个人知识本UUID失败", err)
	}

	// Step 5: 查询公开知识本UUID（优化：只查询UUID）
	publicUUIDs, err := s.kbRepo.FindUUIDsByVisibility(nil, constants.VisibilityPublic)
	if err != nil {
		return nil, 0, errors.Wrap(errors.ErrCodeInternalError, "查询公开知识本UUID失败", err)
	}

	// Step 6: 合并去重所有UUID
	allKBUUIDs := mergeDeduplicate(directKBUUIDs, emailGroupKBUUIDs, chatGroupKBUUIDs, personalUUIDs, publicUUIDs)

	// 如果没有任何权限，返回空列表
	if len(allKBUUIDs) == 0 {
		return []*KnowledgebaseWithRole{}, 0, nil
	}

	// Step 7: 若需要 role 过滤，全量查询角色后在分页前完成过滤
	var roleMap map[string]string
	if req.Role != "" {
		roleMap, err = s.memberRepo.GetUserRolesBatchWithHint(nil, allKBUUIDs, username, matchedGroupIDs)
		if err != nil {
			return nil, 0, errors.Wrap(errors.ErrCodeInternalError, "批量查询用户角色失败", err)
		}

		// 公开知识本中非成员默认视为 viewer
		publicUUIDSet := make(map[string]struct{}, len(publicUUIDs))
		for _, uuid := range publicUUIDs {
			publicUUIDSet[uuid] = struct{}{}
		}
		for _, uuid := range allKBUUIDs {
			if _, isPublic := publicUUIDSet[uuid]; isPublic && roleMap[uuid] == "" {
				roleMap[uuid] = constants.KnowledgebaseMemberRoleViewer
			}
		}

		// 按 role 过滤 UUID 列表
		filtered := make([]string, 0, len(allKBUUIDs))
		for _, uuid := range allKBUUIDs {
			if roleMap[uuid] == req.Role {
				filtered = append(filtered, uuid)
			}
		}
		allKBUUIDs = filtered
	}

	// 过滤后为空，直接返回
	if len(allKBUUIDs) == 0 {
		return []*KnowledgebaseWithRole{}, 0, nil
	}

	// Step 8: 批量查询知识本详情（应用过滤和分页）
	kbs, total, err := s.kbRepo.FindByUUIDsWithFilters(
		nil,
		allKBUUIDs,
		req.Keyword,
		req.Visibility,
		offset,
		req.PageSize,
	)
	if err != nil {
		return nil, 0, errors.Wrap(errors.ErrCodeInternalError, "批量查询知识本详情失败", err)
	}

	// 如果没有查到数据，返回空列表
	if len(kbs) == 0 {
		return []*KnowledgebaseWithRole{}, total, nil
	}

	// Step 9: 批量查询用户角色（若 role 过滤已提前查询则直接复用，否则此处查询）
	kbUUIDs := make([]string, len(kbs))
	for i, kb := range kbs {
		kbUUIDs[i] = kb.UUID
	}

	if roleMap == nil {
		roleMap, err = s.memberRepo.GetUserRolesBatchWithHint(nil, kbUUIDs, username, matchedGroupIDs)
		if err != nil {
			return nil, 0, errors.Wrap(errors.ErrCodeInternalError, "批量查询用户角色失败", err)
		}
	}

	// Step 10: 对于公开知识本，如果用户不是成员，默认设置为viewer
	for _, kb := range kbs {
		if kb.Visibility == constants.VisibilityPublic && roleMap[kb.UUID] == "" {
			roleMap[kb.UUID] = constants.KnowledgebaseMemberRoleViewer
		}
	}

	// Step 11: 批量查询统计信息
	memberCountMap, err := s.memberRepo.CountByKBUUIDsBatch(nil, kbUUIDs)
	if err != nil {
		return nil, 0, errors.Wrap(errors.ErrCodeInternalError, "批量查询成员数量失败", err)
	}

	// tagCountMap, err := s.tagRepo.CountByKBUUIDsBatch(nil, kbUUIDs)
	// if err != nil {
	// 	return nil, 0, errors.Wrap(errors.ErrCodeInternalError, "批量查询标签数量失败", err)
	// }

	knowledgeCountMap, err := s.knowRepo.CountByKBUUIDsBatch(nil, kbUUIDs)
	if err != nil {
		return nil, 0, errors.Wrap(errors.ErrCodeInternalError, "批量查询知识数量失败", err)
	}

	// Step 12: 组装结果
	result := make([]*KnowledgebaseWithRole, 0, len(kbs))
	for _, kb := range kbs {
		result = append(result, &KnowledgebaseWithRole{
			KB:          kb,
			Role:        roleMap[kb.UUID],
			MemberCount: memberCountMap[kb.UUID],
			// TagCount:    tagCountMap[kb.UUID],
			KnowledgeCount: knowledgeCountMap[kb.UUID],
		})
	}

	return result, total, nil
}

// ListGroupKnowledgebases 查询群聊有权限的知识本列表
// 查询以 groupID 作为 chat_group 类型成员注册到知识本的列表，以及该群对应的角色
func (s *knowledgebaseServiceImpl) ListGroupKnowledgebases(ctx context.Context, groupID string, keyword string, role string, page, pageSize int) ([]*KnowledgebaseWithRole, int64, error) {
	// 处理分页参数
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	offset := (page - 1) * pageSize

	// 查询该群登记的知识本 UUID 列表
	kbUUIDs, err := s.memberRepo.FindKBUUIDsByMemberValues(nil, constants.KnowledgebaseMemberTypeChatGroup, []string{groupID})
	if err != nil {
		return nil, 0, errors.Wrap(errors.ErrCodeInternalError, "查询群聊知识本关系失败", err)
	}

	// 该群未关联任何知识本，返回空列表
	if len(kbUUIDs) == 0 {
		return []*KnowledgebaseWithRole{}, 0, nil
	}

	// 批量查询该群在每个知识本中的角色（前置，用于 role 过滤）
	roleMap, err := s.memberRepo.FindMemberRolesByKBUUIDs(nil, kbUUIDs, constants.KnowledgebaseMemberTypeChatGroup, groupID)
	if err != nil {
		return nil, 0, errors.Wrap(errors.ErrCodeInternalError, "批量查询成员角色失败", err)
	}

	// 若传入了 role 参数，在分页前按角色过滤 UUID 列表
	filteredUUIDs := kbUUIDs
	if role != "" {
		filteredUUIDs = make([]string, 0, len(kbUUIDs))
		for _, uuid := range kbUUIDs {
			if roleMap[uuid] == role {
				filteredUUIDs = append(filteredUUIDs, uuid)
			}
		}
	}

	// 过滤后为空，直接返回
	if len(filteredUUIDs) == 0 {
		return []*KnowledgebaseWithRole{}, 0, nil
	}

	// 分页过滤查询知识本详情
	kbs, total, err := s.kbRepo.FindByUUIDsWithFilters(nil, filteredUUIDs, keyword, "", offset, pageSize)
	if err != nil {
		return nil, 0, errors.Wrap(errors.ErrCodeInternalError, "批量查询知识本详情失败", err)
	}

	if len(kbs) == 0 {
		return []*KnowledgebaseWithRole{}, total, nil
	}

	result := make([]*KnowledgebaseWithRole, 0, len(kbs))
	for _, kb := range kbs {
		result = append(result, &KnowledgebaseWithRole{
			KB:   kb,
			Role: roleMap[kb.UUID],
		})
	}

	return result, total, nil
}

// mergeDeduplicate 合并并去重UUID列表
func mergeDeduplicate(slices ...[]string) []string {
	seen := make(map[string]struct{})
	result := []string{}
	for _, slice := range slices {
		for _, uuid := range slice {
			if _, exists := seen[uuid]; !exists {
				seen[uuid] = struct{}{}
				result = append(result, uuid)
			}
		}
	}
	return result
}

// DeleteKnowledgebase 删除知识本
// 删除知识本时会级联删除：
// 1. 知识本下的所有知识（软删除）
// 2. 知识本下的所有目录（软删除）
// 3. 知识本下的所有标签（软删除）
// 4. 知识本下的所有成员关系
// 5. 知识本本身（软删除）
func (s *knowledgebaseServiceImpl) DeleteKnowledgebase(ctx context.Context, uuid, username string, isAdmin bool) error {
	// 获取知识本（非事务操作）
	kb, err := s.kbRepo.FindByUUID(nil, uuid)
	if err != nil {
		return errors.Wrap(errors.ErrCodeInternalError, "查询知识本失败", err)
	}
	if kb == nil {
		return errors.ErrKnowledgebaseNotFound
	}

	// 检查删除权限
	if err := s.permissionService.CheckDeleteKnowledgebasePermission(kb, kb.UUID, username, isAdmin); err != nil {
		return err
	}

	// 在事务中执行删除操作
	err = s.txManager.ExecuteInTransactionWithContext(ctx, func(txCtx *database.TxContext) error {
		// 软删除知识本
		if err := s.kbRepo.SoftDelete(txCtx, kb.UUID); err != nil {
			return errors.Wrap(errors.ErrCodeInternalError, "删除知识本失败", err)
		}

		// 记录操作日志
		detail, err := json.Marshal(map[string]interface{}{
			"name":        kb.Name,
			"description": kb.Description,
			"visibility":  kb.Visibility,
		})
		if err != nil {
			return errors.Wrap(errors.ErrCodeInternalError, "序列化操作日志详情失败", err)
		}

		opLog := &domain.KnowledgebaseOperationLog{
			KnowledgebaseUUID: kb.UUID,
			Operator:          username,
			Operation:         constants.OperationDeleteKB,
			ResourceType:      constants.ResourceTypeKnowledgebase,
			ResourceUUID:      kb.UUID,
			Detail:            string(detail),
		}
		if err := s.operationRepo.Create(txCtx, opLog); err != nil {
			return errors.Wrap(errors.ErrCodeInternalError, "记录操作日志失败", err)
		}

		return nil
	})

	if err != nil {
		return err
	}

	// 事务成功后，删除相关缓存
	s.invalidateKBCache(kb.UUID, kb.Name)

	log.Logger.InfofCtx(ctx, "删除知识本成功: uuid=%s, name=%s", kb.UUID, kb.Name)

	return nil
}

// validateMembersExist 校验成员列表中每个成员是否真实存在
// 跳过创建者自身（创建者无需校验）
func (s *knowledgebaseServiceImpl) validateMembersExist(members []request.CreateKnowledgebaseMember, creator string) error {
	for _, m := range members {
		// 跳过创建者自身
		if m.MemberType == constants.KnowledgebaseMemberTypeUsername && m.MemberValue == creator {
			continue
		}
		if err := validateMemberExists(s.uuapClient, s.infoFlowClient, m.MemberType, m.MemberValue); err != nil {
			return err
		}
	}
	return nil
}

// invalidateKBCache 删除知识本相关缓存
// 包括知识本信息缓存和用户角色缓存
func (s *knowledgebaseServiceImpl) invalidateKBCache(uuid, name string) {
	// 删除知识本信息缓存（UUID和Name两个key）
	s.kbRepo.InvalidateKBCache(uuid, name)

	// 删除该知识本下所有用户的角色缓存
	if uuid != "" {
		s.memberRepo.InvalidateUserRoleCacheByKBUUID(uuid)
	}
}
