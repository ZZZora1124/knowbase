// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package brain

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"github.com/spf13/cobra"
	br "icode.baidu.com/baidu/devops-ai/knowbase-client/internal/brain"
	"icode.baidu.com/baidu/devops-ai/knowbase-client/internal/utils"
	"icode.baidu.com/baidu/devops-ai/knowbase-client/pkg/types"
)

var (
	canonicalSlugType     string
	canonicalSlugNoCheck  bool
	canonicalSlugForceNew bool
	canonicalSlugTTL      int
)

// canonicalSlugCacheTTL 索引缓存的默认 TTL（秒）
const canonicalSlugCacheTTL = 60

// canonicalIndexFile 缓存路径：~/.knowbase/canonical_index.json
const canonicalIndexFile = ".knowbase/canonical_index.json"

// canonicalIndex 内存中的索引结构
type canonicalIndex struct {
	BuiltAt int64             `json:"built_at"`          // unix 秒
	UUID    string            `json:"uuid"`              // brain UUID（缓存隔离用）
	Pages   map[string]string `json:"pages"`             // canonical_key -> slug
	ByType  map[string]string `json:"by_type,omitempty"` // "<type>|<canonical_key>" -> slug
}

// canonicalSlugResult 命令输出 JSON
type canonicalSlugResult struct {
	CanonicalKey string `json:"canonical_key"`
	ProposedSlug string `json:"proposed_slug"`
	ExistingSlug string `json:"existing_slug,omitempty"`
	Exists       bool   `json:"exists"`
	IndexAge     int64  `json:"index_age_seconds"`
}

var canonicalSlugCmd = &cobra.Command{
	Use:   "canonical-slug <name>",
	Short: "Compute canonical slug + check existing page (计算归一化 slug 并检测重复)",
	Long: `Compute a canonical key for the given name (lowercase, no separators,
Chinese converted to pinyin) and optionally check if an existing brain page
matches the same canonical key.

Output is JSON for machine parsing by the signal-detector sidecar:
  {
    "canonical_key": "zhangsan",
    "proposed_slug": "people/zhangsan",
    "existing_slug": "people/zhang-san",
    "exists": true,
    "index_age_seconds": 12
  }

Behavior:
  - "zhang-san" / "zhang_san" / "Zhang San" all map to canonical_key "zhangsan"
  - "张三" also maps to canonical_key "zhangsan"
  - When --type is given, lookup is restricted to that type prefix
  - The index is cached at ~/.knowbase/canonical_index.json with a short TTL
  - --no-check skips brain lookup (pure normalization, fastest)
  - --force-new returns proposed_slug as-is with exists=false (caller wants new page)

Examples:
  knowbase brain canonical-slug "agent-brain" --type project
  knowbase brain canonical-slug "张三" --type person
  knowbase brain canonical-slug "Quick Note" --no-check`,
	Args: cobra.ExactArgs(1),
	Run:  runCanonicalSlug,
}

func init() {
	canonicalSlugCmd.Flags().StringVar(&canonicalSlugType, "type",
		"", "Restrict lookup to a page type (person | concept | project | company | original | idea)")
	canonicalSlugCmd.Flags().BoolVar(&canonicalSlugNoCheck, "no-check",
		false, "Skip brain lookup; only compute canonical key")
	canonicalSlugCmd.Flags().BoolVar(&canonicalSlugForceNew, "force-new",
		false, "Skip dedup, always return exists=false and a fresh proposed slug")
	canonicalSlugCmd.Flags().IntVar(&canonicalSlugTTL, "cache-ttl",
		canonicalSlugCacheTTL, "Canonical index cache TTL in seconds")
}

func runCanonicalSlug(cmd *cobra.Command, args []string) {
	name := args[0]
	key := br.CanonicalKey(name)
	if key == "" {
		utils.PrintlnError("canonical key is empty for input %q", name)
		os.Exit(types.ExitRunSubCMDError)
	}

	slug := readableSlug(name)
	proposed := slug
	if canonicalSlugType != "" {
		proposed = typeToPrefix(canonicalSlugType) + "/" + slug
	}

	result := canonicalSlugResult{
		CanonicalKey: key,
		ProposedSlug: proposed,
		Exists:       false,
	}

	if canonicalSlugNoCheck || canonicalSlugForceNew {
		emitCanonicalResult(result)
		return
	}

	// 加载或重建索引，做命中检查
	idx, age, err := loadOrBuildCanonicalIndex(canonicalSlugTTL)
	if err != nil {
		utils.PrintlnError("Failed to build canonical index: %v", err)
		os.Exit(types.ExitRunSubCMDError)
	}
	result.IndexAge = age

	existing := lookupCanonicalIndex(idx, key, canonicalSlugType)
	if existing != "" {
		result.ExistingSlug = existing
		result.Exists = true
	}

	emitCanonicalResult(result)
}

