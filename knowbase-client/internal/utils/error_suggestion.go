// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package utils

import (
	"fmt"
	"strings"
)

// GetAPISuggestion 根据 API 状态码返回建议
func GetAPISuggestion(statusCode int) string {
	suggestions := map[int]string{
		400: "请检查配置文件中的参数是否正确",
		401: "认证失败，请检查 COMATE_AUTH_TOKEN 环境变量是否正确",
		403: "权限不足，请确认您有访问该资源的权限",
		404: "资源不存在，请检查配置中的资源 ID 是否正确",
		429: "请求过于频繁，请稍后重试",
		500: "服务器内部错误，请稍后重试或联系管理员",
		502: "网关错误，请检查网络连接",
		503: "服务暂时不可用，请稍后重试",
		504: "网关超时，请检查网络连接或稍后重试",
	}

	if suggestion, ok := suggestions[statusCode]; ok {
		return suggestion
	}
	return "API 调用失败，请检查配置，确认权限"
}

// GetGitSuggestion 根据 Git 错误输出返回建议
func GetGitSuggestion(output string) string {
	lowerOutput := strings.ToLower(output)

	suggestions := map[string]string{
		"could not read username":     "请配置 SSH 密钥或使用 HTTPS 认证",
		"permission denied":           "权限被拒绝，请检查 SSH 密钥配置或仓库访问权限",
		"repository not found":        "仓库不存在或无权限访问",
		"remote: not found":           "仓库不存在或无权限访问",
		"connection refused":          "连接被拒绝，请检查网络连接",
		"timeout":                     "连接超时，请检查网络连接或稍后重试",
		"timed out":                   "连接超时，请检查网络连接或稍后重试",
		"already exists":              "目录已存在，请删除后重试或使用增量更新",
		"branch not found":            "分支不存在，请检查配置中的分支名是否正确",
		"authentication failed":       "认证失败，请检查 SSH 密钥或访问令牌配置",
		"fatal: not a git repository": "不是有效的 Git 仓库",
	}

	for keyword, suggestion := range suggestions {
		if strings.Contains(lowerOutput, keyword) {
			return suggestion
		}
	}

	return "Git 操作失败，请检查网络连接、SSH 密钥配置和仓库访问权限"
}

// GetNetworkSuggestion 网络错误建议
func GetNetworkSuggestion(err error) string {
	if err == nil {
		return ""
	}

	errMsg := err.Error()
	lowerMsg := strings.ToLower(errMsg)

	if strings.Contains(lowerMsg, "timeout") || strings.Contains(lowerMsg, "timed out") {
		return "网络超时，请检查网络连接或稍后重试"
	}
	if strings.Contains(lowerMsg, "connection refused") {
		return "连接被拒绝，请检查网络连接和目标服务是否可访问"
	}
	if strings.Contains(lowerMsg, "no such host") {
		return "域名解析失败，请检查网络配置和 DNS 设置"
	}
	if strings.Contains(lowerMsg, "connection reset") {
		return "连接被重置，请检查网络连接或稍后重试"
	}
	if strings.Contains(lowerMsg, "eof") {
		return "连接意外关闭，请检查网络连接或稍后重试"
	}

	return "网络错误，请检查网络连接"
}

// FormatGitOutput 格式化 Git 输出
func FormatGitOutput(output string) string {
	if output == "" {
		return ""
	}

	// 去除过长输出
	lines := strings.Split(output, "\n")
	const maxLines = 10

	if len(lines) > maxLines {
		lines = lines[:maxLines]
		lines = append(lines, "... (输出过长已截断)")
	}

	return strings.TrimSpace(strings.Join(lines, "\n"))
}

// TruncateString 截断过长的字符串
func TruncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}

	// 如果是多行文本，只显示第一行加省略号
	if strings.Contains(s, "\n") {
		firstLine := strings.Split(s, "\n")[0]
		if len(firstLine) > maxLen {
			return firstLine[:maxLen] + "... (已截断)"
		}
		return firstLine + "... (多行已截断)"
	}

	return s[:maxLen] + "... (已截断)"
}

// GetHTTPSuggestion 根据 HTTP 状态码返回建议（通用版本）
func GetHTTPSuggestion(statusCode int, responseBody string) string {
	// 首先尝试从响应体中提取错误信息
	if responseBody != "" {
		lowerBody := strings.ToLower(responseBody)
		if strings.Contains(lowerBody, "token") || strings.Contains(lowerBody, "auth") {
			return "认证失败，请检查访问令牌或认证信息"
		}
		if strings.Contains(lowerBody, "permission") || strings.Contains(lowerBody, "forbidden") {
			return "权限不足，请确认您有访问该资源的权限"
		}
	}

	// 使用通用建议
	return GetAPISuggestion(statusCode)
}

// FormatErrorContext 格式化错误上下文，用于日志输出
func FormatErrorContext(contextInfo map[string]string) string {
	if len(contextInfo) == 0 {
		return ""
	}

	var parts []string
	for key, value := range contextInfo {
		parts = append(parts, fmt.Sprintf("%s=%s", key, TruncateString(value, 100)))
	}

	return strings.Join(parts, ", ")
}
