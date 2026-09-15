// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package loader

import (
	"fmt"
	"os"
	"path/filepath"

	"icode.baidu.com/baidu/devops-ai/knowbase-client/internal/storage"
	"icode.baidu.com/baidu/devops-ai/knowbase-client/internal/utils"
	"icode.baidu.com/baidu/devops-ai/knowbase-client/pkg/types"
)

// SkillGenerator skill生成器
type SkillGenerator struct {
	storage *storage.LocalStorage
	config  *types.KnowledgeSkillConfig
}

// NewSkillGenerator 创建skill生成器
func NewSkillGenerator(storage *storage.LocalStorage, config *types.KnowledgeSkillConfig) *SkillGenerator {
	return &SkillGenerator{
		storage: storage,
		config:  config,
	}
}

// Generate 生成skill文件
func (g *SkillGenerator) Generate() error {
	// 1. 确定目标路径
	targetPath, err := g.determineTargetPath()
	if err != nil {
		return fmt.Errorf("failed to determine target path: %w", err)
	}

	// 2. 检查是否已存在，如果存在则跳过
	skillFile := filepath.Join(targetPath, "SKILL.md")
	if utils.FileExists(skillFile) {
		fmt.Printf("Knowledge skill already exists at: %s, skipping generation\n", targetPath)
		return nil
	}

	// 3. 获取skill内容
	content, isCreateRefs := g.getSkillContent()

	// 4. 创建skill文件
	if err := g.createSkillFiles(targetPath, content, isCreateRefs); err != nil {
		return fmt.Errorf("failed to create skill files: %w", err)
	}

	fmt.Printf("Knowledge skill generated successfully at: %s\n", targetPath)
	return nil
}

// determineTargetPath 确定skill目标路径
func (g *SkillGenerator) determineTargetPath() (string, error) {
	skillName := g.config.Title
	if skillName == "" {
		skillName = types.DefaultSkillName
	}

	var basePath string

	// 判断集成模式还是独立模式
	if g.storage.IsIntegratedMode() {
		// 集成模式: 主代码库根目录/.claude/skills/<skill-name>/
		primaryPath, err := g.storage.GetPrimaryRepoPath()
		if err != nil {
			return "", fmt.Errorf("failed to get primary repo path: %w", err)
		}
		basePath = filepath.Join(primaryPath, ".claude", "skills", skillName)
	} else {
		// 独立模式: ~/.claude/skills/<skill-name>/
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}
		basePath = filepath.Join(homeDir, ".claude", "skills", skillName)
	}

	return basePath, nil
}

// getSkillContent 获取skill内容
// 返回自定义内容和是否创建references目录
func (g *SkillGenerator) getSkillContent() (string, bool) {
	// 如果用户配置了自定义内容,使用用户配置
	if g.config.Content != "" {
		return g.config.Content, false
	}

	// 否则使用默认模板
	return types.DefaultSkillContent, true
}

// createSkillFiles 创建skill文件和目录结构
func (g *SkillGenerator) createSkillFiles(targetPath string, content string, isCreateRefs bool) error {
	// 1. 确保目标目录存在
	if err := utils.EnsureDir(targetPath); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	// 2. 写入SKILL.md
	skillFile := filepath.Join(targetPath, "SKILL.md")
	if err := os.WriteFile(skillFile, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write SKILL.md: %w", err)
	}

	// 3. 创建references目录
	if isCreateRefs {
		referencesDir := filepath.Join(targetPath, "references")
		if err := utils.EnsureDir(referencesDir); err != nil {
			return fmt.Errorf("failed to create references directory: %w", err)
		}

		// 4. 写入knowledge-structure.md
		knowledgeStructureFile := filepath.Join(referencesDir, "knowledge-structure.md")
		if err := os.WriteFile(knowledgeStructureFile, []byte(types.KnowledgeStructureContent), 0644); err != nil {
			return fmt.Errorf("failed to write knowledge-structure.md: %w", err)
		}

		// 5. 写入source-types.md
		sourceTypesFile := filepath.Join(referencesDir, "source-types.md")
		if err := os.WriteFile(sourceTypesFile, []byte(types.SourceTypesContent), 0644); err != nil {
			return fmt.Errorf("failed to write source-types.md: %w", err)
		}
	}
	return nil
}
