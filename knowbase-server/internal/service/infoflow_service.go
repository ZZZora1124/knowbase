// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/api/request"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/api/response"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/domain"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/infrastructure/cache"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/repository"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/log"
)

const (
	// MaxQueryDays 最大查询天数
	MaxQueryDays = 90
	// MillisecondsPerDay 每天的毫秒数
	MillisecondsPerDay = 24 * 60 * 60 * 1000
)

// InfoFlowService 如流消息服务接口
type InfoFlowService interface {
	// ProcessGroupMessage 处理如流群消息回调
	ProcessGroupMessage(req *request.InfoFlowCallbackRequest, body string) error

	// ListGroupMessages 查询群消息列表
	ListGroupMessages(req *request.ListGroupMessagesRequest) (*response.ListGroupMessagesResponse, error)

	// DeleteGroupMessages 删除群消息
	DeleteGroupMessages(groupID int64, dayBefore int, operator string) (int64, error)

	// CleanupExpiredMessages 定时清理过期消息
	CleanupExpiredMessages(ctx context.Context, retentionDays int, lockExpireSeconds int) (int64, error)
}

// infoFlowServiceImpl 如流消息服务实现
type infoFlowServiceImpl struct {
	repo repository.InfoFlowRepository
}

// NewInfoFlowService 创建如流消息服务实例
func NewInfoFlowService(repo repository.InfoFlowRepository) InfoFlowService {
	return &infoFlowServiceImpl{
		repo: repo,
	}
}

// ProcessGroupMessage 处理如流群消息回调
func (s *infoFlowServiceImpl) ProcessGroupMessage(req *request.InfoFlowCallbackRequest, body string) error {
	// 提取基础信息
	header := req.Body.Data.Message.Header
	bodyList := req.Body.Data.Message.Body

	// 方便排查问题
	if req.Body.UserID == "hekai09" {
		log.Logger.Infof("ProcessGroupMessage, body: %v", body)
	}

	if len(bodyList) == 0 {
		return fmt.Errorf("消息体为空")
	}

	// 构建消息列表（每个 body 元素一条记录）
	messages := make([]*domain.InfoFlowMessage, 0, len(bodyList))

	for _, bodyItem := range bodyList {
		// 提取 content_type
		contentType, _ := bodyItem["type"].(string)
		if contentType == "" {
			continue // 跳过无效的消息
		}

		// 序列化完整的 body 元素到 JSON
		contentJSON, err := json.Marshal(bodyItem)
		if err != nil {
			continue // 序列化失败，跳过
		}

		// 根据类型提取文本内容
		contentText := extractContentText(contentType, bodyItem)

		// 构建消息对象
		msg := &domain.InfoFlowMessage{
			// 基础信息
			GroupID:   req.Body.GroupID,
			GroupName: req.Body.GroupName,
			UserID:    req.Body.UserID,
			Time:      req.Body.Data.Time,

			// 头部信息
			FromUserID: header.FromUserID,
			ToID:       header.ToID,
			ToType:     header.ToType,
			MsgType:    header.MsgType,

			// 内容信息
			ContentType: contentType,
			ContentText: contentText,
			ContentJSON: string(contentJSON),
		}
		if header.MessageID != nil {
			msg.MessageID = fmt.Sprintf("%v", header.MessageID)
		}

		messages = append(messages, msg)
	}

	if len(messages) == 0 {
		return fmt.Errorf("没有有效的消息可保存")
	}

	// 批量保存到数据库
	return s.repo.BatchCreate(messages)
}

// extractContentText 根据消息类型提取文本内容
func extractContentText(contentType string, bodyItem map[string]interface{}) string {
	switch contentType {
	case domain.MessageTypeText, domain.MessageTypeReplyData:
		if content, ok := bodyItem["content"].(string); ok {
			return content
		}
	case domain.MessageTypeImage:
		if url, ok := bodyItem["downloadurl"].(string); ok {
			return url
		}
	case domain.MessageTypeAT:
		if name, ok := bodyItem["name"].(string); ok {
			return fmt.Sprintf("@%s", name)
		}
		if atAll, ok := bodyItem["atall"].(bool); ok {
			if atAll {
				return fmt.Sprintf("@全体成员")
			}
			return ""
		}
	case domain.MessageTypeLink:
		if label, ok := bodyItem["label"].(string); ok {
			return label
		}
	case domain.MessageTypeCommand:
		if cmd, ok := bodyItem["commandname"].(string); ok {
			return cmd
		}
	case domain.MessageTypeMerged:
		message := ""
		if title, ok := bodyItem["title"].(string); ok {
			message += fmt.Sprintf("这是一条转发的%s,", title)
		}
		if aidesc, ok := bodyItem["aidesc"].(string); ok {
			message += fmt.Sprintf("记录总结:%s", aidesc)
		}
		if desc, ok := bodyItem["desc"].(string); ok {
			message += fmt.Sprintf("记录详情是%s", desc)
		}
		return message
	}
	return ""
}

