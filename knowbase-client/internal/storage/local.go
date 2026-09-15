// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"icode.baidu.com/baidu/devops-ai/knowbase-client/internal/utils"
	"icode.baidu.com/baidu/devops-ai/knowbase-client/pkg/types"
)

// LocalStorage 本地存储管理器
type LocalStorage struct {
	basePath    string
	primaryRepo *types.PrimaryRepoConfig
}

// NewLocalStorage 创建本地存储管理器
func NewLocalStorage(basePath string, primaryRepo *types.PrimaryRepoConfig) (*LocalStorage, error) {
	// 计算实际的basePath
	actualBasePath, err := calculateBasePath(basePath, primaryRepo)
	if err != nil {
		return nil, err
	}

	// 展开路径
	expandedPath, err := utils.ExpandPath(actualBasePath)
	if err != nil {
		return nil, fmt.Errorf("failed to expand path %s: %w", actualBasePath, err)
	}

	// 创建基础目录
	if err := utils.EnsureDir(expandedPath); err != nil {
		return nil, fmt.Errorf("failed to create base directory %s: %w", expandedPath, err)
	}

	return &LocalStorage{
		basePath:    expandedPath,
		primaryRepo: primaryRepo,
	}, nil
}

// calculateBasePath 计算实际的basePath
func calculateBasePath(basePath string, primaryRepo *types.PrimaryRepoConfig) (string, error) {
	// 独立模式：直接使用basePath
	if primaryRepo == nil || !primaryRepo.Enabled {
		return basePath, nil
	}

	// 集成模式 - 本地：使用 primaryRepo.path + knowledgeSubDir
	if primaryRepo.Type == "local" {
		if primaryRepo.Path == "" {
			return "", fmt.Errorf("primaryRepo.path is required when type is 'local'")
		}
		knowledgeSubDir := primaryRepo.KnowledgeSubDir
		if knowledgeSubDir == "" {
			knowledgeSubDir = ".knowledge"
		}
		return filepath.Join(primaryRepo.Path, knowledgeSubDir), nil
	}

	// 集成模式 - 远程：使用 basePath + repo + knowledgeSubDir
	if primaryRepo.Type == "remote" {
		if primaryRepo.Repo == "" {
			return "", fmt.Errorf("primaryRepo.repo is required when type is 'remote'")
		}
		if basePath == "" {
			return "", fmt.Errorf("storage.basePath is required when primaryRepo.type is 'remote'")
		}
		knowledgeSubDir := primaryRepo.KnowledgeSubDir
		if knowledgeSubDir == "" {
			knowledgeSubDir = ".knowledge"
		}
		return filepath.Join(basePath, primaryRepo.Repo, knowledgeSubDir), nil
	}

	return "", fmt.Errorf("unknown primaryRepo.type: %s", primaryRepo.Type)
}

// GetBasePath 获取基础路径
func (s *LocalStorage) GetBasePath() string {
	return s.basePath
}

// GetSubDir 获取子目录路径
func (s *LocalStorage) GetSubDir(subDir string) string {
	return filepath.Join(s.basePath, subDir)
}

// EnsureSubDir 确保子目录存在
func (s *LocalStorage) EnsureSubDir(subDir string) (string, error) {
	subPath := s.GetSubDir(subDir)
	if err := utils.EnsureDir(subPath); err != nil {
		return "", fmt.Errorf("failed to create sub directory %s: %w", subPath, err)
	}
	return subPath, nil
}

// WriteFile 写入文件到本地存储
//
// path: 文件路径，可以是相对路径或绝对路径
// content: 要写入的文件内容
// usePathDirect: 是否直接使用path参数而不拼接basePath
//
// 返回值: 成功返回nil，失败返回错误信息
func (s *LocalStorage) WriteFile(path string, content []byte, usePathDirect bool) error {
	realPath, err := utils.ExpandPath(path)
	if err != nil {
		return fmt.Errorf("failed to expand path %s: %w", path, err)
	}
	if usePathDirect {
		// 直接使用path，不进行拼接
	} else {
		if !strings.HasPrefix(realPath, s.basePath) {
			realPath = filepath.Join(s.basePath, realPath)
		}
	}

	// 确保父目录存在
	dir := filepath.Dir(realPath)
	if err := utils.EnsureDir(dir); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// 写入文件
	if err := os.WriteFile(realPath, content, 0644); err != nil {
		return fmt.Errorf("failed to write file %s: %w", realPath, err)
	}

	return nil
}

// ReadFile 读取文件
func (s *LocalStorage) ReadFile(relativePath string) ([]byte, error) {
	filePath := filepath.Join(s.basePath, relativePath)
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", filePath, err)
	}
	return content, nil
}

// FileExists 检查文件是否存在
func (s *LocalStorage) FileExists(relativePath string) bool {
	filePath := filepath.Join(s.basePath, relativePath)
	return utils.FileExists(filePath)
}

