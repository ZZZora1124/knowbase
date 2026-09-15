// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package handler

import (
	"encoding/json"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/api/middleware"
	"io"

	"github.com/gin-gonic/gin"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/api/request"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/infrastructure/database"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/infrastructure/infoflow"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/repository"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/service"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/errors"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/log"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/response"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/types/constants"
)

// InfoFlowHandler 如流消息处理器
type InfoFlowHandler struct {
	service        service.InfoFlowService
	oneOneService  service.InfoFlowOneOnOneService
	infoFlowClient *infoflow.Client
}

// NewInfoFlowHandler 创建如流消息处理器实例
func NewInfoFlowHandler() *InfoFlowHandler {
	// 初始化依赖
	ormer := database.GetOrm()

	// 初始化群消息服务
	repo := repository.NewInfoFlowRepository(ormer)
	svc := service.NewInfoFlowService(repo)

	// 初始化单聊消息服务
	oneOneRepo := repository.NewInfoFlowOneOnOneRepository(ormer)
	oneOneSvc := service.NewInfoFlowOneOnOneService(oneOneRepo)

	return &InfoFlowHandler{
		service:        svc,
		oneOneService:  oneOneSvc,
		infoFlowClient: infoflow.GetClient(),
	}
}

// HandleGroupMessageUpload 处理如流群消息上报
func (h *InfoFlowHandler) HandleGroupMessageUpload(c *gin.Context) {
	// 读取原始请求体用于日志
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		log.Logger.Errorf("读取请求体失败: %v", err)
		response.ErrorWithMessage(c, errors.ErrCodeInvalidParam, "读取请求体失败")
		return
	}

	// 打印完整请求体用于调试
	bodyStr := string(bodyBytes)
	log.Logger.Debugf("收到如流群消息上报请求, body: %s", bodyStr)

	// 解析请求体
	var req request.InfoFlowCallbackRequest
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		log.Logger.Errorf("解析请求体失败: %v, body: %s", err, bodyStr)
		response.Error(c, errors.ErrInvalidRequest)
		return
	}

	// 调用服务层处理
	if err := h.service.ProcessGroupMessage(&req, bodyStr); err != nil {
		log.Logger.Errorf("处理如流群消息失败: %v, body: %s", err, bodyStr)
		response.ErrorWithMessage(c, errors.ErrCodeInternalError, err.Error())
		return
	}

	// 返回成功响应
	response.Success(c, gin.H{
		"message": "消息上报成功",
	})
}

// HandleListGroupMessages 查询如流群消息列表
func (h *InfoFlowHandler) HandleListGroupMessages(c *gin.Context) {
	// 解析请求体
	var req request.ListGroupMessagesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Logger.Errorf("解析查询参数失败: %v", err)
		response.Error(c, errors.ErrInvalidRequest)
		return
	}

	userID, exists := c.Get(constants.AuthUsername)
	if !exists {
		log.Logger.Errorf("获取鉴权用户ID失败")
		response.Error(c, errors.ErrInvalidToken)
		return
	}
	userIDStr := userID.(string)
	isAdminClient := middleware.GetClientIsAdmin(c)
	if isAdminClient {
		log.Logger.Info("admin-client query")
	} else {
		// 验证用户是否有权限访问该群消息
		infoFlowClient := infoflow.GetClient()
		if infoFlowClient == nil {
			log.Logger.Errorf("如流客户端未初始化，跳过群成员验证")
			// 如流客户端未配置时，允许访问（优雅降级）
		} else {
			isInGroup, err := infoFlowClient.IsUserInGroup(c.Request.Context(), req.GroupID, userIDStr)
			if err != nil {
				log.Logger.Errorf("验证用户群成员身份失败: %v", err)
				response.ErrorWithMessage(c, errors.ErrCodeInfoFlowAPIError, "验证群成员身份失败: "+err.Error())
				return
			}
			if !isInGroup {
				log.Logger.Warnf("用户 %s 不是群 %d 的成员，拒绝访问", userIDStr, req.GroupID)
				response.Error(c, errors.ErrNotGroupMember)
				return
			}
		}
	}

	// 打印查询参数用于调试
	log.Logger.Infof("%s查询如流群消息, groupId: %d, startTime: %d, endTime: %d, page: %d",
		userIDStr, req.GroupID, req.StartTime, req.EndTime, req.Page)

	// 调用服务层
	resp, err := h.service.ListGroupMessages(&req)
	if err != nil {
		log.Logger.Errorf("查询如流群消息失败: %v", err)
		response.ErrorWithMessage(c, errors.ErrCodeInternalError, err.Error())
		return
	}

	// 返回成功响应
	response.Success(c, resp)
}