// readableSlug 把原始名称转为可读 slug：保留中文、字母、数字，其余字符替换为连字符，去除首尾连字符。
func readableSlug(name string) string {
	var sb strings.Builder
	prevDash := false
	for _, r := range strings.TrimSpace(name) {
		switch {
		case unicode.Is(unicode.Han, r) || unicode.IsLetter(r) || unicode.IsDigit(r):
			if prevDash && sb.Len() > 0 {
				sb.WriteRune('-')
			}
			sb.WriteRune(unicode.ToLower(r))
			prevDash = false
		default:
			prevDash = true
		}
	}
	return sb.String()
}

// typeToPrefix 把单数 type（person）映射为 slug 路径前缀（people）
func typeToPrefix(t string) string {
	switch strings.ToLower(t) {
	case "person":
		return "people"
	case "company":
		return "companies"
	case "project":
		return "projects"
	case "concept":
		return "concepts"
	case "idea":
		return "ideas"
	case "original":
		return "originals"
	case "signal":
		return "signals"
	default:
		return strings.ToLower(t)
	}
}

func emitCanonicalResult(r canonicalSlugResult) {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		utils.PrintlnError("marshal result: %v", err)
		os.Exit(types.ExitRunSubCMDError)
	}
	fmt.Println(string(data))
}

// loadOrBuildCanonicalIndex 加载索引；过期或 UUID 不匹配时重建。
// 返回 (index, ageSeconds, error)。
func loadOrBuildCanonicalIndex(ttlSec int) (*canonicalIndex, int64, error) {
	uuid := brainKbUUID
	if uuid == "" {
		uuid = os.Getenv(types.EnvKnowbaseBrainUUID)
	}
	idxPath, err := canonicalIndexPath()
	if err != nil {
		return nil, 0, err
	}

	if data, readErr := os.ReadFile(idxPath); readErr == nil {
		var cached canonicalIndex
		if jsonErr := json.Unmarshal(data, &cached); jsonErr == nil {
			age := time.Now().Unix() - cached.BuiltAt
			if cached.UUID == uuid && age < int64(ttlSec) {
				return &cached, age, nil
			}
		}
	}

	idx, err := buildCanonicalIndex(uuid)
	if err != nil {
		return nil, 0, err
	}

	// 落盘缓存，失败仅打印 warning
	if data, mErr := json.Marshal(idx); mErr == nil {
		_ = os.MkdirAll(filepath.Dir(idxPath), 0o755)
		_ = os.WriteFile(idxPath, data, 0o600)
	}
	return idx, 0, nil
}

// buildCanonicalIndex 拉取 brain list，按 canonical_key 建索引。
// 同一 canonical_key 命中多个 slug 时，保留首个（按 list 返回顺序）。
func buildCanonicalIndex(uuid string) (*canonicalIndex, error) {
	c, ctx, cancel := initBrainEnv()
	defer cancel()

	pages, err := c.ListAllPages(ctx, "", "")
	if err != nil {
		return nil, fmt.Errorf("list pages: %w", err)
	}

	idx := &canonicalIndex{
		BuiltAt: time.Now().Unix(),
		UUID:    uuid,
		Pages:   make(map[string]string, len(pages)),
		ByType:  make(map[string]string, len(pages)),
	}
	for _, p := range pages {
		key := br.CanonicalKeyFromSlug(p.Slug)
		if key == "" {
			continue
		}
		if _, exists := idx.Pages[key]; !exists {
			idx.Pages[key] = p.Slug
		}
		if p.Type != "" {
			typeKey := strings.ToLower(p.Type) + "|" + key
			if _, exists := idx.ByType[typeKey]; !exists {
				idx.ByType[typeKey] = p.Slug
			}
		}
	}
	return idx, nil
}

