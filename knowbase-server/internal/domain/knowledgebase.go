// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package domain

import (
	"time"

	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/types/constants"
)

// Knowledgebase 知识本模型
type Knowledgebase struct {
	ID          int64      `orm:"column(id);pk;auto" json:"-"`
	UUID        string     `orm:"column(uuid);size(36);unique" json:"uuid"`
	Name        string     `orm:"column(name);size(128)" json:"name"`
	Description string     `orm:"column(description);size(512)" json:"description"`
	Visibility  string     `orm:"column(visibility);size(20)" json:"visibility"`
	Owner       string     `orm:"column(owner);size(100)" json:"owner"`
	Status      int        `orm:"column(status);default(1)" json:"status"`
	CreatedAt   time.Time  `orm:"column(created_at);auto_now_add;type(datetime)" json:"createdAt"`
	UpdatedAt   time.Time  `orm:"column(updated_at);auto_now;type(datetime)" json:"updatedAt"`
	DeletedAt   *time.Time `orm:"column(deleted_at);null" json:"-"`
}

// TableName 指定表名
func (m *Knowledgebase) TableName() string {
	return "knowledgebase"
}

// IsActive 检查知识本是否处于激活状态
func (m *Knowledgebase) IsActive() bool {
	return m.Status == constants.KnowledgebaseStatusEnabled && m.DeletedAt == nil
}

// ValidateName 验证知识本名称格式
func ValidateName(name string) bool {
	// 名称长度1-128
	if len(name) < 1 || len(name) > 128 {
		return false
	}
	// 支持中英文、数字、中划线、下划线
	for _, r := range name {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == '-' || r == '_' ||
			(r >= 0x4e00 && r <= 0x9fa5)) { // 支持中文
			return false
		}
	}
	return true
}
