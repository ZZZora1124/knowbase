// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/api/middleware"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/api/request"

	kbResp "icode.baidu.com/baidu/devops-ai/knowbase-server/internal/api/response"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/domain"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/infrastructure/database"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/infrastructure/infoflow"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/infrastructure/uuap"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/repository"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/service"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/errors"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/log"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/response"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/types/constants"
)

// KnowledgebaseMemberHandler 知识本成员管理处理器
type KnowledgebaseMemberHandler struct {
	memberService service.KnowledgebaseMemberService
}

// NewKnowledgebaseMemberHandler 创建知识本成员管理处理器实例
func NewKnowledgebaseMemberHandler() *KnowledgebaseMemberHandler {
	ormer := database.GetOrm()

	kbRepo := repository.NewKnowledgebaseRepository(ormer)
	memberRepo := repository.NewKnowledgebaseMemberRepository(ormer)
	logRepo := repository.NewKnowledgebaseOperationLogRepository(ormer)
	txManager := database.NewTransactionManager()

	uuapClient, _ := uuap.GetClient()
	infoFlowClient := infoflow.GetClient()
	memberService := service.NewKnowledgebaseMemberService(kbRepo, memberRepo, logRepo, txManager, uuapClient, infoFlowClient)

	return &KnowledgebaseMemberHandler{
		memberService: memberService,
	}
}

// AddMember 添加成员
// POST /api/v1/knowledgebases/:kb_uuid/members
func (h *KnowledgebaseMemberHandler) AddMember(c *gin.Context) {
	ctx := middleware.GetContext(c)

	kbUUID := c.Param("kb_uuid")
	if kbUUID == "" {
		response.Error(c, errors.ErrInvalidParam)
		return
	}

	var req request.AddMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Logger.ErrorfCtx(ctx, "解析请求体失败: %v", err)
		response.Error(c, errors.ErrInvalidRequest)
		return
	}

	// 获取当前用户
	username, exists := c.Get(constants.AuthUsername)
	if !exists {
		log.Logger.ErrorfCtx(ctx, "获取用户名失败")
		response.Error(c, errors.ErrInvalidToken)
		return
	}

	log.Logger.InfofCtx(ctx, "用户 %s 请求添加成员到知识本 %s: %+v", username, kbUUID, req)

	// 获取是否为Admin客户端
	isAdmin := middleware.GetClientIsAdmin(c)

	// 调用服务层
	member, err := h.memberService.AddMember(ctx, kbUUID, req.MemberType, req.MemberValue, req.Role, username.(string), isAdmin)
	if err != nil {
		log.Logger.ErrorfCtx(ctx, "添加成员失败: %v", err)
		response.Error(c, err)
		return
	}

	response.Success(c, toMemberResponse(member))
}

// ListMembers 查询成员列表
// GET /api/v1/knowledgebases/:kb_uuid/members
func (h *KnowledgebaseMemberHandler) ListMembers(c *gin.Context) {
	ctx := middleware.GetContext(c)

	kbUUID := c.Param("kb_uuid")
	if kbUUID == "" {
		response.Error(c, errors.ErrInvalidParam)
		return
	}

	// 解析分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	// 获取当前用户
	username, exists := c.Get(constants.AuthUsername)
	if !exists {
		log.Logger.ErrorfCtx(ctx, "获取用户名失败")
		response.Error(c, errors.ErrInvalidToken)
		return
	}

	// 调用服务层
	members, total, err := h.memberService.ListMembers(ctx, kbUUID, username.(string), page, pageSize)
	if err != nil {
		log.Logger.ErrorfCtx(ctx, "查询成员列表失败: %v", err)
		response.Error(c, err)
		return
	}

	// 转换为响应格式
	items := make([]*kbResp.MemberResponse, 0, len(members)+1)

	// 若当前用户无直接 username 成员记录，但通过邮件组/群聊持有权限，
	// 在列表最前插入一条虚拟记录（UUID 为空），方便前端展示当前用户身份
	operatorStr := username.(string)
	hasDirectMembership := false
	for _, m := range members {
		if m.MemberType == constants.KnowledgebaseMemberTypeUsername && m.MemberValue == operatorStr {
			hasDirectMembership = true
			break
		}
	}
	if !hasDirectMembership {
		role, err := h.memberService.GetCurrentUserRole(ctx, kbUUID, operatorStr)
		if err == nil && role != "" {
			items = append(items, &kbResp.MemberResponse{
				KnowledgebaseUUID: kbUUID,
				MemberType:        constants.KnowledgebaseMemberTypeUsername,
				MemberValue:       operatorStr,
				Role:              role,
			})
			total++
		}
	}

	for _, member := range members {
		items = append(items, toMemberResponse(member))
	}

	resp := &kbResp.ListMembersResponse{
		Total:    total,
		Page:     page,
		PageSize: pageSize,
		Items:    items,
	}

	response.Success(c, resp)
}

