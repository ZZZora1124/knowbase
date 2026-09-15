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

// memberCmd 成员管理父命令
var memberCmd = &cobra.Command{
	Use:   "member",
	Short: "Manage knowledgebase members (成员管理)",
}

var memberAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a member to knowledgebase (添加成员)",
	Long: `Add a member to knowledgebase.

Examples:
  knowbase notebook member add --kb-uuid kb-123456 --member-type username --member-value lisi --role editor
  knowbase notebook member add --kb-uuid kb-123456 --member-type email_group --member-value dl@baidu.com --role viewer`,
	Run: runMemberAdd,
}

var memberListCmd = &cobra.Command{
	Use:   "list",
	Short: "List knowledgebase members (查询成员列表)",
	Long: `List members of a knowledgebase.

Examples:
  knowbase notebook member list --kb-uuid kb-123456
  knowbase notebook member list --kb-uuid kb-123456 --page 1 --page-size 50`,
	Run: runMemberList,
}

var memberUpdateRoleCmd = &cobra.Command{
	Use:   "update-role",
	Short: "Update member role (更新成员角色)",
	Long: `Update a member's role in the knowledgebase.

Examples:
  knowbase notebook member update-role --kb-uuid kb-123456 --member-uuid member-789 --role admin`,
	Run: runMemberUpdateRole,
}

func init() {
	memberCmd.AddCommand(memberAddCmd, memberListCmd, memberUpdateRoleCmd)

	memberAddCmd.Flags().String("kb-uuid", "", "Knowledgebase UUID (required)")
	memberAddCmd.Flags().String("member-type", "", "Member type: username/email_group (required)")
	memberAddCmd.Flags().String("member-value", "", "Member value (required)")
	memberAddCmd.Flags().String("role", "", "Role: admin/editor/viewer (required)")
	_ = memberAddCmd.MarkFlagRequired("kb-uuid")
	_ = memberAddCmd.MarkFlagRequired("member-type")
	_ = memberAddCmd.MarkFlagRequired("member-value")
	_ = memberAddCmd.MarkFlagRequired("role")

	memberListCmd.Flags().String("kb-uuid", "", "Knowledgebase UUID (required)")
	memberListCmd.Flags().Int("page", 1, "Page number")
	memberListCmd.Flags().Int("page-size", 20, "Page size (1-100)")
	_ = memberListCmd.MarkFlagRequired("kb-uuid")

	memberUpdateRoleCmd.Flags().String("kb-uuid", "", "Knowledgebase UUID (required)")
	memberUpdateRoleCmd.Flags().String("member-uuid", "", "Member UUID (required)")
	memberUpdateRoleCmd.Flags().String("role", "", "New role: admin/editor/viewer (required)")
	_ = memberUpdateRoleCmd.MarkFlagRequired("kb-uuid")
	_ = memberUpdateRoleCmd.MarkFlagRequired("member-uuid")
	_ = memberUpdateRoleCmd.MarkFlagRequired("role")
}

func runMemberAdd(cmd *cobra.Command, args []string) {
	svc, ctx := initNotebookEnv()

	kbUUID, _ := cmd.Flags().GetString("kb-uuid")
	memberType, _ := cmd.Flags().GetString("member-type")
	memberValue, _ := cmd.Flags().GetString("member-value")
	role, _ := cmd.Flags().GetString("role")

	result, err := svc.AddMember(ctx, kbUUID, &nb.AddMemberRequest{
		MemberType: memberType, MemberValue: memberValue, Role: role,
	})
	if err != nil {
		utils.PrintlnError("Failed to add member: %v", err)
		os.Exit(types.ExitRunSubCMDError)
	}

	outputResult(result, func() {
		fmt.Println("Member added successfully!")
		fmt.Printf("  UUID:        %s\n", result.UUID)
		fmt.Printf("  Type:        %s\n", result.MemberType)
		fmt.Printf("  Value:       %s\n", result.MemberValue)
		fmt.Printf("  Role:        %s\n", result.Role)
	})
}

func runMemberList(cmd *cobra.Command, args []string) {
	svc, ctx := initNotebookEnv()

	kbUUID, _ := cmd.Flags().GetString("kb-uuid")
	page, _ := cmd.Flags().GetInt("page")
	pageSize, _ := cmd.Flags().GetInt("page-size")

	result, err := svc.ListMembers(ctx, kbUUID, page, pageSize)
	if err != nil {
		utils.PrintlnError("Failed to list members: %v", err)
		os.Exit(types.ExitRunSubCMDError)
	}

	outputResult(result, func() {
		if result.Total == 0 {
			fmt.Println("No members found.")
			return
		}
		totalPages := (int(result.Total) + result.PageSize - 1) / result.PageSize
		fmt.Printf("Total: %d (Page %d/%d)\n\n", result.Total, result.Page, totalPages)
		w := newTabWriter()
		fmt.Fprintf(w, "UUID\tTYPE\tVALUE\tROLE\tCREATED\n")
		for _, m := range result.Items {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
				m.UUID, m.MemberType, m.MemberValue, m.Role, m.CreatedAt)
		}
		w.Flush()
	})
}

func runMemberUpdateRole(cmd *cobra.Command, args []string) {
	svc, ctx := initNotebookEnv()

	kbUUID, _ := cmd.Flags().GetString("kb-uuid")
	memberUUID, _ := cmd.Flags().GetString("member-uuid")
	role, _ := cmd.Flags().GetString("role")

	err := svc.UpdateMemberRole(ctx, kbUUID, memberUUID, &nb.UpdateMemberRoleRequest{Role: role})
	if err != nil {
		utils.PrintlnError("Failed to update member role: %v", err)
		os.Exit(types.ExitRunSubCMDError)
	}

	outputResult(map[string]string{"status": "success"}, func() {
		fmt.Printf("Member %s role updated to %s successfully!\n", memberUUID, role)
	})
}
