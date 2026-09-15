// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package loader

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"icode.baidu.com/baidu/devops-ai/knowbase-client/internal/client"
	"icode.baidu.com/baidu/devops-ai/knowbase-client/internal/storage"
	"icode.baidu.com/baidu/devops-ai/knowbase-client/internal/utils"
	"icode.baidu.com/baidu/devops-ai/knowbase-client/pkg/types"
)

// Manager 加载器管理器
type Manager struct {
	factory    *Factory
	storage    *storage.LocalStorage
	config     *types.KnowledgeConfig
	collector  *utils.StatsCollector
	credential *types.Credential
}

// DirectoryNode 目录树节点
type DirectoryNode struct {
	Name     string
	IsDir    bool
	Children []*DirectoryNode
	RelPath  string
}

// NewManager 创建加载器管理器
func NewManager(cred *types.Credential, storage *storage.LocalStorage, config *types.KnowledgeConfig) *Manager {
	collector := utils.NewStatsCollector(cred.Username)

	// 创建并设置指标上报客户端
	metricClient := client.NewMetricClientWithCredential(cred)
	collector.SetMetricReporter(metricClient)

	return &Manager{
		factory:    NewFactory(cred),
		storage:    storage,
		config:     config,
		collector:  collector,
		credential: cred,
	}
}

// LoadAll 加载所有知识源
func (m *Manager) LoadAll(ctx context.Context) error {
	// 记录开始时间
	startTime := time.Now()

	fmt.Printf("Starting knowledge loading, user: %s, sources: %d\n", m.credential.Username, len(m.config.Sources))

	// 第一步：初始化主代码库（如果启用了集成模式）
	if err := m.initializePrimaryRepo(ctx); err != nil {
		return fmt.Errorf("failed to initialize primary repository: %w", err)
	}

	// 使用channel收集结果
	resultChan := make(chan types.LoadResult, len(m.config.Sources))
	var wg sync.WaitGroup

	// 并发加载各个知识源
	for _, source := range m.config.Sources {
		// 跳过未启用的知识源
		if !source.Enabled {
			fmt.Printf("Skipping disabled source, type: %s\n", source.Type)
			continue
		}

		wg.Add(1)
		go func(src types.SourceConfig) {
			defer wg.Done()
			result := m.loadSource(ctx, src)
			resultChan <- result
		}(source)
	}

	// 等待所有加载完成
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// 收集结果
	for result := range resultChan {
		m.collector.AddResult(result)
	}

	// 计算实际总耗时（墙钟时间）
	actualDuration := time.Since(startTime)
	m.collector.SetActualDuration(actualDuration)

	// 生成入口使用说明
	if err := m.generateEntrypointUsage(); err != nil {
		utils.PrintlnError("Failed to generate entrypoint usage: %v", err.Error())
	}

	// 生成Agent引导内容(仅集成模式)
	if m.storage.IsIntegratedMode() {
		if err := m.generateAgentGuide(); err != nil {
			utils.PrintlnError("Failed to generate agent guide: %v", err.Error())
		}
	}

	// 生成Knowledge Skill(如果启用)
	if err := m.generateKnowledgeSkill(); err != nil {
		// skill生成失败不影响整体流程,只打印警告
		utils.PrintlnError("Failed to generate knowledge skill: %v", err.Error())
	}

	// 上报指标到服务端（不影响主流程）
	m.collector.ReportMetrics()

	// 打印统计信息
	m.collector.PrintStats()

	// 检查是否有失败的加载
	stats := m.collector.GetStats()
	if stats.FailedSources > 0 {
		return fmt.Errorf("failed to load %d source(s)", stats.FailedSources)
	}

	return nil
}

