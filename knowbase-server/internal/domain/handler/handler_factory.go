// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package handler

import (
	"fmt"
	"strings"

	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/infrastructure/storage"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/errors"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/types/constants"
)

// HandlerFactory 处理器工厂
type HandlerFactory struct {
	bosClient *storage.BOSClient
}

// NewHandlerFactory 创建处理器工厂
func NewHandlerFactory(bosClient *storage.BOSClient) *HandlerFactory {
	return &HandlerFactory{
		bosClient: bosClient,
	}
}

// Create 根据知识类型创建处理器
func (f *HandlerFactory) Create(knowledgeType string) (KnowledgeHandler, error) {
	switch knowledgeType {
	case constants.KnowledgeTypeContent, constants.KnowledgeTypeReadme:
		// readme 和 content 使用相同的处理逻辑
		return &ContentHandler{}, nil
	case constants.KnowledgeTypeTextFile,
		constants.KnowledgeTypeAudioFile,
		constants.KnowledgeTypeImageFile,
		constants.KnowledgeTypeVideoFile:
		return NewFileHandler(f.bosClient, knowledgeType), nil
	case constants.KnowledgeTypeConfig:
		return &ConfigHandler{}, nil
	default:
		validTypes := []string{
			constants.KnowledgeTypeContent,
			constants.KnowledgeTypeReadme,
			constants.KnowledgeTypeTextFile,
			constants.KnowledgeTypeAudioFile,
			constants.KnowledgeTypeImageFile,
			constants.KnowledgeTypeVideoFile,
			constants.KnowledgeTypeConfig,
		}
		return nil, errors.New(errors.ErrCodeInvalidKnowledgeType,
			fmt.Sprintf("不支持的知识类型'%s'，支持的类型: %s", knowledgeType, strings.Join(validTypes, "、")))
	}
}
