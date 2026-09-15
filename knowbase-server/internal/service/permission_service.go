// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package service

import (
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/domain"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/repository"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/errors"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/types/constants"
)

// PermissionService 权限服务接口
type PermissionService interface {
	// CheckViewPermission 检查查看权限
	CheckViewPermission(kb *domain.Knowledgebase, kbUUID, operator string, isAdmin bool) bool

	// CheckCreatePermission 检查创建权限
	CheckCreatePermission(kb *domain.Knowledgebase, kbUUID, operator string, isAdmin bool) error

	// CheckUpdateKnowledgebasePermission 检查更新知识本权限
	CheckUpdateKnowledgebasePermission(kb *domain.Knowledgebase, kbUUID, operator string, isAdmin bool) error

	// CheckDeleteKnowledgebasePermission 检查删除知识本权限
	CheckDeleteKnowledgebasePermission(kb *domain.Knowledgebase, kbUUID, operator string, isAdmin bool) error

	// CheckEditPermission 检查编辑权限（知识本级别，用于目录等资源）
	CheckEditPermission(kb *domain.Knowledgebase, kbUUID, operator string, isAdmin bool) error

	// CheckEditPermissionWithResource 检查编辑权限（资源级别，用于知识等需要检查创建者的资源）
	CheckEditPermissionWithResource(kb *domain.Knowledgebase, kbUUID, creatorUsername, operator string, isAdmin bool) error

	// CheckDeletePermission 检查删除权限（资源级别）
	CheckDeletePermissionWithResource(kb *domain.Knowledgebase, kbUUID, creatorUsername, operator string, isAdmin bool) error

	// CheckManageTagPermission 检查标签管理权限（只有admin可以管理标签）
	CheckManageTagPermission(kb *domain.Knowledgebase, kbUUID, operator string, isAdmin bool) error

	// CheckManageMemberPermission 检查成员管理权限（只有admin可以管理成员）
	CheckManageMemberPermission(kb *domain.Knowledgebase, kbUUID, operator string, isAdmin bool) error
}

// permissionServiceImpl 权限服务实现
type permissionServiceImpl struct {
	memberRepo repository.KnowledgebaseMemberRepository
}

// NewPermissionService 创建权限服务实例
func NewPermissionService(memberRepo repository.KnowledgebaseMemberRepository) PermissionService {
	return &permissionServiceImpl{
		memberRepo: memberRepo,
	}
}

// CheckViewPermission 检查查看权限
func (s *permissionServiceImpl) CheckViewPermission(kb *domain.Knowledgebase, kbUUID, operator string, isAdmin bool) bool {
	// Admin客户端直接通过
	if isAdmin {
		return true
	}

	// 公开知识本允许访问
	if kb.Visibility == constants.VisibilityPublic {
		return true
	}

	// 个人知识本只允许owner访问
	if kb.Visibility == constants.VisibilityPersonal {
		return kb.Owner == operator
	}

	// 检查用户是否为成员
	role, err := s.memberRepo.GetUserRole(nil, kbUUID, operator)
	if err != nil {
		return false
	}

	// 空字符串表示非成员，不允许访问
	return role != ""
}

// CheckCreatePermission 检查创建权限
func (s *permissionServiceImpl) CheckCreatePermission(kb *domain.Knowledgebase, kbUUID, operator string, isAdmin bool) error {
	// Admin客户端直接通过
	if isAdmin {
		return nil
	}

	// 个人知识本只有owner可创建
	if kb.Visibility == constants.VisibilityPersonal && kb.Owner != operator {
		return errors.ErrNoPermission
	}

	// 用户可以操作自己的知识本
	if kb.Visibility == constants.VisibilityPersonal && kb.Owner == operator {
		return nil
	}

	role, err := s.memberRepo.GetUserRole(nil, kbUUID, operator)
	if err != nil {
		return errors.Wrap(errors.ErrCodeInternalError, "查询用户角色失败", err)
	}

	// editor及以上可以创建
	if role != constants.KnowledgebaseMemberRoleAdmin &&
		role != constants.KnowledgebaseMemberRoleEditor {
		return errors.ErrNoPermission
	}

	return nil
}

