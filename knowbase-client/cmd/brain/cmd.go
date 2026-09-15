// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package brain

import (
	"github.com/spf13/cobra"
	"icode.baidu.com/baidu/devops-ai/knowbase-client/pkg/types"
)

// BrainCmd brain 父命令
var BrainCmd = &cobra.Command{
	Use:   "brain",
	Short: "Brain knowledge accumulation operations (知识沉淀操作)",
	Long: `Commands for Brain PG knowledge graph operations, including:
  - put:          write/update a brain page
  - get:          read a brain page
  - delete:       delete a brain page
  - search:       keyword search
  - query:        hybrid semantic+keyword retrieval
  - list:         list pages by type/tag
  - graph-query:  traverse the knowledge graph
  - timeline:     read timeline entries
  - timeline-add: append a timeline entry
  - backlinks:    get reverse links

  - canonical-slug: compute canonical slug and check for existing duplicate

  - notify:       session file side-car knowledge extraction

Examples:
  knowbase brain put --slug people/alice < page.md
  knowbase brain get people/alice
  knowbase brain delete people/alice
  knowbase brain search "alice acme"
  knowbase brain query "who is alice" --limit 5
  knowbase brain graph-query companies/acme --type works_at --direction in`,
}

// brainTimeout is the command timeout in seconds, inherited by all subcommands.
var brainTimeout int

// brainKbUUID is the brain UUID passed via --kb-uuid flag, inherited by all subcommands.
var brainKbUUID string

func init() {
	BrainCmd.PersistentFlags().IntVar(&brainTimeout, "timeout", 30, "Command timeout in seconds (0 = no timeout)")
	BrainCmd.PersistentFlags().StringVar(&brainKbUUID, "kb-uuid", "", "Brain UUID (overrides env "+types.EnvKnowbaseBrainUUID+")")
	BrainCmd.AddCommand(
		putCmd,
		getCmd,
		deleteCmd,
		searchCmd,
		queryCmd,
		listCmd,
		graphQueryCmd,
		timelineCmd,
		timelineAddCmd,
		backlinksCmd,
		notifyCmd,
		canonicalSlugCmd,
	)
}
