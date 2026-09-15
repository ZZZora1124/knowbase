// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package domain

import (
	"time"
)

// InfoFlowMessage 如流群消息模型
type InfoFlowMessage struct {
	ID int64 `orm:"column(id);pk;auto" json:"id"`

	// 消息基础信息
	GroupID   int64  `orm:"column(group_id);index" json:"group_id"`
	GroupName string `orm:"column(group_name);size(255);null" json:"group_name"`
	UserID    string `orm:"column(user_id);size(32);index" json:"user_id"`
	Time      int64  `orm:"column(time);index" json:"time"`
	MessageID string `orm:"column(message_id);size(50)" json:"message_id"`

	// 消息头部信息
	FromUserID string `orm:"column(from_user_id);size(32);null" json:"from_user_id"`
	ToID       int64  `orm:"column(to_id);null" json:"to_id"`
	ToType     string `orm:"column(to_type);size(20);null" json:"to_type"`
	MsgType    string `orm:"column(msg_type);size(20);null" json:"msg_type"`

	// 消息内容
	ContentType string `orm:"column(content_type);size(20)" json:"content_type"`
	ContentText string `orm:"column(content_text);type(text);null" json:"content_text"`
	ContentJSON string `orm:"column(content_json);type(json);null" json:"content_json"`

	// 通用字段
	CreatedAt time.Time `orm:"column(created_at);auto_now_add;type(datetime)" json:"created_at"`
}

// TableName 指定表名
func (m *InfoFlowMessage) TableName() string {
	return "infoflow_messages"
}

// InfoFlowMessageType 消息内容类型常量
const (
	MessageTypeText      = "TEXT"          // 文本消息
	MessageTypeImage     = "IMAGE"         // 图片消息
	MessageTypeAT        = "AT"            // @消息
	MessageTypeLink      = "LINK"          // 链接消息
	MessageTypeCommand   = "command"       // 斜杠指令
	MessageTypeReplyData = "replyData"     // 回复数据
	MessageTypeMerged    = "MERGEDMESSAGE" // 合并消息
)

// InfoFlowToType 接收者类型常量
const (
	ToTypeGroup = "GROUP" // 群组
	ToTypeUser  = "USER"  // 个人
)
