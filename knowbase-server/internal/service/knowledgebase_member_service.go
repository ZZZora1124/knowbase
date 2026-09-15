// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package service

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/domain"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/infrastructure/database"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/infrastructure/infoflow"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/infrastructure/uuap"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/repository"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/errors"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/log"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/types/constants"
)

// KnowledgebaseMemberService 知识本成员管理服务接口
type KnowledgebaseMemberService interface {
	// AddMember 添加成员到知识本
	AddMember(ctx context.Context, kbUUID, memberType, memberValue, role, operator string, isAdmin bool) (*domain.KnowledgebaseMember, error)

	// ListMembers 查询知识本成员列表（原始记录）
	ListMembers(ctx context.Context, kbUUID, operator string, page, pageSize int) ([]*domain.KnowledgebaseMember, int64, error)

	// ListAdmins 查询知识本管理员列表（role=admin）
	// 与 ListMembers 不同：已登录用户即可查询，不要求是知识本成员
	// 仅返回管理员原始记录（不展开邮件组/部门），用于"找谁申请"场景
	ListAdmins(ctx context.Context, kbUUID string) ([]*domain.KnowledgebaseMember, error)

	// ListExpandedMembers 查询展开后的成员列表
	// 将邮件组和部门成员展开为实际用户列表
	ListExpandedMembers(ctx context.Context, kbUUID, operator string, page, pageSize int) ([]*ExpandedMember, int64, error)

	// UpdateMemberRole 更新成员角色
	UpdateMemberRole(ctx context.Context, kbUUID, memberUUID, newRole, operator string, isAdmin bool) error

	// RemoveMember 移除成员
	RemoveMember(ctx context.Context, kbUUID, memberUUID, operator string, isAdmin bool) error

	// GetCurrentUserRole 获取当前用户在知识本中的有效角色
	// 返回 admin/editor/viewer 或空字符串（无权限）
	// 权限来源包括：直接 username 成员、邮件组成员、群聊成员
	GetCurrentUserRole(ctx context.Context, kbUUID, operator string) (string, error)
}

// ExpandedMember 展开后的成员信息
type ExpandedMember struct {
	Username          string
	Role              string
	MemberSource      string // username/email_group/department
	SourceName        string // 来源名称
	KnowledgebaseUUID string
	JoinedAt          time.Time
}

// knowledgebaseMemberServiceImpl 知识本成员管理服务实现
type knowledgebaseMemberServiceImpl struct {
	kbRepo            repository.KnowledgebaseRepository
	memberRepo        repository.KnowledgebaseMemberRepository
	logRepo           repository.KnowledgebaseOperationLogRepository
	permissionService PermissionService
	txManager         *database.TransactionManager
	uuapClient        *uuap.UUAPClient // UUAP客户端
	infoFlowClient    *infoflow.Client // 如流客户端
}

// NewKnowledgebaseMemberService 创建知识本成员管理服务实例
func NewKnowledgebaseMemberService(
	kbRepo repository.KnowledgebaseRepository,
	memberRepo repository.KnowledgebaseMemberRepository,
	logRepo repository.KnowledgebaseOperationLogRepository,
	txManager *database.TransactionManager,
	uuapClient *uuap.UUAPClient,
	infoFlowClient *infoflow.Client,
) KnowledgebaseMemberService {
	return &knowledgebaseMemberServiceImpl{
		kbRepo:            kbRepo,
		memberRepo:        memberRepo,
		logRepo:           logRepo,
		permissionService: NewPermissionService(memberRepo),
		txManager:         txManager,
		uuapClient:        uuapClient,
		infoFlowClient:    infoFlowClient,
	}
}

