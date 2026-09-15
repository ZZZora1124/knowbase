// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package notebook

import (
	"github.com/spf13/cobra"
)

// outputFormat 输出格式
var outputFormat string

// NotebookCmd notebook 父命令
var NotebookCmd = &cobra.Command{
	Use:   "notebook",
	Short: "Manage knowledge notebooks (知识方舟管理)",
	Long: `Commands for managing knowledge notebooks, including:
  - Knowledgebase: create, list, get, update
  - Members: add, list, update role
  - Tags: create, list, update
  - Directories: create, list, get, update
  - Knowledge: create, list, get, update, move, tag operations

Examples:
  knowbase notebook create --name "DevOps最佳实践" --visibility private
  knowbase notebook list --keyword DevOps
  knowbase notebook knowledge list --kb-uuid kb-123456`,
}

func init() {
	NotebookCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", "table",
		"Output format: json or table")

	// 注册知识方舟管理子命令
	NotebookCmd.AddCommand(nbCreateCmd, nbListCmd, nbGetCmd, nbGetByNameCmd, nbUpdateCmd)

	// 注册成员管理子命令组
	NotebookCmd.AddCommand(memberCmd)

	// 注册标签管理子命令组
	NotebookCmd.AddCommand(tagCmd)

	// 注册目录管理子命令组
	NotebookCmd.AddCommand(dirCmd)

	// 注册知识管理子命令组
	NotebookCmd.AddCommand(knowledgeCmd)
}
