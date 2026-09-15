// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package request

// CreateTagRequest 创建标签请求
type CreateTagRequest struct {
	Name        string `json:"name" binding:"required,min=1,max=100"`
	Description string `json:"description" binding:"max=500"`
}

// UpdateTagRequest 更新标签请求
type UpdateTagRequest struct {
	Name        string `json:"name" binding:"omitempty,min=1,max=100"`
	Description string `json:"description" binding:"omitempty,max=500"`
}

// BindTagsRequest 绑定标签请求(知识绑定标签时使用)
type BindTagsRequest struct {
	TagUUIDs []string `json:"tagUuids" binding:"required,dive,min=1"`
}

