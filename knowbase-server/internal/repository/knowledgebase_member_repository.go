// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package repository

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/beego/beego/v2/client/orm"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/domain"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/infrastructure/cache"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/infrastructure/database"

	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/infrastructure/infoflow"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/infrastructure/uuap"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/errors"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/log"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/types/constants"
)

// cacheRoleNonMember 是角色缓存中表示"用户不是成员"的哨兵值，用于防止缓存穿透
const cacheRoleNonMember = "non-member"

// userRoleCacheTTL 返回 [10min, 20min) 的随机 TTL，避免大量缓存同时过期
func userRoleCacheTTL() time.Duration {
	const min, max = 10 * time.Minute, 20 * time.Minute
	return min + time.Duration(rand.Int63n(int64(max-min)))
}

// KnowledgebaseMemberRepository 知识本成员仓储接口
type KnowledgebaseMemberRepository interface {
	// Create 创建成员
	Create(ctx *database.TxContext, member *domain.KnowledgebaseMember) error

	// FindByKBUUIDAndMember 根据知识本UUID和成员值查询成员
	FindByKBUUIDAndMember(ctx *database.TxContext, kbUUID, memberType, memberValue string) (*domain.KnowledgebaseMember, error)

	// ListByKBUUID 根据知识本UUID查询成员列表
	ListByKBUUID(ctx *database.TxContext, kbUUID string) ([]*domain.KnowledgebaseMember, error)

	// ListAdminsByKBUUID 根据知识本UUID查询管理员列表（role=admin）
	ListAdminsByKBUUID(ctx *database.TxContext, kbUUID string) ([]*domain.KnowledgebaseMember, error)

	// GetUserRole 获取用户在知识本中的角色
	// 返回: admin/editor/viewer 或空字符串(非成员)
	GetUserRole(ctx *database.TxContext, kbUUID, username string) (string, error)

	// GetUserRolesBatch 批量获取用户在多个知识本中的角色
	// 参数: kbUUIDs - 知识本UUID列表, username - 用户名
	// 返回: map[知识本UUID]角色, 如果用户不是某知识本成员则map中无对应key
	GetUserRolesBatch(ctx *database.TxContext, kbUUIDs []string, username string) (map[string]string, error)

	// GetUserRolesBatchWithHint 批量获取用户在多个知识本中的角色（携带群成员匹配结果 hint）
	// matchedGroupIDs: 上层已查好的用户所在群 ID 列表，非 nil 时直接使用，跳过如流 API 调用
	GetUserRolesBatchWithHint(ctx *database.TxContext, kbUUIDs []string, username string, matchedGroupIDs []string) (map[string]string, error)

	// FindKBUUIDsByMemberValues 根据成员值批量查询知识本UUID
	// 参数: memberType - 成员类型(username/email_group/department), memberValues - 成员值列表
	// 返回: 去重的知识本UUID列表
	FindKBUUIDsByMemberValues(ctx *database.TxContext, memberType string, memberValues []string) ([]string, error)

	// FindDistinctMemberValues 查询指定类型的所有不重复成员值
	// 参数: memberType - 成员类型(例如 chat_group)
	// 返回: 去重的成员值列表
	FindDistinctMemberValues(ctx *database.TxContext, memberType string) ([]string, error)

	// InvalidateUserRoleCache 删除用户角色缓存
	// 在成员添加/删除/角色变更时调用
	InvalidateUserRoleCache(kbUUID, username string)

	// InvalidateUserRoleCacheByKBUUID 删除指定知识本下所有用户的角色缓存
	// 在知识本更新/删除时调用
	InvalidateUserRoleCacheByKBUUID(kbUUID string)

	// Update 更新成员信息
	Update(ctx *database.TxContext, member *domain.KnowledgebaseMember) error

	// DeleteByUUID 根据UUID删除成员
	DeleteByUUID(ctx *database.TxContext, uuid string) error

	// FindByUUID 根据UUID查询成员
	FindByUUID(ctx *database.TxContext, uuid string) (*domain.KnowledgebaseMember, error)

	// CountByKBUUID 统计知识本成员数量
	CountByKBUUID(ctx *database.TxContext, kbUUID string) (int64, error)

	// CountAdminsByKBUUID 统计知识本管理员数量
	CountAdminsByKBUUID(ctx *database.TxContext, kbUUID string) (int64, error)

	// CountByKBUUIDsBatch 批量统计多个知识本的成员数量
	// 返回: map[知识本UUID]成员数量
	CountByKBUUIDsBatch(ctx *database.TxContext, kbUUIDs []string) (map[string]int64, error)

	// FindMemberRolesByKBUUIDs 批量查询指定成员在多个知识本中的角色
	// 参数: kbUUIDs - 知识本UUID列表, memberType - 成员类型, memberValue - 成员值
	// 返回: map[知识本UUID]角色
	FindMemberRolesByKBUUIDs(ctx *database.TxContext, kbUUIDs []string, memberType, memberValue string) (map[string]string, error)

	// InvalidateMemberCountCache 删除知识本成员数量缓存
	// 在成员添加/删除时调用
	InvalidateMemberCountCache(kbUUID string)

	// DeleteMembersByTypeAndValue 删除指定类型+值的所有成员记录，返回受影响的知识本UUID列表（去重）
	// 用于群成员长期不可达时自动清理知识本成员数据
	DeleteMembersByTypeAndValue(ctx *database.TxContext, memberType, memberValue string) ([]string, error)
}

