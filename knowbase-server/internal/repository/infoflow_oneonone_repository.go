// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package repository

import (
	"fmt"
	"strings"

	"github.com/beego/beego/v2/client/orm"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/domain"
)

// InfoFlowOneOnOneRepository 如流单聊消息仓储接口
type InfoFlowOneOnOneRepository interface {
	// Create 创建单条单聊消息记录
	Create(msg *domain.InfoFlowOneOnOneMessage) error

	// ListByConditions 根据条件查询消息列表
	ListByConditions(
		userID string,
		startTime int64,
		endTime int64,
		msgType string,
		offset int,
		limit int,
	) ([]*domain.InfoFlowOneOnOneMessage, int64, error)
}

// infoFlowOneOnOneRepositoryImpl 如流单聊消息仓储实现
type infoFlowOneOnOneRepositoryImpl struct {
	ormer orm.Ormer
}

// NewInfoFlowOneOnOneRepository 创建如流单聊消息仓储实例
func NewInfoFlowOneOnOneRepository(ormer orm.Ormer) InfoFlowOneOnOneRepository {
	return &infoFlowOneOnOneRepositoryImpl{
		ormer: ormer,
	}
}

// Create 创建单条单聊消息记录
func (r *infoFlowOneOnOneRepositoryImpl) Create(msg *domain.InfoFlowOneOnOneMessage) error {
	// 尝试插入，如果msg_id已存在（唯一索引冲突），则忽略错误
	_, err := r.ormer.Insert(msg)
	if err != nil {
		// 检查是否是唯一索引冲突
		if strings.Contains(err.Error(), "Duplicate entry") || strings.Contains(err.Error(), "unique constraint") {
			// 消息已存在，认为是幂等操作，返回成功
			return nil
		}
		return fmt.Errorf("插入单聊消息失败: %w", err)
	}
	return nil
}

// ListByConditions 根据条件查询消息列表
func (r *infoFlowOneOnOneRepositoryImpl) ListByConditions(
	userID string,
	startTime int64,
	endTime int64,
	msgType string,
	offset int,
	limit int,
) ([]*domain.InfoFlowOneOnOneMessage, int64, error) {
	qs := r.ormer.QueryTable(new(domain.InfoFlowOneOnOneMessage))

	// 构建查询条件
	// userID 查询：匹配发送者或接收者
	if userID != "" {
		cond := orm.NewCondition()
		cond = cond.Or("user_id", userID)
		qs = qs.SetCond(cond)
	}

	// 时间范围过滤
	if startTime > 0 {
		qs = qs.Filter("time__gte", startTime)
	}
	if endTime > 0 {
		qs = qs.Filter("time__lte", endTime)
	}

	// 消息类型过滤
	if msgType != "" {
		qs = qs.Filter("msg_type", msgType)
	}

	// 查询总数
	total, err := qs.Count()
	if err != nil {
		return nil, 0, fmt.Errorf("查询单聊消息总数失败: %w", err)
	}

	// 查询数据（按时间倒序）
	var messages []*domain.InfoFlowOneOnOneMessage
	_, err = qs.OrderBy("-time").Limit(limit, offset).All(&messages)
	if err != nil {
		return nil, 0, fmt.Errorf("查询单聊消息列表失败: %w", err)
	}

	return messages, total, nil
}
