// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package notebook

import "encoding/json"

// APIResponse 通用API响应
type APIResponse struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

// Knowledgebase 知识方舟实体
type Knowledgebase struct {
	UUID           string `json:"uuid"`
	Name           string `json:"name"`
	Description    string `json:"description"`
	Visibility     string `json:"visibility"`
	Owner          string `json:"owner"`
	Status         int    `json:"status"`
	MemberCount    int64  `json:"memberCount"`
	TagCount       int64  `json:"tagCount"`
	KnowledgeCount int64  `json:"knowledgeCount"`
	Role           string `json:"role,omitempty"`
	CreatedAt      string `json:"createdAt"`
	UpdatedAt      string `json:"updatedAt"`
}

// CreateKnowledgebaseRequest 创建知识方舟请求
type CreateKnowledgebaseRequest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Visibility  string `json:"visibility"`
}

// UpdateKnowledgebaseRequest 更新知识方舟请求
type UpdateKnowledgebaseRequest struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	Visibility  string `json:"visibility,omitempty"`
}

// KnowledgebaseListResponse 知识方舟列表响应
type KnowledgebaseListResponse struct {
	List     []Knowledgebase `json:"list"`
	Total    int64           `json:"total"`
	Page     int             `json:"page"`
	PageSize int             `json:"pageSize"`
}

// Member 成员实体
type Member struct {
	UUID              string `json:"uuid"`
	KnowledgebaseUUID string `json:"knowledgebaseUuid"`
	MemberType        string `json:"memberType"`
	MemberValue       string `json:"memberValue"`
	Role              string `json:"role"`
	CreatedAt         string `json:"createdAt"`
	UpdatedAt         string `json:"updatedAt"`
}

// AddMemberRequest 添加成员请求
type AddMemberRequest struct {
	MemberType  string `json:"memberType"`
	MemberValue string `json:"memberValue"`
	Role        string `json:"role"`
}

// UpdateMemberRoleRequest 更新成员角色请求
type UpdateMemberRoleRequest struct {
	Role string `json:"role"`
}

// MemberListResponse 成员列表响应
type MemberListResponse struct {
	Items    []Member `json:"items"`
	Total    int64    `json:"total"`
	Page     int      `json:"page"`
	PageSize int      `json:"pageSize"`
}

// Tag 标签实体
type Tag struct {
	UUID              string `json:"uuid"`
	KnowledgebaseUUID string `json:"knowledgebaseUuid"`
	Name              string `json:"name"`
	Description       string `json:"description"`
	CreatedBy         string `json:"createdBy"`
	KnowledgeCount    int64  `json:"knowledgeCount"`
	CreatedAt         string `json:"createdAt"`
	UpdatedAt         string `json:"updatedAt"`
}

// CreateTagRequest 创建标签请求
type CreateTagRequest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// UpdateTagRequest 更新标签请求
type UpdateTagRequest struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
}

// TagListResponse 标签列表响应
type TagListResponse struct {
	Items []Tag `json:"items"`
	Total int64 `json:"total"`
}

// Directory 目录实体
type Directory struct {
	UUID              string `json:"uuid"`
	KnowledgebaseUUID string `json:"knowledgebaseUuid"`
	ParentUUID        string `json:"parentUuid"`
	Name              string `json:"name"`
	KnowledgeCount    int64  `json:"knowledgeCount"`
	CreatedAt         string `json:"createdAt"`
	UpdatedAt         string `json:"updatedAt"`
}

// CreateDirectoryRequest 创建目录请求
type CreateDirectoryRequest struct {
	Name       string  `json:"name"`
	ParentUUID *string `json:"parentUuid,omitempty"`
}

// UpdateDirectoryRequest 更新目录请求
type UpdateDirectoryRequest struct {
	Name string `json:"name"`
}

// DirectoryListResponse 目录列表响应
type DirectoryListResponse struct {
	Items []Directory `json:"items"`
	Total int64       `json:"total"`
}

// DirectoryTreeNode 目录树节点
type DirectoryTreeNode struct {
	UUID              string              `json:"uuid"`
	KnowledgebaseUUID string              `json:"knowledgebaseUuid"`
	ParentUUID        string              `json:"parentUuid"`
	Name              string              `json:"name"`
	KnowledgeCount    int64               `json:"knowledgeCount"`
	Children          []DirectoryTreeNode `json:"children"`
	CreatedAt         string              `json:"createdAt"`
	UpdatedAt         string              `json:"updatedAt"`
}

// DirectoryTreeResponse 目录树响应
type DirectoryTreeResponse struct {
	Items []DirectoryTreeNode `json:"items"`
	Total int64               `json:"total"`
}

// MoveDirectoryRequest 移动目录请求
type MoveDirectoryRequest struct {
	ParentUUID *string `json:"parentUuid"`
}

// Knowledge 知识实体
type Knowledge struct {
	UUID              string                 `json:"uuid"`
	KnowledgebaseUUID string                 `json:"knowledgebaseUuid"`
	DirectoryUUID     string                 `json:"directoryUuid"`
	CreatorUsername   string                 `json:"creatorUsername"`
	Title             string                 `json:"title"`
	KnowledgeType     string                 `json:"knowledgeType"`
	KnowledgeSource   string                 `json:"knowledgeSource"`
	Content           string                 `json:"content"`
	FilePath          string                 `json:"filePath"`
	FileSize          int64                  `json:"fileSize"`
	FileFormat        string                 `json:"fileFormat"`
	SourceMethod      string                 `json:"sourceMethod"`
	Metadata          map[string]interface{} `json:"metadata"`
	Tags              []TagBrief             `json:"tags"`
	CreatedAt         string                 `json:"createdAt"`
	UpdatedAt         string                 `json:"updatedAt"`
}

// TagBrief 标签简要信息
type TagBrief struct {
	UUID string `json:"uuid"`
	Name string `json:"name"`
}

// UpdateKnowledgeRequest 更新知识请求
type UpdateKnowledgeRequest struct {
	Title    string                 `json:"title,omitempty"`
	Content  string                 `json:"content,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// MoveKnowledgeRequest 移动知识请求
type MoveKnowledgeRequest struct {
	DirectoryUUID *string `json:"directoryUuid"`
}

// BindTagsRequest 绑定/解绑标签请求
type BindTagsRequest struct {
	TagUUIDs []string `json:"tagUuids"`
}

// KnowledgeListResponse 知识列表响应
type KnowledgeListResponse struct {
	List     []Knowledge `json:"list"`
	Total    int64       `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"pageSize"`
}

