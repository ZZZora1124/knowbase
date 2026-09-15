// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package handler

import (
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/domain"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/errors"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/types/constants"
)

// ConfigHandler Config类型处理器
type ConfigHandler struct{}

// Validate 验证配置
func (h *ConfigHandler) Validate(data *KnowledgeProcessData) error {
	// config类型不能用于自定义知识
	if data.KnowledgeSource == constants.SourceTypeCustom {
		return errors.New(errors.ErrCodeInvalidKnowledgeType,
			"config类型仅用于外部知识源同步，自定义知识请选择content或file类型")
	}

	// config类型必须提供metadata
	if len(data.Metadata) == 0 {
		return errors.New(errors.ErrCodeInvalidParam, "config类型必须提供metadata配置信息")
	}

	// 根据知识源验证必需字段
	switch data.KnowledgeSource {
	case constants.SourceTypeICodeRepo, constants.SourceTypeICodeDeepWiki:
		return h.validateICodeConfig(data.Metadata)
	case constants.SourceTypeKuDoc:
		return h.validateKuConfig(data.Metadata)
	case constants.SourceTypeICafeCard:
		return h.validateICafeConfig(data.Metadata)
	case constants.SourceTypeKnowledgeBook:
		return h.validateKnowledgeBookConfig(data.Metadata)
	}

	return nil
}

// validateICodeConfig 验证iCode配置
func (h *ConfigHandler) validateICodeConfig(metadata map[string]interface{}) error {
	if _, ok := metadata["repoUrl"]; !ok {
		return errors.New(errors.ErrCodeInvalidParam, "iCode知识源缺少必需字段'repoUrl'（仓库地址）")
	}
	if _, ok := metadata["branch"]; !ok {
		return errors.New(errors.ErrCodeInvalidParam, "iCode知识源缺少必需字段'branch'（分支名称）")
	}
	return nil
}

// validateKuConfig 验证知识库配置
func (h *ConfigHandler) validateKuConfig(metadata map[string]interface{}) error {
	if _, ok := metadata["docId"]; !ok {
		return errors.New(errors.ErrCodeInvalidParam, "知识库知识源缺少必需字段'docId'（文档ID）")
	}
	return nil
}

// validateICafeConfig 验证iCafe配置
func (h *ConfigHandler) validateICafeConfig(metadata map[string]interface{}) error {
	if _, ok := metadata["cardId"]; !ok {
		return errors.New(errors.ErrCodeInvalidParam, "iCafe知识源缺少必需字段'cardId'（卡片ID）")
	}
	if _, ok := metadata["spaceId"]; !ok {
		return errors.New(errors.ErrCodeInvalidParam, "iCafe知识源缺少必需字段'spaceId'（空间ID）")
	}
	return nil
}

// validateKnowledgeBookConfig 验证知识方舟知识本配置
func (h *ConfigHandler) validateKnowledgeBookConfig(metadata map[string]interface{}) error {
	knowledgebaseUuid, ok := metadata["knowledgebaseUuid"]
	if !ok {
		return errors.New(errors.ErrCodeInvalidParam, "知识方舟知识本知识源缺少必需字段'knowledgebaseUuid'（知识本UUID）")
	}
	// 验证knowledgeUuid不为空
	if uuidStr, ok := knowledgebaseUuid.(string); !ok || uuidStr == "" {
		return errors.New(errors.ErrCodeInvalidParam, "知识方舟知识本知识源的'knowledgebaseUuid'字段不能为空")
	}
	return nil
}

// Process 处理配置
func (h *ConfigHandler) Process(data *KnowledgeProcessData) (*ProcessResult, error) {
	return &ProcessResult{
		Metadata: data.Metadata,
	}, nil
}

// Delete 删除配置（无需清理外部资源）
func (h *ConfigHandler) Delete(knowledge *domain.Knowledge) error {
	return nil
}
