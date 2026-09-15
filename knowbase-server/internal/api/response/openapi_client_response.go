// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package response

import (
	"time"
)

// OpenAPIClientResponse OpenAPI客户端响应
type OpenAPIClientResponse struct {
	UUID         string     `json:"uuid"`
	PlatformName string     `json:"platformName"`
	ClientToken  string     `json:"clientToken"` // 创建时返回，其他场景不返回
	Description  string     `json:"description"`
	Owner        string     `json:"owner"`
	Status       int8       `json:"status"`
	IsAdmin      int8       `json:"isAdmin"`   // Admin权限标识: 0-否, 1-是
	RateLimit    int        `json:"rateLimit"` // QPM
	LastUsedAt   *time.Time `json:"lastUsedAt"`
	ExpiresAt    *time.Time `json:"expiresAt"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
}

// OpenAPIClientCreatedResponse 创建OpenAPI客户端响应
type OpenAPIClientCreatedResponse struct {
	UUID         string    `json:"uuid"`
	PlatformName string    `json:"platformName"`
	ClientToken  string    `json:"clientToken"` // 仅创建时返回Token
	Description  string    `json:"description"`
	Owner        string    `json:"owner"`
	Status       int8      `json:"status"`
	IsAdmin      int8      `json:"isAdmin"` // Admin权限标识: 0-否, 1-是
	RateLimit    int       `json:"rateLimit"`
	CreatedAt    time.Time `json:"createdAt"`
}

// ListOpenAPIClientsResponse OpenAPI客户端列表响应
type ListOpenAPIClientsResponse struct {
	Total int64                    `json:"total"`
	Items []*OpenAPIClientResponse `json:"items"`
}