// AddMember 添加成员到知识本
func (s *knowledgebaseMemberServiceImpl) AddMember(
	ctx context.Context, kbUUID, memberType, memberValue, role, operator string, isAdmin bool,
) (*domain.KnowledgebaseMember, error) {
	// 1. 检查知识本是否存在
	kb, err := s.kbRepo.FindByUUID(nil, kbUUID)
	if err != nil {
		return nil, err
	}
	if kb == nil {
		return nil, errors.ErrKnowledgebaseNotFound
	}

	// 2. 检查是否为个人知识本（personal类型不允许成员管理）
	if kb.Visibility == constants.VisibilityPersonal {
		return nil, errors.ErrCannotManagePersonalKB
	}

	// 3. 检查操作者权限（只有admin可以添加成员）
	if err := s.permissionService.CheckManageMemberPermission(kb, kbUUID, operator, isAdmin); err != nil {
		return nil, err
	}

	// 4. 验证成员类型
	if !isValidMemberType(memberType) {
		return nil, errors.ErrInvalidMemberType
	}

	// 5. 验证角色类型
	if !isValidRole(role) {
		return nil, errors.ErrInvalidRoleType
	}

	// 6. 检查成员是否已存在
	existingMember, err := s.memberRepo.FindByKBUUIDAndMember(nil, kbUUID, memberType, memberValue)
	if err != nil {
		return nil, err
	}
	if existingMember != nil {
		return nil, errors.ErrMemberAlreadyExists
	}

	// 7. 验证成员是否存在（根据类型调用不同的验证接口）
	if err := validateMemberExists(s.uuapClient, s.infoFlowClient, memberType, memberValue); err != nil {
		return nil, err
	}

	// 8. 如果是群成员类型，验证操作人是否在群里
	if memberType == constants.KnowledgebaseMemberTypeChatGroup {
		if err := s.validateOperatorInChatGroup(operator, memberValue); err != nil {
			return nil, err
		}
	}

	// 9. 创建成员对象
	member := &domain.KnowledgebaseMember{
		UUID:              uuid.New().String(),
		KnowledgebaseUUID: kbUUID,
		MemberType:        memberType,
		MemberValue:       memberValue,
		Role:              role,
	}

	// 8. 在事务中执行创建成员和记录日志
	err = s.txManager.ExecuteInTransactionWithContext(ctx, func(txCtx *database.TxContext) error {
		// 创建成员记录
		if err := s.memberRepo.Create(txCtx, member); err != nil {
			return err
		}

		// 记录操作日志
		detail := map[string]interface{}{
			"memberType":  memberType,
			"memberValue": memberValue,
			"role":        role,
		}
		detailJSON, err := json.Marshal(detail)
		if err != nil {
			return errors.Wrap(errors.ErrCodeInternalError, "序列化操作日志详情失败", err)
		}
		opLog := &domain.KnowledgebaseOperationLog{
			KnowledgebaseUUID: kbUUID,
			Operator:          operator,
			Operation:         constants.OperationAddMember,
			ResourceType:      constants.ResourceTypeMember,
			ResourceUUID:      member.UUID,
			Detail:            string(detailJSON),
		}
		return s.logRepo.Create(txCtx, opLog)
	})

	if err != nil {
		return nil, err
	}

	// 9. 清除角色缓存
	// 如果是邮件组或群聊类型成员，权限可继承，需要清除所有用户的缓存
	if memberType == constants.KnowledgebaseMemberTypeEmailGroup || memberType == constants.KnowledgebaseMemberTypeChatGroup {
		s.memberRepo.InvalidateUserRoleCacheByKBUUID(kbUUID)
	} else {
		s.memberRepo.InvalidateUserRoleCache(kbUUID, memberValue)
	}

	// 10. 清除成员数量缓存
	s.memberRepo.InvalidateMemberCountCache(kbUUID)

	// 11. 群成员添加成功后，异步刷新群成员缓存
	if memberType == constants.KnowledgebaseMemberTypeChatGroup && s.infoFlowClient != nil {
		if groupID, parseErr := strconv.ParseInt(memberValue, 10, 64); parseErr == nil {
			go func() {
				if err := s.infoFlowClient.ForceRefreshGroupCache(context.Background(), groupID); err != nil {
					log.Logger.Warnf("AddMember 后刷新群 %d 成员缓存失败: %v", groupID, err)
				}
			}()
		}
	}

	log.Logger.InfofCtx(ctx, "添加成员成功 uuid=%s, kbUUID=%s, memberType=%s, memberValue=%s", member.UUID, kbUUID, memberType, memberValue)

	return member, nil
}

