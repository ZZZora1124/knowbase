// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package domain

import (
	"time"
)

// KnowledgeTagRelation 知识-标签关联模型
type KnowledgeTagRelation struct {
	ID            int64     `orm:"column(id);pk;auto" json:"-"`
	UUID          string    `orm:"column(uuid);size(36);unique" json:"uuid"`
	KnowledgeUUID string    `orm:"column(knowledge_uuid);size(36);index" json:"knowledgeUuid"`
	TagUUID       string    `orm:"column(tag_uuid);size(36);index" json:"tagUuid"`
	CreatedAt     time.Time `orm:"column(created_at);auto_now_add;type(datetime)" json:"createdAt"`
}

// TableName 指定表名
func (m *KnowledgeTagRelation) TableName() string {
	return "knowledge_tag_relation"
}
