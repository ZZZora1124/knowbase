// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package domain

import (
	"time"
)

// KnowledgebaseMember 知识本成员模型
type KnowledgebaseMember struct {
	ID                int64     `orm:"column(id);pk;auto" json:"-"`
	UUID              string    `orm:"column(uuid);size(36);unique" json:"uuid"`
	KnowledgebaseUUID string    `orm:"column(knowledgebase_uuid);size(36);index" json:"knowledgebaseUuid"`
	MemberType        string    `orm:"column(member_type);size(50)" json:"memberType"`
	MemberValue       string    `orm:"column(member_value);size(255)" json:"memberValue"`
	Role              string    `orm:"column(role);size(20)" json:"role"`
	CreatedAt         time.Time `orm:"column(created_at);auto_now_add;type(datetime)" json:"createdAt"`
	UpdatedAt         time.Time `orm:"column(updated_at);auto_now;type(datetime)" json:"updatedAt"`
}

// TableName 指定表名
func (m *KnowledgebaseMember) TableName() string {
	return "knowledgebase_member"
}