// ListMembers 查询知识本成员列表
func (s *knowledgebaseMemberServiceImpl) ListMembers(
	ctx context.Context, kbUUID, operator string, page, pageSize int,
) ([]*domain.KnowledgebaseMember, int64, error) {
	// 1. 检查知识本是否存在
	kb, err := s.kbRepo.FindByUUID(nil, kbUUID)
	if err != nil {
		return nil, 0, err
	}
	if kb == nil {
		return nil, 0, errors.ErrKnowledgebaseNotFound
	}

	// 2. 权限检查
	// - 公开知识本：所有人可以查看成员列表
	// - 私密知识本：只有成员可以查看
	if kb.Visibility == constants.VisibilityPrivate {
		role, err := s.memberRepo.GetUserRole(nil, kbUUID, operator)
		if err != nil {
			return nil, 0, err
		}
		if role == "" {
			return nil, 0, errors.ErrNotKnowledgebaseMember
		}
	}

	// 3. 查询成员列表
	members, err := s.memberRepo.ListByKBUUID(nil, kbUUID)
	if err != nil {
		return nil, 0, err
	}

	// 4. 计算总数
	total := int64(len(members))

	// 5. 分页处理
	start := (page - 1) * pageSize
	end := start + pageSize
	if start >= len(members) {
		return []*domain.KnowledgebaseMember{}, total, nil
	}
	if end > len(members) {
		end = len(members)
	}

	return members[start:end], total, nil
}

