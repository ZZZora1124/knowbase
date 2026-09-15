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

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Keyword search brain pages (关键词搜索)",
	Long: `Full-text keyword search across brain pages.
Used by signal-detector Phase 2 to check for existing entity pages.

Examples:
  knowbase brain search "alice"
  knowbase brain search "acme ai" --limit 5`,
	Args: cobra.ExactArgs(1),
	Run:  runSearch,
}

var searchLimit int

func init() {
	searchCmd.Flags().IntVar(&searchLimit, "limit", 20, "Max results to return")
}

func runSearch(cmd *cobra.Command, args []string) {
	c, ctx, cancel := initBrainEnv()
	defer cancel()

	results, err := c.Search(ctx, args[0], searchLimit)
	if err != nil {
		utils.PrintlnError("Failed to search: %v", err)
		os.Exit(types.ExitRunSubCMDError)
	}

	if len(results) == 0 {
		fmt.Println("No results.")
		return
	}
	printResultRows(results)
}
