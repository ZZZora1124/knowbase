// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"go.uber.org/zap"
	"io"
	"net/http"
	"net/url"
	"time"

	"icode.baidu.com/baidu/devops-ai/knowbase-client/pkg/types"
)

// AuthValidateResponse 认证验证响应
type AuthValidateResponse struct {
	XRequestID string `json:"X-Request-ID"`
	Code       int    `json:"code"`
	Msg        string `json:"msg"`
	Data       struct {
		Username string `json:"username"`
		ClientID string `json:"client_id"`
	} `json:"data"`
}

// ValidateToken 验证 token 并获取用户名
// 使用内置的认证服务配置
func ValidateToken(authToken string) (string, error) {
	if authToken == "" {
		return "", fmt.Errorf("access token is empty")
	}

	// 构造请求 URL
	baseURL := types.GetAuthHost() + types.AuthenticationAPIPath
	params := url.Values{}
	params.Add("clientId", types.GetAuthClientKey())
	params.Add("clientSecret", types.GetAuthClientValue())
	params.Add("accessToken", authToken)

	requestURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())

	// 创建 HTTP 客户端（设置 30 秒超时）
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// 创建请求上下文
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send authentication request: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read authentication response: %w", err)
	}

	// 检查 HTTP 状态码
	if resp.StatusCode != http.StatusOK {
		if logger != nil {
			logger.Debug("Authentication HTTP error",
				zap.Int("statusCode", resp.StatusCode),
				zap.String("response", string(body)))
		}
		return "", fmt.Errorf("authentication service returned status %d: %s", resp.StatusCode, string(body))
	}

	// 解析 JSON 响应
	var authResp AuthValidateResponse
	if err := json.Unmarshal(body, &authResp); err != nil {
		return "", fmt.Errorf("failed to parse authentication response: %w", err)
	}

	// 检查业务返回码
	if authResp.Code != 200 {
		return "", fmt.Errorf("authentication failed (code %d): %s", authResp.Code, authResp.Msg)
	}

	// 检查用户名是否为空
	if authResp.Data.Username == "" {
		return "", fmt.Errorf("authentication response does not contain username")
	}

	return authResp.Data.Username, nil
}
