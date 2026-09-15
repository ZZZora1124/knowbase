// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package loader

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"icode.baidu.com/baidu/devops-ai/knowbase-client/internal/utils"
)

// AgentGuideTargets 检测到的目标文件/目录
type AgentGuideTargets struct {
	ClaudeMdPath   string // CLAUDE.md 文件路径
	AgentMdPath    string // agent.md 文件路径
	ComateRulesDir string // .comate/rules 目录路径
}

// generateAgentGuide 生成Agent引导内容(主入口函数)
func (m *Manager) generateAgentGuide() error {
	// 获取主代码库路径
	primaryRepoPath, err := m.storage.GetPrimaryRepoPath()
	if err != nil {
		return fmt.Errorf("failed to get primary repo path: %w", err)
	}

	// 获取知识目录路径
	knowledgeBasePath := m.storage.GetBasePath()

	fmt.Printf("Generating agent guide, primaryRepo: %s, knowledgeBase: %s\n", primaryRepoPath, knowledgeBasePath)

	// 检测目标文件/目录
	targets := detectAgentGuideTargets(primaryRepoPath)

	// 如果没有任何目标,跳过
	if targets.ClaudeMdPath == "" && targets.AgentMdPath == "" && targets.ComateRulesDir == "" {
		fmt.Println("No agent guide targets found (CLAUDE.md, agent.md, or .comate/rules/), skipping")
		return nil
	}

	// 处理 CLAUDE.md
	if targets.ClaudeMdPath != "" {
		if err := appendAgentGuideToMarkdown(targets.ClaudeMdPath, knowledgeBasePath); err != nil {
			utils.PrintlnError("Failed to append guide to CLAUDE.md: %v", err)
		} else {
			fmt.Printf("Successfully appended agent guide to CLAUDE.md: %s\n", targets.ClaudeMdPath)
		}
	}

	// 处理 agent.md
	if targets.AgentMdPath != "" {
		if err := appendAgentGuideToMarkdown(targets.AgentMdPath, knowledgeBasePath); err != nil {
			utils.PrintlnError("Failed to append guide to agent.md: %v\n", err)
		} else {
			fmt.Printf("Successfully appended agent guide to agent.md: %s\n", targets.AgentMdPath)
		}
	}

	// 处理 .comate/rules/
	if targets.ComateRulesDir != "" {
		mdrPath := filepath.Join(targets.ComateRulesDir, "knowledge_usage_readme.mdr")
		if err := createComateRuleFile(mdrPath, knowledgeBasePath); err != nil {
			utils.PrintlnError("Failed to create .comate/rules/knowledge_usage_readme.mdr: %v", err)
		} else {
			fmt.Printf("Successfully created .comate/rules/knowledge_usage_readme.mdr: %s\n", mdrPath)
		}
	}

	return nil
}

// detectAgentGuideTargets 检测主代码库中存在的目标文件/目录
func detectAgentGuideTargets(primaryRepoPath string) AgentGuideTargets {
	targets := AgentGuideTargets{}

	// 检查 CLAUDE.md
	claudeMdPath := filepath.Join(primaryRepoPath, "CLAUDE.md")
	if utils.FileExists(claudeMdPath) {
		targets.ClaudeMdPath = claudeMdPath
	}

	// 检查 agent.md
	agentMdPath := filepath.Join(primaryRepoPath, "agent.md")
	if utils.FileExists(agentMdPath) {
		targets.AgentMdPath = agentMdPath
	}

	// 检查 .comate/rules/ 目录
	comateRulesDir := filepath.Join(primaryRepoPath, ".comate", "rules")
	if utils.FileExists(comateRulesDir) {
		if info, err := os.Stat(comateRulesDir); err == nil && info.IsDir() {
			targets.ComateRulesDir = comateRulesDir
		}
	}

	return targets
}

// generateClaudeAgentGuideContent 生成追加到 CLAUDE.md/agent.md 的内容
func generateClaudeAgentGuideContent(knowledgeBasePath string) string {
	var sb strings.Builder

	sb.WriteString("\n\n")
	sb.WriteString("## 知识使用说明\n")
	sb.WriteString("在处理任何任务前，必须先理解各知识源的组织方式与使用规则。\n")
	sb.WriteString(fmt.Sprintf("%s下包含一个知识目录说明文件knowledge_usage_readme.md，此外每个知识项目录下均包含一个以目录名命名的 `{directory}_usage.md` 文件，**描述该目录的用途、结构及使用方法**。\n\n", knowledgeBasePath))

	sb.WriteString("## 执行原则\n")
	sb.WriteString("1. 按需读取用法说明：\n")
	sb.WriteString("当需要访问某一类知识(如需求描述、代码、接口等)时，必须先读取其对应目录下的用法文档：\n")
	sb.WriteString(fmt.Sprintf("- 需求卡片：读 `%s/icafe_card/icafe_card_usage.md`\n", knowledgeBasePath))
	sb.WriteString(fmt.Sprintf("- 源码类：读 `%s/icode_repo/icode_repo_usage.md`\n", knowledgeBasePath))
	sb.WriteString(fmt.Sprintf("- 接口契约：读 `%s/iapi_doc/iapi_doc_usage.md`\n", knowledgeBasePath))
	sb.WriteString(fmt.Sprintf("- deepwiki(代码总结归纳后的文件)：读 `%s/icode_deepwiki/icode_deepwiki_usage.md`\n", knowledgeBasePath))
	sb.WriteString(fmt.Sprintf("- GitHub仓库：读 `%s/github_repo/github_repo_usage.md`\n", knowledgeBasePath))
	sb.WriteString("- 其他知识内容类似，以此类推。\n\n")

	sb.WriteString("2. 不得凭经验猜测目录内容，一切操作必须以对应 `{directory}_usage.md` 的说明为准。\n\n")

	sb.WriteString("非常重要：不读 xx_usage.md，就不知道怎么用。用错知识源，等于无源输出。\n")

	return sb.String()
}

