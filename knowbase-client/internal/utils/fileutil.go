// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package utils

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// CreateJunction 创建 Windows 目录链接（Junction）
func CreateJunction(src, dest string) error {
	if runtime.GOOS != "windows" {
		return fmt.Errorf("junction is only supported on Windows")
	}

	// 使用 mklink 命令创建 Junction
	cmd := exec.Command("cmd", "/c", "mklink", "/J", dest, src)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create junction: %w, output: %s", err, string(output))
	}

	return nil
}

// CopyRecursive 递归复制文件或目录
func CopyRecursive(src, dest string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("failed to stat source: %w", err)
	}

	if srcInfo.IsDir() {
		return copyDir(src, dest)
	}
	return copyFile(src, dest)
}

// copyDir 递归复制目录
func copyDir(src, dest string) error {
	// 创建目标目录
	if err := os.MkdirAll(dest, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// 读取源目录
	entries, err := os.ReadDir(src)
	if err != nil {
		return fmt.Errorf("failed to read source directory: %w", err)
	}

	// 递归复制每个条目
	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		destPath := filepath.Join(dest, entry.Name())

		if entry.IsDir() {
			if err := copyDir(srcPath, destPath); err != nil {
				return err
			}
		} else {
			if err := copyFile(srcPath, destPath); err != nil {
				return err
			}
		}
	}

	return nil
}

// copyFile 复制单个文件
func copyFile(src, dest string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer srcFile.Close()

	destFile, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, srcFile); err != nil {
		return fmt.Errorf("failed to copy file content: %w", err)
	}

	// 复制文件权限
	srcInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("failed to stat source file: %w", err)
	}
	if err := os.Chmod(dest, srcInfo.Mode()); err != nil {
		return fmt.Errorf("failed to set file permissions: %w", err)
	}

	return nil
}
