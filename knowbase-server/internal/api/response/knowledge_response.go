// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package response

import (
	"encoding/json"
	"time"

	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/domain"
)

// KnowledgeResponse 知识响应
type KnowledgeResponse struct {
	UUID              string                 `json:"uuid"`
	KnowledgebaseUUID string                 `json:"knowledgebaseUuid"`
	DirectoryUUID     *string                `json:"directoryUuid"`
	DirectoryName     string                 `json:"directoryName,omitempty"`
	DirPath           string                 `json:"dirPath,omitempty"`
	CreatorUsername   string                 `json:"creatorUsername"`
	UpdaterUsername   string                 `json:"updaterUsername"`
	Title             string                 `json:"title"`
	KnowledgeType     string                 `json:"knowledgeType"`
	TemplateName      string                 `json:"templateName"`
	KnowledgeSource   string                 `json:"knowledgeSource"`
	Content           string                 `json:"content"`
	FilePath          string                 `json:"filePath,omitempty"`
	FileSize          int64                  `json:"fileSize,omitempty"`
	FileFormat        string                 `json:"fileFormat,omitempty"`
	SourceMethod      string                 `json:"sourceMethod"`
	Metadata          map[string]interface{} `json:"metadata,omitempty"`
	CurrentVersion    string                 `json:"currentVersion"`
	Tags              []*TagBriefResponse    `json:"tags"`
	CreatedAt         time.Time              `json:"createdAt"`
	UpdatedAt         time.Time              `json:"updatedAt"`
}

// TagBriefResponse 标签简要响应
type TagBriefResponse struct {
	UUID string `json:"uuid"`
	Name string `json:"name"`
}

// ListKnowledgeResponse 知识列表响应
type ListKnowledgeResponse struct {
	List     []*KnowledgeListItemResponse `json:"list"`
	Total    int64                        `json:"total"`
	Page     int                          `json:"page"`
	PageSize int                          `json:"pageSize"`
}

// KnowledgeListItemResponse 知识列表项响应
type KnowledgeListItemResponse struct {
	UUID              string                 `json:"uuid"`
	KnowledgebaseUUID string                 `json:"knowledgebaseUuid"`
	DirectoryUUID     *string                `json:"directoryUuid"`
	DirPath           string                 `json:"dirPath,omitempty"`
	CreatorUsername   string                 `json:"creatorUsername"`
	UpdaterUsername   string                 `json:"updaterUsername"`
	Title             string                 `json:"title"`
	Content           string                 `json:"content"`
	Metadata          map[string]interface{} `json:"metadata,omitempty"`
	KnowledgeType     string                 `json:"knowledgeType"`
	TemplateName      string                 `json:"templateName"`
	KnowledgeSource   string                 `json:"knowledgeSource"`
	FileSize          int64                  `json:"fileSize,omitempty"`
	FileFormat        string                 `json:"fileFormat,omitempty"`
	CurrentVersion    string                 `json:"currentVersion"`
	Tags              []*TagBriefResponse    `json:"tags"`
	MatchedTags       []*TagBriefResponse    `json:"matchedTags,omitempty"` // 匹配的标签（标签查询场景）
	CreatedAt         time.Time              `json:"createdAt"`
	UpdatedAt         time.Time              `json:"updatedAt"`
}

// NewKnowledgeResponse 从领域模型创建响应
func NewKnowledgeResponse(k *domain.Knowledge, tags []*domain.KnowledgebaseTag, directoryName, dirPath string) *KnowledgeResponse {
	if k == nil {
		return nil
	}

	var metadata map[string]interface{}
	if k.Metadata != "" {
		json.Unmarshal([]byte(k.Metadata), &metadata)
	}

	updaterUsername := k.UpdaterUsername
	if updaterUsername == "" {
		updaterUsername = k.CreatorUsername
	}

	return &KnowledgeResponse{
		UUID:              k.UUID,
		KnowledgebaseUUID: k.KnowledgebaseUUID,
		DirectoryUUID:     k.DirectoryUUID,
		DirectoryName:     directoryName,
		DirPath:           dirPath,
		CreatorUsername:   k.CreatorUsername,
		UpdaterUsername:   updaterUsername,
		Title:             k.Title,
		KnowledgeType:     k.KnowledgeType,
		TemplateName:      k.TemplateName,
		KnowledgeSource:   k.KnowledgeSource,
		Content:           k.Content,
		FilePath:          k.FilePath,
		FileSize:          k.FileSize,
		FileFormat:        k.FileFormat,
		SourceMethod:      k.SourceMethod,
		Metadata:          metadata,
		CurrentVersion:    k.CurrentVersion,
		Tags:              toTagBriefResponses(tags),
		CreatedAt:         k.CreatedAt,
		UpdatedAt:         k.UpdatedAt,
	}
}

