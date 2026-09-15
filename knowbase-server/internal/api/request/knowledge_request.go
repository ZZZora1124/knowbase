// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package request

import "mime/multipart"

// CreateKnowledgeRequest 创建知识请求
type CreateKnowledgeRequest struct {
	KnowledgebaseUUID string                 `form:"knowledgebaseUuid" binding:"required"`
	DirectoryUUID     *string                `form:"directoryUuid"`
	Title             string                 `form:"title" binding:"required,min=1,max=500"`
	KnowledgeType     string                 `form:"knowledgeType" binding:"required,oneof=content readme text_file audio_file image_file video_file config"`
	TemplateName      string                 `form:"templateName" binding:"omitempty,max=128"`
	KnowledgeSource   string                 `form:"knowledgeSource" binding:"required,oneof=custom icode_repo icode_deepwiki icafe_card ku_doc github_repo infoflow_group iapi_doc knowledge_book"`
	Content           string                 `form:"content"`
	File              *multipart.FileHeader  `form:"file"`
	Metadata          map[string]interface{} `form:"metadata"`
	TagUUIDs          []string               `form:"tagUuids"`
	SourceMethod      string                 `form:"-"` // 来源方式：manual(手动创建) 或 api(API创建)，由handler层根据认证类型设置
}

// ListKnowledgeRequest 查询知识列表请求
type ListKnowledgeRequest struct {
	KnowledgebaseUUID string `form:"knowledgebaseUuid"` // 从URL路径参数获取，不需要required
	DirectoryUUID     string `form:"directoryUuid"`
	KnowledgeSource   string `form:"knowledgeSource"`
	KnowledgeType     string `form:"knowledgeType"`
	TemplateName      string `form:"templateName"`
	Keyword           string `form:"keyword"`
	CreatorUsername   string `form:"creatorUsername"`
	Page              int    `form:"page" binding:"omitempty,min=1"`
	PageSize          int    `form:"pageSize" binding:"omitempty,min=1,max=500"`
}

// ListKnowledgeByTagsRequest 根据标签查询知识请求
type ListKnowledgeByTagsRequest struct {
	KnowledgebaseUUID string   `json:"knowledgebaseUuid"` // 从URL路径参数获取，不需要required
	DirectoryUUIDs    []string `json:"directoryUuids"`    // 可选，目录UUID列表过滤
	DirectoryNames    []string `json:"directoryNames"`    // 可选，目录名称列表过滤（与directoryUuids互斥）
	TagUUIDs          []string `json:"tagUuids"`
	TagNames          []string `json:"tagNames"`
	MatchType         string   `json:"matchType"` // any 或 all，默认any
	KnowledgeSource   string   `json:"knowledgeSource"`
	KnowledgeType     string   `json:"knowledgeType"`
	TemplateName      string   `json:"templateName"`
	Page              int      `json:"page" binding:"omitempty,min=1"`
	PageSize          int      `json:"pageSize" binding:"omitempty,min=1,max=500"`
}

// UpdateKnowledgeRequest 更新知识请求
type UpdateKnowledgeRequest struct {
	Title         string                 `json:"title" binding:"omitempty,min=1,max=500"`
	DirectoryUUID *string                `json:"directoryUuid"` // 目录UUID，null表示移到默认目录，不传表示不更新目录
	Content       *string                `json:"content"`       // 指针类型：nil 表示不更新，"" 表示清空
	File          *multipart.FileHeader  `form:"file"`
	Metadata      map[string]interface{} `json:"metadata"`
}

// MoveKnowledgeRequest 移动知识请求
type MoveKnowledgeRequest struct {
	DirectoryUUID *string `json:"directoryUuid"` // null表示移到默认目录
}

// GetKnowledgeBatchRequest 批量查询知识请求
type GetKnowledgeBatchRequest struct {
	UUIDs []string `json:"uuids" binding:"required,min=1,max=100"` // 知识UUID列表，最多100个
}
