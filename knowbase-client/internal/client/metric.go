// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package client

import (
	"context"
	"encoding/json"
	"icode.baidu.com/baidu/devops-ai/knowbase-client/pkg/types"
	"time"
)

// MetricClient 指标上报客户端
type MetricClient struct {
	httpClient *HTTPClient
	apiURL     string
}

// NewMetricClient 创建指标上报客户端（兼容旧接口）
func NewMetricClient(authToken string) *MetricClient {
	host := types.GetKnowbaseHost()
	apiURL := host + types.PathMetricsDownload

	return &MetricClient{
		httpClient: NewHTTPClient(authToken),
		apiURL:     apiURL,
	}
}

// NewMetricClientWithCredential 创建支持双模式认证的指标上报客户端
func NewMetricClientWithCredential(cred *types.Credential) *MetricClient {
	host := types.GetKnowbaseHost()
	apiURL := host + types.PathMetricsDownload

	return &MetricClient{
		httpClient: NewHTTPClientWithCredential(cred),
		apiURL:     apiURL,
	}
}

// ReportDownloadMetrics 同步批量上报下载指标
func (c *MetricClient) ReportDownloadMetrics(ctx context.Context, results []types.LoadResult) error {
	if len(results) == 0 {
		return nil
	}

	// 转换为上报格式
	metrics := make([]types.MetricDownloadRequest, 0, len(results))
	for _, result := range results {
		metric := types.MetricDownloadRequest{
			SourceType: result.SourceType,
			Count:      result.Count,
			DurationMs: result.Duration.Milliseconds(),
			Success:    result.Success,
		}
		if !result.Success && result.ErrorMessage != "" {
			metric.ErrorMessage = truncateString(result.ErrorMessage, 500)
		}
		metrics = append(metrics, metric)
	}

	// 构造批量上报请求
	batchRequest := types.MetricDownloadBatchRequest{
		Metrics: metrics,
	}

	// 序列化为JSON
	data, err := json.Marshal(batchRequest)
	if err != nil {
		// 静默处理序列化错误
		return nil
	}

	// 创建带超时的上下文
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	// 发送请求，忽略所有错误
	_, _ = c.httpClient.Post(ctxWithTimeout, c.apiURL, data)

	return nil
}

// ReportDownloadMetric 同步上报单个下载指标
func (c *MetricClient) ReportDownloadMetric(ctx context.Context, result types.LoadResult) error {
	return c.ReportDownloadMetrics(ctx, []types.LoadResult{result})
}

// truncateString 截断字符串到指定长度
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}