// ListExpandedMembers 查询展开后的成员列表
// GET /api/v1/knowledgebases/:kb_uuid/members/expanded
// 将邮件组和部门成员展开为实际用户列表
func (h *KnowledgebaseMemberHandler) ListExpandedMembers(c *gin.Context) {
	ctx := middleware.GetContext(c)

	kbUUID := c.Param("kb_uuid")
	if kbUUID == "" {
		response.Error(c, errors.ErrInvalidParam)
		return
	}

	// 解析分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	// 获取当前用户
	username, exists := c.Get(constants.AuthUsername)
	if !exists {
		log.Logger.ErrorfCtx(ctx, "获取用户名失败")
		response.Error(c, errors.ErrInvalidToken)
		return
	}

	// 调用服务层
	members, total, err := h.memberService.ListExpandedMembers(ctx, kbUUID, username.(string), page, pageSize)
	if err != nil {
		log.Logger.ErrorfCtx(ctx, "查询展开成员列表失败: %v", err)
		response.Error(c, err)
		return
	}

	// 转换为响应格式
	items := make([]*kbResp.ExpandedMemberResponse, 0, len(members))
	for _, member := range members {
		items = append(items, &kbResp.ExpandedMemberResponse{
			Username:          member.Username,
			Role:              member.Role,
			MemberSource:      member.MemberSource,
			SourceName:        member.SourceName,
			KnowledgebaseUUID: member.KnowledgebaseUUID,
			JoinedAt:          member.JoinedAt,
		})
	}

	resp := &kbResp.ListExpandedMembersResponse{
		Total:    total,
		Page:     page,
		PageSize: pageSize,
		Items:    items,
	}

	response.Success(c, resp)
}

// ListAdmins 查询知识本管理员列表
// GET /api/v1/knowledgebases/:kb_uuid/admins
// 仅返回 role=admin 的成员（不展开邮件组/部门），用于"找谁申请"场景；
// 已登录用户即可访问，不要求是知识本成员。
func (h *KnowledgebaseMemberHandler) ListAdmins(c *gin.Context) {
	ctx := middleware.GetContext(c)

	kbUUID := c.Param("kb_uuid")
	if kbUUID == "" {
		response.Error(c, errors.ErrInvalidParam)
		return
	}

	admins, err := h.memberService.ListAdmins(ctx, kbUUID)
	if err != nil {
		log.Logger.ErrorfCtx(ctx, "查询管理员列表失败: %v", err)
		response.Error(c, err)
		return
	}

	items := make([]*kbResp.AdminResponse, 0, len(admins))
	for _, m := range admins {
		items = append(items, &kbResp.AdminResponse{
			MemberType:  m.MemberType,
			MemberValue: m.MemberValue,
			Role:        m.Role,
		})
	}

	response.Success(c, &kbResp.ListAdminsResponse{
		Total: int64(len(items)),
		Items: items,
	})
}

// UpdateMemberRole 更新成员角色
// PUT /api/v1/knowledgebases/:kb_uuid/members/:uuid
func (h *KnowledgebaseMemberHandler) UpdateMemberRole(c *gin.Context) {
	ctx := middleware.GetContext(c)

	kbUUID := c.Param("kb_uuid")
	memberUUID := c.Param("uuid")
	if kbUUID == "" || memberUUID == "" {
		response.Error(c, errors.ErrInvalidParam)
		return
	}

	var req request.UpdateMemberRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Logger.ErrorfCtx(ctx, "解析请求体失败: %v", err)
		response.Error(c, errors.ErrInvalidRequest)
		return
	}

	// 获取当前用户
	username, exists := c.Get(constants.AuthUsername)
	if !exists {
		log.Logger.ErrorfCtx(ctx, "获取用户名失败")
		response.Error(c, errors.ErrInvalidToken)
		return
	}

	log.Logger.InfofCtx(ctx, "用户 %s 请求更新成员 %s 的角色为 %s", username, memberUUID, req.Role)

	// 获取是否为Admin客户端
	isAdmin := middleware.GetClientIsAdmin(c)

	// 调用服务层
	err := h.memberService.UpdateMemberRole(ctx, kbUUID, memberUUID, req.Role, username.(string), isAdmin)
	if err != nil {
		log.Logger.ErrorfCtx(ctx, "更新成员角色失败: %v", err)
		response.Error(c, err)
		return
	}

	response.Success(c, nil)
}

// RemoveMember 移除成员
// DELETE /api/v1/knowledgebases/:kb_uuid/members/:uuid
func (h *KnowledgebaseMemberHandler) RemoveMember(c *gin.Context) {
	ctx := middleware.GetContext(c)

	kbUUID := c.Param("kb_uuid")
	memberUUID := c.Param("uuid")
	if kbUUID == "" || memberUUID == "" {
		response.Error(c, errors.ErrInvalidParam)
		return
	}

	// 获取当前用户
	username, exists := c.Get(constants.AuthUsername)
	if !exists {
		log.Logger.ErrorfCtx(ctx, "获取用户名失败")
		response.Error(c, errors.ErrInvalidToken)
		return
	}

	log.Logger.InfofCtx(ctx, "用户 %s 请求移除成员 %s", username, memberUUID)

	// 获取是否为Admin客户端
	isAdmin := middleware.GetClientIsAdmin(c)

	// 调用服务层
	err := h.memberService.RemoveMember(ctx, kbUUID, memberUUID, username.(string), isAdmin)
	if err != nil {
		log.Logger.ErrorfCtx(ctx, "移除成员失败: %v", err)
		response.Error(c, err)
		return
	}

	response.Success(c, nil)
}

// toMemberResponse 将领域模型转换为响应模型
func toMemberResponse(member *domain.KnowledgebaseMember) *kbResp.MemberResponse {
	if member == nil {
		return nil
	}

	return &kbResp.MemberResponse{
		UUID:              member.UUID,
		KnowledgebaseUUID: member.KnowledgebaseUUID,
		MemberType:        member.MemberType,
		MemberValue:       member.MemberValue,
		Role:              member.Role,
		CreatedAt:         member.CreatedAt,
		UpdatedAt:         member.UpdatedAt,
	}
}

