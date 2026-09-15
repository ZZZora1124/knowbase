// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package loader

import (
	"context"

	"icode.baidu.com/baidu/devops-ai/knowbase-client/internal/storage"
	"icode.baidu.com/baidu/devops-ai/knowbase-client/pkg/types"
)

// KnowledgeLoader 知识加载器接口
type KnowledgeLoader interface {
	// Load 加载知识
	Load(ctx context.Context, source types.SourceConfig) ([]types.RawKnowledge, error)

	// Convert 转换格式（如果needConvert=true）
	Convert(sourceType string, raw []types.RawKnowledge, output types.OutputConfig) ([]types.KnowledgeItem, error)

	// Store 存储到本地，返回存储的知识项详细信息列表
	Store(ctx context.Context, items []types.KnowledgeItem, storage *storage.LocalStorage, source types.SourceConfig) ([]types.StoredKnowledgeInfo, error)

	// GenerateUsage 生成使用说明
	GenerateUsage(storage *storage.LocalStorage, usage types.UsageConfig, sourceType string, storedItems []types.StoredKnowledgeInfo) error

	// Type 获取加载器类型
	Type() string
}