// knowledgebaseMemberRepositoryImpl 知识本成员仓储实现
type knowledgebaseMemberRepositoryImpl struct {
	ormer          orm.Ormer
	uuapClient     *uuap.UUAPClient // UUAP客户端
	infoFlowClient *infoflow.Client // 如流客户端
}

// NewKnowledgebaseMemberRepository 创建知识本成员仓储实例
func NewKnowledgebaseMemberRepository(ormer orm.Ormer) KnowledgebaseMemberRepository {
	uuapClient, _ := uuap.GetClient()
	infoFlowClient := infoflow.GetClient()
	return &knowledgebaseMemberRepositoryImpl{
		ormer:          ormer,
		uuapClient:     uuapClient,
		infoFlowClient: infoFlowClient,
	}
}

// getOrmer 获取实际的 ormer，优先使用事务上下文中的 ormer
func (r *knowledgebaseMemberRepositoryImpl) getOrmer(ctx *database.TxContext) orm.QueryExecutor {
	if ctx != nil && ctx.Ormer() != nil {
		return ctx.Ormer()
	}
	return r.ormer
}

// Create 创建成员
func (r *knowledgebaseMemberRepositoryImpl) Create(ctx *database.TxContext, member *domain.KnowledgebaseMember) error {
	_, err := r.getOrmer(ctx).Insert(member)
	return err
}

// FindByKBUUIDAndMember 根据知识本UUID和成员值查询成员
func (r *knowledgebaseMemberRepositoryImpl) FindByKBUUIDAndMember(
	ctx *database.TxContext,
	kbUUID, memberType, memberValue string,
) (*domain.KnowledgebaseMember, error) {
	member := &domain.KnowledgebaseMember{}
	err := r.getOrmer(ctx).QueryTable(new(domain.KnowledgebaseMember)).
		Filter("knowledgebase_uuid", kbUUID).
		Filter("member_type", memberType).
		Filter("member_value", memberValue).
		One(member)
	if err == orm.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return member, nil
}

// FindMemberRolesByKBUUIDs 批量查询指定成员在多个知识本中的角色
func (r *knowledgebaseMemberRepositoryImpl) FindMemberRolesByKBUUIDs(
	ctx *database.TxContext,
	kbUUIDs []string,
	memberType, memberValue string,
) (map[string]string, error) {
	if len(kbUUIDs) == 0 {
		return make(map[string]string), nil
	}

	var members []*domain.KnowledgebaseMember
	_, err := r.getOrmer(ctx).QueryTable(new(domain.KnowledgebaseMember)).
		Filter("knowledgebase_uuid__in", kbUUIDs).
		Filter("member_type", memberType).
		Filter("member_value", memberValue).
		All(&members)
	if err != nil && err != orm.ErrNoRows {
		return nil, err
	}

	roleMap := make(map[string]string, len(members))
	for _, m := range members {
		roleMap[m.KnowledgebaseUUID] = m.Role
	}
	return roleMap, nil
}

// ListByKBUUID 根据知识本UUID查询成员列表
func (r *knowledgebaseMemberRepositoryImpl) ListByKBUUID(ctx *database.TxContext, kbUUID string) ([]*domain.KnowledgebaseMember, error) {
	members := make([]*domain.KnowledgebaseMember, 0)
	_, err := r.getOrmer(ctx).QueryTable(new(domain.KnowledgebaseMember)).
		Filter("knowledgebase_uuid", kbUUID).
		All(&members)
	return members, err
}