// ListGroupMessages 查询群消息列表
func (s *infoFlowServiceImpl) ListGroupMessages(req *request.ListGroupMessagesRequest) (*response.ListGroupMessagesResponse, error) {
	// 1. 参数校验和默认值处理
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}
	// 限制最大每页数量，防止查询过多数据
	if req.PageSize > 100 {
		req.PageSize = 100
	}

	// 2. 处理时间范围，确保不超过90天
	now := time.Now().UnixNano() / 1e6 // 当前时间戳（毫秒）
	maxQueryMilliseconds := int64(MaxQueryDays * MillisecondsPerDay)

	// 2.1 如果未提供任何时间，默认查询最近90天
	if req.StartTime == 0 && req.EndTime == 0 {
		req.EndTime = now
		req.StartTime = now - maxQueryMilliseconds
	} else if req.StartTime == 0 {
		// 2.2 只提供了结束时间，计算开始时间
		req.StartTime = req.EndTime - maxQueryMilliseconds
		if req.StartTime < 0 {
			req.StartTime = 0
		}
	} else if req.EndTime == 0 {
		// 2.3 只提供了开始时间，计算结束时间
		req.EndTime = req.StartTime + maxQueryMilliseconds
		if req.EndTime > now {
			req.EndTime = now
		}
	} else {
		// 2.4 同时提供了开始和结束时间，校验范围
		if req.StartTime > req.EndTime {
			return nil, fmt.Errorf("开始时间不能大于结束时间")
		}
		if req.EndTime-req.StartTime > maxQueryMilliseconds {
			return nil, fmt.Errorf("查询时间范围不能超过%d天", MaxQueryDays)
		}
	}

	// 3. 计算偏移量
	offset := (req.Page - 1) * req.PageSize

	// 4. 调用 repository 查询
	messages, total, err := s.repo.ListByConditions(
		req.GroupID,
		req.StartTime,
		req.EndTime,
		req.ContentType,
		req.UserID,
		offset,
		req.PageSize,
	)
	if err != nil {
		return nil, err
	}

	// 5. 组装响应
	return &response.ListGroupMessagesResponse{
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
		Messages: messages,
	}, nil
}

// DeleteGroupMessages 删除群消息
func (s *infoFlowServiceImpl) DeleteGroupMessages(groupID int64, dayBefore int, operator string) (int64, error) {
	// 1. 计算时间边界
	var beforeTime int64
	if dayBefore > 0 {
		beforeTime = time.Now().Add(-time.Duration(dayBefore)*24*time.Hour).UnixNano() / 1e6
	} else {
		beforeTime = 0 // 删除全部
	}

	// 2. 删除消息
	deletedCount, err := s.repo.DeleteByGroupAndTime(groupID, beforeTime)
	if err != nil {
		return 0, fmt.Errorf("删除消息失败: %w", err)
	}

	// 3. 记录审计日志
	deletionLog := &domain.InfoFlowMessageDeletionLog{
		GroupID:       groupID,
		DayBefore:     dayBefore,
		DeletedCount:  deletedCount,
		Operator:      operator,
		OperationType: domain.OperationTypeManual,
	}

	if err := s.repo.CreateDeletionLog(deletionLog); err != nil {
		// 审计日志失败不影响主流程
		log.Logger.Errorf("创建删除审计日志失败: %v", err)
	}

	return deletedCount, nil
}

// CleanupExpiredMessages 定时清理过期消息（所有群）
// 返回值：删除的记录数，-1 表示未获取到锁
func (s *infoFlowServiceImpl) CleanupExpiredMessages(ctx context.Context, retentionDays int, lockExpireSeconds int) (int64, error) {
	// 1. 尝试获取分布式锁
	locked := s.tryAcquireLock(ctx, domain.CleanupLockKey, lockExpireSeconds)
	if !locked {
		return -1, nil // 未获取到锁，返回 -1 表示其他实例正在执行
	}
	defer s.releaseLock(ctx, domain.CleanupLockKey)

	// 2. 计算时间边界（毫秒）
	beforeTime := time.Now().Add(-time.Duration(retentionDays)*24*time.Hour).UnixNano() / 1e6

	// 3. 删除过期消息
	deletedCount, err := s.repo.DeleteByTime(beforeTime)
	if err != nil {
		return 0, fmt.Errorf("删除消息失败: %w", err)
	}

	// 4. 记录审计日志（group_id=0表示全局清理）
	deletionLog := &domain.InfoFlowMessageDeletionLog{
		GroupID:       0,
		DayBefore:     retentionDays,
		DeletedCount:  deletedCount,
		Operator:      "system",
		OperationType: domain.OperationTypeScheduled,
	}

	if err := s.repo.CreateDeletionLog(deletionLog); err != nil {
		// 审计日志失败不影响主流程
		log.Logger.Errorf("创建删除审计日志失败: %v", err)
	}

	return deletedCount, nil
}

// tryAcquireLock 尝试获取分布式锁
func (s *infoFlowServiceImpl) tryAcquireLock(ctx context.Context, key string, expireSeconds int) bool {
	cacheClient := cache.GetCache()
	if cacheClient == nil {
		// Redis 未配置，返回 true 允许执行（单实例模式）
		log.Logger.Warnf("Redis 缓存未配置，分布式锁降级为单实例模式")
		return true
	}

	err := cacheClient.SetNX(ctx, key, time.Now().Unix(), time.Duration(expireSeconds)*time.Second)
	if err != nil {
		log.Logger.Warnf("获取分布式锁失败: %v", err)
		return false
	}

	return true
}

// releaseLock 释放分布式锁
func (s *infoFlowServiceImpl) releaseLock(ctx context.Context, key string) {
	cacheClient := cache.GetCache()
	if cacheClient == nil {
		return
	}

	if err := cacheClient.Delete(ctx, key); err != nil {
		log.Logger.Errorf("释放分布式锁失败: %v", err)
	}
}