// ListAdmins 查询知识本管理员列表（role=admin）
// 已登录即可访问，不要求调用者是知识本成员；
// 仅用于"找谁申请"场景，返回管理员原始记录（不展开邮件组/部门）
func (s *knowledgebaseMemberServiceImpl) ListAdmins(
	ctx context.Context, kbUUID string,
) ([]*domain.KnowledgebaseMember, error) {
	// 1. 检查知识本是否存在
	kb, err := s.kbRepo.FindByUUID(nil, kbUUID)
	if err != nil {
		return nil, err
	}
	if kb == nil {
		return nil, errors.ErrKnowledgebaseNotFound
	}

	// 2. 查询管理员列表
	admins, err := s.memberRepo.ListAdminsByKBUUID(nil, kbUUID)
	if err != nil {
		return nil, err
	}

	return admins, nil
}
// 将邮件组和部门成员展开为实际用户列表
func (s *knowledgebaseMemberServiceImpl) ListExpandedMembers(
	ctx context.Context, kbUUID, operator string, page, pageSize int,
) ([]*ExpandedMember, int64, error) {
	// 1. 检查知识本是否存在
	kb, err := s.kbRepo.FindByUUID(nil, kbUUID)
	if err != nil {
		return nil, 0, err
	}
	if kb == nil {
		return nil, 0, errors.ErrKnowledgebaseNotFound
	}

	// 2. 权限检查
	if kb.Visibility == constants.VisibilityPrivate {
		role, err := s.memberRepo.GetUserRole(nil, kbUUID, operator)
		if err != nil {
			return nil, 0, err
		}
		if role == "" {
			return nil, 0, errors.ErrNotKnowledgebaseMember
		}
	}

	// 3. 查询成员列表
	members, err := s.memberRepo.ListByKBUUID(nil, kbUUID)
	if err != nil {
		return nil, 0, err
	}

	// 4. 展开成员列表
	expandedMap := make(map[string]*ExpandedMember) // 用去重

	for _, member := range members {
		switch member.MemberType {
		case constants.KnowledgebaseMemberTypeUsername:
			// 直接添加用户
			if existing, ok := expandedMap[member.MemberValue]; !ok {
				expandedMap[member.MemberValue] = &ExpandedMember{
					Username:          member.MemberValue,
					Role:              member.Role,
					MemberSource:      constants.KnowledgebaseMemberTypeUsername,
					SourceName:        member.MemberValue,
					KnowledgebaseUUID: kbUUID,
					JoinedAt:          member.CreatedAt,
				}
			} else {
				// 已存在，保留最高权限
				if getRolePriority(member.Role) > getRolePriority(existing.Role) {
					existing.Role = member.Role
				}
			}

		case constants.KnowledgebaseMemberTypeEmailGroup:
			// 获取邮件组成员列表（带缓存）
			if s.uuapClient != nil {
				users, err := s.uuapClient.GetUserNamesByEmailGroupWithCache(context.Background(), member.MemberValue)
				if err != nil {
					// 记录错误但不阻塞
					continue
				}
				for _, username := range users {
					if existing, ok := expandedMap[username]; !ok {
						expandedMap[username] = &ExpandedMember{
							Username:          username,
							Role:              member.Role,
							MemberSource:      constants.KnowledgebaseMemberTypeEmailGroup,
							SourceName:        member.MemberValue,
							KnowledgebaseUUID: kbUUID,
							JoinedAt:          member.CreatedAt,
						}
					} else {
						// 已存在，保留最高权限
						if getRolePriority(member.Role) > getRolePriority(existing.Role) {
							existing.Role = member.Role
							existing.MemberSource = constants.KnowledgebaseMemberTypeEmailGroup
							existing.SourceName = member.MemberValue
						}
					}
				}
			}

		case constants.KnowledgebaseMemberTypeDepartment:
			// 解析部门信息
			parts := strings.SplitN(member.MemberValue, ":", 2)
			if len(parts) != 2 {
				continue
			}
			deptId, err := strconv.ParseInt(parts[0], 10, 64)
			if err != nil {
				continue
			}
			deptName := parts[1]

			// 获取部门成员列表（带缓存）
			if s.uuapClient != nil {
				users, err := s.uuapClient.GetUserNamesByDepartmentWithCache(context.Background(), deptId, deptName)
				if err != nil {
					// 记录错误但不阻塞
					continue
				}
				for _, username := range users {
					if existing, ok := expandedMap[username]; !ok {
						expandedMap[username] = &ExpandedMember{
							Username:          username,
							Role:              member.Role,
							MemberSource:      constants.KnowledgebaseMemberTypeDepartment,
							SourceName:        member.MemberValue,
							KnowledgebaseUUID: kbUUID,
							JoinedAt:          member.CreatedAt,
						}
					} else {
						// 已存在，保留最高权限
						if getRolePriority(member.Role) > getRolePriority(existing.Role) {
							existing.Role = member.Role
							existing.MemberSource = constants.KnowledgebaseMemberTypeDepartment
							existing.SourceName = member.MemberValue
						}
					}
				}
			}
		}
	}

	// 5. 转换为列表
	allUsers := make([]*ExpandedMember, 0, len(expandedMap))
	for _, user := range expandedMap {
		allUsers = append(allUsers, user)
	}

	// 6. 计算总数
	total := int64(len(allUsers))

	// 7. 分页处理
	start := (page - 1) * pageSize
	end := start + pageSize
	if start >= len(allUsers) {
		return []*ExpandedMember{}, total, nil
	}
	if end > len(allUsers) {
		end = len(allUsers)
	}

	return allUsers[start:end], total, nil
}

// getRolePriority 获取角色优先级（数值越大权限越高）
func getRolePriority(role string) int {
	switch role {
	case constants.KnowledgebaseMemberRoleAdmin:
		return 3
	case constants.KnowledgebaseMemberRoleEditor:
		return 2
	case constants.KnowledgebaseMemberRoleViewer:
		return 1
	default:
		return 0
	}
}

