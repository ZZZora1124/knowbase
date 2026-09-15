// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/api/request"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/api/response"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/domain"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/infrastructure/cache"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/infrastructure/database"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/repository"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/errors"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/log"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/types/constants"
)

// OpenAPIClientService OpenAPI客户端管理服务接口
type OpenAPIClientService interface {
	// CreateClient 创建客户端
	CreateClient(ctx context.Context, req *request.CreateOpenAPIClientRequest) (*response.OpenAPIClientCreatedResponse, error)

	// GetClient 查询单个客户端（按 token）
	GetClient(ctx context.Context, token string) (*response.OpenAPIClientResponse, error)

	// ListClients 查询客户端列表
	ListClients(ctx context.Context, page, pageSize int) (*response.ListOpenAPIClientsResponse, error)

	// UpdateClient 更新客户端（按 token）
	UpdateClient(ctx context.Context, token string, req *request.UpdateOpenAPIClientRequest) (*response.OpenAPIClientResponse, error)

	// DeleteClient 删除客户端（按 token）
	DeleteClient(ctx context.Context, token string) error

	// DisableClient 禁用客户端，并立即淘汰缓存（按 token）
	DisableClient(ctx context.Context, token string) error
}

// openAPIClientServiceImpl OpenAPI客户端管理服务实现
type openAPIClientServiceImpl struct {
	repo      repository.OpenAPIClientRepository
	txManager *database.TransactionManager
}

// NewOpenAPIClientService 创建OpenAPI客户端管理服务实例
func NewOpenAPIClientService(
	repo repository.OpenAPIClientRepository,
	txManager *database.TransactionManager,
) OpenAPIClientService {
	return &openAPIClientServiceImpl{
		repo:      repo,
		txManager: txManager,
	}
}

// CreateClient 创建客户端
func (s *openAPIClientServiceImpl) CreateClient(ctx context.Context, req *request.CreateOpenAPIClientRequest) (*response.OpenAPIClientCreatedResponse, error) {
	// 1. 验证平台名称
	if !domain.ValidatePlatformName(req.PlatformName) {
		return nil, errors.New(errors.ErrCodeInvalidParam, "平台名称格式无效")
	}

	// 2. 检查平台名称是否已存在
	existingClient, err := s.repo.FindByName(nil, req.PlatformName)
	if err != nil {
		return nil, errors.Wrap(errors.ErrCodeInternalError, "查询客户端失败", err)
	}
	if existingClient != nil {
		return nil, errors.New(errors.ErrCodeInvalidParam, "平台名称已存在")
	}

	// 3. 创建客户端对象（使用UUID生成Token）
	client := &domain.OpenAPIClient{
		UUID:        uuid.New().String(),
		ClientName:  req.PlatformName,
		ClientToken: uuid.New().String(),
		Description: req.Description,
		Owner:       req.Applicant,
		Status:      domain.OpenAPIClientStatusEnabled,
		RateLimit:   req.QPM,
	}

	// 设置IsAdmin（默认为0）
	if req.IsAdmin != nil {
		client.IsAdmin = *req.IsAdmin
	}

	// 4. 在事务中创建客户端
	err = s.txManager.ExecuteInTransaction(func(txCtx *database.TxContext) error {
		return s.repo.Create(txCtx, client)
	})
	if err != nil {
		return nil, errors.Wrap(errors.ErrCodeInternalError, "创建客户端失败", err)
	}

	// 5. 返回创建响应（包含Token，仅此一次）
	return &response.OpenAPIClientCreatedResponse{
		UUID:         client.UUID,
		PlatformName: client.ClientName,
		ClientToken:  client.ClientToken,
		Description:  client.Description,
		Owner:        client.Owner,
		Status:       client.Status,
		IsAdmin:      client.IsAdmin,
		RateLimit:    client.RateLimit,
		CreatedAt:    client.CreatedAt,
	}, nil
}

