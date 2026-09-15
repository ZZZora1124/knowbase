// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package notebook

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	nb "icode.baidu.com/baidu/devops-ai/knowbase-client/internal/notebook"
	"icode.baidu.com/baidu/devops-ai/knowbase-client/internal/utils"
	"icode.baidu.com/baidu/devops-ai/knowbase-client/pkg/types"
)

// knowledgeCmd 知识管理父命令
var knowledgeCmd = &cobra.Command{
	Use:   "knowledge",
	Short: "Manage knowledge items (知识管理)",
}

var knowledgeCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create knowledge (创建知识)",
	Long: `Create knowledge in a knowledgebase. Supports text content and file upload.

Examples:
  knowbase notebook knowledge create --kb-uuid kb-123456 --title "部署流程" --knowledge-type content --knowledge-source custom --content "内容..."
  knowbase notebook knowledge create --kb-uuid kb-123456 --title "架构图" --knowledge-type text_file --knowledge-source custom --file /path/to/file`,
	Run: runKnowledgeCreate,
}

var knowledgeListCmd = &cobra.Command{
	Use:   "list",
	Short: "List knowledge (查询知识列表)",
	Long: `List knowledge in a knowledgebase with optional filters.

Examples:
  knowbase notebook knowledge list --kb-uuid kb-123456
  knowbase notebook knowledge list --kb-uuid kb-123456 --keyword "部署" --page 1 --page-size 50`,
	Run: runKnowledgeList,
}

var knowledgeListByTagsCmd = &cobra.Command{
	Use:   "list-by-tags",
	Short: "List knowledge by tags (按标签查询知识)",
	Long: `List knowledge filtered by tags.

Examples:
  knowbase notebook knowledge list-by-tags --kb-uuid kb-123456 --tag-uuids tag-001,tag-002 --match-type any
  knowbase notebook knowledge list-by-tags --kb-uuid kb-123456 --tag-names CICD,Deploy --match-type all`,
	Run: runKnowledgeListByTags,
}

var knowledgeGetCmd = &cobra.Command{
	Use:   "get <uuid> <dirPath>",
	Short: "Get knowledge detail (获取知识详情,如果包含文件则下载到指定路径)",
	Long: `Get knowledge detail. If the knowledge type contains file, it will download to the specified path.

Examples:
  knowbase notebook knowledge get know-001 /path/to/save`,
	Args: cobra.ExactArgs(2),
	Run:  runKnowledgeGet,
}

var knowledgeUpdateCmd = &cobra.Command{
	Use:   "update <uuid>",
	Short: "Update knowledge (更新知识)",
	Long: `Update knowledge information.

Examples:
  knowbase notebook knowledge update know-001 --title "新标题" --content "新内容"
  knowbase notebook knowledge update know-001 --file /path/to/file.md
  knowbase notebook knowledge update know-001 --title "新标题" --file /path/to/file.md`,
	Args: cobra.ExactArgs(1),
	Run:  runKnowledgeUpdate,
}

var knowledgeMoveCmd = &cobra.Command{
	Use:   "move <uuid>",
	Short: "Move knowledge to directory (移动知识)",
	Long: `Move knowledge to a directory or root.

Examples:
  knowbase notebook knowledge move know-001 --directory-uuid dir-002
  knowbase notebook knowledge move know-001 --to-root`,
	Args: cobra.ExactArgs(1),
	Run:  runKnowledgeMove,
}

var knowledgeBindTagsCmd = &cobra.Command{
	Use:   "bind-tags <uuid>",
	Short: "Bind tags to knowledge (绑定标签)",
	Long: `Bind tags to knowledge.

Examples:
  knowbase notebook knowledge bind-tags know-001 --tag-uuids tag-001,tag-002`,
	Args: cobra.ExactArgs(1),
	Run:  runKnowledgeBindTags,
}

var knowledgeUnbindTagsCmd = &cobra.Command{
	Use:   "unbind-tags <uuid>",
	Short: "Unbind tags from knowledge (解绑标签)",
	Long: `Unbind tags from knowledge.

Examples:
  knowbase notebook knowledge unbind-tags know-001 --tag-uuids tag-001,tag-002`,
	Args: cobra.ExactArgs(1),
	Run:  runKnowledgeUnbindTags,
}

