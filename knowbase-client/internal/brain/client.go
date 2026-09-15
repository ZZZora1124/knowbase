// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package brain

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"icode.baidu.com/baidu/devops-ai/knowbase-client/pkg/types"
)

type BrainClient struct {
	http       *http.Client
	credential *types.Credential
	brainUUID  string
	baseURL    string
}

// NewBrainClient 创建 BrainClient（兼容旧接口）
func NewBrainClient(authToken, brainUUID string) *BrainClient {
	return &BrainClient{
		http: &http.Client{Timeout: 5 * time.Minute},
		credential: &types.Credential{
			Mode:  types.AuthModeComate,
			Token: authToken,
		},
		brainUUID: brainUUID,
		baseURL:   types.GetKnowbaseHost(),
	}
}

// NewBrainClientWithCredential 创建支持双模式认证的 BrainClient
func NewBrainClientWithCredential(cred *types.Credential, brainUUID string) *BrainClient {
	return &BrainClient{
		http:       &http.Client{Timeout: 5 * time.Minute},
		credential: cred,
		brainUUID:  brainUUID,
		baseURL:    types.GetKnowbaseHost(),
	}
}

func (c *BrainClient) buildURL(path string, args ...interface{}) string {
	if len(args) > 0 {
		path = fmt.Sprintf(path, args...)
	}
	return c.baseURL + path
}

// do 执行带鉴权 header 的 HTTP 请求
func (c *BrainClient) do(ctx context.Context, method, apiURL string, body []byte) ([]byte, error) {
	var bodyReader io.Reader
	hasBody := len(body) > 0
	if hasBody {
		bodyReader = bytes.NewReader(body)
	}

	// UGate 模式下需要改写 URL
	finalURL := apiURL
	if c.credential != nil && c.credential.Mode == types.AuthModeUgate {
		finalURL = c.rewriteURL(apiURL)
	}

	req, err := http.NewRequestWithContext(ctx, method, finalURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// 设置认证 header
	if c.credential != nil {
		if c.credential.Mode == types.AuthModeUgate {
			req.Header.Set("Ugate-Token", c.credential.Token)
		} else {
			req.Header.Set("x-ac-Authorization", c.credential.Token)
		}
	}
	if c.brainUUID != "" {
		req.Header.Set("x-kb-uuid", c.brainUUID)
	}
	if hasBody {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, truncate(string(respBody), 500))
	}

	return respBody, nil
}

// rewriteURL 根据 UGate 路由规则改写 URL
func (c *BrainClient) rewriteURL(originalURL string) string {
	u, err := url.Parse(originalURL)
	if err != nil {
		return originalURL
	}

	path := u.Path
	query := u.RawQuery

	var newPath string

	switch {
	case strings.HasPrefix(path, "/open-api/v1/"):
		newPath = "/devops/openapi/proxy/knowledge" + path
	default:
		return originalURL
	}

	newURL := types.UgateProxyHost + newPath
	if query != "" {
		newURL += "?" + query
	}
	return newURL
}

func truncate(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max]) + "..."
}

// PutPage 写入/更新 brain page，触发服务端完整副作用链
func (c *BrainClient) PutPage(ctx context.Context, slug, content string) (*PutPageResponse, error) {
	req := PutPageRequest{Slug: slug, Content: content}
	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	body, err := c.do(ctx, http.MethodPut, c.buildURL(types.PathBrainPage), data)
	if err != nil {
		return nil, err
	}

	var result PutPageResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}
	return &result, nil
}

// GetPage 读取单页全文
func (c *BrainClient) GetPage(ctx context.Context, slug string) (*PageResult, error) {
	apiURL := c.buildURL(types.PathBrainPage) + "/" + url.PathEscape(slug)
	body, err := c.do(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, err
	}

	var result PageResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}
	return &result, nil
}

// DeletePage 删除 brain page
func (c *BrainClient) DeletePage(ctx context.Context, slug string) (*DeletePageResponse, error) {
	apiURL := c.buildURL(types.PathBrainPage) + "/" + url.PathEscape(slug)
	body, err := c.do(ctx, http.MethodDelete, apiURL, nil)
	if err != nil {
		return nil, err
	}

	var result DeletePageResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}
	return &result, nil
}

// Search 关键词搜索
func (c *BrainClient) Search(ctx context.Context, q string, limit int) ([]SearchResult, error) {
	params := url.Values{}
	params.Set("q", q)
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	body, err := c.do(ctx, http.MethodGet, c.buildURL(types.PathBrainSearch)+"?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}

	var results []SearchResult
	if err := json.Unmarshal(body, &results); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}
	return results, nil
}

// Query 混合检索召回（brain-ops 主链路）
func (c *BrainClient) Query(ctx context.Context, q string, limit int, expand bool, detail string) ([]SearchResult, error) {
	params := url.Values{}
	params.Set("q", q)
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	params.Set("expand", strconv.FormatBool(expand))
	if detail != "" {
		params.Set("detail", detail)
	}
	body, err := c.do(ctx, http.MethodGet, c.buildURL(types.PathBrainQuery)+"?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}

	var results []SearchResult
	if err := json.Unmarshal(body, &results); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}
	return results, nil
}

