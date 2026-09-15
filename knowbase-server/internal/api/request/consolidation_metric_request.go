// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package request

// ConsolidationMetricItem 单条知识沉淀性能指标
type ConsolidationMetricItem struct {
	KbUUID       string `json:"kbUuid" binding:"required"`
	OpType       string `json:"opType" binding:"required,oneof=query warmup"`
	InputTokens  int    `json:"inputTokens"`
	OutputTokens int    `json:"outputTokens"`
	TotalTokens  int    `json:"totalTokens"`
	Model        string `json:"model"`
	DurationS    int    `json:"durationS"`
	PageCount    int    `json:"pageCount"`
	Success      bool   `json:"success"`
	ErrorMessage string `json:"errorMessage,omitempty"`
}

// ReportConsolidationMetricRequest 批量上报知识沉淀性能指标请求
type ReportConsolidationMetricRequest struct {
	Platform string                    `json:"platform"`
	Username string                    `json:"username"`
	Metrics  []ConsolidationMetricItem `json:"metrics" binding:"required,dive"`
}