// loadSource 加载单个知识源
func (m *Manager) loadSource(ctx context.Context, source types.SourceConfig) types.LoadResult {
	startTime := time.Now()
	result := types.LoadResult{
		SourceType: source.Type,
		Success:    false,
	}

	fmt.Printf("Start loading source, type: %s\n", source.Type)

	// 创建加载器
	loader, err := m.factory.CreateLoader(source.Type)
	if err != nil {
		// 提取详细错误信息
		m.extractErrorDetails(&result, err)
		if result.ErrorMessage == "" {
			result.ErrorMessage = fmt.Sprintf("failed to create loader: %v", err)
		}
		utils.PrintlnError("Failed to create loader, source type: %v, error: %v", source.Type, err.Error())
		return result
	}

	// 1. 加载原始数据
	rawKnowledge, err := loader.Load(ctx, source)
	if err != nil {
		// 提取详细错误信息
		m.extractErrorDetails(&result, err)
		if result.ErrorMessage == "" {
			result.ErrorMessage = fmt.Sprintf("failed to load: %v", err)
		}
		utils.PrintlnError("Failed to load knowledge, source type: %v, error: %v", source.Type, err.Error())
		result.Duration = time.Since(startTime)
		return result
	}

	// 2. 转换格式（如果需要）
	var items []types.KnowledgeItem
	if source.Output.NeedConvert {
		items, err = loader.Convert(source.Type, rawKnowledge, source.Output)
		if err != nil {
			// 提取详细错误信息
			m.extractErrorDetails(&result, err)
			if result.ErrorMessage == "" {
				result.ErrorMessage = fmt.Sprintf("failed to convert: %v", err)
			}
			utils.PrintlnError("Failed to convert knowledge, source type: %v, error: %v", source.Type, err.Error())
			result.Duration = time.Since(startTime)
			return result
		}
	} else {
		// 如果不需要转换，创建简单的知识项
		items = make([]types.KnowledgeItem, len(rawKnowledge))
		for i, raw := range rawKnowledge {
			items[i] = types.KnowledgeItem{
				SourceType: raw.SourceType,
				Metadata:   raw.Data,
			}
		}
	}

	// 3. 存储到本地，并获取存储的知识项详细信息
	storedItems, err := loader.Store(ctx, items, m.storage, source)
	if err != nil {
		// 提取详细错误信息
		m.extractErrorDetails(&result, err)
		if result.ErrorMessage == "" {
			result.ErrorMessage = fmt.Sprintf("failed to store: %v", err)
		}
		utils.PrintlnError("Failed to store knowledge, source type: %v, error: %v", source.Type, err.Error())
		result.Duration = time.Since(startTime)
		return result
	}

	// 4. 生成使用说明(传入storedItems)
	// 确定usage路径
	usagePath := ""
	if source.Usage.Path != "" {
		usagePath = source.Usage.Path
	} else {
		usagePath = filepath.Join(source.Type, fmt.Sprintf("%s_usage.md", source.Type))
	}

	if err := loader.GenerateUsage(m.storage, source.Usage, source.Type, storedItems); err != nil {
		// 使用说明生成失败不算致命错误，只记录警告
		utils.PrintlnError("Failed to generate usage, source type: %v, error: %v", source.Type, err.Error())
	}

	// 5. 收集知识结构信息(包含usagePath)
	m.collector.AddKnowledgeStructure(types.KnowledgeStructure{
		SourceType:  source.Type,
		SubDir:      source.Type,
		Description: getSourceDescription(source),
		StoredItems: storedItems,
		UsagePath:   usagePath, // 记录usage路径
	})

	// 成功，使用实际存储的数量
	result.Success = true
	result.Count = len(storedItems)
	result.StoredItems = storedItems
	result.Duration = time.Since(startTime)

	fmt.Printf("Finish loading source, type: %s, count: %d, duration: %v\n", source.Type, result.Count, result.Duration)

	return result
}

// extractErrorDetails 从错误中提取详细信息到 LoadResult
func (m *Manager) extractErrorDetails(result *types.LoadResult, err error) {
	if err == nil {
		return
	}

	// 尝试转换为 DetailedError
	if detailErr, ok := err.(*types.DetailedError); ok {
		result.ErrorType = detailErr.ErrorType
		result.ErrorMessage = detailErr.ErrorMessage
		result.ErrorDetails = detailErr.Details
		result.Suggestion = detailErr.Suggestion
	} else {
		// 不是 DetailedError，使用默认值
		result.ErrorType = types.ErrorTypeUnknown
		result.ErrorMessage = err.Error()
		result.ErrorDetails = nil
		result.Suggestion = ""
	}
}