// HandleOneOnOneMessageUpload 处理如流单聊消息上报
func (h *InfoFlowHandler) HandleOneOnOneMessageUpload(c *gin.Context) {
	// 读取原始请求体用于日志
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		log.Logger.Errorf("读取请求体失败: %v", err)
		response.ErrorWithMessage(c, errors.ErrCodeInvalidParam, "读取请求体失败")
		return
	}

	// 打印完整请求体用于调试
	bodyStr := string(bodyBytes)
	log.Logger.Debugf("收到如流单聊消息上报请求, body: %s", bodyStr)

	// 解析请求体
	var req request.InfoFlowOneOnOneCallbackRequest
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		log.Logger.Errorf("解析请求体失败: %v, body: %s", err, bodyStr)
		response.Error(c, errors.ErrInvalidRequest)
		return
	}

	// 调用服务层处理
	if err := h.oneOneService.ProcessOneOnOneMessage(&req, bodyStr); err != nil {
		log.Logger.Errorf("处理如流单聊消息失败: %v, body: %s", err, bodyStr)
		response.ErrorWithMessage(c, errors.ErrCodeInternalError, err.Error())
		return
	}

	// 返回成功响应
	response.Success(c, gin.H{
		"message": "消息上报成功",
	})
}

// HandleGetGroupInfo 获取群聊信息
func (h *InfoFlowHandler) HandleGetGroupInfo(c *gin.Context) {
	var req request.GetGroupInfoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Logger.Errorf("解析请求体失败: %v", err)
		response.Error(c, errors.ErrInvalidRequest)
		return
	}

	log.Logger.Infof("获取群聊信息, groupId: %d", req.GroupID)

	infoFlowClient := infoflow.GetClient()
	if infoFlowClient == nil {
		log.Logger.Errorf("如流客户端未初始化")
		response.ErrorWithMessage(c, errors.ErrCodeInfoFlowAPIError, "如流客户端未初始化，请检查如流配置")
		return
	}

	groupInfo, err := infoFlowClient.GetGroupInfo(c.Request.Context(), req.GroupID)
	if err != nil {
		log.Logger.Errorf("获取群聊信息失败, groupId: %d, err: %v", req.GroupID, err)
		response.ErrorWithMessage(c, errors.ErrCodeInfoFlowAPIError, err.Error())
		return
	}

	response.Success(c, groupInfo.Data.Data)
}

// HandleListOneOnOneMessages 查询如流单聊消息列表
func (h *InfoFlowHandler) HandleListOneOnOneMessages(c *gin.Context) {
	// 解析请求体
	var req request.ListOneOnOneMessagesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Logger.Errorf("解析查询参数失败: %v", err)
		response.Error(c, errors.ErrInvalidRequest)
		return
	}

	userID, exists := c.Get(constants.AuthUsername)
	if !exists {
		log.Logger.Errorf("获取鉴权用户ID失败")
		response.Error(c, errors.ErrInvalidToken)
		return
	}
	req.UserID = userID.(string)

	// 打印查询参数用于调试
	log.Logger.Infof("查询如流单聊消息, userId: %s, fromUserId: %s, startTime: %d, endTime: %d, page: %d",
		req.UserID, req.StartTime, req.EndTime, req.Page)

	// 调用服务层
	resp, err := h.oneOneService.ListOneOnOneMessages(&req)
	if err != nil {
		log.Logger.Errorf("查询如流单聊消息失败: %v", err)
		response.ErrorWithMessage(c, errors.ErrCodeInternalError, err.Error())
		return
	}

	// 返回成功响应
	response.Success(c, resp)
}

