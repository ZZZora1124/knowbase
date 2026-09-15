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

// tagCmd 标签管理父命令
var tagCmd = &cobra.Command{
	Use:   "tag",
	Short: "Manage knowledgebase tags (标签管理)",
}

var tagCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a tag (创建标签)",
	Long: `Create a tag in a knowledgebase.

Examples:
  knowbase notebook tag create --kb-uuid kb-123456 --name "CICD"
  knowbase notebook tag create --kb-uuid kb-123456 --name "CICD" --description "持续集成持续部署相关知识"`,
	Run: runTagCreate,
}

var tagListCmd = &cobra.Command{
	Use:   "list",
	Short: "List tags (查询标签列表)",
	Long: `List tags of a knowledgebase.

Examples:
  knowbase notebook tag list --kb-uuid kb-123456`,
	Run: runTagList,
}

var tagUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update a tag (更新标签)",
	Long: `Update a tag in a knowledgebase.

Examples:
  knowbase notebook tag update --kb-uuid kb-123456 --tag-uuid tag-001 --name "新名称"`,
	Run: runTagUpdate,
}

func init() {
	tagCmd.AddCommand(tagCreateCmd, tagListCmd, tagUpdateCmd)

	tagCreateCmd.Flags().String("kb-uuid", "", "Knowledgebase UUID (required)")
	tagCreateCmd.Flags().String("name", "", "Tag name (required)")
	tagCreateCmd.Flags().String("description", "", "Tag description")
	_ = tagCreateCmd.MarkFlagRequired("kb-uuid")
	_ = tagCreateCmd.MarkFlagRequired("name")

	tagListCmd.Flags().String("kb-uuid", "", "Knowledgebase UUID (required)")
	_ = tagListCmd.MarkFlagRequired("kb-uuid")

	tagUpdateCmd.Flags().String("kb-uuid", "", "Knowledgebase UUID (required)")
	tagUpdateCmd.Flags().String("tag-uuid", "", "Tag UUID (required)")
	tagUpdateCmd.Flags().String("name", "", "New tag name")
	tagUpdateCmd.Flags().String("description", "", "New tag description")
	_ = tagUpdateCmd.MarkFlagRequired("kb-uuid")
	_ = tagUpdateCmd.MarkFlagRequired("tag-uuid")
}

func runTagCreate(cmd *cobra.Command, args []string) {
	svc, ctx := initNotebookEnv()

	kbUUID, _ := cmd.Flags().GetString("kb-uuid")
	name, _ := cmd.Flags().GetString("name")
	description, _ := cmd.Flags().GetString("description")

	result, err := svc.CreateTag(ctx, kbUUID, &nb.CreateTagRequest{
		Name: name, Description: description,
	})
	if err != nil {
		utils.PrintlnError("Failed to create tag: %v", err)
		os.Exit(types.ExitRunSubCMDError)
	}

	outputResult(result, func() {
		fmt.Println("Tag created successfully!")
		fmt.Printf("  UUID:        %s\n", result.UUID)
		fmt.Printf("  Name:        %s\n", result.Name)
		fmt.Printf("  Description: %s\n", result.Description)
	})
}

func runTagList(cmd *cobra.Command, args []string) {
	svc, ctx := initNotebookEnv()

	kbUUID, _ := cmd.Flags().GetString("kb-uuid")

	result, err := svc.ListTags(ctx, kbUUID)
	if err != nil {
		utils.PrintlnError("Failed to list tags: %v", err)
		os.Exit(types.ExitRunSubCMDError)
	}

	outputResult(result, func() {
		if result.Total == 0 {
			fmt.Println("No tags found.")
			return
		}
		fmt.Printf("Total: %d\n\n", result.Total)
		w := newTabWriter()
		fmt.Fprintf(w, "UUID\tNAME\tDESCRIPTION\tKNOWLEDGE_COUNT\tCREATED_BY\n")
		for _, t := range result.Items {
			fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\n",
				t.UUID, t.Name, t.Description, t.KnowledgeCount, t.CreatedBy)
		}
		w.Flush()
	})
}

func runTagUpdate(cmd *cobra.Command, args []string) {
	svc, ctx := initNotebookEnv()

	kbUUID, _ := cmd.Flags().GetString("kb-uuid")
	tagUUID, _ := cmd.Flags().GetString("tag-uuid")
	req := &nb.UpdateTagRequest{}
	if v, _ := cmd.Flags().GetString("name"); v != "" {
		req.Name = v
	}
	if v, _ := cmd.Flags().GetString("description"); v != "" {
		req.Description = v
	}

	result, err := svc.UpdateTag(ctx, kbUUID, tagUUID, req)
	if err != nil {
		utils.PrintlnError("Failed to update tag: %v", err)
		os.Exit(types.ExitRunSubCMDError)
	}

	outputResult(result, func() {
		fmt.Println("Tag updated successfully!")
		fmt.Printf("  UUID:        %s\n", result.UUID)
		fmt.Printf("  Name:        %s\n", result.Name)
		fmt.Printf("  Description: %s\n", result.Description)
	})
}