// CheckUpdateKnowledgebasePermission 检查更新知识本权限
// 个人知识本：只有owner可以更新
// 公开/私密知识本：admin和editor可以更新
func (s *permissionServiceImpl) CheckUpdateKnowledgebasePermission(kb *domain.Knowledgebase, kbUUID, operator string, isAdmin bool) error {
	// Admin客户端直接通过
	if isAdmin {
		return nil
	}

	// 个人知识本只允许owner更新
	if kb.Visibility == constants.VisibilityPersonal && kb.Owner != operator {
		return errors.ErrNoPermission
	}

	// 用户可以操作自己的知识本
	if kb.Visibility == constants.VisibilityPersonal && kb.Owner == operator {
		return nil
	}

	role, err := s.memberRepo.GetUserRole(nil, kbUUID, operator)
	if err != nil {
		return errors.Wrap(errors.ErrCodeInternalError, "查询用户角色失败", err)
	}

	// editor及以上可以更新知识本信息
	if role != constants.KnowledgebaseMemberRoleAdmin &&
		role != constants.KnowledgebaseMemberRoleEditor {
		return errors.ErrNoPermission
	}

	return nil
}

// CheckDeleteKnowledgebasePermission 检查删除知识本权限
// 个人知识本：只有owner可以删除
// 公开/私密知识本：只有admin可以删除
func (s *permissionServiceImpl) CheckDeleteKnowledgebasePermission(kb *domain.Knowledgebase, kbUUID, operator string, isAdmin bool) error {
	// Admin客户端直接通过
	if isAdmin {
		return nil
	}

	// 个人知识本只允许owner删除
	if kb.Visibility == constants.VisibilityPersonal && kb.Owner != operator {
		return errors.ErrNoPermission
	}

	// 用户可以操作自己的知识本
	if kb.Visibility == constants.VisibilityPersonal && kb.Owner == operator {
		return nil
	}

	role, err := s.memberRepo.GetUserRole(nil, kbUUID, operator)
	if err != nil {
		return errors.Wrap(errors.ErrCodeInternalError, "查询用户角色失败", err)
	}

	// 只有admin可以删除知识本
	if role != constants.KnowledgebaseMemberRoleAdmin {
		return errors.ErrNoPermission
	}

	return nil
}

// CheckEditPermission 检查编辑权限（知识本级别）
// 个人知识本：只有owner可以编辑
// 公开/私密知识本：admin和editor可以编辑
func (s *permissionServiceImpl) CheckEditPermission(kb *domain.Knowledgebase, kbUUID, operator string, isAdmin bool) error {
	// Admin客户端直接通过
	if isAdmin {
		return nil
	}

	// 个人知识本只允许owner编辑
	if kb.Visibility == constants.VisibilityPersonal && kb.Owner != operator {
		return errors.ErrNoPermission
	}

	// 用户可以操作自己的知识本
	if kb.Visibility == constants.VisibilityPersonal && kb.Owner == operator {
		return nil
	}

	role, err := s.memberRepo.GetUserRole(nil, kbUUID, operator)
	if err != nil {
		return errors.Wrap(errors.ErrCodeInternalError, "查询用户角色失败", err)
	}

	// editor及以上可以编辑
	if role != constants.KnowledgebaseMemberRoleAdmin &&
		role != constants.KnowledgebaseMemberRoleEditor {
		return errors.ErrNoPermission
	}

	return nil
}

