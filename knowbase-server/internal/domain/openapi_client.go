// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package domain

import (
	"time"

	"github.com/google/uuid"
)

// OpenAPIClient OpenAPI客户端模型
type OpenAPIClient struct {
	ID          int64      `orm:"column(id);pk;auto" json:"-"`
	UUID        string     `orm:"column(uuid);size(36);unique" json:"uuid"`
	ClientName  string     `orm:"column(client_name);size(128)" json:"clientName"`
	ClientToken string     `orm:"column(client_token);size(256);unique" json:"-"`
	Description string     `orm:"column(description);size(512)" json:"description"`
	Owner       string     `orm:"column(owner);size(100)" json:"owner"`
	Status      int8       `orm:"column(status);default(1)" json:"status"`
	IsAdmin     int8       `orm:"column(is_admin);default(0)" json:"isAdmin"`
	RateLimit   int        `orm:"column(rate_limit);default(1000)" json:"rateLimit"`
	LastUsedAt  *time.Time `orm:"column(last_used_at);null" json:"lastUsedAt"`
	ExpiresAt   *time.Time `orm:"column(expires_at);null" json:"expiresAt"`
	CreatedAt   time.Time  `orm:"column(created_at);auto_now_add;type(datetime)" json:"createdAt"`
	UpdatedAt   time.Time  `orm:"column(updated_at);auto_now;type(datetime)" json:"updatedAt"`
	DeletedAt   *time.Time `orm:"column(deleted_at);null" json:"-"`
}

// TableName 指定表名
func (c *OpenAPIClient) TableName() string {
	return "openapi_client"
}

// IsActive 检查客户端是否有效
func (c *OpenAPIClient) IsActive() bool {
	// 检查状态是否启用
	if c.Status != 1 {
		return false
	}
	// 检查是否已过期
	if c.ExpiresAt != nil && c.ExpiresAt.Before(time.Now()) {
		return false
	}
	// 检查是否已删除
	if c.DeletedAt != nil {
		return false
	}
	return true
}

// OpenAPIClientStatus 客户端状态常量
const (
	OpenAPIClientStatusDisabled = 0 // 禁用
	OpenAPIClientStatusEnabled  = 1 // 启用
)

// NewOpenAPIClient 创建新的OpenAPI客户端
func NewOpenAPIClient(platformName, description, owner string, rateLimit int) *OpenAPIClient {
	return &OpenAPIClient{
		UUID:        uuid.New().String(),
		ClientName:  platformName,
		ClientToken: uuid.New().String(),
		Description: description,
		Owner:       owner,
		Status:      OpenAPIClientStatusEnabled,
		RateLimit:   rateLimit,
	}
}

// ValidatePlatformName 验证平台名称
func ValidatePlatformName(name string) bool {
	if len(name) == 0 || len(name) > 128 {
		return false
	}
	return true
}
