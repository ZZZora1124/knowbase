// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package brain

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
	"icode.baidu.com/baidu/devops-ai/knowbase-client/internal/utils"
	"icode.baidu.com/baidu/devops-ai/knowbase-client/pkg/types"
)

var (
	putContent string
	putFile    string
)

func init() {
	putCmd.Flags().StringVar(&putContent, "content", "", "Page content (if not provided, reads from stdin)")
	putCmd.Flags().StringVar(&putFile, "file", "", "Read page content from file path (file is removed after successful write)")
}

var putCmd = &cobra.Command{
	Use:   "put <slug>",
	Short: "Write/update a brain page (写入/更新页面)",
	Long: `Write or update a brain page. The slug is a positional argument.
Content can be provided via --file, --content flag, or stdin (pipe).
When --file is used, the file is automatically removed after successful write.
Triggers server-side: chunking → embedding → autolink → timeline parsing.

Examples:
  knowbase brain put people/alice --file /tmp/page.md
  knowbase brain put people/alice --content 'short content here'
  cat page.md | knowbase brain put people/alice`,
	Args: cobra.ExactArgs(1),
	Run:  runPut,
}

func runPut(cmd *cobra.Command, args []string) {
	slug := args[0]

	content, err := resolveContent()
	if err != nil {
		utils.PrintlnError("%v", err)
		os.Exit(types.ExitRunSubCMDError)
	}

	if len(content) == 0 {
		utils.PrintlnError("No content provided. Use --file, --content flag or pipe to stdin.")
		os.Exit(types.ExitRunSubCMDError)
	}

	c, ctx, cancel := initBrainEnv()
	defer cancel()

	result, err := c.PutPage(ctx, slug, string(content))
	if err != nil {
		utils.PrintlnError("Failed to put page: %v", err)
		os.Exit(types.ExitRunSubCMDError)
	}

	// 把新写入的 slug 增量追加到 canonical 索引，确保后续 canonical-slug 查询
	// （尤其是同一会话内的下一次 sidecar）能立即看到，避免重复建页。
	// 失败会内部自动回退为整体失效，不影响 put 主流程。
	updateCanonicalIndex(slug)

	// Remove source file after successful write
	if putFile != "" {
		if err := os.Remove(putFile); err != nil {
			utils.PrintlnError("Warning: failed to remove file %s: %v", putFile, err)
		}
	}

	fmt.Printf("slug: %s\nstatus: %s\nchunks: %d\nlinks: +%d -%d\n",
		result.Slug, result.Status, result.Chunks,
		result.AutoLinks.Created, result.AutoLinks.Removed)
}

// resolveContent resolves page content from --file, --content, or stdin (in priority order).
func resolveContent() ([]byte, error) {
	switch {
	case putFile != "":
		content, err := os.ReadFile(putFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read file %s: %w", putFile, err)
		}
		return content, nil
	case putContent != "":
		return []byte(putContent), nil
	default:
		content, err := io.ReadAll(os.Stdin)
		if err != nil {
			return nil, fmt.Errorf("failed to read stdin: %w", err)
		}
		return content, nil
	}
}
