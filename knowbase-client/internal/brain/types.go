// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package brain

// PutPageRequest 写入页面请求
type PutPageRequest struct {
	Slug    string `json:"slug"`
	Content string `json:"content"`
}

// AutoLinks 自动链接统计
type AutoLinks struct {
	Created int `json:"created"`
	Removed int `json:"removed"`
	Errors  int `json:"errors"`
}

// PutPageResponse 写入页面响应
type PutPageResponse struct {
	Slug      string    `json:"slug"`
	Status    string    `json:"status"`
	Chunks    int       `json:"chunks"`
	AutoLinks AutoLinks `json:"auto_links"`
}

// PageResult 页面详情
type PageResult struct {
	Slug          string                 `json:"slug"`
	Type          string                 `json:"type"`
	Title         string                 `json:"title"`
	Content       string                 `json:"content"`
	CompiledTruth string                 `json:"compiled_truth"`
	Frontmatter   map[string]interface{} `json:"frontmatter"`
	Tags          []string               `json:"tags"`
	CreatedAt     string                 `json:"created_at"`
	UpdatedAt     string                 `json:"updated_at"`
}

// SearchResult 搜索结果
type SearchResult struct {
	Slug          string  `json:"slug"`
	Type          string  `json:"type"`
	Title         string  `json:"title"`
	Score         float64 `json:"score"`
	CompiledTruth string  `json:"compiled_truth"`
}

// GraphEdge 图谱边
type GraphEdge struct {
	FromSlug string `json:"from_slug"`
	ToSlug   string `json:"to_slug"`
	LinkType string `json:"link_type"`
	Depth    int    `json:"depth"`
}

// TimelineEntry 时间线条目
type TimelineEntry struct {
	Slug    string `json:"slug"`
	Date    string `json:"date"`
	Source  string `json:"source"`
	Summary string `json:"summary"`
}

// BacklinkResult 反向链接结果
type BacklinkResult struct {
	FromSlug   string `json:"from_slug"`
	ToSlug     string `json:"to_slug"`
	LinkType   string `json:"link_type"`
	LinkSource string `json:"link_source"`
}

// AddTimelineEntryRequest 追加时间线条目请求
type AddTimelineEntryRequest struct {
	Date    string `json:"date"`
	Summary string `json:"summary"`
	Detail  string `json:"detail,omitempty"`
	Source  string `json:"source,omitempty"`
}

// AddTimelineEntryResponse 追加时间线条目响应
type AddTimelineEntryResponse struct {
	Status string `json:"status"`
}

// ListPageItem 页面列表项
type ListPageItem struct {
	Slug      string `json:"slug"`
	Type      string `json:"type"`
	Title     string `json:"title"`
	UpdatedAt string `json:"updated_at"`
}

// DefaultModelResponse 服务端下发的默认模型配置
type DefaultModelResponse struct {
	Model string `json:"model"`
}

// DeletePageResponse 删除页面响应
type DeletePageResponse struct {
	Slug   string `json:"slug"`
	Status string `json:"status"`
}
