// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package handler

import (
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/domain"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/infrastructure/storage"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/errors"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/types/constants"
)

// bosClientInterface BOS客户端接口，用于支持测试mock
type bosClientInterface interface {
	PutObject(objectKey string, fileData []byte, contentType string) (string, error)
	DeleteObject(objectKey string) error
}

// FileHandler 文件类型处理器
type FileHandler struct {
	bosClient bosClientInterface
	maxSize   int64  // 最大文件大小(字节)
	fileType  string // 知识类型
}

// 统一的最大文件大小限制 500MB
const maxFileSize = 500 * 1024 * 1024

// NewFileHandler 创建文件处理器
func NewFileHandler(bosClient *storage.BOSClient, fileType string) *FileHandler {
	return &FileHandler{
		bosClient: bosClient,
		maxSize:   maxFileSize,
		fileType:  fileType,
	}
}

// Validate 验证文件
func (h *FileHandler) Validate(data *KnowledgeProcessData) error {
	if data.File == nil {
		return errors.New(errors.ErrCodeInvalidParam, "文件不能为空")
	}

	// 验证文件名
	if data.File.Filename == "" {
		return errors.New(errors.ErrCodeInvalidParam, "文件名不能为空")
	}

	// 验证文件大小
	if data.File.Size > h.maxSize {
		fileSizeMB := float64(data.File.Size) / 1024 / 1024
		maxSizeMB := float64(h.maxSize) / 1024 / 1024
		return errors.New(errors.ErrCodeFileTooLarge,
			fmt.Sprintf("文件大小(%.2fMB)超过限制，最大支持%.0fMB", fileSizeMB, maxSizeMB))
	}

	return nil
}

// Process 处理文件
func (h *FileHandler) Process(data *KnowledgeProcessData) (*ProcessResult, error) {
	// 打开文件
	file, err := data.File.Open()
	if err != nil {
		return nil, errors.Wrap(errors.ErrCodeUploadFailed, "打开文件失败", err)
	}
	defer file.Close()

	// 读取文件内容
	fileData, err := io.ReadAll(file)
	if err != nil {
		return nil, errors.Wrap(errors.ErrCodeUploadFailed, "读取文件失败", err)
	}

	// 生成BOS路径: knowledge/{kb_uuid}/{yyyy}/{MM}/{dd}/{原始文件名}_{时间戳}.{ext}
	now := time.Now()
	ext := filepath.Ext(data.File.Filename)
	// 获取不带扩展名的原始文件名
	originalName := strings.TrimSuffix(data.File.Filename, ext)
	// 替换文件名中的特殊字符，避免路径问题
	originalName = sanitizeFilename(originalName)
	// 时间戳格式: yyyyMMddHHmmss
	timestamp := now.Format("20060102150405")
	bosPath := fmt.Sprintf("knowledge/%s/%s/%02d/%02d/%s_%s%s",
		data.KnowledgebaseUUID,
		now.Format("2006"),
		int(now.Month()),
		now.Day(),
		originalName,
		timestamp,
		ext,
	)

	// 上传到BOS
	contentType := getContentType(ext)
	fileURL, err := h.bosClient.PutObject(bosPath, fileData, contentType)
	if err != nil {
		return nil, errors.Wrap(errors.ErrCodeUploadFailed, "上传文件到BOS失败", err)
	}

	// 仅对文本类文件填充Content字段，音频/视频/图片等二进制文件不填充
	var textContent string
	if h.fileType == constants.KnowledgeTypeTextFile {
		textContent = string(fileData)
	}

	return &ProcessResult{
		Content:    textContent,
		FilePath:   bosPath,
		FileSize:   data.File.Size,
		FileFormat: strings.TrimPrefix(ext, "."),
		Metadata: map[string]interface{}{
			"originalName": data.File.Filename,
			"fileUrl":      fileURL,
		},
	}, nil
}

// Delete 删除文件
func (h *FileHandler) Delete(knowledge *domain.Knowledge) error {
	if knowledge.FilePath != "" {
		return h.bosClient.DeleteObject(knowledge.FilePath)
	}
	return nil
}

// getContentType 根据扩展名获取Content-Type
func getContentType(ext string) string {
	contentTypes := map[string]string{
		".pdf":  "application/pdf",
		".doc":  "application/msword",
		".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		".txt":  "text/plain",
		".md":   "text/markdown",
		".rtf":  "application/rtf",
		".mp3":  "audio/mpeg",
		".wav":  "audio/wav",
		".m4a":  "audio/mp4",
		".flac": "audio/flac",
		".aac":  "audio/aac",
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".png":  "image/png",
		".gif":  "image/gif",
		".webp": "image/webp",
		".bmp":  "image/bmp",
		".mp4":  "video/mp4",
		".avi":  "video/x-msvideo",
		".mov":  "video/quicktime",
		".wmv":  "video/x-ms-wmv",
		".flv":  "video/x-flv",
	}
	if ct, ok := contentTypes[ext]; ok {
		return ct
	}
	return "application/octet-stream"
}

// sanitizeFilename 清理文件名中的特殊字符，避免路径问题
func sanitizeFilename(name string) string {
	// 替换Windows和Unix不允许的字符: \ / : * ? " < > |
	re := regexp.MustCompile(`[\\/:*?"<>|]`)
	name = re.ReplaceAllString(name, "_")
	// 去除首尾空格
	name = strings.TrimSpace(name)
	// 如果文件名为空，使用默认名称
	if name == "" {
		name = "file"
	}
	return name
}
