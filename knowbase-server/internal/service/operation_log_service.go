// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package service

import (
	"context"
	"encoding/json"

	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/api/request"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/api/response"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/repository"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/errors"
)

// OperationLogService 操作日志服务接口
type OperationLogService interface {
	// ListOperationLogs 查询操作日志列表
	ListOperationLogs(ctx context.Context, req *request.ListOperationLogRequest) (*response.ListOperationLogsResponse, error)
}

// operationLogServiceImpl 操作日志服务实现
type operationLogServiceImpl struct {
	logRepo repository.KnowledgebaseOperationLogRepository
}

// NewOperationLogService 创建操作日志服务实例
func NewOperationLogService(logRepo repository.KnowledgebaseOperationLogRepository) OperationLogService {
	return &operationLogServiceImpl{
		logRepo: logRepo,
	}
}

// ListOperationLogs 查询操作日志列表
func (s *operationLogServiceImpl) ListOperationLogs(ctx context.Context, req *request.ListOperationLogRequest) (*response.ListOperationLogsResponse, error) {
	// 至少需要一个筛选条件
	if req.KnowledgebaseUUID == "" && req.Operator == "" && req.Agent == "" &&
		req.ResourceType == "" && req.ResourceUUID == "" {
		return nil, errors.New(errors.ErrCodeInvalidParam, "至少需要提供一个查询条件")
	}

	// 构建过滤条件
	filter := &repository.OperationLogFilter{
		KnowledgebaseUUID: req.KnowledgebaseUUID,
		Operator:          req.Operator,
		Agent:             req.Agent,
		ResourceType:      req.ResourceType,
		ResourceUUID:      req.ResourceUUID,
		Page:              req.Page,
		PageSize:          req.PageSize,
	}

	// 查询日志列表
	logs, total, err := s.logRepo.List(filter)
	if err != nil {
		return nil, errors.Wrap(errors.ErrCodeInternalError, "查询操作日志失败", err)
	}

	// 转换为响应格式
	items := make([]*response.OperationLogResponse, 0, len(logs))
	for _, log := range logs {
		// 解析 Detail JSON
		var detail map[string]interface{}
		if log.Detail != "" {
			_ = json.Unmarshal([]byte(log.Detail), &detail)
		}

		items = append(items, &response.OperationLogResponse{
			KnowledgebaseUUID: log.KnowledgebaseUUID,
			Operator:          log.Operator,
			Agent:             log.Agent,
			Operation:         log.Operation,
			ResourceType:      log.ResourceType,
			ResourceUUID:      log.ResourceUUID,
			Detail:            detail,
			CreatedAt:         log.CreatedAt,
		})
	}

	// 处理分页参数
	page := req.Page
	pageSize := req.PageSize
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}

	return &response.ListOperationLogsResponse{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}