// ListAdminsByKBUUID 根据知识本UUID查询管理员列表（role=admin）
func (r *knowledgebaseMemberRepositoryImpl) ListAdminsByKBUUID(ctx *database.TxContext, kbUUID string) ([]*domain.KnowledgebaseMember, error) {
	members := make([]*domain.KnowledgebaseMember, 0)
	_, err := r.getOrmer(ctx).QueryTable(new(domain.KnowledgebaseMember)).
		Filter("knowledgebase_uuid", kbUUID).
		Filter("role", constants.KnowledgebaseMemberRoleAdmin).
		All(&members)
	return members, err
}

// GetUserRole 获取用户在知识本中的角色
// 返回: admin/editor/viewer 或空字符串(非成员)
// 事务内不使用缓存，保证数据一致性
func (r *knowledgebaseMemberRepositoryImpl) GetUserRole(ctx *database.TxContext, kbUUID, username string) (string, error) {
	// 事务内不使用缓存，直接查询数据库
	if ctx != nil {
		return r.getUserRoleFromDB(ctx, kbUUID, username)
	}

	// 1. 先查缓存
	cacher := cache.GetCache()
	cacheKey := fmt.Sprintf("%s%s:%s", constants.CacheKeyUserRole, kbUUID, username)
	if cacher != nil {
		cachedRole, err := cacher.Get(context.Background(), cacheKey)
		if err == nil && cachedRole != "" {
			// cacheRoleNonMember 表示不是成员
			if cachedRole == cacheRoleNonMember {
				return "", nil
			}
			return cachedRole, nil
		}
	}

	// 2. 缓存未命中，查询数据库
	role, err := r.getUserRoleFromDB(nil, kbUUID, username)
	if err != nil {
		return "", err
	}

	// 3. 写入缓存（空结果缓存为 cacheRoleNonMember 防止穿透，过期时间随机 [10, 20) 分钟）
	if cacher != nil {
		cacheValue := role
		if role == "" {
			cacheValue = cacheRoleNonMember
		}
		cacher.Set(context.Background(), cacheKey, cacheValue, userRoleCacheTTL())
	}

	return role, nil
}

// getUserRoleFromDB 从数据库获取用户角色（支持权限继承）
// TODO: 阶段二需要集成用户系统和部门查询API，遵循设计文档 5.3 节要求：
// 1. 用户可通过 username/email_group/department 三种方式成为成员
// 2. 权限优先级: admin > editor > viewer
// 3. 当用户通过多种方式加入时，取最高权限
func (r *knowledgebaseMemberRepositoryImpl) getUserRoleFromDB(ctx *database.TxContext, kbUUID, username string) (string, error) {
	var allMembers []*domain.KnowledgebaseMember

	// 1. 查询直接用户名匹配
	var userMembers []*domain.KnowledgebaseMember
	_, err := r.getOrmer(ctx).QueryTable(new(domain.KnowledgebaseMember)).
		Filter("knowledgebase_uuid", kbUUID).
		Filter("member_type", constants.KnowledgebaseMemberTypeUsername).
		Filter("member_value", username).
		All(&userMembers)
	if err != nil && err != orm.ErrNoRows {
		return "", err
	}
	allMembers = append(allMembers, userMembers...)

	// 2. 查询邮件组匹配（使用UUAP API）
	if r.uuapClient != nil {
		// 获取用户所属的邮件组名称列表
		userEmailGroupNames, err := r.uuapClient.GetUserEmailGroupAddresses(context.Background(), username)
		if err != nil {
			log.Logger.Errorf("Failed to get user email group names: %v", err)
			// 继续处理，不阻塞
		} else if len(userEmailGroupNames) > 0 {
			// 查询知识本中的邮件组成员
			var emailGroupMembers []*domain.KnowledgebaseMember
			_, err := r.getOrmer(ctx).QueryTable(new(domain.KnowledgebaseMember)).
				Filter("knowledgebase_uuid", kbUUID).
				Filter("member_type", constants.KnowledgebaseMemberTypeEmailGroup).
				All(&emailGroupMembers)
			if err == nil {
				// 匹配用户所属的邮件组名称
				userGroupNameSet := make(map[string]bool)
				for _, name := range userEmailGroupNames {
					userGroupNameSet[name] = true
				}

				for _, m := range emailGroupMembers {
					// member_value存储的是邮件组名称
					if userGroupNameSet[m.MemberValue] {
						allMembers = append(allMembers, m)
					}
				}
			}
		}
	}

	// 3. 查询部门匹配（使用UUAP API）
	/*
		if r.uuapClient != nil {
			// 获取用户所属的部门（包含所有层级）
			userDept, err := r.uuapClient.GetUserBusinessDeptOrg(context.Background(), username)
			if err != nil {
				log.Logger.Errorf("Failed to get user department: %v", err)
				// 继续处理，不阻塞
			} else if userDept != nil {
				// 查询知识本中的部门成员
				var deptMembers []*domain.KnowledgebaseMember
				_, err := r.getOrmer(ctx).QueryTable(new(domain.KnowledgebaseMember)).
					Filter("knowledgebase_uuid", kbUUID).
					Filter("member_type", constants.KnowledgebaseMemberTypeDepartment).
					All(&deptMembers)
				if err == nil {
					// 获取用户所属的所有部门标识（包含各级上级部门）
					// 例如: ["2319:百度", "29563:技术中台群组", "30354:企业效率平台", "1639:企业IT平台部"]
					userDeptKeys := userDept.GetAllDepartmentKeys()
					userDeptKeySet := make(map[string]bool)
					for _, key := range userDeptKeys {
						userDeptKeySet[key] = true
					}

					// 匹配用户所属的任意层级部门
					for _, m := range deptMembers {
						if userDeptKeySet[m.MemberValue] {
							allMembers = append(allMembers, m)
						}
					}
				}
			}
		} */

	// 4. 查询群匹配（使用如流API）
	if r.infoFlowClient != nil {
		// 先查出该知识本配置的所有群成员记录
		var chatGroupMembers []*domain.KnowledgebaseMember
		_, err := r.getOrmer(ctx).QueryTable(new(domain.KnowledgebaseMember)).
			Filter("knowledgebase_uuid", kbUUID).
			Filter("member_type", constants.KnowledgebaseMemberTypeChatGroup).
			All(&chatGroupMembers)
		if err == nil && len(chatGroupMembers) > 0 {
			groupIDs := collectUniqueGroupIDs(chatGroupMembers)
			matchedGroupIDs, err := r.infoFlowClient.IsUserInGroups(context.Background(), groupIDs, username)
			if err != nil {
				log.Logger.Errorf("Failed to check user groups: %v", err)
				// 继续处理，不阻塞
			} else if len(matchedGroupIDs) > 0 {
				matchedSet := make(map[string]bool, len(matchedGroupIDs))
				for _, gid := range matchedGroupIDs {
					matchedSet[gid] = true
				}
				for _, m := range chatGroupMembers {
					if matchedSet[m.MemberValue] {
						allMembers = append(allMembers, m)
					}
				}
			}
		}
	}

	// 如果没有找到任何记录，返回空字符串
	if len(allMembers) == 0 {
		return "", nil
	}

	// 取最高权限：如果用户有多个成员记录，返回权限最高的角色
	return getHighestRole(allMembers), nil
}