// lookupCanonicalIndex 查找命中 slug。
//
// 查询顺序：
//  1. typeFilter 非空 → 先查 by_type 索引（精确按类型匹配）
//  2. by_type 未命中 → 回退到全局 pages 索引；若命中页面的 slug 前缀语义上与 typeFilter
//     一致（如 typeFilter=person，slug=people/xxx），仍视为命中
//  3. 否则返回空字符串
//
// 设计动机：by_type 索引仅由完整 buildCanonicalIndex（brain list 回填）维护；增量
// updateCanonicalIndex 只更新 pages，避免维护 prefix→type 反向映射。回退到 pages
// 时通过 slug 前缀做轻量校验，保证 type 限定语义仍然成立。
func lookupCanonicalIndex(idx *canonicalIndex, key, typeFilter string) string {
	if idx == nil {
		return ""
	}
	if typeFilter == "" {
		return idx.Pages[key]
	}
	if slug, ok := idx.ByType[strings.ToLower(typeFilter)+"|"+key]; ok {
		return slug
	}
	// 回退：全局 pages 命中后校验 slug 前缀与 typeFilter 一致，避免 type 不一致误命中
	if slug, ok := idx.Pages[key]; ok {
		expectedPrefix := typeToPrefix(typeFilter)
		if i := strings.Index(slug, "/"); i > 0 && slug[:i] == expectedPrefix {
			return slug
		}
	}
	return ""
}

// canonicalIndexPath 返回缓存索引文件的绝对路径
func canonicalIndexPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, canonicalIndexFile), nil
}

// invalidateCanonicalIndex 删除本地索引缓存。
// 用于以下场景：
//  1. brain delete 成功后（被删页面如仍残留索引会误导后续查重）；
//  2. Phase 4 脚本主动清缓存以强制下一个 sidecar 重拉 brain list；
//  3. 增量更新 updateCanonicalIndex 写盘失败时的兜底。
//
// 文件不存在或删除失败均静默忽略，不影响主流程。
func invalidateCanonicalIndex() {
	idxPath, err := canonicalIndexPath()
	if err != nil {
		return
	}
	if err := os.Remove(idxPath); err != nil && !os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "[canonical] warning: failed to invalidate index %s: %v\n", idxPath, err)
	}
}

// updateCanonicalIndex 在 brain put 成功后被调用，把新写入的 (canonical_key, slug)
// 追加到本地索引并刷新 built_at（续命 TTL）。
//
// 设计意图（详见 docs/superpowers/specs/2026-05-14-brain-dedup-design.md）：
// 让连续触发的 sidecar（典型 hook 模式下用户快速对话）能立刻看到前序 sidecar 写入，
// 避免对同一实体反复建页；同时避免每次 put 都失效缓存进而触发 brain list。
//
// 失败回退：任一 IO 步骤失败 → invalidateCanonicalIndex 删整个缓存，保证下一次
// canonical-slug 会重新从 brain list 拉取，状态可自愈。
//
// 注意：仅更新 pages（全局索引）；by_type 索引不更新——lookupCanonicalIndex 在
// type 限定未命中时会回退到全局 pages（见 lookupCanonicalIndex 文档），避免额外
// 维护 prefix→type 反向映射的复杂度。
func updateCanonicalIndex(slug string) {
	if slug == "" {
		return
	}
	key := br.CanonicalKeyFromSlug(slug)
	if key == "" {
		return
	}

	idxPath, err := canonicalIndexPath()
	if err != nil {
		invalidateCanonicalIndex()
		return
	}
	uuid := brainKbUUID
	if uuid == "" {
		uuid = os.Getenv(types.EnvKnowbaseBrainUUID)
	}

	var idx canonicalIndex
	if data, readErr := os.ReadFile(idxPath); readErr == nil {
		_ = json.Unmarshal(data, &idx)
	}
	// 缓存不存在 / 解析失败 / 不同 brain → 重置（仅保留新写入项，等下一次 canonical-slug
	// 自然 rebuild 全量；本次 append 不阻塞 put 成功语义）
	if idx.UUID != uuid {
		idx = canonicalIndex{UUID: uuid}
	}
	if idx.Pages == nil {
		idx.Pages = make(map[string]string)
	}

	idx.Pages[key] = slug
	idx.BuiltAt = time.Now().Unix()

	data, mErr := json.Marshal(&idx)
	if mErr != nil {
		invalidateCanonicalIndex()
		return
	}
	if err := os.MkdirAll(filepath.Dir(idxPath), 0o755); err != nil {
		invalidateCanonicalIndex()
		return
	}
	if err := os.WriteFile(idxPath, data, 0o600); err != nil {
		invalidateCanonicalIndex()
	}
}
