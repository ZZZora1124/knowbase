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

var timelineAddDetail string
var timelineAddSource string

var timelineAddCmd = &cobra.Command{
	Use:   "timeline-add <slug> <date> <summary>",
	Short: "Append a timeline entry to a brain page (追加时间线条目)",
	Long: `Append a timeline entry to the specified brain page via dedicated POST endpoint.

The date must be in YYYY-MM-DD format.

Examples:
  knowbase brain timeline-add people/alice 2026-05-01 "Meeting — Discussed Q2 roadmap"
  knowbase brain timeline-add companies/acme 2026-04-30 "Funding — Series A announced" --detail "Lead investor: XYZ"`,
	Args: cobra.ExactArgs(3),
	Run:  runTimelineAdd,
}

func init() {
	timelineAddCmd.Flags().StringVar(&timelineAddDetail, "detail", "", "Optional detail text")
	timelineAddCmd.Flags().StringVar(&timelineAddSource, "source", "", "Optional source reference")
}

func runTimelineAdd(cmd *cobra.Command, args []string) {
	slug := args[0]
	date := args[1]
	summary := args[2]

	c, ctx, cancel := initBrainEnv()
	defer cancel()

	resp, err := c.AddTimelineEntry(ctx, slug, date, summary, timelineAddDetail, timelineAddSource)
	if err != nil {
		utils.PrintlnError("Failed to add timeline entry: %v", err)
		os.Exit(types.ExitRunSubCMDError)
	}

	fmt.Printf("status: %s\n", resp.Status)
}