// NewKnowledgeListItemResponse 从领域模型创建列表项响应
func NewKnowledgeListItemResponse(k *domain.Knowledge, tags []*domain.KnowledgebaseTag, dirPath string) *KnowledgeListItemResponse {
	if k == nil {
		return nil
	}

	var metadata map[string]interface{}
	if k.Metadata != "" {
		json.Unmarshal([]byte(k.Metadata), &metadata)
	}

	updaterUsername := k.UpdaterUsername
	if updaterUsername == "" {
		updaterUsername = k.CreatorUsername
	}

	return &KnowledgeListItemResponse{
		UUID:              k.UUID,
		KnowledgebaseUUID: k.KnowledgebaseUUID,
		DirectoryUUID:     k.DirectoryUUID,
		DirPath:           dirPath,
		CreatorUsername:   k.CreatorUsername,
		UpdaterUsername:   updaterUsername,
		Title:             k.Title,
		Content:           k.Content,
		Metadata:          metadata,
		KnowledgeType:     k.KnowledgeType,
		TemplateName:      k.TemplateName,
		KnowledgeSource:   k.KnowledgeSource,
		FileSize:          k.FileSize,
		FileFormat:        k.FileFormat,
		CurrentVersion:    k.CurrentVersion,
		Tags:              toTagBriefResponses(tags),
		CreatedAt:         k.CreatedAt,
		UpdatedAt:         k.UpdatedAt,
	}
}

// NewKnowledgeListItemResponseWithMatchedTags 从领域模型创建列表项响应（包含匹配标签）
func NewKnowledgeListItemResponseWithMatchedTags(k *domain.Knowledge, tags, matchedTags []*domain.KnowledgebaseTag, dirPath string) *KnowledgeListItemResponse {
	if k == nil {
		return nil
	}

	var metadata map[string]interface{}
	if k.Metadata != "" {
		json.Unmarshal([]byte(k.Metadata), &metadata)
	}

	updaterUsername := k.UpdaterUsername
	if updaterUsername == "" {
		updaterUsername = k.CreatorUsername
	}

	return &KnowledgeListItemResponse{
		UUID:              k.UUID,
		KnowledgebaseUUID: k.KnowledgebaseUUID,
		DirectoryUUID:     k.DirectoryUUID,
		DirPath:           dirPath,
		CreatorUsername:   k.CreatorUsername,
		UpdaterUsername:   updaterUsername,
		Title:             k.Title,
		Content:           k.Content,
		Metadata:          metadata,
		KnowledgeType:     k.KnowledgeType,
		TemplateName:      k.TemplateName,
		KnowledgeSource:   k.KnowledgeSource,
		FileSize:          k.FileSize,
		FileFormat:        k.FileFormat,
		CurrentVersion:    k.CurrentVersion,
		Tags:              toTagBriefResponses(tags),
		MatchedTags:       toTagBriefResponses(matchedTags),
		CreatedAt:         k.CreatedAt,
		UpdatedAt:         k.UpdatedAt,
	}
}

func toTagBriefResponses(tags []*domain.KnowledgebaseTag) []*TagBriefResponse {
	if tags == nil {
		return []*TagBriefResponse{}
	}
	result := make([]*TagBriefResponse, len(tags))
	for i, tag := range tags {
		result[i] = &TagBriefResponse{
			UUID: tag.UUID,
			Name: tag.Name,
		}
	}
	return result
}

// GetKnowledgeBatchResponse 批量查询知识响应
type GetKnowledgeBatchResponse struct {
	List []*KnowledgeResponse `json:"list"` // 知识列表
}
