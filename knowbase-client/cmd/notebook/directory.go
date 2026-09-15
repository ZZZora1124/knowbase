// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package notebook

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	nb "icode.baidu.com/baidu/devops-ai/knowbase-client/internal/notebook"
	"icode.baidu.com/baidu/devops-ai/knowbase-client/internal/utils"
	"icode.baidu.com/baidu/devops-ai/knowbase-client/pkg/types"
)

// dirCmd 目录管理父命令
var dirCmd = &cobra.Command{
	Use:   "dir",
	Short: "Manage knowledgebase directories (目录管理)",
}

var dirCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a directory (创建目录)",
	Long: `Create a directory in a knowledgebase.

Examples:
  knowbase notebook dir create --kb-uuid kb-123456 --name "部署文档"
  knowbase notebook dir create --kb-uuid kb-123456 --name "子目录" --parent-uuid dir-001`,
	Run: runDirCreate,
}

var dirListCmd = &cobra.Command{
	Use:   "list",
	Short: "List directories (查询目录列表)",
	Long: `List directories of a knowledgebase.

Examples:
  knowbase notebook dir list --kb-uuid kb-123456`,
	Run: runDirList,
}

var dirGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get directory detail (获取目录详情)",
	Long: `Get directory detail.

Examples:
  knowbase notebook dir get --kb-uuid kb-123456 --dir-uuid dir-001`,
	Run: runDirGet,
}

var dirUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update a directory (更新目录)",
	Long: `Update a directory name.

Examples:
  knowbase notebook dir update --kb-uuid kb-123456 --dir-uuid dir-001 --name "新名称"`,
	Run: runDirUpdate,
}

var dirTreeCmd = &cobra.Command{
	Use:   "tree",
	Short: "Get directory tree (获取目录树)",
	Long: `Get hierarchical directory tree of a knowledgebase (max 5 levels).

Examples:
  knowbase notebook dir tree --kb-uuid kb-123456`,
	Run: runDirTree,
}

var dirMoveCmd = &cobra.Command{
	Use:   "move",
	Short: "Move a directory (移动目录)",
	Long: `Move a directory to a new parent or promote to root.

Examples:
  knowbase notebook dir move --kb-uuid kb-123456 --dir-uuid dir-002 --parent-uuid dir-003
  knowbase notebook dir move --kb-uuid kb-123456 --dir-uuid dir-002 --to-root`,
	Run: runDirMove,
}

func init() {
	dirCmd.AddCommand(dirCreateCmd, dirListCmd, dirGetCmd, dirUpdateCmd, dirTreeCmd, dirMoveCmd)

	dirCreateCmd.Flags().String("kb-uuid", "", "Knowledgebase UUID (required)")
	dirCreateCmd.Flags().String("name", "", "Directory name (required)")
	dirCreateCmd.Flags().String("parent-uuid", "", "Parent directory UUID (for creating sub-directory)")
	_ = dirCreateCmd.MarkFlagRequired("kb-uuid")
	_ = dirCreateCmd.MarkFlagRequired("name")

	dirListCmd.Flags().String("kb-uuid", "", "Knowledgebase UUID (required)")
	_ = dirListCmd.MarkFlagRequired("kb-uuid")

	dirGetCmd.Flags().String("kb-uuid", "", "Knowledgebase UUID (required)")
	dirGetCmd.Flags().String("dir-uuid", "", "Directory UUID (required)")
	_ = dirGetCmd.MarkFlagRequired("kb-uuid")
	_ = dirGetCmd.MarkFlagRequired("dir-uuid")

	dirUpdateCmd.Flags().String("kb-uuid", "", "Knowledgebase UUID (required)")
	dirUpdateCmd.Flags().String("dir-uuid", "", "Directory UUID (required)")
	dirUpdateCmd.Flags().String("name", "", "New directory name (required)")
	_ = dirUpdateCmd.MarkFlagRequired("kb-uuid")
	_ = dirUpdateCmd.MarkFlagRequired("dir-uuid")
	_ = dirUpdateCmd.MarkFlagRequired("name")


	dirTreeCmd.Flags().String("kb-uuid", "", "Knowledgebase UUID (required)")
	_ = dirTreeCmd.MarkFlagRequired("kb-uuid")

	dirMoveCmd.Flags().String("kb-uuid", "", "Knowledgebase UUID (required)")
	dirMoveCmd.Flags().String("dir-uuid", "", "Directory UUID to move (required)")
	dirMoveCmd.Flags().String("parent-uuid", "", "Target parent directory UUID")
	dirMoveCmd.Flags().Bool("to-root", false, "Move directory to root level")
	_ = dirMoveCmd.MarkFlagRequired("kb-uuid")
	_ = dirMoveCmd.MarkFlagRequired("dir-uuid")
}

