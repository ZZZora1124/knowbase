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

var graphQueryCmd = &cobra.Command{
	Use:   "graph-query <slug>",
	Short: "Traverse the knowledge graph (图谱遍历)",
	Long: `Traverse the knowledge graph from a starting node.
Used by query Skill to answer relationship questions ("who works at Acme?").

Link types: attended, works_at, invested_in, founded, advises, mentions, source

Examples:
  knowbase brain graph-query companies/acme --type works_at --direction in
  knowbase brain graph-query people/alice --type attended --depth 2
  knowbase brain graph-query meetings/demo-day --type attended --direction out`,
	Args: cobra.ExactArgs(1),
	Run:  runGraphQuery,
}

var graphLinkType, graphDirection string
var graphDepth int

func init() {
	graphQueryCmd.Flags().StringVar(&graphLinkType, "type", "", "Edge type filter (works_at|attended|invested_in|founded|advises|mentions|source)")
	graphQueryCmd.Flags().IntVar(&graphDepth, "depth", 1, "Traversal depth (default 1, max 3)")
	graphQueryCmd.Flags().StringVar(&graphDirection, "direction", "both", "Edge direction: in|out|both")
}

func runGraphQuery(cmd *cobra.Command, args []string) {
	c, ctx, cancel := initBrainEnv()
	defer cancel()

	results, err := c.GraphQuery(ctx, args[0], graphLinkType, graphDepth, graphDirection)
	if err != nil {
		utils.PrintlnError("Failed to graph-query: %v", err)
		os.Exit(types.ExitRunSubCMDError)
	}

	if len(results) == 0 {
		fmt.Println("No edges found.")
		return
	}
	for _, e := range results {
		fmt.Printf("%s → %s  [%s, depth %d]\n", e.FromSlug, e.ToSlug, e.LinkType, e.Depth)
	}
}