// GetClient 查询单个客户端（按 token）
func (s *openAPIClientServiceImpl) GetClient(ctx context.Context, token string) (*response.OpenAPIClientResponse, error) {
	// 查询客户端
	client, err := s.repo.FindByToken(token)
	if err != nil {
		return nil, errors.Wrap(errors.ErrCodeInternalError, "查询客户端失败", err)
	}
	if client == nil {
		return nil, errors.New(errors.ErrCodeNotFound, "客户端不存在")
	}

	return &response.OpenAPIClientResponse{
		UUID:         client.UUID,
		PlatformName: client.ClientName,
		Description:  client.Description,
		Owner:        client.Owner,
		Status:       client.Status,
		IsAdmin:      client.IsAdmin,
		RateLimit:    client.RateLimit,
		LastUsedAt:   client.LastUsedAt,
		ExpiresAt:    client.ExpiresAt,
		CreatedAt:    client.CreatedAt,
		UpdatedAt:    client.UpdatedAt,
	}, nil
}

// ListClients 查询客户端列表
func (s *openAPIClientServiceImpl) ListClients(ctx context.Context, page, pageSize int) (*response.ListOpenAPIClientsResponse, error) {
	// 参数校验和默认值处理
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	offset := (page - 1) * pageSize

	// 查询客户端列表
	clients, total, err := s.repo.List(nil, offset, pageSize)
	if err != nil {
		return nil, errors.Wrap(errors.ErrCodeInternalError, "查询客户端列表失败", err)
	}

	// 转换为响应格式
	items := make([]*response.OpenAPIClientResponse, 0, len(clients))
	for _, client := range clients {
		items = append(items, &response.OpenAPIClientResponse{
			UUID:         client.UUID,
			PlatformName: client.ClientName,
			Description:  client.Description,
			Owner:        client.Owner,
			Status:       client.Status,
			IsAdmin:      client.IsAdmin,
			RateLimit:    client.RateLimit,
			LastUsedAt:   client.LastUsedAt,
			ExpiresAt:    client.ExpiresAt,
			CreatedAt:    client.CreatedAt,
			UpdatedAt:    client.UpdatedAt,
		})
	}

	return &response.ListOpenAPIClientsResponse{
		Total: total,
		Items: items,
	}, nil
}

// UpdateClient 更新客户端（按 token）
func (s *openAPIClientServiceImpl) UpdateClient(ctx context.Context, clientToken string, req *request.UpdateOpenAPIClientRequest) (*response.OpenAPIClientResponse, error) {
	// 1. 查询客户端
	client, err := s.repo.FindByToken(clientToken)
	if err != nil {
		return nil, errors.Wrap(errors.ErrCodeInternalError, "查询客户端失败", err)
	}
	if client == nil {
		return nil, errors.New(errors.ErrCodeNotFound, "客户端不存在")
	}

	// 2. 记录变更前的值
	hasChanges := false

	// 3. 更新平台名称
	if req.PlatformName != "" && req.PlatformName != client.ClientName {
		if !domain.ValidatePlatformName(req.PlatformName) {
			return nil, errors.New(errors.ErrCodeInvalidParam, "平台名称格式无效")
		}
		// 检查新名称是否已存在
		existingClient, err := s.repo.FindByName(nil, req.PlatformName)
		if err != nil {
			return nil, errors.Wrap(errors.ErrCodeInternalError, "查询客户端失败", err)
		}
		if existingClient != nil && existingClient.UUID != client.UUID {
			return nil, errors.New(errors.ErrCodeInvalidParam, "平台名称已存在")
		}
		client.ClientName = req.PlatformName
		hasChanges = true
	}

	// 4. 更新描述
	if req.Description != "" && req.Description != client.Description {
		client.Description = req.Description
		hasChanges = true
	}

	// 5. 更新QPM
	if req.QPM > 0 && req.QPM != client.RateLimit {
		client.RateLimit = req.QPM
		hasChanges = true
	}

	// 6. 更新状态
	if req.Status != nil && *req.Status != client.Status {
		if *req.Status != domain.OpenAPIClientStatusDisabled && *req.Status != domain.OpenAPIClientStatusEnabled {
			return nil, errors.New(errors.ErrCodeInvalidParam, "无效的状态值")
		}
		client.Status = *req.Status
		hasChanges = true
	}

	// 7. 更新IsAdmin
	if req.IsAdmin != nil && *req.IsAdmin != client.IsAdmin {
		if *req.IsAdmin != 0 && *req.IsAdmin != 1 {
			return nil, errors.New(errors.ErrCodeInvalidParam, "无效的IsAdmin值")
		}
		client.IsAdmin = *req.IsAdmin
		hasChanges = true
	}

	// 如果没有变更，直接返回
	if !hasChanges {
		return s.GetClient(ctx, clientToken)
	}

	// 8. 在事务中更新客户端
	err = s.txManager.ExecuteInTransaction(func(txCtx *database.TxContext) error {
		return s.repo.Update(txCtx, client)
	})
	if err != nil {
		return nil, errors.Wrap(errors.ErrCodeInternalError, "更新客户端失败", err)
	}

	// 9. 主动淘汰缓存（有变更时清除，避免缓存与DB不一致）
	evictOpenAPIClientCache(client.ClientToken)

	// 10. 返回更新后的客户端信息
	return &response.OpenAPIClientResponse{
		UUID:         client.UUID,
		PlatformName: client.ClientName,
		Description:  client.Description,
		Owner:        client.Owner,
		Status:       client.Status,
		IsAdmin:      client.IsAdmin,
		RateLimit:    client.RateLimit,
		LastUsedAt:   client.LastUsedAt,
		ExpiresAt:    client.ExpiresAt,
		CreatedAt:    client.CreatedAt,
		UpdatedAt:    client.UpdatedAt,
	}, nil
}