// CheckEditPermissionWithResource 检查编辑权限（资源级别）
// 个人知识本：只有owner可以编辑
// 公开/私密知识本：admin和editor都可以编辑任意资源
func (s *permissionServiceImpl) CheckEditPermissionWithResource(kb *domain.Knowledgebase, kbUUID, creatorUsername, operator string, isAdmin bool) error {
	// Admin客户端直接通过
	if isAdmin {
		return nil
	}

	// 个人知识本只允许owner编辑
	if kb.Visibility == constants.VisibilityPersonal && kb.Owner != operator {
		return errors.ErrNoPermission
	}

	// 用户可以操作自己的知识本
	if kb.Visibility == constants.VisibilityPersonal && kb.Owner == operator {
		return nil
	}

	role, err := s.memberRepo.GetUserRole(nil, kbUUID, operator)
	if err != nil {
		return errors.Wrap(errors.ErrCodeInternalError, "查询用户角色失败", err)
	}

	// 公开/私密知识本：admin和editor都可以编辑任意资源
	if role == constants.KnowledgebaseMemberRoleAdmin || role == constants.KnowledgebaseMemberRoleEditor {
		return nil
	}

	return errors.ErrNoPermission
}

// CheckDeletePermissionWithResource 检查删除权限（资源级别）
// 个人知识本：只有owner可以删除
// 公开/私密知识本：admin和editor都可以删除任意资源
func (s *permissionServiceImpl) CheckDeletePermissionWithResource(kb *domain.Knowledgebase, kbUUID, creatorUsername, operator string, isAdmin bool) error {
	// Admin客户端直接通过
	if isAdmin {
		return nil
	}

	// 个人知识本只允许owner删除
	if kb.Visibility == constants.VisibilityPersonal && kb.Owner != operator {
		return errors.ErrNoPermission
	}

	// 用户可以操作自己的知识本
	if kb.Visibility == constants.VisibilityPersonal && kb.Owner == operator {
		return nil
	}

	role, err := s.memberRepo.GetUserRole(nil, kbUUID, operator)
	if err != nil {
		return errors.Wrap(errors.ErrCodeInternalError, "查询用户角色失败", err)
	}

	// 公开/私密知识本：admin和editor都可以删除任意资源
	if role == constants.KnowledgebaseMemberRoleAdmin || role == constants.KnowledgebaseMemberRoleEditor {
		return nil
	}

	return errors.ErrNoPermission
}

// CheckManageTagPermission 检查标签管理权限
// 个人知识本：只有owner可以管理标签
// 公开/私密知识本：只有admin可以管理标签
func (s *permissionServiceImpl) CheckManageTagPermission(kb *domain.Knowledgebase, kbUUID, operator string, isAdmin bool) error {
	// Admin客户端直接通过
	if isAdmin {
		return nil
	}

	// 个人知识本只允许owner管理标签
	if kb.Visibility == constants.VisibilityPersonal && kb.Owner != operator {
		return errors.ErrNoPermission
	}

	// 用户可以操作自己的知识本
	if kb.Visibility == constants.VisibilityPersonal && kb.Owner == operator {
		return nil
	}

	role, err := s.memberRepo.GetUserRole(nil, kbUUID, operator)
	if err != nil {
		return errors.Wrap(errors.ErrCodeInternalError, "查询用户角色失败", err)
	}

	// 只有admin可以管理标签
	if role != constants.KnowledgebaseMemberRoleAdmin {
		return errors.ErrNoPermissionToManageTag
	}

	return nil
}

// CheckManageMemberPermission 检查成员管理权限
// 个人知识本：不允许管理成员（个人知识本不支持多成员）
// 公开/私密知识本：只有admin可以管理成员
func (s *permissionServiceImpl) CheckManageMemberPermission(kb *domain.Knowledgebase, kbUUID, operator string, isAdmin bool) error {
	// Admin客户端直接通过
	if isAdmin {
		return nil
	}

	// 个人知识本不允许管理成员
	if kb.Visibility == constants.VisibilityPersonal {
		return errors.ErrNoPermission
	}

	role, err := s.memberRepo.GetUserRole(nil, kbUUID, operator)
	if err != nil {
		return errors.Wrap(errors.ErrCodeInternalError, "查询用户角色失败", err)
	}

	// 只有admin可以管理成员
	if role != constants.KnowledgebaseMemberRoleAdmin {
		return errors.ErrNoPermissionToManageMember
	}

	return nil
}
