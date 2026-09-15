// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package response

import "time"

// ApplyConfigResponse 应用配置响应
type ApplyConfigResponse struct {
	UUID                    string                  `json:"configUuid"`
	KnowledgebaseUUID       string                  `json:"knowledgebaseUuid"`
	KnowledgebaseName       string                  `json:"knowledgebaseName,omitempty"`
	KnowledgebaseVisibility string                  `json:"knowledgebaseVisibility,omitempty"`
	AgentCode               string                  `json:"agentCode"`
	AgentName               string                  `json:"agentName,omitempty"`
	Scene                   string                  `json:"scene"`
	TargetType              string                  `json:"targetType"`
	TargetID                string                  `json:"targetId"`
	TargetName              string                  `json:"targetName,omitempty"`
	TargetExtra             map[string]any          `json:"targetExtra,omitempty"`
	Version                 string                  `json:"version"`
	Status                  string                  `json:"status"`
	Operator                string                  `json:"operator"`
	Remark                  string                  `json:"remark,omitempty"`
	// ConfiguredEntries 原始配置条目（知识+目录混合），代表用户在保存时实际选择了什么
	ConfiguredEntries []*ApplyConfigEntryItem `json:"configuredEntries"`
	// KnowledgeList 展开后的具体知识列表（目录条目会被动态展开为其子树下的所有知识）
	KnowledgeList []*AppliedKnowledgeItem `json:"knowledgeList"`
	CreatedAt     time.Time               `json:"createdAt"`
	UpdatedAt     time.Time               `json:"updatedAt"`
}

// ApplyConfigEntryItem 原始配置条目项
type ApplyConfigEntryItem struct {
	EntryType        string    `json:"entryType"`
	KnowledgeUUID    string    `json:"knowledgeUuid,omitempty"`
	KnowledgeVersion string    `json:"knowledgeVersion,omitempty"`
	KnowledgeTitle   string    `json:"knowledgeTitle,omitempty"`
	DirectoryUUID    string    `json:"directoryUuid,omitempty"`
	DirectoryName    string    `json:"directoryName,omitempty"`
	Operator         string    `json:"operator"`
	CreatedAt        time.Time `json:"createdAt"`
}

// AppliedKnowledgeItem 已应用的知识项（展开后）
type AppliedKnowledgeItem struct {
	KnowledgeUUID    string `json:"knowledgeUuid"`
	KnowledgeVersion string `json:"knowledgeVersion"`
	KnowledgeTitle   string `json:"knowledgeTitle"`
	KnowledgeContent string `json:"knowledgeContent"`
	KnowledgeType    string `json:"knowledgeType"`
	TemplateName     string `json:"templateName"`
	PushStatus       string `json:"pushStatus"`
	Operator         string `json:"operator"`
	// SourceEntryType 该知识来自哪种类型的配置条目：knowledge(直接选中) 或 directory(目录展开)
	SourceEntryType string `json:"sourceEntryType"`
	// SourceDirectoryUUID 当 SourceEntryType=directory 时，对应的目录UUID
	SourceDirectoryUUID string `json:"sourceDirectoryUuid,omitempty"`
	// SourceDirectoryName 当 SourceEntryType=directory 时，对应的目录名称
	SourceDirectoryName string    `json:"sourceDirectoryName,omitempty"`
	CreatedAt           time.Time `json:"createdAt"`
}

// ApplyConfigListResponse 配置列表响应
type ApplyConfigListResponse struct {
	List  []*ApplyConfigResponse `json:"list"`
	Total int64                  `json:"total"`
}

// ApplyConfigHistoryResponse 配置历史响应
type ApplyConfigHistoryResponse struct {
	History []*ApplyConfigHistoryItem `json:"history"`
}

// ApplyConfigHistoryItem 历史版本项
type ApplyConfigHistoryItem struct {
	Version string    `json:"version"`
	Operator string    `json:"operator"`
	CreatedAt time.Time `json:"createdAt"`
	// ConfiguredEntries 原始配置条目（知识+目录混合），代表该版本保存时用户实际选择了什么
	ConfiguredEntries []*ApplyConfigEntryItem `json:"configuredEntries"`
	// KnowledgeList 展开后的具体知识列表（目录条目会被动态展开为其子树下的所有知识）
	KnowledgeList []*AppliedKnowledgeItem `json:"knowledgeList"`
}

// ApplyResultResponse 应用结果响应
type ApplyResultResponse struct {
	ConfigUUID    string `json:"configUuid"`
	ConfigVersion string `json:"configVersion"`
	TotalCount    int    `json:"totalCount"`
	SuccessCount  int    `json:"successCount"`
	FailedCount   int    `json:"failedCount"`
}

// RetryResultResponse 重试结果响应
type RetryResultResponse struct {
	RetryCount   int `json:"retryCount"`
	SuccessCount int `json:"successCount"`
	FailedCount  int `json:"failedCount"`
}
