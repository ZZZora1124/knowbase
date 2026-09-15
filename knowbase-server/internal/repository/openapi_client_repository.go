// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package repository

import (
	"time"

	"github.com/beego/beego/v2/client/orm"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/domain"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/infrastructure/database"
)

// OpenAPIClientRepository OpenAPI客户端仓储接口
type OpenAPIClientRepository interface {
	// FindByToken 根据Token查询客户端
	FindByToken(token string) (*domain.OpenAPIClient, error)

	// UpdateLastUsed 更新最后使用时间
	UpdateLastUsed(uuid string) error

	// Create 创建客户端
	Create(ctx *database.TxContext, client *domain.OpenAPIClient) error

	// FindByUUID 根据UUID查询客户端
	FindByUUID(ctx *database.TxContext, uuid string) (*domain.OpenAPIClient, error)

	// FindByName 根据名称查询客户端
	FindByName(ctx *database.TxContext, name string) (*domain.OpenAPIClient, error)

	// List 查询客户端列表
	List(ctx *database.TxContext, offset, limit int) ([]*domain.OpenAPIClient, int64, error)

	// Count 统计客户端总数
	Count(ctx *database.TxContext) (int64, error)

	// Update 更新客户端
	Update(ctx *database.TxContext, client *domain.OpenAPIClient) error

	// SoftDelete 软删除客户端
	SoftDelete(ctx *database.TxContext, uuid string) error
}

// openAPIClientRepositoryImpl OpenAPI客户端仓储实现
type openAPIClientRepositoryImpl struct {
	ormer orm.Ormer
}

// NewOpenAPIClientRepository 创建OpenAPI客户端仓储实例
func NewOpenAPIClientRepository(ormer orm.Ormer) OpenAPIClientRepository {
	return &openAPIClientRepositoryImpl{
		ormer: ormer,
	}
}

// FindByToken 根据Token查询客户端
func (r *openAPIClientRepositoryImpl) FindByToken(token string) (*domain.OpenAPIClient, error) {
	client := &domain.OpenAPIClient{}
	err := r.ormer.QueryTable(new(domain.OpenAPIClient)).
		Filter("client_token", token).
		Filter("deleted_at__isnull", true).
		One(client)
	if err == orm.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return client, nil
}

// UpdateLastUsed 更新最后使用时间
func (r *openAPIClientRepositoryImpl) UpdateLastUsed(uuid string) error {
	_, err := r.ormer.QueryTable(new(domain.OpenAPIClient)).
		Filter("uuid", uuid).
		Update(orm.Params{
			"last_used_at": time.Now(),
		})
	return err
}

// Create 创建客户端
func (r *openAPIClientRepositoryImpl) Create(ctx *database.TxContext, client *domain.OpenAPIClient) error {
	o := r.getOrmer(ctx)
	_, err := o.Insert(client)
	return err
}

// FindByUUID 根据UUID查询客户端
func (r *openAPIClientRepositoryImpl) FindByUUID(ctx *database.TxContext, uuid string) (*domain.OpenAPIClient, error) {
	o := r.getOrmer(ctx)
	client := &domain.OpenAPIClient{}
	err := o.QueryTable(new(domain.OpenAPIClient)).
		Filter("uuid", uuid).
		Filter("deleted_at__isnull", true).
		One(client)
	if err == orm.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return client, nil
}

// FindByName 根据名称查询客户端
func (r *openAPIClientRepositoryImpl) FindByName(ctx *database.TxContext, name string) (*domain.OpenAPIClient, error) {
	o := r.getOrmer(ctx)
	client := &domain.OpenAPIClient{}
	err := o.QueryTable(new(domain.OpenAPIClient)).
		Filter("client_name", name).
		Filter("deleted_at__isnull", true).
		One(client)
	if err == orm.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return client, nil
}

// List 查询客户端列表
func (r *openAPIClientRepositoryImpl) List(ctx *database.TxContext, offset, limit int) ([]*domain.OpenAPIClient, int64, error) {
	o := r.getOrmer(ctx)
	var clients []*domain.OpenAPIClient
	_, err := o.QueryTable(new(domain.OpenAPIClient)).
		Filter("deleted_at__isnull", true).
		OrderBy("-created_at").
		Offset(offset).
		Limit(limit).
		All(&clients)
	if err != nil {
		return nil, 0, err
	}

	total, err := r.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	return clients, total, nil
}

// Count 统计客户端总数
func (r *openAPIClientRepositoryImpl) Count(ctx *database.TxContext) (int64, error) {
	o := r.getOrmer(ctx)
	return o.QueryTable(new(domain.OpenAPIClient)).
		Filter("deleted_at__isnull", true).
		Count()
}

// Update 更新客户端
func (r *openAPIClientRepositoryImpl) Update(ctx *database.TxContext, client *domain.OpenAPIClient) error {
	o := r.getOrmer(ctx)
	_, err := o.Update(client)
	return err
}

// SoftDelete 软删除客户端
func (r *openAPIClientRepositoryImpl) SoftDelete(ctx *database.TxContext, uuid string) error {
	o := r.getOrmer(ctx)
	now := time.Now()
	_, err := o.QueryTable(new(domain.OpenAPIClient)).
		Filter("uuid", uuid).
		Update(orm.Params{
			"deleted_at": &now,
			"status":     0,
		})
	return err
}

// getOrmer 获取orm实例，支持事务
func (r *openAPIClientRepositoryImpl) getOrmer(ctx *database.TxContext) orm.QueryExecutor {
	if ctx != nil && ctx.Ormer() != nil {
		return ctx.Ormer()
	}
	return r.ormer
}