// generateEntrypointUsage 生成入口使用说明
func (m *Manager) generateEntrypointUsage() error {
	var content []byte

	// 如果用户没有配置content或为空，自动生成
	if m.config.Entrypoint.Content == "" {
		content = m.generateAutoEntrypointContent()
	} else {
		content = []byte(m.config.Entrypoint.Content)
	}

	// 确定文件路径
	usagePath := m.config.Entrypoint.Path
	if usagePath == "" {
		usagePath = m.storage.GetBasePath() + "/knowledge_usage_readme.md"
		if err := m.storage.WriteFile(usagePath, content, false); err != nil {
			return fmt.Errorf("failed to write entrypoint usage: %w", err)
		}
	} else {
		if err := m.storage.WriteFile(usagePath, content, true); err != nil {
			return fmt.Errorf("failed to write entrypoint usage: %w", err)
		}
	}

	fmt.Printf("Generated entrypoint usage, path: %s\n", usagePath)

	return nil
}

// generateAutoEntrypointContent 自动生成入口使用说明内容
func (m *Manager) generateAutoEntrypointContent() []byte {
	var sb strings.Builder

	sb.WriteString("# 知识目录使用说明\n\n")

	// 1. 项目基本信息
	if m.config.Meta.Description != "" {
		sb.WriteString(fmt.Sprintf("**项目**: %s\n\n", m.config.Meta.Description))
	}
	sb.WriteString(fmt.Sprintf("**知识根目录**: `%s`\n\n", m.storage.GetBasePath()))

	// 2. 目录结构树
	sb.WriteString("## 目录结构\n\n")
	sb.WriteString("```\n")

	structures := m.collector.GetKnowledgeStructures()
	tree := m.buildDirectoryTree(structures)
	sb.WriteString(m.renderTree(tree, "", true))

	sb.WriteString("```\n\n")

	// 3. 各知识子项详细说明
	sb.WriteString("## 知识分类说明\n\n")

	for _, st := range structures {
		sb.WriteString(fmt.Sprintf("### %s\n\n", st.SubDir))

		// 类型说明
		if st.Description != "" {
			sb.WriteString(fmt.Sprintf("**类型**: %s\n\n", st.Description))
		}

		// Usage文档路径
		if st.UsagePath != "" {
			sb.WriteString(fmt.Sprintf("**使用文档**: [`%s`](%s)\n\n",
				filepath.Base(st.UsagePath), st.UsagePath))
		}

		// 包含的文件/目录说明
		if len(st.StoredItems) > 0 {
			sb.WriteString(fmt.Sprintf("**包含内容** (%d项):\n\n", len(st.StoredItems)))

			for _, item := range st.StoredItems {
				// 判断是文件还是目录
				itemType := "文件"
				if isDirectory(item.RelPath, item.FullPath) {
					itemType = "目录"
				}

				sb.WriteString(fmt.Sprintf("- **%s** (%s)\n", item.Name, itemType))

				if item.Description != "" {
					sb.WriteString(fmt.Sprintf("  - 说明: %s\n", item.Description))
				}

				sb.WriteString(fmt.Sprintf("  - 路径: `%s`\n", item.RelPath))
				sb.WriteString("\n")
			}
		} else {
			sb.WriteString("*暂无内容*\n\n")
		}

		sb.WriteString("---\n\n")
	}

	return []byte(sb.String())
}

// getSourceDescription 从source配置中提取描述信息
func getSourceDescription(source types.SourceConfig) string {
	// 优先使用usage中的描述
	if source.Usage.Content != "" {
		// 提取第一行作为简短描述
		lines := strings.Split(source.Usage.Content, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" && !strings.HasPrefix(line, "#") {
				return line
			}
		}
	}

	// 否则根据类型生成默认描述
	switch source.Type {
	case types.SourceTypeICafeCard:
		return "iCafe卡片知识"
	case types.SourceTypeICodeDeepWiki:
		return "iCode Wiki文档"
	case types.SourceTypeICodeRepo:
		return "iCode代码仓库"
	case types.SourceTypeIApiDoc:
		return "iAPI接口文档"
	case types.SourceTypeLocalKnowledge:
		return "本地知识"
	case types.SourceTypeGithubRepo:
		return "GitHub开源代码仓库"
	case types.SourceTypeKuDoc:
		return "知识库文档"
	case types.SourceTypeInfoflowGroupMessage:
		return "如流群组消息"
	case types.SourceTypeICodeCRDiff:
		return "iCode代码评审变更"
	case types.SourceTypeKnowledgeBook:
		return "知识方舟知识"
	case types.SourceTypeIDataset:
		return "数据集"
	default:
		return fmt.Sprintf("%s知识", source.Type)
	}
}

