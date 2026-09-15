// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package loader

import (
	"fmt"
	"path/filepath"
	"strings"

	"icode.baidu.com/baidu/devops-ai/knowbase-client/internal/storage"
	"icode.baidu.com/baidu/devops-ai/knowbase-client/pkg/types"
)

// BaseLoader 基础加载器，提供默认实现
type BaseLoader struct {
	credential *types.Credential
}

// NewBaseLoader 创建基础加载器
func NewBaseLoader(cred *types.Credential) *BaseLoader {
	return &BaseLoader{
		credential: cred,
	}
}

// GetCredential 获取认证凭证
func (b *BaseLoader) GetCredential() *types.Credential {
	return b.credential
}

// GetAuthToken 获取认证令牌（兼容旧接口）
func (b *BaseLoader) GetAuthToken() string {
	if b.credential != nil {
		return b.credential.Token
	}
	return ""
}

// GetUsername 获取用户名
func (b *BaseLoader) GetUsername() string {
	if b.credential != nil {
		return b.credential.Username
	}
	return ""
}

// GenerateUsage 生成使用说明（支持自动生成）
func (b *BaseLoader) GenerateUsage(storage *storage.LocalStorage, usage types.UsageConfig, sourceType string, storedItems []types.StoredKnowledgeInfo) error {
	var content []byte

	// 如果用户没有配置content或为空，自动生成
	if usage.Content == "" {
		content = b.generateAutoUsageContent(sourceType, storedItems)
	} else {
		content = []byte(usage.Content)
	}

	// 确定文件路径并写入文件
	var usagePath string
	if usage.Path != "" {
		usagePath = usage.Path
		if err := storage.WriteFile(usagePath, content, true); err != nil {
			return fmt.Errorf("failed to write usage file: %w", err)
		}
	} else {
		// 使用默认路径: sourceType子目录下的sourceType_usage.md
		usagePath = filepath.Join(sourceType, fmt.Sprintf("%s_usage.md", sourceType))
		if err := storage.WriteFile(usagePath, content, false); err != nil {
			return fmt.Errorf("failed to write usage file: %w", err)
		}
	}

	return nil
}

// generateAutoUsageContent 自动生成usage内容
func (b *BaseLoader) generateAutoUsageContent(sourceType string, items []types.StoredKnowledgeInfo) []byte {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# %s 知识使用说明\n\n", sourceType))
	sb.WriteString(fmt.Sprintf("本目录包含 %d 个知识项:\n\n", len(items)))

	for i, item := range items {
		sb.WriteString(fmt.Sprintf("## %d. %s\n\n", i+1, item.Name))
		if item.Description != "" {
			sb.WriteString(fmt.Sprintf("**描述**: %s\n\n", item.Description))
		}
		sb.WriteString(fmt.Sprintf("**存储路径**: `%s`\n\n", item.FullPath))
	}

	return []byte(sb.String())
}

// Convert 转换格式（默认不支持转换）
func (b *BaseLoader) Convert(sourceType string, raw []types.RawKnowledge, output types.OutputConfig) ([]types.KnowledgeItem, error) {
	return nil, fmt.Errorf("conversion not supported for this loader type, sourceType: %s", sourceType)
}
