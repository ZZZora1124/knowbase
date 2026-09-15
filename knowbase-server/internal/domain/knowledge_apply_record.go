// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package domain

import "time"

// KnowledgeApplyRecord 知识应用记录模型
// EntryType 为 "directory" 时，DirectoryUUID 有效，KnowledgeUUID/KnowledgeVersion 为空串
type KnowledgeApplyRecord struct {
	ID                int64      `orm:"column(id);pk;auto" json:"-"`
	UUID              string     `orm:"column(uuid);size(36);unique" json:"uuid"`
	ConfigUUID        string     `orm:"column(config_uuid);size(36)" json:"configUuid"`
	ConfigVersion     string     `orm:"column(config_version);size(50)" json:"configVersion"`
	KnowledgebaseUUID string     `orm:"column(knowledge_base_uuid);size(36)" json:"knowledgebaseUuid"`
	KnowledgeUUID     string     `orm:"column(knowledge_uuid);size(36)" json:"knowledgeUuid"`
	KnowledgeVersion  string     `orm:"column(knowledge_version);size(50)" json:"knowledgeVersion"`
	EntryType         string     `orm:"column(entry_type);size(20);default(knowledge)" json:"entryType"`
	DirectoryUUID     string     `orm:"column(directory_uuid);size(36);null" json:"directoryUuid,omitempty"`
	Operator          string     `orm:"column(operator);size(100)" json:"operator"`
	PushStatus        string     `orm:"column(push_status);size(20);default(pending)" json:"pushStatus"`
	PushErrorMsg      string     `orm:"column(push_error_msg);type(text);null" json:"pushErrorMsg,omitempty"`
	PushedAt          *time.Time `orm:"column(pushed_at);null" json:"pushedAt,omitempty"`
	CreatedAt         time.Time  `orm:"column(created_at);auto_now_add;type(datetime)" json:"createdAt"`
}

// TableName 指定表名
func (m *KnowledgeApplyRecord) TableName() string {
	return "knowledge_apply_record"
}

// 推送状态常量
const (
	PushStatusPending = "pending"
	PushStatusSuccess = "success"
	PushStatusFailed  = "failed"
)

// 条目类型常量
const (
	EntryTypeKnowledge = "knowledge"
	EntryTypeDirectory = "directory"
)