var knowledgeGetTagsCmd = &cobra.Command{
	Use:   "get-tags <uuid>",
	Short: "Get knowledge tags (获取知识标签)",
	Args:  cobra.ExactArgs(1),
	Run:   runKnowledgeGetTags,
}

func init() {
	knowledgeCmd.AddCommand(knowledgeCreateCmd, knowledgeListCmd, knowledgeListByTagsCmd,
		knowledgeGetCmd, knowledgeUpdateCmd, knowledgeMoveCmd,
		knowledgeBindTagsCmd, knowledgeUnbindTagsCmd, knowledgeGetTagsCmd)

	// create flags
	knowledgeCreateCmd.Flags().String("kb-uuid", "", "Knowledgebase UUID (required)")
	knowledgeCreateCmd.Flags().String("title", "", "Knowledge title (required)")
	knowledgeCreateCmd.Flags().String("knowledge-type", "", "Type: content/text_file/config (required)")
	knowledgeCreateCmd.Flags().String("knowledge-source", "", "Source: custom/icode_repo/... (required)")
	knowledgeCreateCmd.Flags().String("directory-uuid", "", "Directory UUID")
	knowledgeCreateCmd.Flags().String("content", "", "Text content")
	knowledgeCreateCmd.Flags().String("file", "", "File path for upload")
	_ = knowledgeCreateCmd.MarkFlagRequired("kb-uuid")
	_ = knowledgeCreateCmd.MarkFlagRequired("title")
	_ = knowledgeCreateCmd.MarkFlagRequired("knowledge-type")
	_ = knowledgeCreateCmd.MarkFlagRequired("knowledge-source")

	// list flags
	knowledgeListCmd.Flags().String("kb-uuid", "", "Knowledgebase UUID (required)")
	knowledgeListCmd.Flags().String("directory-uuid", "", "Directory UUID filter")
	knowledgeListCmd.Flags().String("knowledge-source", "", "Knowledge source filter")
	knowledgeListCmd.Flags().String("knowledge-type", "", "Knowledge type filter")
	knowledgeListCmd.Flags().String("keyword", "", "Search keyword")
	knowledgeListCmd.Flags().String("creator", "", "Creator username filter")
	knowledgeListCmd.Flags().Int("page", 1, "Page number")
	knowledgeListCmd.Flags().Int("page-size", 20, "Page size (1-100)")
	_ = knowledgeListCmd.MarkFlagRequired("kb-uuid")

	// list-by-tags flags
	knowledgeListByTagsCmd.Flags().String("kb-uuid", "", "Knowledgebase UUID (required)")
	knowledgeListByTagsCmd.Flags().String("tag-uuids", "", "Tag UUIDs (comma separated)")
	knowledgeListByTagsCmd.Flags().String("tag-names", "", "Tag names (comma separated)")
	knowledgeListByTagsCmd.Flags().String("directory-uuids", "", "Directory UUIDs (comma separated)")
	knowledgeListByTagsCmd.Flags().String("match-type", "any", "Match type: any/all")
	knowledgeListByTagsCmd.Flags().String("knowledge-source", "", "Knowledge source filter")
	knowledgeListByTagsCmd.Flags().String("knowledge-type", "", "Knowledge type filter")
	knowledgeListByTagsCmd.Flags().Int("page", 1, "Page number")
	knowledgeListByTagsCmd.Flags().Int("page-size", 20, "Page size (1-100)")
	_ = knowledgeListByTagsCmd.MarkFlagRequired("kb-uuid")

	// update flags
	knowledgeUpdateCmd.Flags().String("title", "", "New title")
	knowledgeUpdateCmd.Flags().String("content", "", "New content")
	knowledgeUpdateCmd.Flags().String("file", "", "File path for upload")

	// move flags
	knowledgeMoveCmd.Flags().String("directory-uuid", "", "Target directory UUID")
	knowledgeMoveCmd.Flags().Bool("to-root", false, "Move to root directory")

	// bind/unbind tags flags
	knowledgeBindTagsCmd.Flags().String("tag-uuids", "", "Tag UUIDs (comma separated, required)")
	_ = knowledgeBindTagsCmd.MarkFlagRequired("tag-uuids")

	knowledgeUnbindTagsCmd.Flags().String("tag-uuids", "", "Tag UUIDs (comma separated, required)")
	_ = knowledgeUnbindTagsCmd.MarkFlagRequired("tag-uuids")
}

