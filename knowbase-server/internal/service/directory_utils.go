// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package service

import (
	"strings"

	"icode.baidu.com/baidu/devops-ai/knowbase-server/internal/domain"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/log"
)

const maxDirDepth = 6

// BuildDirPath 根据 dirUUID 和目录 map 构建完整目录路径（如 "父目录/子目录"）。
// dirUUID 为 nil 时，查找 is_default=true 的目录作为路径；Name 为空则返回 "默认目录"。
// 最大向上遍历深度为 maxDirDepth，防止脏数据导致死循环。
func BuildDirPath(dirUUID *string, dirMap map[string]*domain.KnowledgebaseDirectory) string {
	const defaultDirName = "默认目录"

	if dirUUID == nil || *dirUUID == "" {
		// 查找默认目录
		for _, d := range dirMap {
			if d.IsDefault {
				if d.Name == "" {
					return defaultDirName
				}
				return d.Name
			}
		}
		return defaultDirName
	}

	// 向上遍历收集路径段
	parts := make([]string, 0, maxDirDepth)
	visited := make(map[string]bool)
	cur := *dirUUID

	for depth := 0; depth < maxDirDepth; depth++ {
		if visited[cur] {
			log.Logger.Warnf("BuildDirPath: 检测到目录循环引用, uuid=%s", cur)
			break
		}
		visited[cur] = true

		d, ok := dirMap[cur]
		if !ok {
			break
		}

		name := d.Name
		if name == "" {
			name = defaultDirName
		}
		parts = append(parts, name)

		if d.ParentUUID == nil || *d.ParentUUID == "" {
			break
		}
		cur = *d.ParentUUID
	}

	if len(parts) == 0 {
		return defaultDirName
	}

	// 反转：从根到叶
	for i, j := 0, len(parts)-1; i < j; i, j = i+1, j-1 {
		parts[i], parts[j] = parts[j], parts[i]
	}
	return strings.Join(parts, "/")
}

// CollectSubtreeUUIDs 使用 BFS 收集以 rootUUID 为根的所有后代目录 UUID（不含 rootUUID 本身）
// 供 directory_service 和 knowledge_apply_service 共享
func CollectSubtreeUUIDs(rootUUID string, allDirs []*domain.KnowledgebaseDirectory) []string {
	// 构建 parent -> children 映射
	childMap := make(map[string][]string)
	for _, d := range allDirs {
		if d.ParentUUID != nil {
			childMap[*d.ParentUUID] = append(childMap[*d.ParentUUID], d.UUID)
		}
	}

	// visited 防止脏数据中的环导致死循环
	visited := make(map[string]bool)
	visited[rootUUID] = true

	// 使用索引指针代替 queue[1:] 截断，避免底层数组无法 GC 的问题
	result := make([]string, 0)
	queue := []string{rootUUID}
	head := 0
	for head < len(queue) {
		cur := queue[head]
		head++
		for _, child := range childMap[cur] {
			if visited[child] {
				continue
			}
			visited[child] = true
			result = append(result, child)
			queue = append(queue, child)
		}
	}
	return result
}