// UpdateMemberRole 更新成员角色
func (s *knowledgebaseMemberServiceImpl) UpdateMemberRole(
	ctx context.Context, kbUUID, memberUUID, newRole, operator string, isAdmin bool,
) error {
	// 1. 检查知识本是否存在
	kb, err := s.kbRepo.FindByUUID(nil, kbUUID)
	if err != nil {
		return err
	}
	if kb == nil {
		return errors.ErrKnowledgebaseNotFound
	}

	// 2. 检查是否为个人知识本（personal类型不允许成员管理）
	if kb.Visibility == constants.VisibilityPersonal {
		return errors.ErrCannotManagePersonalKB
	}

	// 3. 检查操作者权限（只有admin可以更新角色）
	if err := s.permissionService.CheckManageMemberPermission(kb, kbUUID, operator, isAdmin); err != nil {
		return err
	}

	// 4. 验证新角色类型
	if !isValidRole(newRole) {
		return errors.ErrInvalidRoleType
	}

	// 5. 查询目标成员
	member, err := s.memberRepo.FindByUUID(nil, memberUUID)
	if err != nil {
		return err
	}
	if member == nil {
		return errors.ErrMemberNotFound
	}

	// 6. 检查成员是否属于该知识本
	if member.KnowledgebaseUUID != kbUUID {
		return errors.ErrMemberNotFound
	}

	// 7. 如果目标成员是群组成员，验证操作人是否仍在该群中
	if member.MemberType == constants.KnowledgebaseMemberTypeChatGroup {
		if err := s.validateOperatorInChatGroup(operator, member.MemberValue); err != nil {
			return err
		}
	}

	// 8. 如果角色没有变化，直接返回
	if member.Role == newRole {
		return nil
	}

	// 9. 检查是否是知识本所有者（owner不能被降级）
	if member.MemberValue == kb.Owner && member.MemberType == constants.KnowledgebaseMemberTypeUsername {
		// owner 始终保持 admin 角色，不能被降级
		if newRole != constants.KnowledgebaseMemberRoleAdmin {
			return errors.ErrCannotModifyOwnerRole
		}
	}

	// 10. 如果是降级admin，检查是否是最后一个admin
	if member.Role == constants.KnowledgebaseMemberRoleAdmin && newRole != constants.KnowledgebaseMemberRoleAdmin {
		adminCount, err := s.memberRepo.CountAdminsByKBUUID(nil, kbUUID)
		if err != nil {
			return err
		}
		if adminCount <= 1 {
			return errors.ErrCannotRemoveLastAdmin
		}
	}

	// 11. 在事务中执行更新角色和记录日志
	oldRole := member.Role
	err = s.txManager.ExecuteInTransactionWithContext(ctx, func(txCtx *database.TxContext) error {
		// 更新成员角色
		member.Role = newRole
		if err := s.memberRepo.Update(txCtx, member); err != nil {
			return err
		}

		// 记录操作日志
		detail := map[string]interface{}{
			"memberValue":   member.MemberValue,
			"before":        map[string]string{"role": oldRole},
			"after":         map[string]string{"role": newRole},
			"changedFields": []string{"role"},
		}
		detailJSON, err := json.Marshal(detail)
		if err != nil {
			return errors.Wrap(errors.ErrCodeInternalError, "序列化操作日志详情失败", err)
		}
		opLog := &domain.KnowledgebaseOperationLog{
			KnowledgebaseUUID: kbUUID,
			Operator:          operator,
			Operation:         constants.OperationUpdateMemberRole,
			ResourceType:      constants.ResourceTypeMember,
			ResourceUUID:      member.UUID,
			Detail:            string(detailJSON),
		}
		return s.logRepo.Create(txCtx, opLog)
	})

	if err != nil {
		return err
	}

	// 11. 清除角色缓存
	// 如果是邮件组或群聊类型成员，权限可继承，需要清除所有用户的缓存
	if member.MemberType == constants.KnowledgebaseMemberTypeEmailGroup || member.MemberType == constants.KnowledgebaseMemberTypeChatGroup {
		s.memberRepo.InvalidateUserRoleCacheByKBUUID(kbUUID)
	} else {
		s.memberRepo.InvalidateUserRoleCache(kbUUID, member.MemberValue)
	}

	log.Logger.InfofCtx(ctx, "更新成员角色成功: uuid=%s, role=%s", member.UUID, newRole)

	return nil
}