// GetUserRolesBatch 批量获取用户在多个知识本中的角色
// 参数: kbUUIDs - 知识本UUID列表, username - 用户名
// 返回: map[知识本UUID]角色, 如果用户不是某知识本成员则map中无对应key
func (r *knowledgebaseMemberRepositoryImpl) GetUserRolesBatch(ctx *database.TxContext, kbUUIDs []string, username string) (map[string]string, error) {
	return r.GetUserRolesBatchWithHint(ctx, kbUUIDs, username, nil)
}

// GetUserRolesBatchWithHint 批量获取用户在多个知识本中的角色（携带群成员匹配结果 hint）
// matchedGroupIDs 非 nil 时直接使用，跳过如流 IsUserInGroups 调用；nil 时走原有逻辑
// 事务内不使用缓存，保证数据一致性
func (r *knowledgebaseMemberRepositoryImpl) GetUserRolesBatchWithHint(ctx *database.TxContext, kbUUIDs []string, username string, matchedGroupIDs []string) (map[string]string, error) {
	if len(kbUUIDs) == 0 {
		return make(map[string]string), nil
	}

	// 将 hint 转为 set，nil 表示不携带 hint
	var userMatchedGroups map[string]bool
	if matchedGroupIDs != nil {
		userMatchedGroups = make(map[string]bool, len(matchedGroupIDs))
		for _, gid := range matchedGroupIDs {
			userMatchedGroups[gid] = true
		}
	}

	// 事务内不使用缓存，直接查询数据库
	if ctx != nil {
		return r.getUserRolesBatchFromDB(ctx, kbUUIDs, username, userMatchedGroups)
	}

	// 1. 尝试从缓存批量获取
	cacher := cache.GetCache()
	roleMap := make(map[string]string)
	missingKBUUIDs := make([]string, 0)

	for _, kbUUID := range kbUUIDs {
		if cacher == nil {
			missingKBUUIDs = append(missingKBUUIDs, kbUUID)
			continue
		}
		cacheKey := fmt.Sprintf("%s%s:%s", constants.CacheKeyUserRole, kbUUID, username)
		cachedRole, err := cacher.Get(context.Background(), cacheKey)
		if err == nil && cachedRole != "" {
			if cachedRole != cacheRoleNonMember {
				roleMap[kbUUID] = cachedRole
			}
		} else {
			missingKBUUIDs = append(missingKBUUIDs, kbUUID)
		}
	}

	// 2. 缓存未命中的，查询数据库（传入 hint）
	if len(missingKBUUIDs) > 0 {
		dbRoleMap, err := r.getUserRolesBatchFromDB(nil, missingKBUUIDs, username, userMatchedGroups)
		if err != nil {
			return nil, err
		}

		// 3. 写入缓存并合并结果
		for _, kbUUID := range missingKBUUIDs {
			role := dbRoleMap[kbUUID]
			if cacher != nil {
				cacheKey := fmt.Sprintf("%s%s:%s", constants.CacheKeyUserRole, kbUUID, username)
				cacheValue := role
				if role == "" {
					cacheValue = cacheRoleNonMember
				}
				cacher.Set(context.Background(), cacheKey, cacheValue, userRoleCacheTTL())
			}
			if role != "" {
				roleMap[kbUUID] = role
			}
		}
	}

	return roleMap, nil
}

