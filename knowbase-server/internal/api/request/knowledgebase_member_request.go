// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package request

// AddMemberRequest 添加成员请求
type AddMemberRequest struct {
	MemberType  string `json:"memberType" binding:"required,oneof=username email_group department chat_group"`
	MemberValue string `json:"memberValue" binding:"required,min=1,max=255"`
	Role        string `json:"role" binding:"required,oneof=admin editor viewer"`
}

// UpdateMemberRoleRequest 更新成员角色请求
type UpdateMemberRoleRequest struct {
	Role string `json:"role" binding:"required,oneof=admin editor viewer"`
}

// ListMembersRequest 查询成员列表请求
type ListMembersRequest struct {
	Page     int `form:"page" binding:"omitempty,min=1"`
	PageSize int `form:"pageSize" binding:"omitempty,min=1,max=100"`
}
