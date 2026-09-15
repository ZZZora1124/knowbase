// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package handler

import (
	"mime/multipart"

	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/domain"
)

// ProcessResult 处理结果
type ProcessResult struct {
	Content    string                 // 短文本内容(content类型使用)
	FilePath   string                 // BOS文件路径(file类型使用)
	FileSize   int64                  // 文件大小
	FileFormat string                 // 文件格式
	Metadata   map[string]interface{} // 扩展元数据
}

// KnowledgeProcessData 处理数据
type KnowledgeProcessData struct {
	Content           string                 // 短文本内容
	File              *multipart.FileHeader  // 上传的文件
	Metadata          map[string]interface{} // 元数据
	KnowledgeSource   string                 // 知识来源
	KnowledgeType     string                 // 知识类型
	KnowledgebaseUUID string                 // 知识本UUID
}

// KnowledgeHandler 知识处理器接口
type KnowledgeHandler interface {
	// Validate 验证知识数据
	Validate(data *KnowledgeProcessData) error

	// Process 处理知识(上传文件、转换格式等)
	Process(data *KnowledgeProcessData) (*ProcessResult, error)

	// Delete 删除知识(清理文件等)
	Delete(knowledge *domain.Knowledge) error
}
