// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package response

import (
	"time"

	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/domain"
)

// DirectoryResponse 目录响应（扁平列表用，不含 children）
type DirectoryResponse struct {
	UUID              string    `json:"uuid"`
	KnowledgebaseUUID string    `json:"knowledgebaseUuid"`
	ParentUUID        *string   `json:"parentUuid"`
	Name              string    `json:"name"`
	KnowledgeCount    int64     `json:"knowledgeCount"`
	CreatedAt         time.Time `json:"createdAt"`
	UpdatedAt         time.Time `json:"updatedAt"`
}

// DirectoryTreeNode 目录树节点（树形接口用，含 children）
type DirectoryTreeNode struct {
	UUID              string               `json:"uuid"`
	KnowledgebaseUUID string               `json:"knowledgebaseUuid"`
	ParentUUID        *string              `json:"parentUuid"`
	Name              string               `json:"name"`
	KnowledgeCount    int64                `json:"knowledgeCount"`
	Children          []*DirectoryTreeNode `json:"children"`
	CreatedAt         time.Time            `json:"createdAt"`
	UpdatedAt         time.Time            `json:"updatedAt"`
}

// ListDirectoriesResponse 目录列表响应
type ListDirectoriesResponse struct {
	Items []*DirectoryResponse `json:"items"`
	Total int64                `json:"total"`
}

// DirectoryTreeResponse 目录树响应
type DirectoryTreeResponse struct {
	Items []*DirectoryTreeNode `json:"items"`
	Total int64                `json:"total"`
}

// NewDirectoryResponse 从领域模型创建扁平响应
func NewDirectoryResponse(dir *domain.KnowledgebaseDirectory, knowledgeCount int64) *DirectoryResponse {
	if dir == nil {
		return nil
	}
	return &DirectoryResponse{
		UUID:              dir.UUID,
		KnowledgebaseUUID: dir.KnowledgebaseUUID,
		ParentUUID:        dir.ParentUUID,
		Name:              dir.Name,
		KnowledgeCount:    knowledgeCount,
		CreatedAt:         dir.CreatedAt,
		UpdatedAt:         dir.UpdatedAt,
	}
}

// NewDirectoryTreeNode 从领域模型创建树节点（children 初始化为空切片）
func NewDirectoryTreeNode(dir *domain.KnowledgebaseDirectory, knowledgeCount int64) *DirectoryTreeNode {
	if dir == nil {
		return nil
	}
	return &DirectoryTreeNode{
		UUID:              dir.UUID,
		KnowledgebaseUUID: dir.KnowledgebaseUUID,
		ParentUUID:        dir.ParentUUID,
		Name:              dir.Name,
		KnowledgeCount:    knowledgeCount,
		Children:          []*DirectoryTreeNode{},
		CreatedAt:         dir.CreatedAt,
		UpdatedAt:         dir.UpdatedAt,
	}
}
