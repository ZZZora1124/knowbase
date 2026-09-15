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

var getCmd = &cobra.Command{
	Use:   "get <slug>",
	Short: "Read a brain page (读取页面全文)",
	Long: `Fetch the full content of a brain page by slug.
Outputs the page as GBrain standard Markdown (frontmatter + compiled_truth + timeline).

Examples:
  knowbase brain get people/alice
  knowbase brain get companies/acme`,
	Args: cobra.ExactArgs(1),
	Run:  runGet,
}

func runGet(cmd *cobra.Command, args []string) {
	c, ctx, cancel := initBrainEnv()
	defer cancel()

	result, err := c.GetPage(ctx, args[0])
	if err != nil {
		utils.PrintlnError("Failed to get page: %v", err)
		os.Exit(types.ExitRunSubCMDError)
	}

	// 输出完整 markdown 原文，与 gbrain get 保持一致
	fmt.Print(result.Content)
}
