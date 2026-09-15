// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package auth

import (
	"fmt"
	"os"

	"icode.baidu.com/baidu/devops-ai/go-cli-lib/oneauth"
	"icode.baidu.com/baidu/devops-ai/knowbase-client/internal/utils"
	"icode.baidu.com/baidu/devops-ai/knowbase-client/pkg/types"
)

// GetCredential 获取认证凭证
// 优先级：COMATE_AUTH_TOKEN 环境变量 > UGate 本地登录状态
func GetCredential() (*types.Credential, error) {
	// 1. 优先检查 COMATE_AUTH_TOKEN 环境变量
	comateToken := os.Getenv(types.EnvComateAuthToken)
	if comateToken != "" {
		username, err := utils.ValidateToken(comateToken)
		if err != nil {
			return nil, fmt.Errorf("COMATE_AUTH_TOKEN 验证失败: %w", err)
		}
		return &types.Credential{
			Mode:     types.AuthModeComate,
			Token:    comateToken,
			Username: username,
		}, nil
	}

	// 2. 检查 UGate 登录状态
	auth := oneauth.New(oneauth.WithMethod(oneauth.NewUgateAuth()))
	if err := auth.SetCurrentMethod("ugate"); err != nil {
		return nil, newAuthError()
	}

	method := auth.CurrentMethod()
	ok, err := method.CheckLogin()
	if err != nil || !ok {
		return nil, newAuthError()
	}

	// 获取凭证
	cred := method.GetCredential()
	token, _ := cred["token"]
	if token == "" {
		return nil, newAuthError()
	}

	return &types.Credential{
		Mode:     types.AuthModeUgate,
		Token:    token,
		Username: auth.GetUsername(),
	}, nil
}

// GetCredentialWithValidation 获取认证凭证（Comate 模式会验证 token）
// 这是 GetCredential 的别名，保持向后兼容
func GetCredentialWithValidation() (*types.Credential, error) {
	return GetCredential()
}

// newAuthError 创建统一的认证错误提示
func newAuthError() error {
	return fmt.Errorf("未检测到认证信息，请使用以下方式之一进行认证:\n" +
		" 方式一，设置环境变量: export COMATE_AUTH_TOKEN=，token 从 https://console.cloud.baidu-int.com/onetool/auth-manage/my-services 复制个人token \n" +
		" 方式二， UGate 登录: knowbase login <username> <token>，token 从 https://uuap.baidu.com/agent/token 获取\n")
}