func runKnowledgeCreate(cmd *cobra.Command, args []string) {
	svc, ctx := initNotebookEnv()

	kbUUID, _ := cmd.Flags().GetString("kb-uuid")
	title, _ := cmd.Flags().GetString("title")
	knowledgeType, _ := cmd.Flags().GetString("knowledge-type")
	knowledgeSource, _ := cmd.Flags().GetString("knowledge-source")
	directoryUUID, _ := cmd.Flags().GetString("directory-uuid")
	content, _ := cmd.Flags().GetString("content")
	filePath, _ := cmd.Flags().GetString("file")

	result, err := svc.CreateKnowledge(ctx, kbUUID, title, knowledgeType, knowledgeSource, directoryUUID, content, filePath)
	if err != nil {
		utils.PrintlnError("Failed to create knowledge: %v", err)
		os.Exit(types.ExitRunSubCMDError)
	}

	outputResult(result, func() {
		fmt.Println("Knowledge created successfully!")
		printKnowledgeDetail(result, "")
	})
}

func runKnowledgeList(cmd *cobra.Command, args []string) {
	svc, ctx := initNotebookEnv()

	kbUUID, _ := cmd.Flags().GetString("kb-uuid")
	directoryUUID, _ := cmd.Flags().GetString("directory-uuid")
	knowledgeSource, _ := cmd.Flags().GetString("knowledge-source")
	knowledgeType, _ := cmd.Flags().GetString("knowledge-type")
	keyword, _ := cmd.Flags().GetString("keyword")
	creator, _ := cmd.Flags().GetString("creator")
	page, _ := cmd.Flags().GetInt("page")
	pageSize, _ := cmd.Flags().GetInt("page-size")

	result, err := svc.ListKnowledge(ctx, kbUUID, directoryUUID, knowledgeSource, knowledgeType, keyword, creator,
		page, pageSize)
	if err != nil {
		utils.PrintlnError("Failed to list knowledge: %v", err)
		os.Exit(types.ExitRunSubCMDError)
	}

	outputResult(result, func() {
		if result.Total == 0 {
			fmt.Println("No knowledge found.")
			return
		}
		totalPages := (int(result.Total) + result.PageSize - 1) / result.PageSize
		fmt.Printf("Total: %d (Page %d/%d)\n\n", result.Total, result.Page, totalPages)
		w := newTabWriter()
		fmt.Fprintf(w, "UUID\tTITLE\tTYPE\tSOURCE\tCREATOR\tCREATED\n")
		for _, k := range result.List {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
				k.UUID, k.Title, k.KnowledgeType, k.KnowledgeSource, k.CreatorUsername, k.CreatedAt)
		}
		w.Flush()
	})
}

