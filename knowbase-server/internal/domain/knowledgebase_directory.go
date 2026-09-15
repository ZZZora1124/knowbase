// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package domain

import "time"

// KnowledgebaseDirectory 目录模型，支持无限级树形结构（邻接表）
// 目录只用于"自定义知识"，parent_uuid=NULL 表示根目录（一级目录）
type KnowledgebaseDirectory struct {
	ID                int64      `orm:"column(id);pk;auto" json:"-"`
	UUID              string     `orm:"column(uuid);size(36);unique" json:"uuid"`
	KnowledgebaseUUID string     `orm:"column(knowledgebase_uuid);size(36);index" json:"knowledgebaseUuid"`
	ParentUUID        *string    `orm:"column(parent_uuid);size(36);null" json:"parentUuid"`
	Name              string     `orm:"column(name);size(255)" json:"name"`
	IsDefault         bool       `orm:"column(is_default);default(0)" json:"isDefault"`
	CreatedAt         time.Time  `orm:"column(created_at);auto_now_add;type(datetime)" json:"createdAt"`
	UpdatedAt         time.Time  `orm:"column(updated_at);auto_now;type(datetime)" json:"updatedAt"`
	DeletedAt         *time.Time `orm:"column(deleted_at);null" json:"-"`
}

// TableName 指定表名
func (m *KnowledgebaseDirectory) TableName() string {
	return "knowledgebase_directory"
}

// ValidateDirectoryName 验证目录名称格式
// 支持中英文、数字、中划线、下划线
func ValidateDirectoryName(name string) bool {
	if len(name) < 1 || len(name) > 255 {
		return false
	}
	for _, r := range name {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == '-' || r == '_' ||
			(r >= 0x4e00 && r <= 0x9fa5)) {
			return false
		}
	}
	return true
}
