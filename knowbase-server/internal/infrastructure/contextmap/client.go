// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package contextmap

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/config"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/log"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/utils"
)

const (
	defaultTimeout    = 30 * time.Second
	defaultRetryCount = 3
	headerAuth        = "Authorization"

	// HeaderKbUUID x-kb-uuid 请求头名称（供外部使用）
	HeaderKbUUID = "x-kb-uuid"
)

// ContextMapClient context-map 服务客户端接口
type ContextMapClient interface {
	// PutPage 写入/更新 brain page，触发完整副作用链
	PutPage(ctx context.Context, kbUUID, slug, content string) ([]byte, int, error)
	// GetPage 读取单页全文
	GetPage(ctx context.Context, kbUUID, slug string, fuzzy bool) ([]byte, int, error)
	// SearchPages 关键词搜索
	SearchPages(ctx context.Context, kbUUID, q string, limit, offset int) ([]byte, int, error)
	// QueryPages 混合检索
	QueryPages(ctx context.Context, kbUUID, q string, limit int, expand bool, detail string) ([]byte, int, error)
	// ListPages 枚举页面
	ListPages(ctx context.Context, kbUUID, pageType, tag string, limit, offset int) ([]byte, int, error)
	// GetBacklinks 获取反向链接
	GetBacklinks(ctx context.Context, kbUUID, slug string) ([]byte, int, error)
	// TraverseGraph 图谱遍历
	TraverseGraph(ctx context.Context, kbUUID, slug, linkType, direction string, depth int) ([]byte, int, error)
	// GetTimeline 读取时间线
	GetTimeline(ctx context.Context, kbUUID, slug, since, until string, limit int) ([]byte, int, error)
	// AddTimelineEntry 追加单条时间线事件（不触发分块/embedding/autolink）
	AddTimelineEntry(ctx context.Context, kbUUID, slug, date, summary, detail, source string) ([]byte, int, error)
	// DeletePage 删除单页（透传 context-map 删除链路）
	DeletePage(ctx context.Context, kbUUID, slug string) ([]byte, int, error)
}

// client context-map HTTP 客户端实现
type client struct {
	baseURL       string
	platformToken string
	httpClient    *http.Client
	retryCount    int           // 最大重试次数（不含首次请求），0 表示不重试
	retryDelay    time.Duration // 指数退避初始间隔
}

var (
	defaultClient     *client
	defaultClientOnce sync.Once
)

// GetClient 获取全局 context-map 客户端单例
func GetClient() ContextMapClient {
	defaultClientOnce.Do(func() {
		cfg, err := config.GetContextMapConfig()
		if err != nil {
			log.Logger.Warnf("context-map 配置未找到，brain 功能不可用: %v", err)
			return
		}
		defaultClient = newClient(cfg)
	})
	return defaultClient
}

// newClient 创建 context-map 客户端
func newClient(cfg *config.ContextMapConfig) *client {
	timeout := time.Duration(cfg.Timeout) * time.Second
	if timeout <= 0 {
		timeout = defaultTimeout
	}

	retryCount := cfg.RetryCount
	if retryCount < 0 {
		retryCount = defaultRetryCount
	}

	retryDelay := time.Duration(cfg.RetryDelayMs) * time.Millisecond
	if retryDelay <= 0 {
		retryDelay = 200 * time.Millisecond
	}

	return &client{
		baseURL:       cfg.BaseURL,
		platformToken: cfg.PlatformToken,
		httpClient:    &http.Client{Timeout: timeout},
		retryCount:    retryCount,
		retryDelay:    retryDelay,
	}
}

// do 执行 HTTP 请求，统一注入鉴权和 x-kb-uuid 头，支持带 context 感知的指数退避重试
// 重试条件：网络错误（err != nil）或 5xx 响应；4xx 不重试
func (c *client) do(ctx context.Context, method, path, kbUUID string, body interface{}) ([]byte, int, error) {
	// 序列化 body（仅一次，重试时复用字节）
	var bodyData []byte
	if body != nil {
		var err error
		bodyData, err = json.Marshal(body)
		if err != nil {
			return nil, 0, fmt.Errorf("序列化请求体失败: %w", err)
		}
	}

	backOff := utils.GetRetryBackOff()

	var (
		lastBody       []byte
		lastStatusCode int
		lastErr        error
	)

	// 总共最多执行 retryCount+1 次（1次原始 + retryCount次重试）
	for attempt := 0; attempt <= c.retryCount; attempt++ {
		if attempt > 0 {
			delay := backOff.NextBackOff()
			log.Logger.Warnf("context-map 请求失败，第 %d 次重试: path=%s, status=%d, err=%v, delay=%v",
				attempt, path, lastStatusCode, lastErr, delay)
			// context 感知等待，避免已取消的请求继续等待
			select {
			case <-ctx.Done():
				return nil, 0, ctx.Err()
			case <-time.After(delay):
			}
		}

		var reqBody io.Reader
		if bodyData != nil {
			reqBody = bytes.NewBuffer(bodyData)
		}

		req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reqBody)
		if err != nil {
			// 请求构建失败不重试
			return nil, 0, fmt.Errorf("创建请求失败: %w", err)
		}

		req.Header.Set(headerAuth, c.platformToken)
		req.Header.Set(HeaderKbUUID, kbUUID)
		if bodyData != nil {
			req.Header.Set("Content-Type", "application/json")
		}

		log.Logger.InfofCtx(ctx, "context-map request: method=%s url=%s kbUUID=%s body=%s",
			method, c.baseURL+path, kbUUID, string(bodyData))

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("请求 context-map 失败: %w", err)
			lastStatusCode = 0
			lastBody = nil
			continue // 网络错误，重试
		}

		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = fmt.Errorf("读取响应体失败: %w", err)
			lastStatusCode = resp.StatusCode
			lastBody = nil
			continue // 读取失败，重试
		}

		log.Logger.InfofCtx(ctx, "context-map response: method=%s url=%s status=%d body=%s",
			method, c.baseURL+path, resp.StatusCode, string(respBody))

		lastBody = respBody
		lastStatusCode = resp.StatusCode
		lastErr = nil

		if resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("context-map 返回 %d", resp.StatusCode)
			continue // 5xx，重试
		}

		// 成功（包含 2xx / 3xx / 4xx），直接返回，不再重试
		return respBody, resp.StatusCode, nil
	}

	return lastBody, lastStatusCode, lastErr
}