func runKnowledgeListByTags(cmd *cobra.Command, args []string) {
	svc, ctx := initNotebookEnv()

	kbUUID, _ := cmd.Flags().GetString("kb-uuid")
	tagUUIDsStr, _ := cmd.Flags().GetString("tag-uuids")
	tagNamesStr, _ := cmd.Flags().GetString("tag-names")
	dirUUIDsStr, _ := cmd.Flags().GetString("directory-uuids")
	matchType, _ := cmd.Flags().GetString("match-type")
	knowledgeSource, _ := cmd.Flags().GetString("knowledge-source")
	knowledgeType, _ := cmd.Flags().GetString("knowledge-type")
	page, _ := cmd.Flags().GetInt("page")
	pageSize, _ := cmd.Flags().GetInt("page-size")

	tagUUIDs := splitComma(tagUUIDsStr)
	tagNames := splitComma(tagNamesStr)
	dirUUIDs := splitComma(dirUUIDsStr)

	result, err := svc.ListKnowledgeByTags(ctx, kbUUID, tagUUIDs, tagNames, dirUUIDs,
		matchType, knowledgeSource, knowledgeType, page, pageSize)
	if err != nil {
		utils.PrintlnError("Failed to list knowledge by tags: %v", err)
		os.Exit(types.ExitRunSubCMDError)
	}

	outputResult(result, func() {
		if result.Total == 0 {
			fmt.Println("No knowledge found.")
			return
		}
		totalPages := (int(result.Total) + result.PageSize - 1) / result.PageSize
		fmt.Printf("Total: %d (Page %d/%d)\n\n", result.Total, result.Page, totalPages)
		w := newTabWriter()
		fmt.Fprintf(w, "UUID\tTITLE\tTYPE\tSOURCE\tCREATOR\tCREATED\n")
		for _, k := range result.List {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
				k.UUID, k.Title, k.KnowledgeType, k.KnowledgeSource, k.CreatorUsername, k.CreatedAt)
		}
		w.Flush()
	})
}

func runKnowledgeGet(cmd *cobra.Command, args []string) {
	svc, ctx := initNotebookEnv()

	result, err := svc.GetKnowledgeWithDownload(ctx, args[0], args[1])
	if err != nil {
		utils.PrintlnError("Failed to get knowledge: %v", err)
		os.Exit(types.ExitRunSubCMDError)
	}

	outputResult(result, func() { printKnowledgeDetail(result, args[1]) })
}

func runKnowledgeUpdate(cmd *cobra.Command, args []string) {
	svc, ctx := initNotebookEnv()

	title, _ := cmd.Flags().GetString("title")
	content, _ := cmd.Flags().GetString("content")
	filePath, _ := cmd.Flags().GetString("file")

	if content != "" && filePath != "" {
		utils.PrintlnError("--content and --file are mutually exclusive")
		os.Exit(types.ExitRunSubCMDError)
	}

	if filePath != "" {
		if _, err := os.Stat(filePath); err != nil {
			utils.PrintlnError("Cannot access file %s: %v", filePath, err)
			os.Exit(types.ExitRunSubCMDError)
		}
		if title == "" {
			title = filepath.Base(filePath)
		}
		result, err := svc.UpdateKnowledgeFile(ctx, args[0], title, filePath)
		if err != nil {
			utils.PrintlnError("Failed to update knowledge: %v", err)
			os.Exit(types.ExitRunSubCMDError)
		}
		outputResult(result, func() {
			fmt.Println("Knowledge updated successfully!")
			printKnowledgeDetail(result, "")
		})
		return
	}

	req := &nb.UpdateKnowledgeRequest{}
	if title != "" {
		req.Title = title
	}
	if content != "" {
		req.Content = content
	}

	result, err := svc.UpdateKnowledge(ctx, args[0], req)
	if err != nil {
		utils.PrintlnError("Failed to update knowledge: %v", err)
		os.Exit(types.ExitRunSubCMDError)
	}

	outputResult(result, func() {
		fmt.Println("Knowledge updated successfully!")
		printKnowledgeDetail(result, "")
	})
}

func runKnowledgeMove(cmd *cobra.Command, args []string) {
	svc, ctx := initNotebookEnv()

	toRoot, _ := cmd.Flags().GetBool("to-root")
	dirUUID, _ := cmd.Flags().GetString("directory-uuid")

	var dirPtr *string
	if toRoot {
		dirPtr = nil
	} else if dirUUID != "" {
		dirPtr = &dirUUID
	}

	err := svc.MoveKnowledge(ctx, args[0], dirPtr)
	if err != nil {
		utils.PrintlnError("Failed to move knowledge: %v", err)
		os.Exit(types.ExitRunSubCMDError)
	}

	outputResult(map[string]string{"status": "success"}, func() {
		if toRoot {
			fmt.Printf("Knowledge %s moved to root directory successfully!\n", args[0])
		} else {
			fmt.Printf("Knowledge %s moved to directory %s successfully!\n", args[0], dirUUID)
		}
	})
}