func runDirCreate(cmd *cobra.Command, args []string) {
	svc, ctx := initNotebookEnv()

	kbUUID, _ := cmd.Flags().GetString("kb-uuid")
	name, _ := cmd.Flags().GetString("name")
	parentUUID, _ := cmd.Flags().GetString("parent-uuid")

	req := &nb.CreateDirectoryRequest{Name: name}
	if parentUUID != "" {
		req.ParentUUID = &parentUUID
	}

	result, err := svc.CreateDirectory(ctx, kbUUID, req)
	if err != nil {
		utils.PrintlnError("Failed to create directory: %v", err)
		os.Exit(types.ExitRunSubCMDError)
	}

	outputResult(result, func() {
		fmt.Println("Directory created successfully!")
		fmt.Printf("  UUID:        %s\n", result.UUID)
		fmt.Printf("  Name:        %s\n", result.Name)
		if result.ParentUUID != "" {
			fmt.Printf("  Parent UUID: %s\n", result.ParentUUID)
		}
	})
}

func runDirList(cmd *cobra.Command, args []string) {
	svc, ctx := initNotebookEnv()

	kbUUID, _ := cmd.Flags().GetString("kb-uuid")

	result, err := svc.ListDirectories(ctx, kbUUID)
	if err != nil {
		utils.PrintlnError("Failed to list directories: %v", err)
		os.Exit(types.ExitRunSubCMDError)
	}

	outputResult(result, func() {
		if result.Total == 0 {
			fmt.Println("No directories found.")
			return
		}
		fmt.Printf("Total: %d\n\n", result.Total)
		w := newTabWriter()
		fmt.Fprintf(w, "UUID\tNAME\tPARENT_UUID\tKNOWLEDGE_COUNT\tCREATED\n")
		for _, d := range result.Items {
			parentDisplay := "-"
			if d.ParentUUID != "" {
				parentDisplay = d.ParentUUID
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\n", d.UUID, d.Name, parentDisplay, d.KnowledgeCount, d.CreatedAt)
		}
		w.Flush()
	})
}

func runDirGet(cmd *cobra.Command, args []string) {
	svc, ctx := initNotebookEnv()

	kbUUID, _ := cmd.Flags().GetString("kb-uuid")
	dirUUID, _ := cmd.Flags().GetString("dir-uuid")

	result, err := svc.GetDirectory(ctx, kbUUID, dirUUID)
	if err != nil {
		utils.PrintlnError("Failed to get directory: %v", err)
		os.Exit(types.ExitRunSubCMDError)
	}

	outputResult(result, func() {
		fmt.Printf("  UUID:           %s\n", result.UUID)
		fmt.Printf("  Name:           %s\n", result.Name)
		if result.ParentUUID != "" {
			fmt.Printf("  Parent UUID:    %s\n", result.ParentUUID)
		}
		fmt.Printf("  Knowledge:      %d\n", result.KnowledgeCount)
		fmt.Printf("  Created:        %s\n", result.CreatedAt)
		fmt.Printf("  Updated:        %s\n", result.UpdatedAt)
	})
}

func runDirUpdate(cmd *cobra.Command, args []string) {
	svc, ctx := initNotebookEnv()

	kbUUID, _ := cmd.Flags().GetString("kb-uuid")
	dirUUID, _ := cmd.Flags().GetString("dir-uuid")
	name, _ := cmd.Flags().GetString("name")

	result, err := svc.UpdateDirectory(ctx, kbUUID, dirUUID, &nb.UpdateDirectoryRequest{Name: name})
	if err != nil {
		utils.PrintlnError("Failed to update directory: %v", err)
		os.Exit(types.ExitRunSubCMDError)
	}

	outputResult(result, func() {
		fmt.Println("Directory updated successfully!")
		fmt.Printf("  UUID:   %s\n", result.UUID)
		fmt.Printf("  Name:   %s\n", result.Name)
	})
}


func runDirTree(cmd *cobra.Command, args []string) {
	svc, ctx := initNotebookEnv()

	kbUUID, _ := cmd.Flags().GetString("kb-uuid")

	result, err := svc.GetDirectoryTree(ctx, kbUUID)
	if err != nil {
		utils.PrintlnError("Failed to get directory tree: %v", err)
		os.Exit(types.ExitRunSubCMDError)
	}

	outputResult(result, func() {
		if result.Total == 0 {
			fmt.Println("No directories found.")
			return
		}
		fmt.Printf("Total root directories: %d\n\n", result.Total)
		for i, node := range result.Items {
			printTreeNode(node, "", i == len(result.Items)-1, 0)
		}
	})
}

func printTreeNode(node nb.DirectoryTreeNode, prefix string, isLast bool, depth int) {
	connector := "├── "
	if isLast {
		connector = "└── "
	}
	if depth == 0 {
		fmt.Printf("%s [%s] (%d knowledge)\n", node.Name, node.UUID, node.KnowledgeCount)
	} else {
		fmt.Printf("%s%s%s [%s] (%d knowledge)\n", prefix, connector, node.Name, node.UUID, node.KnowledgeCount)
	}

	var childPrefix string
	if depth == 0 {
		// 根节点的子节点不需要前缀缩进，但需要纵向连线
		if isLast {
			childPrefix = "    "
		} else {
			childPrefix = "│   "
		}
	} else {
		if isLast {
			childPrefix = prefix + "    "
		} else {
			childPrefix = prefix + "│   "
		}
	}
	for i, child := range node.Children {
		printTreeNode(child, childPrefix, i == len(node.Children)-1, depth+1)
	}
}

func runDirMove(cmd *cobra.Command, args []string) {
	svc, ctx := initNotebookEnv()

	kbUUID, _ := cmd.Flags().GetString("kb-uuid")
	dirUUID, _ := cmd.Flags().GetString("dir-uuid")
	parentUUID, _ := cmd.Flags().GetString("parent-uuid")
	toRoot, _ := cmd.Flags().GetBool("to-root")

	if parentUUID == "" && !toRoot {
		utils.PrintlnError("Either --parent-uuid or --to-root must be specified")
		os.Exit(types.ExitRunSubCMDError)
	}

	req := &nb.MoveDirectoryRequest{}
	if toRoot {
		req.ParentUUID = nil
	} else {
		req.ParentUUID = &parentUUID
	}

	result, err := svc.MoveDirectory(ctx, kbUUID, dirUUID, req)
	if err != nil {
		utils.PrintlnError("Failed to move directory: %v", err)
		os.Exit(types.ExitRunSubCMDError)
	}

	outputResult(result, func() {
		fmt.Println("Directory moved successfully!")
		fmt.Printf("  UUID:        %s\n", result.UUID)
		fmt.Printf("  Name:        %s\n", result.Name)
		if result.ParentUUID != "" {
			fmt.Printf("  Parent UUID: %s\n", result.ParentUUID)
		} else {
			fmt.Printf("  Parent UUID: (root)\n")
		}
	})
}