// HandleDeleteGroupMessages 删除群消息（需群成员验证）
func (h *InfoFlowHandler) HandleDeleteGroupMessages(c *gin.Context) {
	var req request.DeleteGroupMessagesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Logger.Errorf("解析请求参数失败: %v", err)
		response.Error(c, errors.ErrInvalidRequest)
		return
	}

	if req.GroupID == 0 && req.DayBefore == 0 {
		log.Logger.Errorf("群聊ID和天数不能同时为空")
		response.Error(c, errors.ErrInvalidRequest)
		return
	}

	// 获取操作人
	userID, exists := c.Get(constants.AuthUsername)
	if !exists {
		log.Logger.Errorf("获取鉴权用户ID失败")
		response.Error(c, errors.ErrInvalidToken)
		return
	}
	userIDStr := userID.(string)

	// 验证用户是否为群成员
	infoFlowClient := infoflow.GetClient()
	if infoFlowClient != nil {
		isInGroup, err := infoFlowClient.IsUserInGroup(c.Request.Context(), req.GroupID, userIDStr)
		if err != nil {
			log.Logger.Errorf("验证用户群成员身份失败: %v", err)
			response.ErrorWithMessage(c, errors.ErrCodeInfoFlowAPIError, "验证群成员身份失败: "+err.Error())
			return
		}
		if !isInGroup {
			log.Logger.Warnf("用户 %s 不是群 %d 的成员，拒绝删除", userIDStr, req.GroupID)
			response.Error(c, errors.ErrNotGroupMember)
			return
		}
	}

	log.Logger.Infof("用户 %s 请求删除群 %d 的消息，dayBefore: %d", userIDStr, req.GroupID, req.DayBefore)

	// 调用服务层
	deletedCount, err := h.service.DeleteGroupMessages(req.GroupID, req.DayBefore, userIDStr)
	if err != nil {
		log.Logger.Errorf("删除群消息失败: %v", err)
		response.ErrorWithMessage(c, errors.ErrCodeInternalError, err.Error())
		return
	}

	response.Success(c, gin.H{
		"deletedCount": deletedCount,
	})
}

// HandleDeleteGroupMessagesInternal 删除群消息（内部接口，无群成员验证）
func (h *InfoFlowHandler) HandleDeleteGroupMessagesInternal(c *gin.Context) {
	var req request.DeleteGroupMessagesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Logger.Errorf("解析请求参数失败: %v", err)
		response.Error(c, errors.ErrInvalidRequest)
		return
	}
	if req.GroupID == 0 && req.DayBefore == 0 {
		log.Logger.Errorf("群聊ID和天数不能同时为空")
		response.Error(c, errors.ErrInvalidRequest)
		return
	}

	// 内部接口使用固定操作人
	operator := "internal"

	log.Logger.Infof("内部接口请求删除群 %d 的消息，dayBefore: %d", req.GroupID, req.DayBefore)

	// 调用服务层
	deletedCount, err := h.service.DeleteGroupMessages(req.GroupID, req.DayBefore, operator)
	if err != nil {
		log.Logger.Errorf("删除群消息失败: %v", err)
		response.ErrorWithMessage(c, errors.ErrCodeInternalError, err.Error())
		return
	}

	response.Success(c, gin.H{
		"deletedCount": deletedCount,
	})
}

