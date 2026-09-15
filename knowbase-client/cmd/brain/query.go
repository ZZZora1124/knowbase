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

var queryCmd = &cobra.Command{
	Use:   "query <question>",
	Short: "Hybrid semantic+keyword retrieval (混合检索召回)",
	Long: `Hybrid retrieval: pgvector HNSW + FTS + RRF fusion + multi-query expansion.
Used by brain-ops as the primary retrieval path before each agent response.

Examples:
  knowbase brain query "who is alice"
  knowbase brain query "acme ai founding story" --limit 5 --detail high
  knowbase brain query "recent events" --no-expand`,
	Args: cobra.ExactArgs(1),
	Run:  runQuery,
}

var queryLimit int
var queryDetail string
var queryExpand bool

func init() {
	queryCmd.Flags().IntVar(&queryLimit, "limit", 20, "Max results to return")
	queryCmd.Flags().StringVar(&queryDetail, "detail", "medium", "Detail level: low|medium|high")
	queryCmd.Flags().BoolVar(&queryExpand, "expand", true, "Enable multi-query expansion (Haiku rewrites 3 variants)")
}

func runQuery(cmd *cobra.Command, args []string) {
	c, ctx, cancel := initBrainEnv()
	defer cancel()

	results, err := c.Query(ctx, args[0], queryLimit, queryExpand, queryDetail)
	if err != nil {
		utils.PrintlnError("Failed to query: %v", err)
		os.Exit(types.ExitRunSubCMDError)
	}

	if len(results) == 0 {
		fmt.Println("No results.")
		return
	}
	printResultRows(results)
}
