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

var backlinksCmd = &cobra.Command{
	Use:   "backlinks <slug>",
	Short: "Get reverse links to a brain page (获取反向链接)",
	Long: `Fetch all pages that link to the specified page.
Used by brain-ops Phase 1 to assess entity importance.

Examples:
  knowbase brain backlinks people/alice
  knowbase brain backlinks companies/acme`,
	Args: cobra.ExactArgs(1),
	Run:  runBacklinks,
}

func runBacklinks(cmd *cobra.Command, args []string) {
	c, ctx, cancel := initBrainEnv()
	defer cancel()

	results, err := c.GetBacklinks(ctx, args[0])
	if err != nil {
		utils.PrintlnError("Failed to get backlinks: %v", err)
		os.Exit(types.ExitRunSubCMDError)
	}

	if len(results) == 0 {
		fmt.Println("No backlinks.")
		return
	}
	for _, b := range results {
		fmt.Printf("%s → %s  (%s)\n", b.FromSlug, b.ToSlug, b.LinkType)
	}
}
