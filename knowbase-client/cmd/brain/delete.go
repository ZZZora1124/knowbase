// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package brain

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"icode.baidu.com/baidu/devops-ai/knowbase-client/internal/utils"
	"icode.baidu.com/baidu/devops-ai/knowbase-client/pkg/types"
)

var deleteCmd = &cobra.Command{
	Use:   "delete <slug>",
	Short: "Delete a brain page (删除页面)",
	Long: `Delete a brain page by its slug. This removes the page and all associated
chunks, embeddings, and autolinks.

Examples:
  knowbase brain delete people/alice
  knowbase brain delete concepts/old-concept`,
	Args: cobra.ExactArgs(1),
	Run:  runDelete,
}

func runDelete(cmd *cobra.Command, args []string) {
	slug := args[0]

	c, ctx, cancel := initBrainEnv()
	defer cancel()

	result, err := c.DeletePage(ctx, slug)
	if err != nil {
		utils.PrintlnError("Failed to delete page: %v", err)
		os.Exit(types.ExitRunSubCMDError)
	}

	// 删除是低频操作，简单做整体失效；下一次 canonical-slug 自然 rebuild。
	invalidateCanonicalIndex()

	fmt.Printf("slug: %s\nstatus: %s\n", result.Slug, result.Status)
}
