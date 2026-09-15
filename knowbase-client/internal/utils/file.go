// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package utils

import (
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"
)

// ExpandPath 展开路径（支持～和相对路径）
func ExpandPath(path string) (string, error) {
	if path == "" {
		return "", nil
	}

	// 处理～开头的路径
	if strings.HasPrefix(path, "~/") || path == "~" {
		usr, err := user.Current()
		if err != nil {
			return "", err
		}
		if path == "~" {
			return usr.HomeDir, nil
		}
		return filepath.Join(usr.HomeDir, path[2:]), nil
	}

	return path, nil
}

// SanitizeFilename 清理文件名，只保留中文、英文、数字
func SanitizeFilename(filename string) string {
	// 只保留中文、英文、数字
	// \p{Han} 匹配中文字符
	// a-zA-Z 匹配英文字母
	// 0-9 匹配数字
	validChars := regexp.MustCompile(`[^\p{Han}a-zA-Z0-9._-]+`)
	sanitized := validChars.ReplaceAllString(filename, "")

	// 去除前后空格（虽然空格已经被移除，但保留以防万一）
	sanitized = strings.TrimSpace(sanitized)

	// 限制文件名长度（考虑UTF-8编码，中文字符占3字节）
	// 最多200字节，避免超过文件系统限制
	if len(sanitized) > 200 {
		// 安全截断，避免截断多字节UTF-8字符
		sanitized = truncateUTF8(sanitized, 200)
	}

	// 如果文件名为空，使用默认名称
	if sanitized == "" {
		sanitized = "untitled"
	}

	return sanitized
}

// truncateUTF8 安全地截断UTF-8字符串到指定字节数
func truncateUTF8(s string, maxBytes int) string {
	if len(s) <= maxBytes {
		return s
	}

	// 从maxBytes位置向前查找，找到完整的UTF-8字符边界
	for i := maxBytes; i > 0; i-- {
		if utf8.RuneStart(s[i]) {
			return s[:i]
		}
	}
	return s[:maxBytes]
}

// FileExists 检查文件或目录是否存在
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// EnsureDir 确保目录存在
func EnsureDir(path string) error {
	if !FileExists(path) {
		return os.MkdirAll(path, 0755)
	}
	return nil
}

// IsDir 判断路径是否为目录
func IsDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// check whether file is symbol link
// Args:
//
//	filepath: file path
//
// Returns:
//
//	isSymbolLink: true or false
//	err          : nil stands for success, otherwise stands for fail
func IsSymbolLink(filePath string) (isSymbolLink bool, err error) {
	fileInfo, err := os.Lstat(filePath)
	if err != nil {
		errMsg := fmt.Sprintf("Fail to get file info of %s: %s", filePath, err.Error())
		return false, errors.New(errMsg)
	}
	return fileInfo.Mode()&os.ModeSymlink != 0, nil
}