// buildDirectoryTree 构建目录树结构
func (m *Manager) buildDirectoryTree(structures []types.KnowledgeStructure) *DirectoryNode {
	root := &DirectoryNode{
		Name:     filepath.Base(m.storage.GetBasePath()),
		IsDir:    true,
		Children: make([]*DirectoryNode, 0),
	}

	// 为每个知识源创建子目录节点
	for _, st := range structures {
		sourceNode := &DirectoryNode{
			Name:     st.SubDir,
			IsDir:    true,
			Children: make([]*DirectoryNode, 0),
			RelPath:  st.SubDir,
		}

		// 添加usage文档节点(如果存在)
		if st.UsagePath != "" {
			usageNode := &DirectoryNode{
				Name:    filepath.Base(st.UsagePath),
				IsDir:   false,
				RelPath: st.UsagePath,
			}
			sourceNode.Children = append(sourceNode.Children, usageNode)
		}

		// 添加知识项节点（支持层级路径）
		for _, item := range st.StoredItems {
			m.insertHierarchicalPath(sourceNode, item.RelPath, st.SubDir)
		}

		root.Children = append(root.Children, sourceNode)
	}

	return root
}

// insertHierarchicalPath 将层级路径插入目录树，创建中间目录节点
func (m *Manager) insertHierarchicalPath(parent *DirectoryNode, relPath string, subDir string) {
	// 去掉源类型前缀，得到相对路径
	remaining := strings.TrimPrefix(relPath, subDir+"/")
	if remaining == "" {
		return
	}

	// 按路径分隔符拆分
	parts := strings.Split(remaining, string(filepath.Separator))

	current := parent
	for i, part := range parts {
		if i == len(parts)-1 {
			// 叶节点（文件）
			current.Children = append(current.Children, &DirectoryNode{
				Name:    part,
				IsDir:   false,
				RelPath: relPath,
			})
		} else {
			// 目录 - 查找或创建
			found := false
			for _, child := range current.Children {
				if child.IsDir && child.Name == part {
					current = child
					found = true
					break
				}
			}
			if !found {
				newNode := &DirectoryNode{
					Name:     part,
					IsDir:    true,
					Children: make([]*DirectoryNode, 0),
					RelPath:  filepath.Join(subDir, strings.Join(parts[:i+1], string(filepath.Separator))),
				}
				current.Children = append(current.Children, newNode)
				current = newNode
			}
		}
	}
}

// renderTree 渲染目录树(类似tree命令的输出)
func (m *Manager) renderTree(node *DirectoryNode, prefix string, isLast bool) string {
	var sb strings.Builder

	// 当前节点
	if node.Name != "" {
		connector := "├── "
		if isLast {
			connector = "└── "
		}

		sb.WriteString(prefix + connector + node.Name + "\n")
	}

	// 递归渲染子节点
	childPrefix := prefix
	if node.Name != "" {
		if isLast {
			childPrefix += "    "
		} else {
			childPrefix += "│   "
		}
	}

	for i, child := range node.Children {
		isChildLast := (i == len(node.Children)-1)
		sb.WriteString(m.renderTree(child, childPrefix, isChildLast))
	}

	return sb.String()
}

