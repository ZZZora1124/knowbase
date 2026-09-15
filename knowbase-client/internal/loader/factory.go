// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package loader

import (
	"fmt"

	"icode.baidu.com/baidu/devops-ai/knowbase-client/pkg/types"
)

// Factory 加载器工厂
type Factory struct {
	credential *types.Credential
}

// NewFactory 创建加载器工厂
func NewFactory(cred *types.Credential) *Factory {
	return &Factory{
		credential: cred,
	}
}

// CreateLoader 根据类型创建加载器
func (f *Factory) CreateLoader(sourceType string) (KnowledgeLoader, error) {
	switch sourceType {
	case types.SourceTypeICafeCard:
		return NewICafeLoader(f.credential), nil
	case types.SourceTypeICodeRepo:
		return NewICodeRepoLoader(f.credential), nil
	case types.SourceTypeICodeDeepWiki:
		return NewICodeWikiLoader(f.credential), nil
	case types.SourceTypeIApiDoc:
		return NewIApiDocLoader(f.credential), nil
	case types.SourceTypeLocalKnowledge:
		return NewLocalKnowledgeLoader(f.credential), nil
	case types.SourceTypeGithubRepo:
		return NewGithubRepoLoader(f.credential), nil
	case types.SourceTypeKuDoc:
		return NewKuDocLoader(f.credential), nil
	case types.SourceTypeInfoflowGroupMessage:
		return NewInfoflowGroupMessageLoader(f.credential), nil
	case types.SourceTypeICodeCRDiff:
		return NewICodeCRLoader(f.credential), nil
	case types.SourceTypeKnowledgeBook:
		return NewKnowledgeBookLoader(f.credential), nil
	case types.SourceTypeIDataset:
		return NewIDatasetLoader(f.credential), nil
	// TODO: 添加其他加载器类型
	default:
		return nil, fmt.Errorf("unsupported source type: %s", sourceType)
	}
}