// getUserRolesBatchFromDB 从数据库批量获取用户角色
// userMatchedGroups 非 nil 时直接用于群匹配，跳过如流 API 调用
// TODO: 阶段二需要集成用户系统和部门查询API，支持邮件组和部门权限继承
func (r *knowledgebaseMemberRepositoryImpl) getUserRolesBatchFromDB(ctx *database.TxContext, kbUUIDs []string, username string, userMatchedGroups map[string]bool) (map[string]string, error) {
	if len(kbUUIDs) == 0 {
		return make(map[string]string), nil
	}

	// 1. 批量查询直接用户名匹配的成员记录
	var userMembers []*domain.KnowledgebaseMember
	_, err := r.getOrmer(ctx).QueryTable(new(domain.KnowledgebaseMember)).
		Filter("knowledgebase_uuid__in", kbUUIDs).
		Filter("member_type", constants.KnowledgebaseMemberTypeUsername).
		Filter("member_value", username).
		All(&userMembers)
	if err != nil && err != orm.ErrNoRows {
		return nil, err
	}

	// 2. 构建 map[知识本UUID][]成员记录 用于后续取最高权限
	kbMembersMap := make(map[string][]*domain.KnowledgebaseMember)
	for _, member := range userMembers {
		kbMembersMap[member.KnowledgebaseUUID] = append(kbMembersMap[member.KnowledgebaseUUID], member)
	}

	// 3. 查询邮件组匹配（使用UUAP API）
	if r.uuapClient != nil {
		userEmailGroupNames, err := r.uuapClient.GetUserEmailGroupAddresses(context.Background(), username)
		if err == nil && len(userEmailGroupNames) > 0 {
			var emailGroupMembers []*domain.KnowledgebaseMember
			_, err := r.getOrmer(ctx).QueryTable(new(domain.KnowledgebaseMember)).
				Filter("knowledgebase_uuid__in", kbUUIDs).
				Filter("member_type", constants.KnowledgebaseMemberTypeEmailGroup).
				All(&emailGroupMembers)
			if err == nil {
				userGroupNameSet := make(map[string]bool)
				for _, name := range userEmailGroupNames {
					userGroupNameSet[name] = true
				}

				for _, m := range emailGroupMembers {
					if userGroupNameSet[m.MemberValue] {
						kbMembersMap[m.KnowledgebaseUUID] = append(kbMembersMap[m.KnowledgebaseUUID], m)
					}
				}
			}
		}
	}

	// 4. 查询部门匹配（使用UUAP API）
	/*
		if r.uuapClient != nil {
			userDept, err := r.uuapClient.GetUserBusinessDeptOrg(context.Background(), username)
			if err == nil && userDept != nil {
				var deptMembers []*domain.KnowledgebaseMember
				_, err := r.getOrmer(ctx).QueryTable(new(domain.KnowledgebaseMember)).
					Filter("knowledgebase_uuid__in", kbUUIDs).
					Filter("member_type", constants.KnowledgebaseMemberTypeDepartment).
					All(&deptMembers)
				if err == nil {
					// 获取用户所属的所有部门标识（包含各级上级部门）
					userDeptKeys := userDept.GetAllDepartmentKeys()
					userDeptKeySet := make(map[string]bool)
					for _, key := range userDeptKeys {
						userDeptKeySet[key] = true
					}

					// 匹配用户所属的任意层级部门
					for _, m := range deptMembers {
						if userDeptKeySet[m.MemberValue] {
							kbMembersMap[m.KnowledgebaseUUID] = append(kbMembersMap[m.KnowledgebaseUUID], m)
						}
					}
				}
			}
		}*/

	// 5. 查询群匹配（使用如流API）
	if r.infoFlowClient != nil {
		// 先查出这批知识本中配置的所有群成员记录
		var chatGroupMembers []*domain.KnowledgebaseMember
		_, err := r.getOrmer(ctx).QueryTable(new(domain.KnowledgebaseMember)).
			Filter("knowledgebase_uuid__in", kbUUIDs).
			Filter("member_type", constants.KnowledgebaseMemberTypeChatGroup).
			All(&chatGroupMembers)
		if err == nil && len(chatGroupMembers) > 0 {
			var matchedSet map[string]bool
			if userMatchedGroups != nil {
				// 直接使用上层传入的 hint，跳过如流 API 调用
				matchedSet = userMatchedGroups
			} else {
				groupIDs := collectUniqueGroupIDs(chatGroupMembers)
				matchedGroupIDs, err := r.infoFlowClient.IsUserInGroups(context.Background(), groupIDs, username)
				if err != nil {
					log.Logger.Errorf("[getUserRolesBatchFromDB] 检查用户群失败: %v", err)
					// 继续处理，不阻塞
				} else {
					matchedSet = make(map[string]bool, len(matchedGroupIDs))
					for _, gid := range matchedGroupIDs {
						matchedSet[gid] = true
					}
				}
			}
			for _, m := range chatGroupMembers {
				if matchedSet[m.MemberValue] {
					kbMembersMap[m.KnowledgebaseUUID] = append(kbMembersMap[m.KnowledgebaseUUID], m)
				}
			}
		}
	} else {
		log.Logger.Warnf("[getUserRolesBatchFromDB] infoFlowClient 为 nil，跳过群匹配")
	}

	// 6. 对每个知识本，取最高权限角色
	roleMap := make(map[string]string)
	for kbUUID, members := range kbMembersMap {
		if len(members) > 0 {
			roleMap[kbUUID] = getHighestRole(members)
		}
	}

	return roleMap, nil
}

