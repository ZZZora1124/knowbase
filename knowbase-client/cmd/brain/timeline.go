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

var timelineCmd = &cobra.Command{
	Use:   "timeline <slug>",
	Short: "Read timeline entries of a brain page (读取时间线)",
	Long: `Read timeline entries for the specified brain page.
Supports filtering by date range and limiting results.

Examples:
  knowbase brain timeline people/alice
  knowbase brain timeline companies/acme --since 2026-01-01 --limit 10
  knowbase brain timeline people/alice --since 2026-01-01 --until 2026-06-01`,
	Args: cobra.ExactArgs(1),
	Run:  runTimeline,
}

var timelineSince string
var timelineUntil string
var timelineLimit int

func init() {
	timelineCmd.Flags().StringVar(&timelineSince, "since", "", "Start date filter (YYYY-MM-DD)")
	timelineCmd.Flags().StringVar(&timelineUntil, "until", "", "End date filter (YYYY-MM-DD)")
	timelineCmd.Flags().IntVar(&timelineLimit, "limit", 50, "Max entries to return")
}

func runTimeline(cmd *cobra.Command, args []string) {
	c, ctx, cancel := initBrainEnv()
	defer cancel()

	entries, err := c.GetTimeline(ctx, args[0], timelineSince, timelineUntil, timelineLimit)
	if err != nil {
		utils.PrintlnError("Failed to get timeline: %v", err)
		os.Exit(types.ExitRunSubCMDError)
	}

	if len(entries) == 0 {
		fmt.Println("No timeline entries.")
		return
	}
	for _, e := range entries {
		fmt.Printf("%s  %s\n", e.Date, e.Summary)
	}
}