// PutPage 写入/更新 brain page
func (c *client) PutPage(ctx context.Context, kbUUID, slug, content string) ([]byte, int, error) {
	body := map[string]string{
		"slug":    slug,
		"content": content,
	}
	return c.do(ctx, http.MethodPut, "/open-api/v1/brain/page", kbUUID, body)
}

// GetPage 读取单页全文
func (c *client) GetPage(ctx context.Context, kbUUID, slug string, fuzzy bool) ([]byte, int, error) {
	path := fmt.Sprintf("/open-api/v1/brain/page/%s", url.PathEscape(slug))
	if fuzzy {
		path += "?fuzzy=true"
	}
	return c.do(ctx, http.MethodGet, path, kbUUID, nil)
}

// SearchPages 关键词搜索
func (c *client) SearchPages(ctx context.Context, kbUUID, q string, limit, offset int) ([]byte, int, error) {
	path := fmt.Sprintf("/open-api/v1/brain/search?q=%s&limit=%d&offset=%d",
		url.QueryEscape(q), limit, offset)
	return c.do(ctx, http.MethodGet, path, kbUUID, nil)
}

// QueryPages 混合检索
func (c *client) QueryPages(ctx context.Context, kbUUID, q string, limit int, expand bool, detail string) ([]byte, int, error) {
	expandStr := "false"
	if expand {
		expandStr = "true"
	}
	path := fmt.Sprintf("/open-api/v1/brain/query?q=%s&limit=%d&expand=%s&detail=%s",
		url.QueryEscape(q), limit, expandStr, url.QueryEscape(detail))
	return c.do(ctx, http.MethodGet, path, kbUUID, nil)
}

// ListPages 枚举页面
func (c *client) ListPages(ctx context.Context, kbUUID, pageType, tag string, limit, offset int) ([]byte, int, error) {
	path := fmt.Sprintf("/open-api/v1/brain/page?type=%s&tag=%s&limit=%d&offset=%d",
		url.QueryEscape(pageType), url.QueryEscape(tag), limit, offset)
	return c.do(ctx, http.MethodGet, path, kbUUID, nil)
}

// GetBacklinks 获取反向链接
func (c *client) GetBacklinks(ctx context.Context, kbUUID, slug string) ([]byte, int, error) {
	path := fmt.Sprintf("/open-api/v1/brain/backlinks/%s", url.PathEscape(slug))
	return c.do(ctx, http.MethodGet, path, kbUUID, nil)
}

// TraverseGraph 图谱遍历
func (c *client) TraverseGraph(ctx context.Context, kbUUID, slug, linkType, direction string, depth int) ([]byte, int, error) {
	path := fmt.Sprintf("/open-api/v1/brain/graph/%s?type=%s&direction=%s&depth=%d",
		url.PathEscape(slug), url.QueryEscape(linkType), url.QueryEscape(direction), depth)
	return c.do(ctx, http.MethodGet, path, kbUUID, nil)
}

// GetTimeline 读取时间线
func (c *client) GetTimeline(ctx context.Context, kbUUID, slug, since, until string, limit int) ([]byte, int, error) {
	path := fmt.Sprintf("/open-api/v1/brain/timeline/%s?since=%s&until=%s&limit=%d",
		url.PathEscape(slug), url.QueryEscape(since), url.QueryEscape(until), limit)
	return c.do(ctx, http.MethodGet, path, kbUUID, nil)
}

// AddTimelineEntry 追加单条时间线事件
func (c *client) AddTimelineEntry(ctx context.Context, kbUUID, slug, date, summary, detail, source string) ([]byte, int, error) {
	body := map[string]string{
		"date":    date,
		"summary": summary,
		"detail":  detail,
		"source":  source,
	}
	path := fmt.Sprintf("/open-api/v1/brain/timeline/%s", url.PathEscape(slug))
	return c.do(ctx, http.MethodPost, path, kbUUID, body)
}

// DeletePage 删除单页
func (c *client) DeletePage(ctx context.Context, kbUUID, slug string) ([]byte, int, error) {
	path := fmt.Sprintf("/open-api/v1/brain/page/%s", url.PathEscape(slug))
	return c.do(ctx, http.MethodDelete, path, kbUUID, nil)
}