// RemoveMember 移除成员
func (s *knowledgebaseMemberServiceImpl) RemoveMember(
	ctx context.Context, kbUUID, memberUUID, operator string, isAdmin bool,
) error {
	// 1. 检查知识本是否存在
	kb, err := s.kbRepo.FindByUUID(nil, kbUUID)
	if err != nil {
		return err
	}
	if kb == nil {
		return errors.ErrKnowledgebaseNotFound
	}

	// 2. 检查是否为个人知识本（personal类型不允许成员管理）
	if kb.Visibility == constants.VisibilityPersonal {
		return errors.ErrCannotManagePersonalKB
	}

	// 3. 检查操作者权限（只有admin可以移除成员）
	if err := s.permissionService.CheckManageMemberPermission(kb, kbUUID, operator, isAdmin); err != nil {
		return err
	}

	// 4. 查询目标成员
	member, err := s.memberRepo.FindByUUID(nil, memberUUID)
	if err != nil {
		return err
	}
	if member == nil {
		return errors.ErrMemberNotFound
	}

	// 5. 检查成员是否属于该知识本
	if member.KnowledgebaseUUID != kbUUID {
		return errors.ErrMemberNotFound
	}

	// 6. 如果目标成员是群组成员，验证操作人是否仍在该群中
	if member.MemberType == constants.KnowledgebaseMemberTypeChatGroup {
		if err := s.validateOperatorInChatGroup(operator, member.MemberValue); err != nil {
			return err
		}
	}

	// 7. 检查是否是知识本所有者（owner不能被移除）
	if member.MemberValue == kb.Owner && member.MemberType == constants.KnowledgebaseMemberTypeUsername {
		return errors.ErrCannotRemoveOwner
	}

	// 8. 如果是admin，检查是否是最后一个admin
	if member.Role == constants.KnowledgebaseMemberRoleAdmin {
		adminCount, err := s.memberRepo.CountAdminsByKBUUID(nil, kbUUID)
		if err != nil {
			return err
		}
		if adminCount <= 1 {
			return errors.ErrCannotRemoveLastAdmin
		}
	}

	// 9. 在事务中执行删除成员和记录日志
	err = s.txManager.ExecuteInTransactionWithContext(ctx, func(txCtx *database.TxContext) error {
		// 删除成员记录
		if err := s.memberRepo.DeleteByUUID(txCtx, memberUUID); err != nil {
			return err
		}

		// 记录操作日志
		detail := map[string]interface{}{
			"memberType":  member.MemberType,
			"memberValue": member.MemberValue,
			"role":        member.Role,
		}
		detailJSON, err := json.Marshal(detail)
		if err != nil {
			return errors.Wrap(errors.ErrCodeInternalError, "序列化操作日志详情失败", err)
		}
		opLog := &domain.KnowledgebaseOperationLog{
			KnowledgebaseUUID: kbUUID,
			Operator:          operator,
			Operation:         constants.OperationRemoveMember,
			ResourceType:      constants.ResourceTypeMember,
			ResourceUUID:      member.UUID,
			Detail:            string(detailJSON),
		}
		return s.logRepo.Create(txCtx, opLog)
	})

	if err != nil {
		return err
	}

	// 10. 清除角色缓存
	// 如果是邮件组或群聊类型成员，权限可继承，需要清除所有用户的缓存
	if member.MemberType == constants.KnowledgebaseMemberTypeEmailGroup || member.MemberType == constants.KnowledgebaseMemberTypeChatGroup {
		s.memberRepo.InvalidateUserRoleCacheByKBUUID(kbUUID)
	} else {
		s.memberRepo.InvalidateUserRoleCache(kbUUID, member.MemberValue)
	}

	// 11. 清除成员数量缓存
	s.memberRepo.InvalidateMemberCountCache(kbUUID)

	log.Logger.InfofCtx(ctx, "移除成员成功: uuid=%s, memberValue=%s", member.UUID, member.MemberValue)

	return nil
}

// GetCurrentUserRole 获取当前用户在知识本中的有效角色
func (s *knowledgebaseMemberServiceImpl) GetCurrentUserRole(ctx context.Context, kbUUID, operator string) (string, error) {
	return s.memberRepo.GetUserRole(nil, kbUUID, operator)
}