// DeleteClient 删除客户端（按 token）
func (s *openAPIClientServiceImpl) DeleteClient(ctx context.Context, clientToken string) error {
	// 1. 查询客户端
	client, err := s.repo.FindByToken(clientToken)
	if err != nil {
		return errors.Wrap(errors.ErrCodeInternalError, "查询客户端失败", err)
	}
	if client == nil {
		return errors.New(errors.ErrCodeNotFound, "客户端不存在")
	}

	// 2. 在事务中软删除客户端
	err = s.txManager.ExecuteInTransaction(func(txCtx *database.TxContext) error {
		return s.repo.SoftDelete(txCtx, client.UUID)
	})
	if err != nil {
		return errors.Wrap(errors.ErrCodeInternalError, "删除客户端失败", err)
	}

	// 3. 主动淘汰缓存
	evictOpenAPIClientCache(client.ClientToken)

	return nil
}

// DisableClient 禁用客户端，并立即淘汰缓存（按 token）
func (s *openAPIClientServiceImpl) DisableClient(ctx context.Context, clientToken string) error {
	// 1. 查询客户端
	client, err := s.repo.FindByToken(clientToken)
	if err != nil {
		return errors.Wrap(errors.ErrCodeInternalError, "查询客户端失败", err)
	}
	if client == nil {
		return errors.New(errors.ErrCodeNotFound, "客户端不存在")
	}

	// 幂等：已禁用则直接返回
	if client.Status == domain.OpenAPIClientStatusDisabled {
		return nil
	}

	// 2. 在事务中更新状态为禁用
	client.Status = domain.OpenAPIClientStatusDisabled
	err = s.txManager.ExecuteInTransaction(func(txCtx *database.TxContext) error {
		return s.repo.Update(txCtx, client)
	})
	if err != nil {
		return errors.Wrap(errors.ErrCodeInternalError, "禁用客户端失败", err)
	}

	// 3. 立即淘汰缓存，使禁用立即生效
	evictOpenAPIClientCache(client.ClientToken)

	return nil
}

// evictOpenAPIClientCache 主动删除 OpenAPI 客户端缓存
func evictOpenAPIClientCache(clientToken string) {
	cacher := cache.GetCache()
	if cacher == nil {
		return
	}
	cacheKey := fmt.Sprintf("%s%s", constants.CacheKeyOpenAPIClient, clientToken)
	if err := cacher.Delete(context.Background(), cacheKey); err != nil {
		log.Logger.Warnf("Failed to evict OpenAPI client cache: %v, token=%s...", err, clientToken[:min(8, len(clientToken))])
	}
}

// min 返回两个整数中的较小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
