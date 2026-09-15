// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package request

// CreateOpenAPIClientRequest 创建OpenAPI客户端请求
type CreateOpenAPIClientRequest struct {
	PlatformName string `json:"platformName" binding:"required,min=1,max=128"` // 平台名称
	Description  string `json:"description" binding:"max=512"`                 // 平台描述
	Applicant    string `json:"applicant" binding:"required,min=1,max=100"`    // 申请人
	QPM          int    `json:"qpm" binding:"required,min=1,max=100000"`       // 每分钟请求限制
	IsAdmin      *int8  `json:"isAdmin" binding:"omitempty,oneof=0 1"`         // Admin权限标识: 0-否, 1-是
}

// UpdateOpenAPIClientRequest 更新OpenAPI客户端请求
type UpdateOpenAPIClientRequest struct {
	PlatformName string `json:"platformName" binding:"omitempty,min=1,max=128"` // 平台名称
	Description  string `json:"description" binding:"omitempty,max=512"`        // 平台描述
	QPM          int    `json:"qpm" binding:"omitempty,min=1,max=100000"`       // 每分钟请求限制
	Status       *int8  `json:"status" binding:"omitempty,oneof=0 1"`           // 状态: 0-禁用, 1-启用
	IsAdmin      *int8  `json:"isAdmin" binding:"omitempty,oneof=0 1"`          // Admin权限标识: 0-否, 1-是
}
