// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package types

// AuthMode 认证模式
type AuthMode string

const (
	// AuthModeComate 使用 COMATE_AUTH_TOKEN 环境变量认证
	AuthModeComate AuthMode = "comate"
	// AuthModeUgate 使用 UGate Token 认证
	AuthModeUgate AuthMode = "ugate"
)

// Credential 认证凭证
type Credential struct {
	Mode     AuthMode // 认证模式
	Token    string   // 认证令牌
	Username string   // 用户名
}

// UGate 代理配置
const (
	// UgateProxyHost UGate 代理域名
	UgateProxyHost = "https://apigo.baidu-int.com"
)