// InvalidateUserRoleCache 删除用户角色缓存
// 在成员添加/删除/角色变更时调用
func (r *knowledgebaseMemberRepositoryImpl) InvalidateUserRoleCache(kbUUID, username string) {
	cacher := cache.GetCache()
	cacheKey := fmt.Sprintf("%s%s:%s", constants.CacheKeyUserRole, kbUUID, username)
	cacher.Delete(context.Background(), cacheKey)
}

// InvalidateUserRoleCacheByKBUUID 删除指定知识本下所有用户的角色缓存
// 在知识本更新/删除时调用
func (r *knowledgebaseMemberRepositoryImpl) InvalidateUserRoleCacheByKBUUID(kbUUID string) {
	cacher := cache.GetCache()
	// 使用 pattern 删除该知识本所有用户角色缓存
	pattern := fmt.Sprintf("%s%s:*", constants.CacheKeyUserRole, kbUUID)
	cacher.DeleteByPattern(context.Background(), pattern)
}

// collectUniqueGroupIDs 从群成员记录中收集不重复的群ID列表
func collectUniqueGroupIDs(members []*domain.KnowledgebaseMember) []string {
	seen := make(map[string]struct{}, len(members))
	result := make([]string, 0, len(members))
	for _, m := range members {
		if _, exists := seen[m.MemberValue]; !exists {
			seen[m.MemberValue] = struct{}{}
			result = append(result, m.MemberValue)
		}
	}
	return result
}

// getHighestRole 从多个成员记录中获取最高角色
// 权限优先级: admin > editor > viewer
// 遵循设计文档 5.3 节要求
func getHighestRole(members []*domain.KnowledgebaseMember) string {
	if len(members) == 0 {
		return ""
	}

	// 检查是否有 admin 角色
	for _, m := range members {
		if m.Role == constants.KnowledgebaseMemberRoleAdmin {
			return constants.KnowledgebaseMemberRoleAdmin
		}
	}

	// 检查是否有 editor 角色
	for _, m := range members {
		if m.Role == constants.KnowledgebaseMemberRoleEditor {
			return constants.KnowledgebaseMemberRoleEditor
		}
	}

	// 默认返回 viewer
	return constants.KnowledgebaseMemberRoleViewer
}

