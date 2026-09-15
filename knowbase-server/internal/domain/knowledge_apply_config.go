// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package domain

import "time"

// KnowledgeApplyConfig 知识应用配置模型
type KnowledgeApplyConfig struct {
	ID                int64      `orm:"column(id);pk;auto" json:"-"`
	UUID              string     `orm:"column(uuid);size(36);unique" json:"uuid"`
	KnowledgebaseUUID string     `orm:"column(knowledge_base_uuid);size(36)" json:"knowledgebaseUuid"`
	AgentCode         string     `orm:"column(agent_code);size(100)" json:"agentCode"`
	AgentName         string     `orm:"column(agent_name);size(200);null" json:"agentName,omitempty"`
	Scene             string     `orm:"column(scene);size(100)" json:"scene"`
	TargetType        string     `orm:"column(target_type);size(100)" json:"targetType"`
	TargetID          string     `orm:"column(target_id);size(255)" json:"targetId"`
	TargetName        string     `orm:"column(target_name);size(500);null" json:"targetName,omitempty"`
	TargetExtra       string     `orm:"column(target_extra);type(json);null" json:"targetExtra,omitempty"` // JSON格式的额外信息
	Version           string     `orm:"column(version);size(50)" json:"version"`
	Status            string     `orm:"column(status);size(20);default(active)" json:"status"`
	Operator          string     `orm:"column(operator);size(100)" json:"operator"`
	Remark            string     `orm:"column(remark);size(500);null" json:"remark,omitempty"`
	CreatedAt         time.Time  `orm:"column(created_at);auto_now_add;type(datetime)" json:"createdAt"`
	UpdatedAt         time.Time  `orm:"column(updated_at);auto_now;type(datetime)" json:"updatedAt"`
	DeletedAt         *time.Time `orm:"column(deleted_at);type(datetime);null" json:"-"`
}

// TableName 指定表名
func (m *KnowledgeApplyConfig) TableName() string {
	return "knowledge_apply_config"
}

// 状态常量
const (
	ApplyConfigStatusActive   = "active"
	ApplyConfigStatusDisabled = "disabled"
)
