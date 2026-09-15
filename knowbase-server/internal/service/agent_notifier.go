// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package service

import (
	"time"
)

// ApplyNotification 应用通知数据
type ApplyNotification struct {
	ConfigUUID        string                   `json:"configUuid"`
	ConfigVersion     string                   `json:"configVersion"`
	KnowledgebaseUUID string                   `json:"knowledgebaseUuid"`
	KnowledgebaseType string                   `json:"knowledgebaseType"`
	AgentCode         string                   `json:"agentCode"`
	Scene             string                   `json:"scene"`
	TargetType        string                   `json:"targetType"`
	TargetID          string                   `json:"targetId"`
	TargetName        string                   `json:"targetName,omitempty"`
	Operator          string                   `json:"operator"`
	Action            string                   `json:"action"` // create/update/delete
	KnowledgeList     []*ApplyNotificationItem `json:"knowledgeList"`
	AppliedAt         time.Time                `json:"appliedAt"`
}

// ApplyNotificationItem 通知中的知识项
type ApplyNotificationItem struct {
	KnowledgebaseUUID string `json:"knowledgebaseUuid"`
	KnowledgeUUID     string `json:"knowledgeUuid"`
	KnowledgeVersion  string `json:"knowledgeVersion"`
	KnowledgeTitle    string `json:"knowledgeTitle"`
	KnowledgeType     string `json:"knowledgeType"`
	Content           string `json:"content,omitempty"`
	FilePath          string `json:"filePath,omitempty"`
	FileDownloadUrl   string `json:"fileDownloadUrl,omitempty"`
	Metadata          string `json:"metadata,omitempty"`
}

// AgentNotifier Agent通知接口
type AgentNotifier interface {
	// Notify 发送应用通知到Agent服务端
	Notify(notification *ApplyNotification) error

	// NotifyBatch 批量发送通知
	NotifyBatch(notifications []*ApplyNotification) []error
}
