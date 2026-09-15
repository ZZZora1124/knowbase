// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package loader

import (
	"context"
	"fmt"
	"path/filepath"

	"icode.baidu.com/baidu/devops-ai/knowbase-client/internal/storage"
	"icode.baidu.com/baidu/devops-ai/knowbase-client/internal/utils"
	"icode.baidu.com/baidu/devops-ai/knowbase-client/pkg/types"
)

// LocalKnowledgeLoader 本地知识加载器
type LocalKnowledgeLoader struct {
	*BaseLoader
}

// NewLocalKnowledgeLoader 创建本地知识加载器
func NewLocalKnowledgeLoader(cred *types.Credential) *LocalKnowledgeLoader {
	return &LocalKnowledgeLoader{
		BaseLoader: NewBaseLoader(cred),
	}
}

// Type 返回加载器类型
func (l *LocalKnowledgeLoader) Type() string {
	return types.SourceTypeLocalKnowledge
}

// Load 验证源路径存在性
func (l *LocalKnowledgeLoader) Load(ctx context.Context, source types.SourceConfig) ([]types.RawKnowledge, error) {
	result := make([]types.RawKnowledge, 0)

	for _, filter := range source.Filters {
		// 展开路径
		expandedSrc, err := utils.ExpandPath(filter.SrcPath)
		if err != nil {
			return nil, fmt.Errorf("failed to expand source path %s: %w", filter.SrcPath, err)
		}

		// 检查路径是否存在
		if !utils.FileExists(expandedSrc) {
			return nil, fmt.Errorf("source path does not exist: %s", expandedSrc)
		}

		// 构造 RawKnowledge，保存 filter 信息
		result = append(result, types.RawKnowledge{
			SourceType: types.SourceTypeLocalKnowledge,
			Data:       map[string]interface{}{"filter": filter}, // 直接保存 FilterConfig
		})
	}

	fmt.Printf("Validated local knowledge sources, count: %d\n", len(result))
	return result, nil
}

// Store 创建符号链接，返回存储的知识项详细信息列表
func (l *LocalKnowledgeLoader) Store(ctx context.Context, items []types.KnowledgeItem, storage *storage.LocalStorage,
	source types.SourceConfig) ([]types.StoredKnowledgeInfo, error) {

	storedInfos := make([]types.StoredKnowledgeInfo, 0)

	for _, item := range items {
		// 从 Metadata 中提取 FilterConfig
		filterOri, exist := item.Metadata["filter"]
		if !exist {
			utils.PrintlnWarn("LocalKnowledgeLoader metadata missing filter, skipping item")
			continue
		}
		filter, ok := filterOri.(types.FilterConfig)
		if !ok {
			utils.PrintlnWarn("LocalKnowledgeLoader invalid metadata format, skipping item")
			continue
		}
		filter.DestPath = filepath.Join(types.SourceTypeLocalKnowledge, filter.DestPath)

		// 根据 editable 决定策略
		var err error
		if filter.Editable {
			// 必须使用符号链接
			err = storage.CreateSymlink(filter.SrcPath, filter.DestPath)
		} else {
			// 允许降级为复制
			err = storage.CreateSymlinkOrCopy(filter.SrcPath, filter.DestPath)
		}

		if err != nil {
			utils.PrintlnError("Failed to link local knowledge %s: %v", filter.SrcPath, err.Error())
			continue
		}

		// 收集存储信息
		fullPath := storage.GetFullPath(filter.DestPath)
		name := filepath.Base(filter.SrcPath)

		// 构建描述信息
		description := fmt.Sprintf("本地知识链接，源路径: %s", filter.SrcPath)
		if filter.Editable {
			description += "（可编辑）"
		} else {
			description += "（只读）"
		}

		storedInfos = append(storedInfos, types.StoredKnowledgeInfo{
			Name:        name,
			Description: description,
			FullPath:    fullPath,
			RelPath:     filter.DestPath,
		})

		fmt.Printf("Successfully linked local knowledge, src: %s, dest: %s\n",
			filter.SrcPath, filter.DestPath)
	}

	fmt.Printf("Stored local knowledge, count: %d\n", len(storedInfos))
	return storedInfos, nil
}
