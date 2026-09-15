// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package service

import (
	"encoding/json"
	"fmt"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/log"
	"strconv"
	"time"

	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/api/request"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/api/response"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/domain"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/repository"
)

// InfoFlowOneOnOneService 如流单聊消息服务接口
type InfoFlowOneOnOneService interface {
	// ProcessOneOnOneMessage 处理如流单聊消息回调
	ProcessOneOnOneMessage(req *request.InfoFlowOneOnOneCallbackRequest, body string) error

	// ListOneOnOneMessages 查询单聊消息列表
	ListOneOnOneMessages(req *request.ListOneOnOneMessagesRequest) (*response.ListOneOnOneMessagesResponse, error)
}

// infoFlowOneOnOneServiceImpl 如流单聊消息服务实现
type infoFlowOneOnOneServiceImpl struct {
	repo repository.InfoFlowOneOnOneRepository
}

// NewInfoFlowOneOnOneService 创建如流单聊消息服务实例
func NewInfoFlowOneOnOneService(repo repository.InfoFlowOneOnOneRepository) InfoFlowOneOnOneService {
	return &infoFlowOneOnOneServiceImpl{
		repo: repo,
	}
}

// ProcessOneOnOneMessage 处理如流单聊消息回调
func (s *infoFlowOneOnOneServiceImpl) ProcessOneOnOneMessage(req *request.InfoFlowOneOnOneCallbackRequest, body string) error {
	// 1. 参数校验
	if req.Body.Data.MsgID == "" {
		return fmt.Errorf("消息ID不能为空")
	}
	if req.Body.Data.MsgType == "" {
		return fmt.Errorf("消息类型不能为空")
	}
	if req.Body.UserID == "" {
		return fmt.Errorf("用户ID不能为空")
	}
	if req.Body.Data.CreateTime == "" {
		return fmt.Errorf("消息创建时间不能为空")
	}

	// debug用
	if req.Body.UserID == "hekai09" {
		log.Logger.Infof("ProcessOneOnOneMessage, body: %s", body)
	}

	// 2. 时间戳转换（字符串 -> int64）
	createTime, err := strconv.ParseInt(req.Body.Data.CreateTime, 10, 64)
	if err != nil {
		return fmt.Errorf("消息创建时间格式错误: %w", err)
	}

	// 3. 序列化完整的 data 字段到 raw_data
	rawDataBytes, err := json.Marshal(req.Body.Data)
	if err != nil {
		return fmt.Errorf("序列化消息数据失败: %w", err)
	}

	// 4. 构建消息对象
	msg := &domain.InfoFlowOneOnOneMessage{
		AIRobotID:  req.Body.AIRobotID,
		UserID:     req.Body.UserID,
		FromUserID: req.Body.Data.FromUserID,
		MsgID:      req.Body.Data.MsgID,
		Time:       createTime,
		MsgType:    req.Body.Data.MsgType,
		Content:    req.Body.Data.Content,
		PicURL:     req.Body.Data.PicURL,
		RawData:    string(rawDataBytes),
	}

	// 5. 保存到数据库
	return s.repo.Create(msg)
}

// ListOneOnOneMessages 查询单聊消息列表
func (s *infoFlowOneOnOneServiceImpl) ListOneOnOneMessages(req *request.ListOneOnOneMessagesRequest) (*response.ListOneOnOneMessagesResponse, error) {
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

	if req.StartTime > 0 {
		req.StartTime = req.StartTime / 1000
	}
	if req.EndTime > 0 {
		req.EndTime = req.EndTime / 1000
	}

	// 2. 处理时间范围，确保不超过90天
	const maxQuerySeconds = MaxQueryDays * 24 * 60 * 60 // 90天的秒数

	// 2.1 如果未提供任何时间，默认查询最近90天
	if req.StartTime == 0 && req.EndTime == 0 {
		// 当前时间戳（秒）
		nowSeconds := getCurrentTimestampSeconds()
		req.EndTime = nowSeconds
		req.StartTime = nowSeconds - maxQuerySeconds
	} else if req.StartTime == 0 {
		// 2.2 只提供了结束时间，计算开始时间
		req.StartTime = req.EndTime - maxQuerySeconds
		if req.StartTime < 0 {
			req.StartTime = 0
		}
	} else if req.EndTime == 0 {
		// 2.3 只提供了开始时间，计算结束时间
		req.EndTime = req.StartTime + maxQuerySeconds
		nowSeconds := getCurrentTimestampSeconds()
		if req.EndTime > nowSeconds {
			req.EndTime = nowSeconds
		}
	} else {
		// 2.4 同时提供了开始和结束时间，校验范围
		if req.StartTime > req.EndTime {
			return nil, fmt.Errorf("开始时间不能大于结束时间")
		}
		if req.EndTime-req.StartTime > maxQuerySeconds {
			return nil, fmt.Errorf("查询时间范围不能超过%d天", MaxQueryDays)
		}
	}

	// 3. 计算偏移量
	offset := (req.Page - 1) * req.PageSize

	// 4. 调用 repository 查询
	messages, total, err := s.repo.ListByConditions(
		req.UserID,
		req.StartTime,
		req.EndTime,
		req.MsgType,
		offset,
		req.PageSize,
	)
	if err != nil {
		return nil, err
	}

	// 5. 组装响应
	return &response.ListOneOnOneMessagesResponse{
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
		Messages: messages,
	}, nil
}

// getCurrentTimestampSeconds 获取当前时间戳（秒）
func getCurrentTimestampSeconds() int64 {
	return time.Now().Unix()
}