// generateComateRuleContent 生成 .comate/rules/knowledge_usage_readme.mdr 的内容
func generateComateRuleContent(knowledgeBasePath string) string {
	var sb strings.Builder

	sb.WriteString("---\n")
	sb.WriteString("alwaysApply: true\n")
	sb.WriteString("---\n\n")

	sb.WriteString("## 知识使用说明\n")
	sb.WriteString("在处理任何任务前，必须先理解各知识源的组织方式与使用规则。\n")
	sb.WriteString(fmt.Sprintf("%s下包含一个知识目录说明文件knowledge_usage_readme.md，此外每个知识项目录下均包含一个以目录名命名的 `{directory}_usage.md` 文件，**描述该目录的用途、结构及使用方法**。\n\n", knowledgeBasePath))

	sb.WriteString("## 执行原则\n")
	sb.WriteString("1. 按需读取用法说明：\n")
	sb.WriteString("   当需要访问某一类知识(如需求描述、代码、接口等)时，必须先读取其对应目录下的用法文档：\n")
	sb.WriteString(fmt.Sprintf("- 需求卡片：读 `%s/icafe_card/icafe_card_usage.md`\n", knowledgeBasePath))
	sb.WriteString(fmt.Sprintf("- 源码类：读 `%s/icode_repo/icode_repo_usage.md`\n", knowledgeBasePath))
	sb.WriteString(fmt.Sprintf("- 接口契约：读 `%s/iapi_doc/iapi_doc_usage.md`\n", knowledgeBasePath))
	sb.WriteString(fmt.Sprintf("- deepwiki(代码总结归纳后的文件)：读 `%s/icode_deepwiki/icode_deepwiki_usage.md`\n", knowledgeBasePath))
	sb.WriteString(fmt.Sprintf("- GitHub仓库：读 `%s/github_repo/github_repo_usage.md`\n", knowledgeBasePath))
	sb.WriteString("- 其他知识内容类似，以此类推。\n\n")

	sb.WriteString("2. 不得凭经验猜测目录内容，一切操作必须以对应 `{directory}_usage.md` 的说明为准。\n\n")

	sb.WriteString("非常重要：不读 xx_usage.md，就不知道怎么用。用错知识源，等于无源输出。\n")

	return sb.String()
}

// appendAgentGuideToMarkdown 追加内容到已存在的 Markdown 文件
func appendAgentGuideToMarkdown(filePath string, knowledgeBasePath string) error {
	// 读取现有文件内容
	existingContent, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	// 检查是否已包含知识使用说明(幂等性保证)
	if strings.Contains(string(existingContent), "## 知识使用说明") {
		fmt.Printf("File %s already contains agent guide, skipping\n", filePath)
		return nil
	}

	// 生成新内容
	guideContent := generateClaudeAgentGuideContent(knowledgeBasePath)

	// 追加到文件末尾
	newContent := string(existingContent) + guideContent

	// 写回文件
	if err := os.WriteFile(filePath, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("failed to write file %s: %w", filePath, err)
	}

	return nil
}

// createComateRuleFile 创建 .comate/rules/knowledge_usage_readme.mdr 文件
func createComateRuleFile(filePath string, knowledgeBasePath string) error {
	// 检查文件是否已存在
	if utils.FileExists(filePath) {
		// 读取现有内容,检查是否已包含知识使用说明
		existingContent, err := os.ReadFile(filePath)
		if err == nil && strings.Contains(string(existingContent), "## 知识使用说明") {
			fmt.Printf("File %s already contains agent guide, skipping\n", filePath)
			return nil
		}
	}

	// 确保父目录存在
	dir := filepath.Dir(filePath)
	if err := utils.EnsureDir(dir); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// 生成内容
	content := generateComateRuleContent(knowledgeBasePath)

	// 写入文件
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write file %s: %w", filePath, err)
	}

	return nil
}
