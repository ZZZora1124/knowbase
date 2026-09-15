// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package domain

import "time"

// KnowledgeDownloadMetric 知识下载记录模型
type KnowledgeDownloadMetric struct {
	ID           int64     `orm:"column(id);pk;auto" json:"id"`
	Platform     string    `orm:"column(platform);size(64)" json:"platform"`
	Username     string    `orm:"column(username);size(128)" json:"username"`
	SourceType   string    `orm:"column(source_type);size(32)" json:"sourceType"`
	Count        int       `orm:"column(count)" json:"count"`
	DurationMs   int       `orm:"column(duration_ms)" json:"durationMs"`
	Success      bool      `orm:"column(success)" json:"success"`
	ErrorMessage string    `orm:"column(error_message);size(512);null" json:"errorMessage,omitempty"`
	CreatedAt    time.Time `orm:"column(created_at);auto_now_add;type(datetime)" json:"createdAt"`
}

func (m *KnowledgeDownloadMetric) TableName() string {
	return "knowledge_download_metric"
}
