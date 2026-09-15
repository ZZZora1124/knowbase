// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package types

import "fmt"

// DetailedError 详细错误信息
type DetailedError struct {
	ErrorType    string            // 错误类型
	ErrorMessage string            // 错误信息（简短）
	Details      map[string]string // 详细信息
	Suggestion   string            // 恢复建议
}

func (e *DetailedError) Error() string {
	if e.Details != nil {
		for k, v := range e.Details {
			if k == "url" {
				continue
			}
			e.ErrorMessage += fmt.Sprintf("\n[%s]: %s", k, v)
		}
	}
	if e.Suggestion != "" {
		e.ErrorMessage += fmt.Sprintf("\nSuggestion: %s", e.Suggestion)
	}
	return e.ErrorMessage
}

// 错误类型常量
const (
	ErrorTypeAuth       = "认证失败"
	ErrorTypePermission = "权限不足"
	ErrorTypeNetwork    = "网络错误"
	ErrorTypeAPI        = "API错误"
	ErrorTypeGit        = "Git操作失败"
	ErrorTypeConfig     = "配置错误"
	ErrorTypeValidation = "数据验证失败"
	ErrorTypeStorage    = "存储失败"
	ErrorTypeUnknown    = "未知错误"
)

// NewDetailedError 创建详细错误
func NewDetailedError(errorType, message string) *DetailedError {
	return &DetailedError{
		ErrorType:    errorType,
		ErrorMessage: message,
		Details:      make(map[string]string),
	}
}

// WithDetail 添加详细信息
func (e *DetailedError) WithDetail(key, value string) *DetailedError {
	if e.Details == nil {
		e.Details = make(map[string]string)
	}
	e.Details[key] = value
	return e
}

// WithSuggestion 添加恢复建议
func (e *DetailedError) WithSuggestion(suggestion string) *DetailedError {
	e.Suggestion = suggestion
	return e
}
