// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package brain

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	br "icode.baidu.com/baidu/devops-ai/knowbase-client/internal/brain"
	"icode.baidu.com/baidu/devops-ai/knowbase-client/internal/utils"
	"icode.baidu.com/baidu/devops-ai/knowbase-client/pkg/types"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List brain pages (枚举页面)",
	Long: `List brain pages with optional type/tag filters.

Pagination 行为：
  - 不传 --limit（或 --limit=0）：客户端自动按 offset 翻页拉取全部页面，--offset 被忽略
  - --limit > 0：单页模式，可配合 --offset 手动翻页（服务端单页上限 100）

Examples:
  knowbase brain list                              # 全量（客户端自动翻页）
  knowbase brain list --type person                # 按类型过滤的全量
  knowbase brain list --limit 50                   # 仅取前 50 条
  knowbase brain list --limit 100 --offset 100     # 单页模式：第 2 页`,
	Run: runList,
}

var listType, listTag string
var listLimit, listOffset int

func init() {
	listCmd.Flags().StringVar(&listType, "type", "", "Filter by frontmatter type (person|company|project|concept|signal|original)")
	listCmd.Flags().StringVar(&listTag, "tag", "", "Filter by tag")
	listCmd.Flags().IntVar(&listLimit, "limit", 0, "Per-page max（不传或 0 表示客户端自动翻页拉全量；>0 时为单页模式，服务端上限 100）")
	listCmd.Flags().IntVar(&listOffset, "offset", 0, "分页偏移量（仅在 --limit > 0 时生效）")
}

func runList(cmd *cobra.Command, args []string) {
	c, ctx, cancel := initBrainEnv()
	defer cancel()

	results, err := fetchListResults(ctx, c, listType, listTag, listLimit, listOffset)
	if err != nil {
		utils.PrintlnError("Failed to list pages: %v", err)
		os.Exit(types.ExitRunSubCMDError)
	}

	if len(results) == 0 {
		fmt.Println("No pages found.")
		return
	}
	for _, p := range results {
		date := "?"
		if len(p.UpdatedAt) >= 10 {
			date = p.UpdatedAt[:10]
		}
		fmt.Printf("%s\t%s\t%s\t%s\n", p.Slug, p.Type, date, p.Title)
	}
}

// fetchListResults 根据 limit 决定走单页或全量分页：
//   - limit <= 0：客户端自动翻页拉全量，offset 被忽略
//   - limit  > 0：单页模式，offset 透传给服务端
//
// 抽离出来便于单测覆盖路由分支。
func fetchListResults(ctx context.Context, c *br.BrainClient, pageType, tag string, limit, offset int) ([]br.ListPageItem, error) {
	if limit <= 0 {
		return c.ListAllPages(ctx, pageType, tag)
	}
	return c.ListPages(ctx, pageType, tag, limit, offset)
}
