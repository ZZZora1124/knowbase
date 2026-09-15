// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package client

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"icode.baidu.com/baidu/devops-ai/knowbase-client/internal/utils"
	"icode.baidu.com/baidu/devops-ai/knowbase-client/pkg/types"
)

// HTTPClient HTTP客户端封装
type HTTPClient struct {
	client     *http.Client
	credential *types.Credential
	// 兼容旧接口，仅在 credential 为 nil 时使用
	authToken string
}

// NewHTTPClient 创建HTTP客户端（兼容旧接口）
func NewHTTPClient(authToken string) *HTTPClient {
	return &HTTPClient{
		client: &http.Client{
			Timeout: 5 * time.Minute,
		},
		authToken: authToken,
	}
}

// NewHTTPClientWithCredential 创建支持双模式认证的HTTP客户端
func NewHTTPClientWithCredential(cred *types.Credential) *HTTPClient {
	return &HTTPClient{
		client: &http.Client{
			Timeout: 5 * time.Minute,
		},
		credential: cred,
	}
}

// getAuthInfo 获取认证信息（header key, header value, 是否需要 URL 改写）
func (c *HTTPClient) getAuthInfo() (headerKey, headerValue string, needRewrite bool) {
	if c.credential != nil {
		if c.credential.Mode == types.AuthModeUgate {
			return "Ugate-Token", c.credential.Token, true
		}
		return "x-ac-Authorization", c.credential.Token, false
	}
	// 兼容旧接口
	return "x-ac-Authorization", c.authToken, false
}

// rewriteURL 根据 UGate 路由规则改写 URL
func (c *HTTPClient) rewriteURL(originalURL string) string {
	u, err := url.Parse(originalURL)
	if err != nil {
		return originalURL
	}

	path := u.Path
	query := u.RawQuery

	var newPath string

	switch {
	// /api/process/ku/ku/openapi/* → /wiki/so/ku/openapi/*
	case strings.HasPrefix(path, types.UgateKuOpenAPIPrefix):
		newPath = types.UgateKuProxyPrefix + strings.TrimPrefix(path, types.UgateKuOpenAPIPrefix)

	// /api/process/* → /devops/openapi/proxy + 剩余路径
	case strings.HasPrefix(path, "/api/process/"):
		newPath = "/devops/openapi/proxy" + strings.TrimPrefix(path, "/api/process")

	// /open-api/v1/* → /devops/openapi/proxy/knowledge + 原路径
	case strings.HasPrefix(path, "/open-api/v1/"):
		newPath = "/devops/openapi/proxy/knowledge" + path

	// /api/v1/infoflow/* → /devops/openapi/proxy/knowledge + 原路径
	case strings.HasPrefix(path, "/api/v1/infoflow/"):
		newPath = "/devops/openapi/proxy/knowledge" + path

	// /api/v1/metrics/* → /devops/openapi/proxy/knowledge + 原路径
	case strings.HasPrefix(path, "/api/v1/metrics/"):
		newPath = "/devops/openapi/proxy/knowledge" + path

	// /api/v1/iapi/* → 不走代理，保持原样
	case strings.HasPrefix(path, "/api/v1/iapi/"):
		return originalURL

	// 其他路径保持原样
	default:
		return originalURL
	}

	// 构建新 URL
	newURL := types.UgateProxyHost + newPath
	if query != "" {
		newURL += "?" + query
	}
	return newURL
}

func isUgateKuOpenAPIRequest(originalURL string) bool {
	u, err := url.Parse(originalURL)
	if err != nil {
		return false
	}
	return strings.HasPrefix(u.Path, types.UgateKuOpenAPIPrefix)
}