// FindKBUUIDsByMemberValues 根据成员值批量查询知识本UUID
// 参数: memberType - 成员类型(username/email_group/department), memberValues - 成员值列表
// 返回: 去重的知识本UUID列表
func (r *knowledgebaseMemberRepositoryImpl) FindKBUUIDsByMemberValues(
	ctx *database.TxContext,
	memberType string,
	memberValues []string,
) ([]string, error) {
	// 1. 空列表快速返回
	if len(memberValues) == 0 {
		return []string{}, nil
	}

	// 2. 限制memberValues列表最大长度为500（防止SQL过大）
	if len(memberValues) > 500 {
		memberValues = memberValues[:500]
	}

	// 3. 查询匹配的成员记录
	var members []*domain.KnowledgebaseMember
	_, err := r.getOrmer(ctx).QueryTable(new(domain.KnowledgebaseMember)).
		Filter("member_type", memberType).
		Filter("member_value__in", memberValues).
		All(&members)
	if err != nil && err != orm.ErrNoRows {
		return nil, err
	}

	// 4. 提取并去重知识本UUID
	uuidSet := make(map[string]struct{})
	for _, m := range members {
		uuidSet[m.KnowledgebaseUUID] = struct{}{}
	}

	// 5. 转换为列表
	uuids := make([]string, 0, len(uuidSet))
	for uuid := range uuidSet {
		uuids = append(uuids, uuid)
	}

	return uuids, nil
}

// FindDistinctMemberValues 查询指定类型的所有不重复成员值
func (r *knowledgebaseMemberRepositoryImpl) FindDistinctMemberValues(ctx *database.TxContext, memberType string) ([]string, error) {
	var members []*domain.KnowledgebaseMember
	_, err := r.getOrmer(ctx).QueryTable(new(domain.KnowledgebaseMember)).
		Filter("member_type", memberType).
		All(&members, "MemberValue")
	if err != nil && err != orm.ErrNoRows {
		return nil, err
	}

	seen := make(map[string]struct{}, len(members))
	result := make([]string, 0, len(members))
	for _, m := range members {
		if _, exists := seen[m.MemberValue]; !exists {
			seen[m.MemberValue] = struct{}{}
			result = append(result, m.MemberValue)
		}
	}
	return result, nil
}

// Update 更新成员信息
func (r *knowledgebaseMemberRepositoryImpl) Update(ctx *database.TxContext, member *domain.KnowledgebaseMember) error {
	_, err := r.getOrmer(ctx).Update(member)
	return err
}

// DeleteByUUID 根据UUID删除成员
func (r *knowledgebaseMemberRepositoryImpl) DeleteByUUID(ctx *database.TxContext, uuid string) error {
	_, err := r.getOrmer(ctx).QueryTable(new(domain.KnowledgebaseMember)).
		Filter("uuid", uuid).
		Delete()
	return err
}

// FindByUUID 根据UUID查询成员
func (r *knowledgebaseMemberRepositoryImpl) FindByUUID(ctx *database.TxContext, uuid string) (*domain.KnowledgebaseMember, error) {
	member := &domain.KnowledgebaseMember{}
	err := r.getOrmer(ctx).QueryTable(new(domain.KnowledgebaseMember)).
		Filter("uuid", uuid).
		One(member)
	if err == orm.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return member, nil
}

// CountByKBUUID 统计知识本成员数量（带缓存）
func (r *knowledgebaseMemberRepositoryImpl) CountByKBUUID(ctx *database.TxContext, kbUUID string) (int64, error) {
	// 事务内不使用缓存
	if ctx != nil {
		count, err := r.getOrmer(ctx).QueryTable(new(domain.KnowledgebaseMember)).
			Filter("knowledgebase_uuid", kbUUID).
			Count()
		return count, err
	}

	// 1. 先查缓存
	cacher := cache.GetCache()
	cacheKey := fmt.Sprintf("%s%s", constants.CacheKeyKBMemberCount, kbUUID)
	cachedData, err := cacher.Get(context.Background(), cacheKey)
	if err == nil && cachedData != "" {
		if count, err := strconv.ParseInt(cachedData, 10, 64); err == nil {
			return count, nil
		}
	}

	// 2. 缓存未命中，查询数据库
	count, err := r.getOrmer(ctx).QueryTable(new(domain.KnowledgebaseMember)).
		Filter("knowledgebase_uuid", kbUUID).
		Count()
	if err != nil {
		return 0, err
	}

	// 3. 写入缓存（过期时间5分钟）
	cacher.Set(context.Background(), cacheKey, strconv.FormatInt(count, 10), 5*time.Minute)

	return count, nil
}

