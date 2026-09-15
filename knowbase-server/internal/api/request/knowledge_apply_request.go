// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package request

// CreateOrUpdateApplyConfigRequest 创建或更新应用配置请求
type CreateOrUpdateApplyConfigRequest struct {
	KnowledgebaseUUID string               `json:"knowledgebaseUuid" binding:"required"`
	AgentCode         string               `json:"agentCode" binding:"required"`
	AgentName         string               `json:"agentName"`
	Scene             string               `json:"scene" binding:"required"`
	TargetType        string               `json:"targetType" binding:"required"`
	TargetID          string               `json:"targetId" binding:"required"`
	TargetName        string               `json:"targetName"`
	TargetExtra       map[string]any       `json:"targetExtra"` // 额外信息，如代码库的分支等
	Version           string               `json:"version"`     // 可选，不填自动生成
	Remark            string               `json:"remark"`
	KnowledgeList     []ApplyKnowledgeItem `json:"knowledgeList" binding:"required,min=1"`
}

// ApplyKnowledgeItem 应用的条目项
// 支持两种类型：
//   - entryType=knowledge (默认): 由 knowledgeUuid (+ 可选 knowledgeVersion) 指定单条知识
//   - entryType=directory       : 由 directoryUuid 指定目录，代表目录及其所有后代目录下的知识
type ApplyKnowledgeItem struct {
	EntryType        string `json:"entryType"`        // knowledge(默认) / directory
	KnowledgeUUID    string `json:"knowledgeUuid"`    // entryType=knowledge 时必填
	KnowledgeVersion string `json:"knowledgeVersion"` // entryType=knowledge 时可选，为空使用最新版本
	DirectoryUUID    string `json:"directoryUuid"`    // entryType=directory 时必填
}

// ListApplyConfigRequest 查询应用配置列表请求
type ListApplyConfigRequest struct {
	KnowledgebaseUUID string `form:"knowledgebaseUuid"`
	AgentCode         string `form:"agentCode"`
	Scene             string `form:"scene"`
	TargetType        string `form:"targetType"`
	TargetID          string `form:"targetId"`
	Status            string `form:"status"`
	Operator          string `form:"operator"`
	Page              int    `form:"page"`
	PageSize          int    `form:"pageSize"`
}

// AddKnowledgeToConfigRequest 增量添加知识请求
type AddKnowledgeToConfigRequest struct {
	ConfigUUID    string               `json:"configUuid"`
	Version       string               `json:"version"` // 可选，不填自动生成
	KnowledgeList []ApplyKnowledgeItem `json:"knowledgeList" binding:"required,min=1"`
}

// RemoveKnowledgeFromConfigRequest 移除条目请求
// 兼容两种入参方式：
//   - 旧接口：KnowledgeUUIDs，仅支持移除 knowledge 条目
//   - 新接口：Entries，支持移除 knowledge / directory 两种条目。若 Entries 非空则优先使用
type RemoveKnowledgeFromConfigRequest struct {
	ConfigUUID     string             `json:"configUuid"`
	Version        string             `json:"version"` // 可选，不填自动生成
	KnowledgeUUIDs []string           `json:"knowledgeUuids"`
	Entries        []ApplyRemoveEntry `json:"entries"`
}

// ApplyRemoveEntry 移除条目
type ApplyRemoveEntry struct {
	EntryType     string `json:"entryType"`     // knowledge / directory
	KnowledgeUUID string `json:"knowledgeUuid"` // entryType=knowledge 时必填
	DirectoryUUID string `json:"directoryUuid"` // entryType=directory 时必填
}

// UpdateConfigStatusRequest 更新配置状态请求
type UpdateConfigStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=active disabled"`
}

// GetConfigByTargetRequest 按目标查询配置请求
type GetConfigByTargetRequest struct {
	KnowledgebaseUUID string `form:"knowledgebaseUuid"`
	AgentCode         string `form:"agentCode" binding:"required"`
	Scene             string `form:"scene" binding:"required"`
	TargetType        string `form:"targetType" binding:"required"`
	TargetID          string `form:"targetId" binding:"required"`
}
