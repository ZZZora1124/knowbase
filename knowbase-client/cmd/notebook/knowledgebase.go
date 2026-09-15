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

// --- 创建知识方舟 ---
var nbCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new knowledgebase (创建知识方舟)",
	Long: `Create a new knowledgebase with specified name, description and visibility.

Examples:
  knowbase notebook create --name "DevOps最佳实践" --visibility private
  knowbase notebook create --name "公共知识库" --description "团队共享" --visibility public`,
	Run: runNbCreate,
}

// --- 列表知识方舟 ---
var nbListCmd = &cobra.Command{
	Use:   "list",
	Short: "List knowledgebases (查询知识方舟列表)",
	Long: `List knowledgebases with optional filters.

Examples:
  knowbase notebook list --visibility private
  knowbase notebook list --keyword DevOps --visibility private 
  knowbase notebook list --owner zhangsan --page 1 --page-size 50 --output json`,
	Run: runNbList,
}

// --- 按名称获取知识方舟 ---
var nbGetByNameCmd = &cobra.Command{
	Use:   "get-by-name <name>",
	Short: "Get knowledgebase by name (按名称查询知识方舟)",
	Args:  cobra.ExactArgs(1),
	Run:   runNbGetByName,
}

// --- 按UUID获取知识方舟 ---
var nbGetCmd = &cobra.Command{
	Use:   "get <uuid>",
	Short: "Get knowledgebase by UUID (按UUID查询知识方舟)",
	Args:  cobra.ExactArgs(1),
	Run:   runNbGet,
}

// --- 更新知识方舟 ---
var nbUpdateCmd = &cobra.Command{
	Use:   "update <uuid>",
	Short: "Update knowledgebase (更新知识方舟)",
	Long: `Update knowledgebase basic information.

Examples:
  knowbase notebook update kb-123456 --name "新名称"
  knowbase notebook update kb-123456 --description "新描述" --visibility public`,
	Args: cobra.ExactArgs(1),
	Run:  runNbUpdate,
}

func init() {
	nbCreateCmd.Flags().String("name", "", "Knowledgebase name (required)")
	nbCreateCmd.Flags().String("description", "", "Knowledgebase description")
	nbCreateCmd.Flags().String("visibility", "", "Visibility: private/public/personal (required)")
	_ = nbCreateCmd.MarkFlagRequired("name")
	_ = nbCreateCmd.MarkFlagRequired("visibility")

	nbListCmd.Flags().String("keyword", "", "Search keyword")
	nbListCmd.Flags().String("visibility", "", "Filter by visibility")
	nbListCmd.Flags().String("owner", "", "Filter by owner")
	nbListCmd.Flags().Int("page", 1, "Page number")
	nbListCmd.Flags().Int("page-size", 20, "Page size (1-100)")

	nbUpdateCmd.Flags().String("name", "", "New name")
	nbUpdateCmd.Flags().String("description", "", "New description")
	nbUpdateCmd.Flags().String("visibility", "", "New visibility")
}

func runNbCreate(cmd *cobra.Command, args []string) {
	svc, ctx := initNotebookEnv()

	name, _ := cmd.Flags().GetString("name")
	description, _ := cmd.Flags().GetString("description")
	visibility, _ := cmd.Flags().GetString("visibility")

	result, err := svc.CreateKnowledgebase(ctx, &nb.CreateKnowledgebaseRequest{
		Name: name, Description: description, Visibility: visibility,
	})
	if err != nil {
		utils.PrintlnError("Failed to create knowledgebase: %v", err)
		os.Exit(types.ExitRunSubCMDError)
	}

	outputResult(result, func() {
		fmt.Println("Knowledgebase created successfully!")
		printKnowledgebaseDetail(result)
	})
}

func runNbList(cmd *cobra.Command, args []string) {
	svc, ctx := initNotebookEnv()

	keyword, _ := cmd.Flags().GetString("keyword")
	visibility, _ := cmd.Flags().GetString("visibility")
	owner, _ := cmd.Flags().GetString("owner")
	page, _ := cmd.Flags().GetInt("page")
	pageSize, _ := cmd.Flags().GetInt("page-size")

	result, err := svc.ListKnowledgebases(ctx, keyword, visibility, owner, page, pageSize)
	if err != nil {
		utils.PrintlnError("Failed to list knowledgebases: %v", err)
		os.Exit(types.ExitRunSubCMDError)
	}

	outputResult(result, func() {
		if result.Total == 0 {
			fmt.Println("No knowledgebases found.")
			return
		}
		totalPages := (int(result.Total) + result.PageSize - 1) / result.PageSize
		fmt.Printf("Total: %d (Page %d/%d)\n\n", result.Total, result.Page, totalPages)
		w := newTabWriter()
		fmt.Fprintf(w, "UUID\tNAME\tVISIBILITY\tOWNER\tKNOWLEDGE\tMEMBERS\tROLE\n")
		for _, kb := range result.List {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%d\t%d\t%s\n",
				kb.UUID, kb.Name, kb.Visibility, kb.Owner, kb.KnowledgeCount, kb.MemberCount, kb.Role)
		}
		w.Flush()
	})
}

func runNbGetByName(cmd *cobra.Command, args []string) {
	svc, ctx := initNotebookEnv()

	result, err := svc.GetKnowledgebaseByName(ctx, args[0])
	if err != nil {
		utils.PrintlnError("Failed to get knowledgebase: %v", err)
		os.Exit(types.ExitRunSubCMDError)
	}

	outputResult(result, func() { printKnowledgebaseDetail(result) })
}

func runNbGet(cmd *cobra.Command, args []string) {
	svc, ctx := initNotebookEnv()

	result, err := svc.GetKnowledgebaseByUUID(ctx, args[0])
	if err != nil {
		utils.PrintlnError("Failed to get knowledgebase: %v", err)
		os.Exit(types.ExitRunSubCMDError)
	}

	outputResult(result, func() { printKnowledgebaseDetail(result) })
}

func runNbUpdate(cmd *cobra.Command, args []string) {
	svc, ctx := initNotebookEnv()

	req := &nb.UpdateKnowledgebaseRequest{}
	if v, _ := cmd.Flags().GetString("name"); v != "" {
		req.Name = v
	}
	if v, _ := cmd.Flags().GetString("description"); v != "" {
		req.Description = v
	}
	if v, _ := cmd.Flags().GetString("visibility"); v != "" {
		req.Visibility = v
	}

	result, err := svc.UpdateKnowledgebase(ctx, args[0], req)
	if err != nil {
		utils.PrintlnError("Failed to update knowledgebase: %v", err)
		os.Exit(types.ExitRunSubCMDError)
	}

	outputResult(result, func() {
		fmt.Println("Knowledgebase updated successfully!")
		printKnowledgebaseDetail(result)
	})
}

func printKnowledgebaseDetail(kb *nb.Knowledgebase) {
	fmt.Printf("  UUID:           %s\n", kb.UUID)
	fmt.Printf("  Name:           %s\n", kb.Name)
	fmt.Printf("  Description:    %s\n", kb.Description)
	fmt.Printf("  Visibility:     %s\n", kb.Visibility)
	fmt.Printf("  Owner:          %s\n", kb.Owner)
	fmt.Printf("  Status:         %d\n", kb.Status)
	fmt.Printf("  Members:        %d\n", kb.MemberCount)
	fmt.Printf("  Tags:           %d\n", kb.TagCount)
	fmt.Printf("  Knowledge:      %d\n", kb.KnowledgeCount)
	fmt.Printf("  Created:        %s\n", kb.CreatedAt)
	fmt.Printf("  Updated:        %s\n", kb.UpdatedAt)
}