// ListPages 枚举页面
func (c *BrainClient) ListPages(ctx context.Context, pageType, tag string, limit, offset int) ([]ListPageItem, error) {
	params := url.Values{}
	if pageType != "" {
		params.Set("type", pageType)
	}
	if tag != "" {
		params.Set("tag", tag)
	}
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	if offset > 0 {
		params.Set("offset", strconv.Itoa(offset))
	}
	body, err := c.do(ctx, http.MethodGet, c.buildURL(types.PathBrainPage)+"?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}

	var results []ListPageItem
	if err := json.Unmarshal(body, &results); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}
	return results, nil
}

// listAllPagesPageSize 单页大小取服务端最大值，减少往返。
const listAllPagesPageSize = 100

// listAllPagesMaxPages 兜底页数，防止服务端排序异常时死循环（10w 条远超合理规模）。
// 用 var 而非 const 以便测试中临时改小，便于覆盖兜底分支。
var listAllPagesMaxPages = 1000

// ListAllPages 循环调用 ListPages，按 offset 翻页直至末页（返回数 < pageSize）。
// 用于需要全量页面的场景（如 canonical 索引重建、brain list 默认行为）。
func (c *BrainClient) ListAllPages(ctx context.Context, pageType, tag string) ([]ListPageItem, error) {
	var all []ListPageItem
	for i := 0; i < listAllPagesMaxPages; i++ {
		offset := i * listAllPagesPageSize
		page, err := c.ListPages(ctx, pageType, tag, listAllPagesPageSize, offset)
		if err != nil {
			return nil, fmt.Errorf("list pages (offset=%d): %w", offset, err)
		}
		all = append(all, page...)
		if len(page) < listAllPagesPageSize {
			return all, nil
		}
	}
	return all, fmt.Errorf("list pages: exceeded max pages (%d)", listAllPagesMaxPages)
}

// GraphQuery 图谱遍历
func (c *BrainClient) GraphQuery(ctx context.Context, slug, linkType string, depth int, direction string) ([]GraphEdge, error) {
	params := url.Values{}
	if linkType != "" {
		params.Set("type", linkType)
	}
	if depth > 0 {
		params.Set("depth", strconv.Itoa(depth))
	}
	if direction != "" {
		params.Set("direction", direction)
	}
	body, err := c.do(ctx, http.MethodGet, c.buildURL(types.PathBrainGraph, url.PathEscape(slug))+"?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}

	var results []GraphEdge
	if err := json.Unmarshal(body, &results); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}
	return results, nil
}

// GetTimeline 读取时间线
func (c *BrainClient) GetTimeline(ctx context.Context, slug, since, until string, limit int) ([]TimelineEntry, error) {
	params := url.Values{}
	if since != "" {
		params.Set("since", since)
	}
	if until != "" {
		params.Set("until", until)
	}
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	body, err := c.do(ctx, http.MethodGet, c.buildURL(types.PathBrainTimeline, url.PathEscape(slug))+"?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}

	var results []TimelineEntry
	if err := json.Unmarshal(body, &results); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}
	return results, nil
}

// GetBacklinks 获取反向链接
func (c *BrainClient) GetBacklinks(ctx context.Context, slug string) ([]BacklinkResult, error) {
	body, err := c.do(ctx, http.MethodGet, c.buildURL(types.PathBrainBacklinks, url.PathEscape(slug)), nil)
	if err != nil {
		return nil, err
	}

	var results []BacklinkResult
	if err := json.Unmarshal(body, &results); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}
	return results, nil
}

// AddTimelineEntry 追加时间线条目（调用专用POST接口）
func (c *BrainClient) AddTimelineEntry(ctx context.Context, slug, date, summary, detail, source string) (*AddTimelineEntryResponse, error) {
	if err := validateDate(date); err != nil {
		return nil, err
	}

	req := AddTimelineEntryRequest{
		Date:    date,
		Summary: summary,
		Detail:  detail,
		Source:  source,
	}
	payload, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	body, err := c.do(ctx, http.MethodPost, c.buildURL(types.PathBrainTimeline, url.PathEscape(slug)), payload)
	if err != nil {
		return nil, err
	}

	var resp AddTimelineEntryResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}
	return &resp, nil
}

// validateDate 验证日期格式为 YYYY-MM-DD 且为有效日历日期
func validateDate(date string) error {
	if _, err := time.Parse("2006-01-02", date); err != nil {
		return fmt.Errorf("date must be in YYYY-MM-DD format, got %q", date)
	}
	return nil
}

// GetDefaultModel 从服务端拉取默认模型名
func (c *BrainClient) GetDefaultModel(ctx context.Context) (string, error) {
	body, err := c.do(ctx, http.MethodGet, c.buildURL(types.PathBrainConfigDefaultModel), nil)
	if err != nil {
		return "", err
	}
	var resp DefaultModelResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("unmarshal response: %w", err)
	}
	return resp.Model, nil
}
