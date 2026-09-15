// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package request

// CreateDirectoryRequest 创建目录请求
type CreateDirectoryRequest struct {
	Name       string  `json:"name" binding:"required,min=1,max=255"`
	ParentUUID *string `json:"parentUuid"` // 可选，nil 表示创建根目录
}

// UpdateDirectoryRequest 更新目录请求
type UpdateDirectoryRequest struct {
	Name string `json:"name" binding:"required,min=1,max=255"`
}

// DeleteDirectoryRequest 删除目录请求（预留，暂无请求体字段）
type DeleteDirectoryRequest struct {
}

// MoveDirectoryRequest 移动目录请求
type MoveDirectoryRequest struct {
	ParentUUID *string `json:"parentUuid"` // 目标父目录UUID，nil 表示移动为根目录
}