func runKnowledgeBindTags(cmd *cobra.Command, args []string) {
	svc, ctx := initNotebookEnv()

	tagUUIDsStr, _ := cmd.Flags().GetString("tag-uuids")
	tagUUIDs := splitComma(tagUUIDsStr)

	result, err := svc.BindTags(ctx, args[0], tagUUIDs)
	if err != nil {
		utils.PrintlnError("Failed to bind tags: %v", err)
		os.Exit(types.ExitRunSubCMDError)
	}

	outputResult(result, func() {
		fmt.Printf("Tags bound successfully! (%d tags)\n", len(result))
		for _, t := range result {
			fmt.Printf("  - %s (%s)\n", t.Name, t.UUID)
		}
	})
}

func runKnowledgeUnbindTags(cmd *cobra.Command, args []string) {
	svc, ctx := initNotebookEnv()

	tagUUIDsStr, _ := cmd.Flags().GetString("tag-uuids")
	tagUUIDs := splitComma(tagUUIDsStr)

	result, err := svc.UnbindTags(ctx, args[0], tagUUIDs)
	if err != nil {
		utils.PrintlnError("Failed to unbind tags: %v", err)
		os.Exit(types.ExitRunSubCMDError)
	}

	outputResult(result, func() {
		fmt.Printf("Tags unbound successfully! Remaining tags: %d\n", len(result))
		for _, t := range result {
			fmt.Printf("  - %s (%s)\n", t.Name, t.UUID)
		}
	})
}

func runKnowledgeGetTags(cmd *cobra.Command, args []string) {
	svc, ctx := initNotebookEnv()

	result, err := svc.GetKnowledgeTags(ctx, args[0])
	if err != nil {
		utils.PrintlnError("Failed to get knowledge tags: %v", err)
		os.Exit(types.ExitRunSubCMDError)
	}

	outputResult(result, func() {
		if len(result) == 0 {
			fmt.Println("No tags found for this knowledge.")
			return
		}
		fmt.Printf("Tags (%d):\n", len(result))
		w := newTabWriter()
		fmt.Fprintf(w, "UUID\tNAME\tDESCRIPTION\n")
		for _, t := range result {
			fmt.Fprintf(w, "%s\t%s\t%s\n", t.UUID, t.Name, t.Description)
		}
		w.Flush()
	})
}

func printKnowledgeDetail(k *nb.Knowledge, dirPath string) {
	fmt.Printf("  UUID:           %s\n", k.UUID)
	fmt.Printf("  Title:          %s\n", k.Title)
	fmt.Printf("  Type:           %s\n", k.KnowledgeType)
	fmt.Printf("  Source:         %s\n", k.KnowledgeSource)
	fmt.Printf("  Creator:        %s\n", k.CreatorUsername)
	fmt.Printf("  Directory:      %s\n", k.DirectoryUUID)
	if k.Content != "" {
		contentPreview := k.Content
		if len(contentPreview) > 200 {
			contentPreview = contentPreview[:200] + "..."
		}
		fmt.Printf("  Content:        %s\n", contentPreview)
	}
	if k.FilePath != "" {
		if dirPath != "" {
			fmt.Printf("  FilePath:       %s\n", dirPath+"/"+k.FilePath)
		} else {
			fmt.Printf("  FilePath:       %s\n", k.FilePath)
		}
		fmt.Printf("  FileSize:       %d\n", k.FileSize)
		fmt.Printf("  FileFormat:     %s\n", k.FileFormat)
	}
	if len(k.Tags) > 0 {
		tagNames := make([]string, 0, len(k.Tags))
		for _, t := range k.Tags {
			tagNames = append(tagNames, t.Name)
		}
		fmt.Printf("  Tags:           %s\n", strings.Join(tagNames, ", "))
	}
	fmt.Printf("  Created:        %s\n", k.CreatedAt)
	fmt.Printf("  Updated:        %s\n", k.UpdatedAt)
}

// splitComma 分割逗号分隔的字符串，过滤空值
func splitComma(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}
