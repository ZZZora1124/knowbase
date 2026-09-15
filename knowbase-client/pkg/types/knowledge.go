// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package types

import "time"

// RawKnowledge 原始知识数据
type RawKnowledge struct {
	SourceType string                 // 知识源类型
	Data       map[string]interface{} // 原始数据
}

// KnowledgeItem 标准化的知识项
type KnowledgeItem struct {
	UUID       string                 // 唯一标识符
	ID         string                 // 知识ID
	Title      string                 // 标题
	CreateTime string                 // 创建时间
	UpdateTime string                 // 更新时间
	Content    string                 // 内容
	SourceType string                 // 来源类型
	Metadata   map[string]interface{} // 元数据
	FilePath   string                 // 存储路径
}

// LoadResult 加载结果
type LoadResult struct {
	SourceType   string                // 知识源类型
	Success      bool                  // 是否成功
	Count        int                   // 加载数量
	Duration     time.Duration         // 耗时
	ErrorMessage string                // 错误信息（简短）
	ErrorType    string                // 错误类型分类
	ErrorDetails map[string]string     // 错误详细信息（如 URL、状态码、响应体等）
	Suggestion   string                // 恢复建议
	StoredItems  []StoredKnowledgeInfo // 存储的知识项详细信息
}

// StoredKnowledgeInfo 存储的知识项信息
type StoredKnowledgeInfo struct {
	Uuid        string // 知识UUID
	Name        string // 知识项名称(如文件名、卡片标题等)
	Description string // 知识项描述信息
	FullPath    string // 存储的完整绝对路径
	RelPath     string // 相对于basePath的路径
}

// KnowledgeStructure 知识结构信息
type KnowledgeStructure struct {
	SourceType  string                // 知识源类型
	SubDir      string                // 子目录名称
	Description string                // 该知识源的整体描述
	StoredItems []StoredKnowledgeInfo // 该源下的所有知识项
	UsagePath   string                // usage文档的路径
}

// LoadStats 加载统计
type LoadStats struct {
	TotalSources   int           // 总知识源数
	SuccessSources int           // 成功数
	FailedSources  int           // 失败数
	TotalItems     int           // 总知识项数
	TotalDuration  time.Duration // 各源耗时之和（用于调试）
	ActualDuration time.Duration // 实际总耗时（并行执行的墙钟时间）
	Results        []LoadResult  // 详细结果
}

// MetricDownloadRequest 指标上报请求数据（单个指标）
type MetricDownloadRequest struct {
	SourceType   string `json:"sourceType"`             // 知识源类型
	Count        int    `json:"count"`                  // 下载数量
	DurationMs   int64  `json:"durationMs"`             // 耗时(毫秒)
	Success      bool   `json:"success"`                // 是否成功
	ErrorMessage string `json:"errorMessage,omitempty"` // 错误信息，成功时省略
}

// MetricDownloadBatchRequest 批量指标上报请求
type MetricDownloadBatchRequest struct {
	Metrics []MetricDownloadRequest `json:"metrics"` // 指标列表
}