// prepareRequest 准备请求（处理 URL 改写和认证 header）
func (c *HTTPClient) prepareRequest(ctx context.Context, method, originalURL string, body io.Reader) (*http.Request, error) {
	headerKey, headerValue, needRewrite := c.getAuthInfo()

	finalURL := originalURL
	if needRewrite {
		finalURL = c.rewriteURL(originalURL)
	}

	req, err := http.NewRequestWithContext(ctx, method, finalURL, body)
	if err != nil {
		return nil, err
	}

	if headerValue != "" {
		req.Header.Set(headerKey, headerValue)
	}
	if c.credential != nil && c.credential.Mode == types.AuthModeUgate && isUgateKuOpenAPIRequest(originalURL) {
		req.Header.Set(types.UgateKuAccessKeyHeader, types.UgateKuAccessKey)
	}

	return req, nil
}

// Get 发送GET请求
func (c *HTTPClient) Get(ctx context.Context, url string) ([]byte, error) {
	req, err := c.prepareRequest(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, types.NewDetailedError(types.ErrorTypeNetwork, "failed to create request").
			WithDetail("url", url).
			WithDetail("rawError", err.Error())
	}

	// 发送请求
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, types.NewDetailedError(types.ErrorTypeNetwork, "failed to send request").
			WithDetail("url", url).
			WithDetail("rawError", err.Error()).
			WithSuggestion(utils.GetNetworkSuggestion(err))
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, types.NewDetailedError(types.ErrorTypeNetwork, "failed to read response body").
			WithDetail("url", url).
			WithDetail("statusCode", fmt.Sprintf("%d", resp.StatusCode)).
			WithDetail("rawError", err.Error())
	}

	// 检查状态码
	if resp.StatusCode != http.StatusOK {
		return nil, types.NewDetailedError(types.ErrorTypeAPI, "unexpected status code").
			WithDetail("url", url).
			WithDetail("statusCode", fmt.Sprintf("%d", resp.StatusCode)).
			WithDetail("response", utils.TruncateString(string(body), 500)).
			WithSuggestion(utils.GetHTTPSuggestion(resp.StatusCode, string(body)))
	}

	return body, nil
}

// Post 发送POST请求
func (c *HTTPClient) Post(ctx context.Context, url string, data []byte) ([]byte, error) {
	req, err := c.prepareRequest(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return nil, types.NewDetailedError(types.ErrorTypeNetwork, "failed to create request").
			WithDetail("url", url).
			WithDetail("rawError", err.Error())
	}

	// 添加Content-Type头
	req.Header.Set("Content-Type", "application/json")

	// 发送请求
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, types.NewDetailedError(types.ErrorTypeNetwork, "failed to send request").
			WithDetail("url", url).
			WithDetail("rawError", err.Error()).
			WithSuggestion(utils.GetNetworkSuggestion(err))
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, types.NewDetailedError(types.ErrorTypeNetwork, "failed to read response body").
			WithDetail("url", url).
			WithDetail("statusCode", fmt.Sprintf("%d", resp.StatusCode)).
			WithDetail("rawError", err.Error())
	}

	// 检查状态码
	if resp.StatusCode != http.StatusOK {
		return nil, types.NewDetailedError(types.ErrorTypeAPI, "unexpected status code").
			WithDetail("url", url).
			WithDetail("statusCode", fmt.Sprintf("%d", resp.StatusCode)).
			WithDetail("response", utils.TruncateString(string(body), 500)).
			WithSuggestion(utils.GetHTTPSuggestion(resp.StatusCode, string(body)))
	}

	return body, nil
}