// isValidMemberType 验证成员类型
func isValidMemberType(memberType string) bool {
	switch memberType {
	case constants.KnowledgebaseMemberTypeUsername,
		constants.KnowledgebaseMemberTypeEmailGroup,
		constants.KnowledgebaseMemberTypeDepartment,
		constants.KnowledgebaseMemberTypeChatGroup:
		return true
	default:
		return false
	}
}

// isValidRole 验证角色类型
func isValidRole(role string) bool {
	switch role {
	case constants.KnowledgebaseMemberRoleAdmin,
		constants.KnowledgebaseMemberRoleEditor,
		constants.KnowledgebaseMemberRoleViewer:
		return true
	default:
		return false
	}
}

// validateMemberExists 验证成员是否存在（包级别公共函数）
// 根据成员类型调用不同的UUAP验证接口
func validateMemberExists(uuapClient *uuap.UUAPClient, infoFlowClient *infoflow.Client, memberType, memberValue string) error {
	if uuapClient == nil {
		return errors.ErrUUAPClientInit
	}
	ctx := context.Background()

	switch memberType {
	case constants.KnowledgebaseMemberTypeUsername:
		user, err := uuapClient.GetUserByUsername(ctx, memberValue)
		if err != nil {
			return errors.Wrap(errors.ErrCodeUserNotFound, "用户验证失败", err)
		}
		if user == nil {
			return errors.ErrUserNotFound
		}
	case constants.KnowledgebaseMemberTypeEmailGroup:
		emailGroup, err := uuapClient.GetEmailGroupByAddress(ctx, memberValue)
		if err != nil {
			return errors.Wrap(errors.ErrCodeEmailGroupNotFound, "邮件组验证失败", err)
		}
		if emailGroup == nil {
			return errors.ErrEmailGroupNotFound
		}
	case constants.KnowledgebaseMemberTypeDepartment:
		parts := strings.SplitN(memberValue, ":", 2)
		if len(parts) != 2 {
			return errors.New(errors.ErrCodeInvalidParam, "部门成员值格式不正确，应为'部门ID:部门名称'")
		}
		departmentId, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			return errors.Wrap(errors.ErrCodeInvalidParam, "部门ID格式不正确", err)
		}
		dept, err := uuapClient.GetDepartmentByIdAndName(ctx, departmentId, parts[1])
		if err != nil {
			return errors.Wrap(errors.ErrCodeDepartmentNotFound, "部门验证失败", err)
		}
		if dept == nil {
			return errors.ErrDepartmentNotFound
		}
	case constants.KnowledgebaseMemberTypeChatGroup:
		// chat_group 使用如流API验证群是否存在
		groupID, err := strconv.ParseInt(memberValue, 10, 64)
		if err != nil {
			return errors.Wrap(errors.ErrCodeInvalidParam, "群ID格式不正确", err)
		}
		if infoFlowClient == nil {
			return errors.New(errors.ErrCodeInvalidParam, "如流客户端未初始化，无法验证群信息")
		}
		_, err = infoFlowClient.GetGroupInfo(ctx, groupID)
		if err != nil {
			return errors.Wrap(errors.ErrCodeChatGroupNotFound, "群验证失败", err)
		}
	}
	return nil
}

// validateOperatorInChatGroup 验证操作人是否在群里
func (s *knowledgebaseMemberServiceImpl) validateOperatorInChatGroup(operator, chatGroupID string) error {
	if s.infoFlowClient == nil {
		return errors.New(errors.ErrCodeInvalidParam, "如流客户端未初始化，无法验证群成员")
	}

	// 将群ID字符串转换为int64
	groupID, err := strconv.ParseInt(chatGroupID, 10, 64)
	if err != nil {
		return errors.New(errors.ErrCodeInvalidParam, "群ID格式不正确，应为数字字符串")
	}

	// 调用如流API判断操作人是否在群中
	inGroup, err := s.infoFlowClient.IsUserInGroup(context.Background(), groupID, operator)
	if err != nil {
		return errors.Wrap(errors.ErrCodeInfoFlowAPIError, "验证群成员失败", err)
	}

	if !inGroup {
		return errors.ErrNotInChatGroup
	}

	return nil
}
