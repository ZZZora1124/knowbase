// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package utils

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"icode.baidu.com/baidu/devops-ai/knowbase-client/pkg/types"
)

// MetricReporter 指标上报接口
type MetricReporter interface {
	ReportDownloadMetrics(ctx context.Context, results []types.LoadResult) error
}

// StatsCollector 统计收集器
type StatsCollector struct {
	stats               *types.LoadStats
	username            string // 当前用户名
	knowledgeStructures []types.KnowledgeStructure
	metricReporter      MetricReporter // 指标上报器
	mu                  sync.Mutex     // 保护并发访问
}

// NewStatsCollector 创建统计收集器
func NewStatsCollector(username string) *StatsCollector {
	return &StatsCollector{
		stats: &types.LoadStats{
			Results: make([]types.LoadResult, 0),
		},
		username:            username,
		knowledgeStructures: make([]types.KnowledgeStructure, 0),
		metricReporter:      nil, // 延迟初始化
	}
}

// SetMetricReporter 设置指标上报器
func (s *StatsCollector) SetMetricReporter(reporter MetricReporter) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.metricReporter = reporter
}

// AddResult 添加加载结果
func (s *StatsCollector) AddResult(result types.LoadResult) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.stats.Results = append(s.stats.Results, result)
	s.stats.TotalSources++
	s.stats.TotalItems += result.Count
	s.stats.TotalDuration += result.Duration

	if result.Success {
		s.stats.SuccessSources++
	} else {
		s.stats.FailedSources++
	}
}

// AddKnowledgeStructure 添加知识结构信息
func (s *StatsCollector) AddKnowledgeStructure(structure types.KnowledgeStructure) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.knowledgeStructures = append(s.knowledgeStructures, structure)
}

// GetKnowledgeStructures 获取所有知识结构信息
func (s *StatsCollector) GetKnowledgeStructures() []types.KnowledgeStructure {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 返回副本以避免并发修改
	result := make([]types.KnowledgeStructure, len(s.knowledgeStructures))
	copy(result, s.knowledgeStructures)
	return result
}

// SetActualDuration 设置实际总耗时（并行执行的墙钟时间）
func (s *StatsCollector) SetActualDuration(duration time.Duration) {
	s.stats.ActualDuration = duration
}

// GetStats 获取统计信息
func (s *StatsCollector) GetStats() *types.LoadStats {
	return s.stats
}

// PrintStats 打印统计信息
func (s *StatsCollector) PrintStats() {
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("知识加载统计报告")
	fmt.Println(strings.Repeat("=", 80))
	fmt.Printf("用户: %s\n", s.username)
	fmt.Printf("总知识源数: %d\n", s.stats.TotalSources)
	fmt.Printf("成功: %d | 失败: %d\n", s.stats.SuccessSources, s.stats.FailedSources)
	fmt.Printf("总知识项数: %d\n", s.stats.TotalItems)

	// 优先显示实际总耗时（并行执行的墙钟时间）
	if s.stats.ActualDuration > 0 {
		fmt.Printf("总耗时: %s\n", s.stats.ActualDuration.Round(time.Millisecond))
	} else {
		// 如果没有设置实际耗时，使用累计耗时
		fmt.Printf("总耗时: %s (累计)\n", s.stats.TotalDuration.Round(time.Millisecond))
	}

	fmt.Println(strings.Repeat("-", 80))

	for _, result := range s.stats.Results {
		status := "✓"
		if !result.Success {
			status = "✗"
		}
		fmt.Printf("[%s] %s - 数量: %d, 耗时: %s\n",
			status,
			result.SourceType,
			result.Count,
			result.Duration.Round(time.Millisecond))

		// 如果失败，展示详细错误信息
		if !result.Success {
			s.printErrorDetails(result)
		}
	}

	fmt.Println(strings.Repeat("=", 80))
}

// ReportMetrics 同步上报所有加载结果到服务端
func (s *StatsCollector) ReportMetrics() {
	s.mu.Lock()
	reporter := s.metricReporter
	results := make([]types.LoadResult, len(s.stats.Results))
	copy(results, s.stats.Results)
	s.mu.Unlock()

	// 如果没有配置上报器，直接返回
	if reporter == nil {
		return
	}

	// 同步批量上报所有结果（忽略所有错误）
	_ = reporter.ReportDownloadMetrics(context.Background(), results)
}

// printErrorDetails 打印错误详情
func (s *StatsCollector) printErrorDetails(result types.LoadResult) {
	// 缩进前缀
	indent := "    "

	// 错误类型
	if result.ErrorType != "" {
		fmt.Printf("%s错误类型: %s\n", indent, result.ErrorType)
	}

	// 错误信息
	if result.ErrorMessage != "" {
		fmt.Printf("%s错误信息: %s\n", indent, result.ErrorMessage)
	}

	// 详细信息
	if len(result.ErrorDetails) > 0 {
		fmt.Printf("%s详细信息:\n", indent)

		// 按关键字优先级排序展示
		priorityKeys := []string{"url", "statusCode", "message", "command", "output", "repo", "branch"}
		displayed := make(map[string]bool)

		for _, key := range priorityKeys {
			if value, ok := result.ErrorDetails[key]; ok && value != "" {
				fmt.Printf("%s  - %s: %s\n", indent, key, TruncateString(value, 200))
				displayed[key] = true
			}
		}

		// 展示其他详情
		for key, value := range result.ErrorDetails {
			if !displayed[key] && value != "" {
				fmt.Printf("%s  - %s: %s\n", indent, key, TruncateString(value, 200))
			}
		}
	}

	// 恢复建议
	if result.Suggestion != "" {
		fmt.Printf("%s💡 建议: %s\n", indent, result.Suggestion)
	}

	fmt.Println() // 空行分隔
}