// isDirectory 判断路径是否为目录，优先检查完整路径，失败时根据相对路径特征推断
func isDirectory(relPath string, fullPath string) bool {
	// 先检查fullPath
	if info, err := os.Stat(fullPath); err == nil {
		return info.IsDir()
	}

	// 如果fullPath检查失败,通过路径特征判断
	// 如果relPath包含多层路径且没有文件扩展名,很可能是目录
	ext := filepath.Ext(relPath)
	if ext == "" && strings.Contains(relPath, string(filepath.Separator)) {
		return true
	}

	return false
}

// GetStats 获取统计信息
func (m *Manager) GetStats() *types.LoadStats {
	return m.collector.GetStats()
}

// initializePrimaryRepo 初始化主代码库
func (m *Manager) initializePrimaryRepo(ctx context.Context) error {
	// 如果没有配置primaryRepo或未启用，跳过
	if m.config.PrimaryRepo == nil || !m.config.PrimaryRepo.Enabled {
		fmt.Println("Primary repository not configured, using standalone mode")
		return nil
	}

	primaryRepo := m.config.PrimaryRepo
	fmt.Printf("Initializing primary repository, type: %s\n", primaryRepo.Type)

	switch primaryRepo.Type {
	case "local":
		// 本地模式：验证路径是否存在
		return m.validateLocalPrimaryRepo(primaryRepo)

	case "remote":
		// 远程模式：克隆代码库
		return m.clonePrimaryRepo(ctx, primaryRepo)

	default:
		return fmt.Errorf("unknown primary repository type: %s", primaryRepo.Type)
	}
}

// validateLocalPrimaryRepo 验证本地主代码库
func (m *Manager) validateLocalPrimaryRepo(primaryRepo *types.PrimaryRepoConfig) error {
	expandedPath, err := utils.ExpandPath(primaryRepo.Path)
	if err != nil {
		return fmt.Errorf("failed to expand primary repo path: %w", err)
	}

	if !utils.FileExists(expandedPath) {
		return fmt.Errorf("primary repository path does not exist: %s", expandedPath)
	}

	fmt.Printf("Local primary repository validated, path: %s\n", expandedPath)
	return nil
}

// clonePrimaryRepo 克隆远程主代码库
func (m *Manager) clonePrimaryRepo(ctx context.Context, primaryRepo *types.PrimaryRepoConfig) error {
	// 计算目标路径
	basePath, err := utils.ExpandPath(m.config.Storage.BasePath)
	if err != nil {
		return fmt.Errorf("failed to expand base path: %w", err)
	}

	destPath := filepath.Join(basePath, primaryRepo.Repo)

	// 检查是否已存在
	if utils.FileExists(destPath) {
		fmt.Printf("Primary repository already exists, skipping clone, path: %s\n", destPath)
		return nil
	}

	// 确保父目录存在
	parentDir := filepath.Dir(destPath)
	if err := utils.EnsureDir(parentDir); err != nil {
		return fmt.Errorf("failed to create parent directory: %w", err)
	}

	// 构建git clone命令
	args := []string{"clone"}

	// 添加分支参数
	if primaryRepo.Branch != "" {
		args = append(args, "--branch", primaryRepo.Branch)
	}

	// 添加深度参数
	depth := primaryRepo.Depth
	if depth == "" {
		depth = "1" // 默认浅克隆
	}
	if depth != "-1" {
		args = append(args, "--depth", depth)
	}

	// 构建仓库URL (使用SSH格式)
	repoURL := fmt.Sprintf("ssh://%s@icode.baidu.com:8235/%s", m.credential.Username, primaryRepo.Repo)
	args = append(args, repoURL, destPath)

	fmt.Printf("Cloning primary repository, args: %v\n", args)

	// 执行git clone
	cmd := exec.CommandContext(ctx, "git", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git clone failed: %w, output: %s", err, string(output))
	}

	fmt.Printf("Primary repository cloned successfully, path: %s\n", destPath)
	return nil
}

// generateKnowledgeSkill 生成knowledge skill
func (m *Manager) generateKnowledgeSkill() error {
	// 检查是否启用
	if m.config.KnowledgeSkill == nil || !m.config.KnowledgeSkill.Enable {
		return nil
	}

	fmt.Println("Generating knowledge skill...")

	generator := NewSkillGenerator(m.storage, m.config.KnowledgeSkill)
	return generator.Generate()
}