// Put 发送PUT请求
func (c *HTTPClient) Put(ctx context.Context, url string, data []byte) ([]byte, error) {
	req, err := c.prepareRequest(ctx, http.MethodPut, url, bytes.NewReader(data))
	if err != nil {
		return nil, types.NewDetailedError(types.ErrorTypeNetwork, "failed to create request").
			WithDetail("url", url).
			WithDetail("rawError", err.Error())
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, types.NewDetailedError(types.ErrorTypeNetwork, "failed to send request").
			WithDetail("url", url).
			WithDetail("rawError", err.Error()).
			WithSuggestion(utils.GetNetworkSuggestion(err))
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, types.NewDetailedError(types.ErrorTypeNetwork, "failed to read response body").
			WithDetail("url", url).
			WithDetail("statusCode", fmt.Sprintf("%d", resp.StatusCode)).
			WithDetail("rawError", err.Error())
	}

	if resp.StatusCode != http.StatusOK {
		return nil, types.NewDetailedError(types.ErrorTypeAPI, "unexpected status code").
			WithDetail("url", url).
			WithDetail("statusCode", fmt.Sprintf("%d", resp.StatusCode)).
			WithDetail("response", utils.TruncateString(string(body), 500)).
			WithSuggestion(utils.GetHTTPSuggestion(resp.StatusCode, string(body)))
	}

	return body, nil
}

// Delete 发送DELETE请求（支持请求体）
func (c *HTTPClient) Delete(ctx context.Context, url string, data []byte) ([]byte, error) {
	var bodyReader io.Reader
	if data != nil {
		bodyReader = bytes.NewReader(data)
	}

	req, err := c.prepareRequest(ctx, http.MethodDelete, url, bodyReader)
	if err != nil {
		return nil, types.NewDetailedError(types.ErrorTypeNetwork, "failed to create request").
			WithDetail("url", url).
			WithDetail("rawError", err.Error())
	}

	if data != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, types.NewDetailedError(types.ErrorTypeNetwork, "failed to send request").
			WithDetail("url", url).
			WithDetail("rawError", err.Error()).
			WithSuggestion(utils.GetNetworkSuggestion(err))
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, types.NewDetailedError(types.ErrorTypeNetwork, "failed to read response body").
			WithDetail("url", url).
			WithDetail("statusCode", fmt.Sprintf("%d", resp.StatusCode)).
			WithDetail("rawError", err.Error())
	}

	if resp.StatusCode != http.StatusOK {
		return nil, types.NewDetailedError(types.ErrorTypeAPI, "unexpected status code").
			WithDetail("url", url).
			WithDetail("statusCode", fmt.Sprintf("%d", resp.StatusCode)).
			WithDetail("response", utils.TruncateString(string(body), 500)).
			WithSuggestion(utils.GetHTTPSuggestion(resp.StatusCode, string(body)))
	}

	return body, nil
}

