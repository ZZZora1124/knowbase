// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package request

// DownloadMetricItem 单条下载记录
type DownloadMetricItem struct {
	SourceType   string `json:"sourceType" binding:"required"`
	Count        int    `json:"count"`
	DurationMs   int    `json:"durationMs"`
	Success      bool   `json:"success"`
	ErrorMessage string `json:"errorMessage,omitempty"`
}

// ReportDownloadMetricRequest 批量上报知识下载记录请求
type ReportDownloadMetricRequest struct {
	Platform string               `json:"platform"`
	Username string               `json:"username"`
	Metrics  []DownloadMetricItem `json:"metrics" binding:"required,dive"`
}
