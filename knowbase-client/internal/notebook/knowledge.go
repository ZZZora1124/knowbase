// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package notebook

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"icode.baidu.com/baidu/devops-ai/knowbase-client/pkg/types"
)

// CreateKnowledge 创建知识（支持文本内容和文件上传）
func (s *NotebookService) CreateKnowledge(ctx context.Context, kbUUID string,
	title, knowledgeType, knowledgeSource, directoryUUID, content, filePath string) (*Knowledge, error) {

	apiURL := s.BuildURL(types.PathNotebookKnowledgeCreate, kbUUID)

	fields := map[string]string{
		"title":           title,
		"knowledgeType":   knowledgeType,
		"knowledgeSource": knowledgeSource,
	}
	if directoryUUID != "" {
		fields["directoryUuid"] = directoryUUID
	}
	if content != "" {
		fields["content"] = content
	}

	fileField := ""
	if filePath != "" {
		fileField = "file"
	}

	body, err := WithRetry(ctx, "create knowledge", func() ([]byte, error) {
		return s.HTTPClient.PostMultipart(ctx, apiURL, fields, fileField, filePath)
	})
	if err != nil {
		return nil, err
	}

	var result Knowledge
	if err := s.ParseResponse(body, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ListKnowledge 查询知识列表
func (s *NotebookService) ListKnowledge(ctx context.Context, kbUUID string,
	directoryUUID, knowledgeSource, knowledgeType, keyword, creatorUsername string,
	page, pageSize int) (*KnowledgeListResponse, error) {

	params := url.Values{}
	if directoryUUID != "" {
		params.Set("directoryUuid", directoryUUID)
	}
	if knowledgeSource != "" {
		params.Set("knowledgeSource", knowledgeSource)
	}
	if knowledgeType != "" {
		params.Set("knowledgeType", knowledgeType)
	}
	if keyword != "" {
		params.Set("keyword", keyword)
	}
	if creatorUsername != "" {
		params.Set("creatorUsername", creatorUsername)
	}
	params.Set("page", fmt.Sprintf("%d", page))
	params.Set("pageSize", fmt.Sprintf("%d", pageSize))

	apiURL := s.BuildURL(types.PathNotebookKnowledgeList, kbUUID) + "?" + params.Encode()
	body, err := WithRetry(ctx, "list knowledge", func() ([]byte, error) {
		return s.HTTPClient.Get(ctx, apiURL)
	})
	if err != nil {
		return nil, err
	}

	var result KnowledgeListResponse
	if err := s.ParseResponse(body, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ListKnowledgeByTags 根据标签查询知识
func (s *NotebookService) ListKnowledgeByTags(ctx context.Context, kbUUID string,
	tagUUIDs, tagNames, directoryUUIDs []string, matchType, knowledgeSource, knowledgeType string,
	page, pageSize int) (*KnowledgeListResponse, error) {

	// 构建请求体
	reqBody := map[string]interface{}{
		"page":     page,
		"pageSize": pageSize,
	}
	if len(tagUUIDs) > 0 {
		reqBody["tagUuids"] = tagUUIDs
	}
	if len(tagNames) > 0 {
		reqBody["tagNames"] = tagNames
	}
	if len(directoryUUIDs) > 0 {
		reqBody["directoryUuids"] = directoryUUIDs
	}
	if matchType != "" {
		reqBody["matchType"] = matchType
	}
	if knowledgeSource != "" {
		reqBody["knowledgeSource"] = knowledgeSource
	}
	if knowledgeType != "" {
		reqBody["knowledgeType"] = knowledgeType
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	apiURL := s.BuildURL(types.PathNotebookKnowledgeByTags, kbUUID)
	body, err := WithRetry(ctx, "list knowledge by tags", func() ([]byte, error) {
		return s.HTTPClient.Post(ctx, apiURL, jsonData)
	})
	if err != nil {
		return nil, err
	}

	var result KnowledgeListResponse
	if err := s.ParseResponse(body, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetKnowledge 获取知识详情
func (s *NotebookService) GetKnowledge(ctx context.Context, uuid string) (*Knowledge, error) {
	apiURL := s.BuildURL(types.PathNotebookKnowledgeGet, uuid)
	body, err := WithRetry(ctx, "get knowledge", func() ([]byte, error) {
		return s.HTTPClient.Get(ctx, apiURL)
	})
	if err != nil {
		return nil, err
	}

	var result Knowledge
	if err := s.ParseResponse(body, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetKnowledgeWithDownload 获取知识详情，如果是文件类型则下载到指定路径
// dirPath: 文件保存目录，如果为空则不下载文件
func (s *NotebookService) GetKnowledgeWithDownload(ctx context.Context, uuid string, dirPath string) (*Knowledge, error) {
	// 先获取知识详情
	knowledge, err := s.GetKnowledge(ctx, uuid)
	if err != nil {
		return nil, err
	}

	// 如果不需要下载文件或知识类型不包含 file，直接返回
	if dirPath == "" || !strings.Contains(strings.ToLower(knowledge.KnowledgeType), "file") {
		return knowledge, nil
	}

	filePath := dirPath + "/" + knowledge.FilePath

	// 从 metadata 中获取 fileUrl
	fileURL, err := s.extractFileURL(knowledge.Metadata)
	if err != nil {
		return knowledge, err
	}

	// 下载文件
	if err := s.downloadFile(ctx, fileURL, filePath); err != nil {
		return knowledge, types.NewDetailedError(types.ErrorTypeNetwork, "failed to download knowledge file").
			WithDetail("fileUrl", fileURL).
			WithDetail("rawError", err.Error())
	}

	knowledge.FilePath = filePath

	return knowledge, nil
}

// extractFileURL 从 metadata 中提取文件下载链接
func (s *NotebookService) extractFileURL(metadata map[string]interface{}) (string, error) {
	if metadata == nil {
		return "", types.NewDetailedError(types.ErrorTypeValidation, "metadata is empty, cannot extract fileUrl")
	}

	fileURL, ok := metadata["fileUrl"]
	if !ok {
		return "", types.NewDetailedError(types.ErrorTypeValidation, "fileUrl not found in metadata")
	}

	urlStr, ok := fileURL.(string)
	if !ok || urlStr == "" {
		return "", types.NewDetailedError(types.ErrorTypeValidation, "fileUrl is not a valid string")
	}

	return urlStr, nil
}

// downloadFile 下载文件到指定路径
func (s *NotebookService) downloadFile(ctx context.Context, fileURL, filePath string) error {
	// 创建目标文件的父目录
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// 创建目标文件
	outFile, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", filePath, err)
	}
	defer outFile.Close()

	// 使用 HTTPClient 下载文件
	return s.HTTPClient.Download(ctx, fileURL, outFile)
}

// UpdateKnowledge 更新知识
func (s *NotebookService) UpdateKnowledge(ctx context.Context, uuid string,
	req *UpdateKnowledgeRequest) (*Knowledge, error) {

	data, err := json.Marshal(req)
	if err != nil {
		return nil, types.NewDetailedError(types.ErrorTypeValidation, "failed to marshal request").
			WithDetail("rawError", err.Error())
	}

	apiURL := s.BuildURL(types.PathNotebookKnowledgeUpdate, uuid)
	body, err := WithRetry(ctx, "update knowledge", func() ([]byte, error) {
		return s.HTTPClient.Put(ctx, apiURL, data)
	})
	if err != nil {
		return nil, err
	}

	var result Knowledge
	if err := s.ParseResponse(body, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// UpdateKnowledgeFile 通过文件更新知识内容
func (s *NotebookService) UpdateKnowledgeFile(ctx context.Context, uuid, title, filePath string) (*Knowledge, error) {
	apiURL := s.BuildURL(types.PathNotebookKnowledgeUpdateFile, uuid)
	fields := map[string]string{}
	if title != "" {
		fields["title"] = title
	}
	body, err := WithRetry(ctx, "update knowledge file", func() ([]byte, error) {
		return s.HTTPClient.PostMultipart(ctx, apiURL, fields, "file", filePath)
	})
	if err != nil {
		return nil, err
	}
	var result Knowledge
	if err := s.ParseResponse(body, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// MoveKnowledge 移动知识
func (s *NotebookService) MoveKnowledge(ctx context.Context, uuid string, directoryUUID *string) error {
	req := &MoveKnowledgeRequest{DirectoryUUID: directoryUUID}
	data, err := json.Marshal(req)
	if err != nil {
		return types.NewDetailedError(types.ErrorTypeValidation, "failed to marshal request").
			WithDetail("rawError", err.Error())
	}

	apiURL := s.BuildURL(types.PathNotebookKnowledgeMove, uuid)
	body, err := WithRetry(ctx, "move knowledge", func() ([]byte, error) {
		return s.HTTPClient.Post(ctx, apiURL, data)
	})
	if err != nil {
		return err
	}

	return s.ParseResponse(body, nil)
}

// BindTags 绑定标签
func (s *NotebookService) BindTags(ctx context.Context, uuid string, tagUUIDs []string) ([]Tag, error) {
	req := &BindTagsRequest{TagUUIDs: tagUUIDs}
	data, err := json.Marshal(req)
	if err != nil {
		return nil, types.NewDetailedError(types.ErrorTypeValidation, "failed to marshal request").
			WithDetail("rawError", err.Error())
	}

	apiURL := s.BuildURL(types.PathNotebookKnowledgeBindTags, uuid)
	body, err := WithRetry(ctx, "bind tags", func() ([]byte, error) {
		return s.HTTPClient.Post(ctx, apiURL, data)
	})
	if err != nil {
		return nil, err
	}

	var result []Tag
	if err := s.ParseResponse(body, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// UnbindTags 解绑标签
func (s *NotebookService) UnbindTags(ctx context.Context, uuid string, tagUUIDs []string) ([]Tag, error) {
	req := &BindTagsRequest{TagUUIDs: tagUUIDs}
	data, err := json.Marshal(req)
	if err != nil {
		return nil, types.NewDetailedError(types.ErrorTypeValidation, "failed to marshal request").
			WithDetail("rawError", err.Error())
	}

	apiURL := s.BuildURL(types.PathNotebookKnowledgeUnbindTags, uuid)
	body, err := WithRetry(ctx, "unbind tags", func() ([]byte, error) {
		return s.HTTPClient.Delete(ctx, apiURL, data)
	})
	if err != nil {
		return nil, err
	}

	var result []Tag
	if err := s.ParseResponse(body, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// GetKnowledgeTags 获取知识的标签
func (s *NotebookService) GetKnowledgeTags(ctx context.Context, uuid string) ([]Tag, error) {
	apiURL := s.BuildURL(types.PathNotebookKnowledgeGetTags, uuid)
	body, err := WithRetry(ctx, "get knowledge tags", func() ([]byte, error) {
		return s.HTTPClient.Get(ctx, apiURL)
	})
	if err != nil {
		return nil, err
	}

	var result []Tag
	if err := s.ParseResponse(body, &result); err != nil {
		return nil, err
	}
	return result, nil
}