// PostMultipart 发送multipart/form-data POST请求
func (c *HTTPClient) PostMultipart(ctx context.Context, url string, fields map[string]string,
	fileField string, filePath string) ([]byte, error) {

	bodyBuf := &bytes.Buffer{}
	writer := multipart.NewWriter(bodyBuf)

	for key, value := range fields {
		if err := writer.WriteField(key, value); err != nil {
			return nil, types.NewDetailedError(types.ErrorTypeNetwork, "failed to write form field").
				WithDetail("field", key).
				WithDetail("rawError", err.Error())
		}
	}

	if fileField != "" && filePath != "" {
		file, err := os.Open(filePath)
		if err != nil {
			return nil, types.NewDetailedError(types.ErrorTypeStorage, "failed to open file").
				WithDetail("filePath", filePath).
				WithDetail("rawError", err.Error()).
				WithSuggestion("请检查文件路径是否正确")
		}
		defer file.Close()

		stat, err := file.Stat()
		if err != nil {
			return nil, types.NewDetailedError(types.ErrorTypeStorage, "failed to stat file").
				WithDetail("filePath", filePath).
				WithDetail("rawError", err.Error())
		}
		if stat.Size() > types.MaxFileUploadSize {
			return nil, types.NewDetailedError(types.ErrorTypeValidation, "file too large").
				WithDetail("filePath", filePath).
				WithDetail("fileSize", fmt.Sprintf("%d", stat.Size())).
				WithDetail("maxSize", fmt.Sprintf("%d", types.MaxFileUploadSize)).
				WithSuggestion("文件大小超过500MB限制，请压缩或拆分文件")
		}

		part, err := writer.CreateFormFile(fileField, filepath.Base(filePath))
		if err != nil {
			return nil, types.NewDetailedError(types.ErrorTypeNetwork, "failed to create form file").
				WithDetail("rawError", err.Error())
		}
		if _, err := io.Copy(part, file); err != nil {
			return nil, types.NewDetailedError(types.ErrorTypeNetwork, "failed to copy file content").
				WithDetail("rawError", err.Error())
		}
	}

	if err := writer.Close(); err != nil {
		return nil, types.NewDetailedError(types.ErrorTypeNetwork, "failed to close multipart writer").
			WithDetail("rawError", err.Error())
	}

	req, err := c.prepareRequest(ctx, http.MethodPost, url, bodyBuf)
	if err != nil {
		return nil, types.NewDetailedError(types.ErrorTypeNetwork, "failed to create request").
			WithDetail("url", url).
			WithDetail("rawError", err.Error())
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, types.NewDetailedError(types.ErrorTypeNetwork, "failed to send request").
			WithDetail("url", url).
			WithDetail("rawError", err.Error()).
			WithSuggestion(utils.GetNetworkSuggestion(err))
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, types.NewDetailedError(types.ErrorTypeNetwork, "failed to read response body").
			WithDetail("url", url).
			WithDetail("statusCode", fmt.Sprintf("%d", resp.StatusCode)).
			WithDetail("rawError", err.Error())
	}

	if resp.StatusCode != http.StatusOK {
		return nil, types.NewDetailedError(types.ErrorTypeAPI, "unexpected status code").
			WithDetail("url", url).
			WithDetail("statusCode", fmt.Sprintf("%d", resp.StatusCode)).
			WithDetail("response", utils.TruncateString(string(body), 500)).
			WithSuggestion(utils.GetHTTPSuggestion(resp.StatusCode, string(body)))
	}

	return body, nil
}

// Download 下载文件
func (c *HTTPClient) Download(ctx context.Context, url string, writer io.Writer) error {
	req, err := c.prepareRequest(ctx, http.MethodGet, url, nil)
	if err != nil {
		return types.NewDetailedError(types.ErrorTypeNetwork, "failed to create request").
			WithDetail("url", url).
			WithDetail("rawError", err.Error())
	}

	// 发送请求
	resp, err := c.client.Do(req)
	if err != nil {
		return types.NewDetailedError(types.ErrorTypeNetwork, "failed to send request").
			WithDetail("url", url).
			WithDetail("rawError", err.Error()).
			WithSuggestion(utils.GetNetworkSuggestion(err))
	}
	defer resp.Body.Close()

	// 检查状态码
	if resp.StatusCode != http.StatusOK {
		// 读取响应体以获取错误详情
		body, _ := io.ReadAll(resp.Body)
		return types.NewDetailedError(types.ErrorTypeAPI, "unexpected status code").
			WithDetail("url", url).
			WithDetail("statusCode", fmt.Sprintf("%d", resp.StatusCode)).
			WithDetail("response", utils.TruncateString(string(body), 500)).
			WithSuggestion(utils.GetHTTPSuggestion(resp.StatusCode, string(body)))
	}

	// 流式写入
	_, err = io.Copy(writer, resp.Body)
	if err != nil {
		return types.NewDetailedError(types.ErrorTypeNetwork, "failed to copy response body").
			WithDetail("url", url).
			WithDetail("rawError", err.Error()).
			WithSuggestion("下载失败，请检查磁盘空间和网络连接")
	}

	return nil
}

// GetContentLength 获取内容长度
func (c *HTTPClient) GetContentLength(ctx context.Context, url string) (int64, error) {
	req, err := c.prepareRequest(ctx, http.MethodHead, url, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	// 发送请求
	resp, err := c.client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	return resp.ContentLength, nil
}