// CreateSymlink 创建符号链接（跨平台优化）
func (s *LocalStorage) CreateSymlink(srcPath, destRelativePath string) error {
	// 展开源路径
	expandedSrc, err := utils.ExpandPath(srcPath)
	if err != nil {
		return fmt.Errorf("failed to expand source path %s: %w", srcPath, err)
	}

	// 检查源路径是否存在
	if !utils.FileExists(expandedSrc) {
		return fmt.Errorf("source path does not exist: %s", expandedSrc)
	}

	// 目标路径
	destPath := filepath.Join(s.basePath, destRelativePath)

	// 确保父目录存在
	dir := filepath.Dir(destPath)
	if err := utils.EnsureDir(dir); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// 如果目标已存在，先删除
	if utils.FileExists(destPath) {
		if err := os.RemoveAll(destPath); err != nil {
			return fmt.Errorf("failed to remove existing path %s: %w", destPath, err)
		}
	}

	// 判断源是文件还是目录
	srcInfo, err := os.Stat(expandedSrc)
	if err != nil {
		return fmt.Errorf("failed to stat source path %s: %w", expandedSrc, err)
	}

	// Windows 特殊处理
	if runtime.GOOS == "windows" {
		return s.createWindowsSymlink(expandedSrc, destPath, srcInfo.IsDir())
	}

	// Unix/Linux/macOS: 使用标准符号链接
	if err := os.Symlink(expandedSrc, destPath); err != nil {
		return fmt.Errorf("failed to create symlink from %s to %s: %w", expandedSrc, destPath, err)
	}

	return nil
}

// createWindowsSymlink Windows 平台符号链接创建
func (s *LocalStorage) createWindowsSymlink(src, dest string, isDir bool) error {
	// 尝试创建符号链接
	if err := os.Symlink(src, dest); err == nil {
		return nil
	}

	// 如果失败，尝试使用 Junction（仅目录）
	if isDir {
		if err := utils.CreateJunction(src, dest); err == nil {
			fmt.Printf("Created directory junction: %s -> %s\n", dest, src)
			return nil
		}
	}

	// 都失败，返回错误（editable=true 时不能降级）
	return fmt.Errorf("failed to create symlink or junction from %s to %s (insufficient permissions)", src, dest)
}

// CreateSymlinkOrCopy 创建符号链接，失败则降级为复制
func (s *LocalStorage) CreateSymlinkOrCopy(srcPath, destRelativePath string) error {
	// 尝试创建符号链接
	if err := s.CreateSymlink(srcPath, destRelativePath); err == nil {
		return nil
	}

	// 符号链接失败，降级为复制
	expandedSrc, err := utils.ExpandPath(srcPath)
	if err != nil {
		return fmt.Errorf("failed to expand source path %s: %w", srcPath, err)
	}

	destPath := filepath.Join(s.basePath, destRelativePath)

	utils.PrintlnWarn("Creating copy instead of symlink: %s -> %s", destPath, expandedSrc)

	// 使用复制
	if err := utils.CopyRecursive(expandedSrc, destPath); err != nil {
		return fmt.Errorf("failed to copy from %s to %s: %w", expandedSrc, destPath, err)
	}

	return nil
}

// GetFullPath 获取完整路径
func (s *LocalStorage) GetFullPath(relativePath string) string {
	return filepath.Join(s.basePath, relativePath)
}

// IsIntegratedMode 判断是否为集成模式
func (s *LocalStorage) IsIntegratedMode() bool {
	return s.primaryRepo != nil && s.primaryRepo.Enabled
}

// GetPrimaryRepoPath 获取主代码库路径（仅集成模式）
func (s *LocalStorage) GetPrimaryRepoPath() (string, error) {
	if !s.IsIntegratedMode() {
		return "", fmt.Errorf("not in integrated mode")
	}

	if s.primaryRepo.Type == "local" {
		expandedPath, err := utils.ExpandPath(s.primaryRepo.Path)
		if err != nil {
			return "", fmt.Errorf("failed to expand primary repo path: %w", err)
		}
		return expandedPath, nil
	}

	// remote类型：需要从basePath中推导出主代码库路径
	// basePath格式: {workDir}/{repo}/.knowledge
	// 需要去掉最后的.knowledge部分
	knowledgeSubDir := s.primaryRepo.KnowledgeSubDir
	if knowledgeSubDir == "" {
		knowledgeSubDir = ".knowledge"
	}

	// 从basePath中移除knowledgeSubDir
	if strings.HasSuffix(s.basePath, knowledgeSubDir) {
		repoPath := strings.TrimSuffix(s.basePath, knowledgeSubDir)
		repoPath = strings.TrimSuffix(repoPath, string(filepath.Separator))
		return repoPath, nil
	}

	return "", fmt.Errorf("failed to calculate primary repo path from basePath: %s", s.basePath)
}
