// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package handler

import (
	"fmt"
	"strings"

	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/domain"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/errors"
)

// ContentHandler Content类型处理器
type ContentHandler struct{}

// 最大内容长度 64KB
const maxContentLength = 64 * 1024

// Validate 验证内容
func (h *ContentHandler) Validate(data *KnowledgeProcessData) error {
	// 验证内容不为空
	if strings.TrimSpace(data.Content) == "" {
		return errors.New(errors.ErrCodeInvalidParam, "内容不能为空")
	}

	// 验证内容长度
	if len(data.Content) > maxContentLength {
		contentLengthKB := len(data.Content) / 1024
		maxLengthKB := maxContentLength / 1024
		return errors.New(errors.ErrCodeInvalidParam,
			fmt.Sprintf("内容长度(%dKB)超过限制，文本内容最大支持%dKB", contentLengthKB, maxLengthKB))
	}

	return nil
}

// Process 处理内容
func (h *ContentHandler) Process(data *KnowledgeProcessData) (*ProcessResult, error) {
	return &ProcessResult{
		Content: data.Content,
		Metadata: map[string]interface{}{
			"length": len(data.Content),
		},
	}, nil
}

// Delete 删除内容（无需清理外部资源）
func (h *ContentHandler) Delete(knowledge *domain.Knowledge) error {
	return nil
}