// HandleGroupMemberEvent 处理如流群成员变更事件
// 事件类型：GROUP_MEMBER_ADD（新增）/ GROUP_MEMBER_DEL（删除）
// 处理策略：刷新群成员缓存，并清理受影响用户的 user_matched_groups 缓存
func (h *InfoFlowHandler) HandleGroupMemberEvent(c *gin.Context) {
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		log.Logger.Errorf("读取群成员变更事件请求体失败: %v", err)
		response.ErrorWithMessage(c, errors.ErrCodeInvalidParam, "读取请求体失败")
		return
	}

	bodyStr := string(bodyBytes)
	log.Logger.Infof("收到如流群成员变更事件, body: %s", bodyStr)

	var req request.InfoFlowGroupEventRequest
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		log.Logger.Errorf("解析群成员变更事件请求体失败: %v, body: %s", err, bodyStr)
		response.Error(c, errors.ErrInvalidRequest)
		return
	}

	groupID := req.Body.GroupId
	eventType := req.Body.EventType
	userIDs := req.Body.Data.UserIds

	if groupID == 0 {
		log.Logger.Warnf("群成员变更事件 groupId 为空，忽略, body: %s", bodyStr)
		response.Success(c, gin.H{"message": "ignored"})
		return
	}
	if eventType != constants.InfoFlowEventGroupMemberAdd && eventType != constants.InfoFlowEventGroupMemberDel {
		log.Logger.Warnf("未知群成员变更事件类型: %s，忽略", eventType)
		response.Success(c, gin.H{"message": "ignored"})
		return
	}

	infoFlowClient := infoflow.GetClient()
	if infoFlowClient == nil {
		log.Logger.Warnf("如流客户端未初始化，跳过群成员缓存刷新")
		response.Success(c, gin.H{"message": "ok"})
		return
	}

	ctx := c.Request.Context()

	// 刷新群成员缓存（ADD/DEL 都需要）
	if refreshErr := infoFlowClient.ForceRefreshGroupCache(ctx, groupID); refreshErr != nil {
		log.Logger.Errorf("刷新群成员缓存失败: groupId=%d, eventType=%s, err=%v", groupID, eventType, refreshErr)
	} else {
		log.Logger.Infof("群成员缓存刷新成功: groupId=%d, eventType=%s", groupID, eventType)
	}

	// 清理受影响用户的 user_matched_groups 缓存（ADD/DEL 都需要，保证用户权限及时生效）
	if len(userIDs) > 0 {
		infoFlowClient.InvalidateUserMatchedGroupsCache(ctx, userIDs)
		log.Logger.Infof("已清理用户群缓存: groupId=%d, eventType=%s, users=%v", groupID, eventType, userIDs)
	}

	response.Success(c, gin.H{"message": "ok"})
}

// HandleRefreshGroupMemberCache 强制刷新群成员缓存
// POST /api/v1/infoflow/group/cache/refresh
func (h *InfoFlowHandler) HandleRefreshGroupMemberCache(c *gin.Context) {
	ctx := middleware.GetContext(c)

	var req request.RefreshGroupMemberCacheRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Logger.ErrorfCtx(ctx, "解析请求体失败: %v", err)
		response.Error(c, errors.ErrInvalidRequest)
		return
	}

	if h.infoFlowClient == nil {
		log.Logger.ErrorfCtx(ctx, "如流客户端未初始化")
		response.ErrorWithMessage(c, errors.ErrCodeInternalError, "如流客户端未初始化")
		return
	}

	if err := h.infoFlowClient.ForceRefreshGroupCache(c.Request.Context(), req.GroupID); err != nil {
		log.Logger.ErrorfCtx(ctx, "刷新群 %d 成员缓存失败: %v", req.GroupID, err)
		response.ErrorWithMessage(c, errors.ErrCodeInternalError, err.Error())
		return
	}

	log.Logger.InfofCtx(ctx, "群 %d 成员缓存刷新成功", req.GroupID)
	response.Success(c, nil)
}
