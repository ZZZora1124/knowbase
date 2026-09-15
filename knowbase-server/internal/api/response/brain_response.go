// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package response

// PutPageLinks brain page 链接操作统计
type PutPageLinks struct {
	Created int `json:"created"`
	Removed int `json:"removed"`
	Errors  int `json:"errors"`
}

// PutPageTimeline brain page 时间线操作统计
type PutPageTimeline struct {
	Created int `json:"created"`
}

// PutPageResponse put_page 接口响应
type PutPageResponse struct {
	Slug     string           `json:"slug"`
	Status   string           `json:"status"`
	Chunks   int              `json:"chunks,omitempty"`
	Links    *PutPageLinks    `json:"links,omitempty"`
	Timeline *PutPageTimeline `json:"timeline,omitempty"`
}

// AddTimelineEntryResponse add_timeline_entry 接口响应
type AddTimelineEntryResponse struct {
	Status string `json:"status"`
}

// DefaultModelResponse brain 全局默认模型配置响应
// 对应 GET /open-api/v1/brain/config/default-model
type DefaultModelResponse struct {
	Model string `json:"model"`
}
