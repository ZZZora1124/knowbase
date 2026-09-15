// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package domain

import "time"

// KnowledgeConsolidationMetric 知识沉淀性能指标模型
type KnowledgeConsolidationMetric struct {
	ID           int64     `orm:"column(id);pk;auto" json:"id"`
	Platform     string    `orm:"column(platform);size(64)" json:"platform"`
	Username     string    `orm:"column(username);size(128)" json:"username"`
	KbUUID       string    `orm:"column(kb_uuid);size(64)" json:"kbUuid"`
	OpType       string    `orm:"column(op_type);size(16)" json:"opType"`
	InputTokens  int       `orm:"column(input_tokens)" json:"inputTokens"`
	OutputTokens int       `orm:"column(output_tokens)" json:"outputTokens"`
	TotalTokens  int       `orm:"column(total_tokens)" json:"totalTokens"`
	Model        string    `orm:"column(model);size(128)" json:"model"`
	DurationS    int       `orm:"column(duration_s)" json:"durationS"`
	PageCount    int       `orm:"column(page_count)" json:"pageCount"`
	Success      bool      `orm:"column(success)" json:"success"`
	ErrorMessage string    `orm:"column(error_message);size(512);null" json:"errorMessage,omitempty"`
	CreatedAt    time.Time `orm:"column(created_at);auto_now_add;type(datetime)" json:"createdAt"`
}

func (m *KnowledgeConsolidationMetric) TableName() string {
	return "knowledge_consolidation_metric"
}
