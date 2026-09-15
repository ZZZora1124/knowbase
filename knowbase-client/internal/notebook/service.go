// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package notebook

import (
	"encoding/json"
	"fmt"

	"icode.baidu.com/baidu/devops-ai/knowbase-client/internal/client"
	"icode.baidu.com/baidu/devops-ai/knowbase-client/internal/utils"
	"icode.baidu.com/baidu/devops-ai/knowbase-client/pkg/types"
)

// NotebookService 知识方舟管理服务
type NotebookService struct {
	HTTPClient *client.HTTPClient
	baseURL    string
}

// NewNotebookService 创建知识方舟管理服务（兼容旧接口）
func NewNotebookService(authToken string) *NotebookService {
	return &NotebookService{
		HTTPClient: client.NewHTTPClient(authToken),
		baseURL:    types.GetKnowbaseHost(),
	}
}

// NewNotebookServiceWithCredential 创建支持双模式认证的知识方舟管理服务
func NewNotebookServiceWithCredential(cred *types.Credential) *NotebookService {
	return &NotebookService{
		HTTPClient: client.NewHTTPClientWithCredential(cred),
		baseURL:    types.GetKnowbaseHost(),
	}
}

// BuildURL 构建完整API URL
func (s *NotebookService) BuildURL(path string, args ...interface{}) string {
	if len(args) > 0 {
		path = fmt.Sprintf(path, args...)
	}
	return s.baseURL + path
}

// ParseResponse 解析统一响应格式
func (s *NotebookService) ParseResponse(body []byte, result interface{}) error {
	var resp APIResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return types.NewDetailedError(types.ErrorTypeValidation, "failed to parse API response").
			WithDetail("rawError", err.Error()).
			WithDetail("responseBody", utils.TruncateString(string(body), 500)).
			WithSuggestion("API 响应格式错误，请稍后重试或联系管理员")
	}
	if resp.Code != 0 {
		return types.NewDetailedError(types.ErrorTypeAPI, "API returned error").
			WithDetail("code", fmt.Sprintf("%d", resp.Code)).
			WithDetail("message", resp.Message).
			WithSuggestion(utils.GetAPISuggestion(resp.Code))
	}
	if result != nil && resp.Data != nil {
		if err := json.Unmarshal(resp.Data, result); err != nil {
			return types.NewDetailedError(types.ErrorTypeValidation, "failed to parse response data").
				WithDetail("rawError", err.Error()).
				WithSuggestion("API 响应数据格式错误，请稍后重试或联系管理员")
		}
	}
	return nil
}
