// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package repository

import (
	"github.com/beego/beego/v2/client/orm"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/domain"
)

// InfoFlowRepository 如流消息仓储接口
type InfoFlowRepository interface {
	// BatchCreate 批量创建消息
	BatchCreate(messages []*domain.InfoFlowMessage) error

	// ListByConditions 根据条件查询消息列表
	ListByConditions(
		groupID int64,
		startTime, endTime int64,
		contentType, userID string,
		offset, limit int,
	) ([]*domain.InfoFlowMessage, int64, error)

	// DeleteByGroupAndTime 删除指定群在指定时间之前的消息
	DeleteByGroupAndTime(groupID int64, beforeTime int64) (int64, error)

	// DeleteByTime 删除所有群在指定时间之前的消息
	DeleteByTime(beforeTime int64) (int64, error)

	// CreateDeletionLog 创建删除审计日志
	CreateDeletionLog(log *domain.InfoFlowMessageDeletionLog) error
}

// infoFlowRepositoryImpl 如流消息仓储实现
type infoFlowRepositoryImpl struct {
	ormer orm.Ormer
}

// NewInfoFlowRepository 创建如流消息仓储实例
func NewInfoFlowRepository(ormer orm.Ormer) InfoFlowRepository {
	return &infoFlowRepositoryImpl{
		ormer: ormer,
	}
}

// BatchCreate 批量创建消息
func (r *infoFlowRepositoryImpl) BatchCreate(messages []*domain.InfoFlowMessage) error {
	if len(messages) == 0 {
		return nil
	}

	// Beego ORM 批量插入
	_, err := r.ormer.InsertMulti(len(messages), messages)
	return err
}

// ListByConditions 根据条件查询消息列表
func (r *infoFlowRepositoryImpl) ListByConditions(
	groupID int64,
	startTime, endTime int64,
	contentType, userID string,
	offset, limit int,
) ([]*domain.InfoFlowMessage, int64, error) {
	qs := r.ormer.QueryTable(new(domain.InfoFlowMessage))

	// 必填条件：群ID
	qs = qs.Filter("group_id", groupID)

	// 可选条件：时间范围
	if startTime > 0 {
		qs = qs.Filter("time__gte", startTime)
	}
	if endTime > 0 {
		qs = qs.Filter("time__lte", endTime)
	}

	// 可选条件：消息类型
	if contentType != "" {
		qs = qs.Filter("content_type", contentType)
	}

	// 可选条件：用户ID
	if userID != "" {
		qs = qs.Filter("user_id", userID)
	}

	// 统计总数
	total, err := qs.Count()
	if err != nil {
		return nil, 0, err
	}

	// 分页 + 排序（按时间逆序）
	messages := make([]*domain.InfoFlowMessage, 0)
	_, err = qs.OrderBy("-time").Limit(limit, offset).All(&messages)
	if err != nil {
		return nil, 0, err
	}

	return messages, total, nil
}

// DeleteByGroupAndTime 删除指定群在指定时间之前的消息
func (r *infoFlowRepositoryImpl) DeleteByGroupAndTime(groupID int64, beforeTime int64) (int64, error) {
	qs := r.ormer.QueryTable(new(domain.InfoFlowMessage)).Filter("group_id", groupID)
	if beforeTime > 0 {
		qs = qs.Filter("time__lt", beforeTime)
	}
	count, err := qs.Delete()
	return count, err
}

// DeleteByTime 删除所有群在指定时间之前的消息
func (r *infoFlowRepositoryImpl) DeleteByTime(beforeTime int64) (int64, error) {
	qs := r.ormer.QueryTable(new(domain.InfoFlowMessage)).Filter("time__lt", beforeTime)
	count, err := qs.Delete()
	return count, err
}

// CreateDeletionLog 创建删除审计日志
func (r *infoFlowRepositoryImpl) CreateDeletionLog(log *domain.InfoFlowMessageDeletionLog) error {
	_, err := r.ormer.Insert(log)
	return err
}