// CountAdminsByKBUUID 统计知识本管理员数量
func (r *knowledgebaseMemberRepositoryImpl) CountAdminsByKBUUID(ctx *database.TxContext, kbUUID string) (int64, error) {
	count, err := r.getOrmer(ctx).QueryTable(new(domain.KnowledgebaseMember)).
		Filter("knowledgebase_uuid", kbUUID).
		Filter("role", constants.KnowledgebaseMemberRoleAdmin).
		Count()
	return count, err
}

// CountByKBUUIDsBatch 批量统计多个知识本的成员数量（复用单个缓存）
func (r *knowledgebaseMemberRepositoryImpl) CountByKBUUIDsBatch(ctx *database.TxContext, kbUUIDs []string) (map[string]int64, error) {
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
		cacheKey := fmt.Sprintf("%s%s", constants.CacheKeyKBMemberCount, kbUUID)
		cachedData, err := cacher.Get(context.Background(), cacheKey)
		if err == nil && cachedData != "" {
			if count, err := strconv.ParseInt(cachedData, 10, 64); err == nil {
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
		cacheKey := fmt.Sprintf("%s%s", constants.CacheKeyKBMemberCount, kbUUID)
		cacher.Set(context.Background(), cacheKey, strconv.FormatInt(count, 10), 5*time.Minute)
		result[kbUUID] = count
	}

	return result, nil
}

// countByKBUUIDsBatchFromDB 从数据库批量统计成员数量
func (r *knowledgebaseMemberRepositoryImpl) countByKBUUIDsBatchFromDB(ctx *database.TxContext, kbUUIDs []string) (map[string]int64, error) {
	result := make(map[string]int64)
	if len(kbUUIDs) == 0 {
		return result, nil
	}

	// 安全校验：限制批量查询数量，防止SQL语句过长和内存消耗过大
	if len(kbUUIDs) > 1000 {
		return nil, errors.New(errors.ErrCodeInvalidParam, "批量查询数量超过限制(最多1000个)")
	}

	// 使用 GROUP BY 一次性查询所有知识本的成员数量
	type countResult struct {
		KnowledgebaseUUID string `orm:"column(knowledgebase_uuid)"`
		Count             int64  `orm:"column(cnt)"`
	}

	var counts []countResult

	placeholders := strings.Repeat("?,", len(kbUUIDs))
	placeholders = placeholders[:len(placeholders)-1]

	sql := fmt.Sprintf(`
	SELECT knowledgebase_uuid, COUNT(id) AS cnt
	FROM knowledgebase_member
	WHERE knowledgebase_uuid IN (%s)
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

	// 对于没有成员的知识本，设置数量为0
	for _, uuid := range kbUUIDs {
		if _, exists := result[uuid]; !exists {
			result[uuid] = 0
		}
	}

	return result, nil
}

// InvalidateMemberCountCache 删除知识本成员数量缓存
// 在成员添加/删除时调用
func (r *knowledgebaseMemberRepositoryImpl) InvalidateMemberCountCache(kbUUID string) {
	cacher := cache.GetCache()
	cacheKey := fmt.Sprintf("%s%s", constants.CacheKeyKBMemberCount, kbUUID)
	cacher.Delete(context.Background(), cacheKey)
}

// DeleteMembersByTypeAndValue 删除指定类型+值的所有成员记录，返回受影响的知识本UUID列表（去重）
func (r *knowledgebaseMemberRepositoryImpl) DeleteMembersByTypeAndValue(ctx *database.TxContext, memberType, memberValue string) ([]string, error) {
	// 1. 先查询受影响的知识本UUID
	var members []*domain.KnowledgebaseMember
	_, err := r.getOrmer(ctx).QueryTable(new(domain.KnowledgebaseMember)).
		Filter("member_type", memberType).
		Filter("member_value", memberValue).
		All(&members, "knowledgebase_uuid")
	if err != nil && err != orm.ErrNoRows {
		return nil, err
	}

	if len(members) == 0 {
		return []string{}, nil
	}

	// 2. 收集去重的知识本UUID
	seen := make(map[string]struct{}, len(members))
	kbUUIDs := make([]string, 0, len(members))
	for _, m := range members {
		if _, exists := seen[m.KnowledgebaseUUID]; !exists {
			seen[m.KnowledgebaseUUID] = struct{}{}
			kbUUIDs = append(kbUUIDs, m.KnowledgebaseUUID)
		}
	}

	// 3. 批量删除
	_, err = r.getOrmer(ctx).QueryTable(new(domain.KnowledgebaseMember)).
		Filter("member_type", memberType).
		Filter("member_value", memberValue).
		Delete()
	if err != nil {
		return nil, err
	}

	return kbUUIDs, nil
}
