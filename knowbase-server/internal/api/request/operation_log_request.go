// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package request

// ListOperationLogRequest 查询操作日志列表请求
type ListOperationLogRequest struct {
	KnowledgebaseUUID string `form:"knowledgebaseUuid"`
	Operator          string `form:"operator"`
	Agent             string `form:"agent"`
	ResourceType      string `form:"resourceType"`
	ResourceUUID      string `form:"resourceUuid"`
	Page              int    `form:"page" binding:"omitempty,min=1"`
	PageSize          int    `form:"pageSize" binding:"omitempty,min=1,max=100"`
}
