package utils

import (
	"fmt"
	"net/http"
	"time"

	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/log"
)

// AuthValidateResponse 第三方认证接口响应结构
type AuthValidateResponse struct {
	XRequestID string `json:"X-Request-ID"`
	Code       int    `json:"code"`
	Msg        string `json:"msg"`
	Data       struct {
		Username string `json:"username"`
		Platform string `json:"platform"`
	} `json:"data"`
}

// ValidateAccessToken 验证访问令牌的有效性
//
// 参数:
//
//	authURL: 认证服务URL
//	clientId: 客户端ID
//	clientSecret: 客户端密钥
//	accessToken: 需要验证的访问令牌
//	timeout: 请求超时时间(秒)
//	retryCount: 重试次数
//
// 返回值:
//
//	string: 验证成功返回用户名
//	string: 验证成功返回平台名
//	error: 验证失败返回错误信息
//
// 处理流程:
//  1. 构建认证请求URL
//  2. 发送HTTP请求到认证服务
//  3. 检查HTTP状态码和业务响应码
//  4. 验证用户名和平台名不为空
//  5. 返回验证结果
func ValidateAccessToken(authURL, clientId, clientSecret string, retryCount, timeout int, accessToken string) (string, string, error) {
	// 构建请求URL
	reqURL := fmt.Sprintf("%s/api/rest/v1/auth/validate?clientId=%s&clientSecret=%s&accessToken=%s",
		authURL, clientId, clientSecret, accessToken)

	log.Logger.Debugf("Calling authentication service: %s", reqURL)

	// 发送HTTP请求
	request := &Request{
		URL:         reqURL,
		Method:      "GET",
		ContentType: ContentTypeJSON,
		Timeout:     time.Duration(timeout) * time.Second,
		Retry:       retryCount,
	}

	var authResp AuthValidateResponse
	_, statusCode, err := SendRequest(request, &authResp)
	if err != nil {
		log.Logger.Errorf("Failed to call authentication service: %v", err)
		return "", "", fmt.Errorf("authentication service call failed: %w", err)
	}

	// 检查响应状态码
	if statusCode != http.StatusOK {
		log.Logger.Errorf("Authentication service returned non-200 status: %d", statusCode)
		return "", "", fmt.Errorf("authentication service returned status: %d", statusCode)
	}

	// 检查业务响应码
	if authResp.Code != 200 {
		log.Logger.Errorf("Authentication service returned business error: code=%d, msg=%s", authResp.Code, authResp.Msg)
		return "", "", fmt.Errorf("authentication failed: code=%d, msg=%s", authResp.Code, authResp.Msg)
	}

	// 检查用户名是否为空
	if authResp.Data.Username == "" {
		log.Logger.Errorf("Authentication service returned empty username")
		return "", "", fmt.Errorf("authentication succeeded but username is empty")
	}

	log.Logger.Debugf("Successfully validated access token for user: %s, platform: %s", authResp.Data.Username, authResp.Data.Platform)
	return authResp.Data.Username, authResp.Data.Platform, nil
}
