// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package request

// CreateKnowledgebaseMember 创建知识本时附带的成员信息
type CreateKnowledgebaseMember struct {
	MemberType  string `json:"memberType" binding:"required,oneof=username email_group department chat_group"`
	MemberValue string `json:"memberValue" binding:"required,min=1,max=255"`
	Role        string `json:"role" binding:"required,oneof=admin editor viewer"`
}

// CreateKnowledgebaseRequest 创建知识本请求
type CreateKnowledgebaseRequest struct {
	Name        string                      `json:"name" binding:"required,min=1,max=128"`
	Description string                      `json:"description" binding:"max=512"`
	Visibility  string                      `json:"visibility" binding:"required,oneof=private public personal"`
	Members     []CreateKnowledgebaseMember `json:"members,omitempty" binding:"omitempty,max=20,dive"`
}

// ListKnowledgebasesRequest 查询知识本列表请求
type ListKnowledgebasesRequest struct {
	Keyword    string `form:"keyword"`                                                      // 模糊搜索关键词（搜索名称、描述、创建者）
	Visibility string `form:"visibility" binding:"omitempty,oneof=private public personal"` // 可见性过滤
	Owner      string `form:"owner"`                                                        // 创建者过滤
	Page       int    `form:"page" binding:"omitempty,min=1"`                               // 页码
	PageSize   int    `form:"pageSize" binding:"omitempty,min=1,max=100"`                   // 每页数量
}

// ListMyKnowledgebasesRequest 查询用户有权限的知识本列表请求
type ListMyKnowledgebasesRequest struct {
	Keyword    string `form:"keyword"`                                                      // 模糊搜索关键词（搜索名称、描述）
	Visibility string `form:"visibility" binding:"omitempty,oneof=private public personal"` // 可见性过滤
	Role       string `form:"role" binding:"omitempty,oneof=admin editor viewer"`            // 角色过滤（非必需）
	Page       int    `form:"page" binding:"omitempty,min=1"`                               // 页码
	PageSize   int    `form:"pageSize" binding:"omitempty,min=1,max=100"`                   // 每页数量
}

// UpdateKnowledgebaseRequest 更新知识本请求
type UpdateKnowledgebaseRequest struct {
	Name        string `json:"name" binding:"omitempty,min=1,max=128"`
	Description string `json:"description" binding:"omitempty,max=512"`
	Visibility  string `json:"visibility" binding:"omitempty,oneof=private public personal"`
}
