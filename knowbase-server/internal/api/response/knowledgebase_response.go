// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package response

import (
	"time"

	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/domain"
)

// KnowledgebaseResponse 知识本响应
type KnowledgebaseResponse struct {
	UUID           string    `json:"uuid"`
	Name           string    `json:"name"`
	Description    string    `json:"description"`
	Visibility     string    `json:"visibility"`
	Owner          string    `json:"owner"`
	Status         int       `json:"status"`
	MemberCount    int64     `json:"memberCount"`    // 成员数量
	TagCount       int64     `json:"tagCount"`       // 标签数量
	KnowledgeCount int64     `json:"knowledgeCount"` // 知识数量
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

// NewKnowledgebaseResponse 从领域模型创建响应
func NewKnowledgebaseResponse(kb *domain.Knowledgebase, memberCount, tagCount, knowledgeCount int64) *KnowledgebaseResponse {
	return &KnowledgebaseResponse{
		UUID:           kb.UUID,
		Name:           kb.Name,
		Description:    kb.Description,
		Visibility:     kb.Visibility,
		Owner:          kb.Owner,
		Status:         kb.Status,
		MemberCount:    memberCount,
		TagCount:       tagCount,
		KnowledgeCount: knowledgeCount,
		CreatedAt:      kb.CreatedAt,
		UpdatedAt:      kb.UpdatedAt,
	}
}

// KnowledgebaseListItemResponse 知识本列表项响应（含权限信息）
type KnowledgebaseListItemResponse struct {
	KnowledgebaseResponse
	Role string `json:"role"` // admin | editor | viewer | ""
}

// ListKnowledgebasesResponse 查询知识本列表响应
type ListKnowledgebasesResponse struct {
	List     []*KnowledgebaseListItemResponse `json:"list"`
	Total    int64                            `json:"total"`
	Page     int                              `json:"page"`
	PageSize int                              `json:"pageSize"`
}

// NewKnowledgebaseListItemResponse 从领域模型和角色创建列表项响应
func NewKnowledgebaseListItemResponse(kb *domain.Knowledgebase, role string, memberCount, tagCount, knowledgeCount int64) *KnowledgebaseListItemResponse {
	return &KnowledgebaseListItemResponse{
		KnowledgebaseResponse: KnowledgebaseResponse{
			UUID:           kb.UUID,
			Name:           kb.Name,
			Description:    kb.Description,
			Visibility:     kb.Visibility,
			Owner:          kb.Owner,
			Status:         kb.Status,
			MemberCount:    memberCount,
			TagCount:       tagCount,
			KnowledgeCount: knowledgeCount,
			CreatedAt:      kb.CreatedAt,
			UpdatedAt:      kb.UpdatedAt,
		},
		Role: role,
	}
}
