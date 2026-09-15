// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package brain

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/gofrs/flock"

	"icode.baidu.com/baidu/devops-ai/knowbase-client/internal/utils"
	"icode.baidu.com/baidu/devops-ai/knowbase-client/pkg/types"
)

// NotifyConfig 通知命令配置
type NotifyConfig struct {
	SessionFile  string
	NotebookUUID string
	CursorFile   string // 空则使用默认路径
	Window       int    // 默认 10
	AuthToken    string // COMATE_AUTH_TOKEN，转发给子进程
	DuccBin      string // 可选，空则自动查找（测试注入用）
	AllowedTools string // 可选，空则使用 DefaultSignalDetectorAllowedTools
	// InputFile：预格式化 markdown 文件路径。非空则跳过 session JSONL 解析与 cursor 增量逻辑，
	// 直接把该文件交给 ducc 子进程；与 SessionFile 互斥，由调用方校验。
	InputFile string
	// Sync：仅在 InputFile 模式下生效。为 true 时阻塞等待 ducc 子进程结束后再返回，
	// 用于 warmup Phase 4 串行派发场景（确保前一个 sidecar 写入完毕后再派发下一个）。
	Sync bool

	// Format 调用方对 session 文件格式的 hint；空则由 pickParser 自动识别
	Format SessionFormat

	// ChatMode 聊天模式："personal"（默认） / "group"。空字符串等价 personal。
	// 由 cmd 层 DetectChatMode 解析后填入；group 模式下 buildExtractFile 走群头剥离 +
	// speaker 提取路径，spawnDucc 追加 groupModePromptFragment。
	ChatMode string

	// Model 指定知识沉淀使用的模型；空则不向 ducc 子进程传递 --model
	Model string
}

// Cursor 游标文件内容
type Cursor struct {
	Format        string `json:"format,omitempty"`
	ByteOffset    int64  `json:"byteOffset,omitempty"`
	MessageCount  int    `json:"messageCount,omitempty"`
	TurnCount     int    `json:"turnCount"`
	LastExtractAt string `json:"lastExtractAt"`
}

// Turn 一轮对话（user + assistant 配对）
type Turn struct {
	Index     int
	UserMsg   string
	AssistMsg string
}

// brainWorkDir ducc 子进程的工作目录名（位于 ~/ark-brain），避免 stop hook 循环触发
const brainWorkDir = "ark-brain"

// defaultDuccModel 兜底模型：正常情况下由服务端下发（见 BrainClient.GetDefaultModel），
// 服务端不可达或未配置时使用该值保证 notify 流程不被阻塞。
const defaultDuccModel = "GLM-5.1"

// defaultModelFetchTimeout 拉取服务端默认模型配置的超时时间，默认短时间（5秒），为了不阻塞 stop hook，超时则回退默认
const defaultModelFetchTimeout = 5 * time.Second

// ensureWorkDir 设置 cmd 的工作目录为 ~/ark-brain，目录不存在时自动创建
func ensureWorkDir(cmd *exec.Cmd) error {
	home, err := getHomeDir()
	if err != nil {
		return fmt.Errorf("get home dir: %w", err)
	}
	workDir := filepath.Join(home, brainWorkDir)
	if err := utils.EnsureDir(workDir); err != nil {
		return fmt.Errorf("create workdir %s: %w", workDir, err)
	}
	cmd.Dir = workDir
	return nil
}

// getHomeDir 获取用户 home 目录，优先 os.UserHomeDir，回退 HOME 环境变量
func getHomeDir() (string, error) {
	home, err := os.UserHomeDir()
	if err == nil && home != "" {
		return home, nil
	}
	if h := os.Getenv("HOME"); h != "" {
		return h, nil
	}
	return "", fmt.Errorf("home directory not found")
}

// computeAddDirs 返回需要通过 --add-dir 注入给 ducc 子进程的目录列表。
// 包含：系统临时目录（存放 extract 文件）、~/.knowbase（cursors / canonical_index 等本地状态）。
// 当 home 目录解析失败时只返回临时目录，不阻塞 sidecar 启动。
func computeAddDirs() []string {
	dirs := []string{os.TempDir()}
	if home, err := getHomeDir(); err == nil {
		dirs = append(dirs, filepath.Join(home, ".knowbase"))
	}
	return dirs
}

// resolveCursorPath 确定游标文件路径
func resolveCursorPath(sessionFile, cursorFile string, parser SessionParser) string {
	if cursorFile != "" {
		return cursorFile
	}
	home, err := getHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".knowbase", "cursors", parser.SessionUUID(sessionFile)+".cursor")
}

// readCursor 读取游标，文件不存在时返回零值 Cursor
func readCursor(path string) (Cursor, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return Cursor{}, nil
	}
	if err != nil {
		return Cursor{}, err
	}
	var c Cursor
	if err := json.Unmarshal(data, &c); err != nil {
		return Cursor{}, err
	}
	return c, nil
}

// writeCursor 写入游标（自动创建父目录）
func writeCursor(path string, c Cursor) error {
	c.LastExtractAt = time.Now().UTC().Format(time.RFC3339)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// buildExtractFile 构造临时提取文件，返回文件路径。
// chatMode=group 时输出含群元数据 frontmatter 与 **Speaker**/**Said** 双行 turn 格式；
// personal 模式（默认）保持原 **User**:/**Assistant**: 格式不变。
// groupMeta 仅在 chatMode=group 时使用——personal 模式调用方可传零值。
func buildExtractFile(sessionUUID, notebookUUID string, turns []Turn, totalTurns int,
	chatMode string, groupMeta GroupTurnMeta) (string, error) {
	f, err := os.CreateTemp("", fmt.Sprintf("knowbase-extract-%s-*.md", sessionUUID))
	if err != nil {
		return "", err
	}
	defer f.Close()

	startIdx, endIdx := 0, 0
	if len(turns) > 0 {
		startIdx = turns[0].Index
		endIdx = turns[len(turns)-1].Index
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "# Session Context for Knowledge Extraction\n\n")
	fmt.Fprintf(&sb, "Session ID: %s\n", sessionUUID)
	fmt.Fprintf(&sb, "Notebook UUID: %s\n", notebookUUID)
	if chatMode == types.ChatModeGroup {
		fmt.Fprintf(&sb, "Chat Mode: group\n")
		if groupMeta.GroupID != "" {
			fmt.Fprintf(&sb, "Group ID: %s\n", groupMeta.GroupID)
		}
		if len(groupMeta.Participants) > 0 {
			fmt.Fprintf(&sb, "Group Participants: %s\n", strings.Join(groupMeta.Participants, ", "))
		}
	}
	fmt.Fprintf(&sb, "Extracted At: %s\n", time.Now().UTC().Format(time.RFC3339))
	fmt.Fprintf(&sb, "Window: turns %d-%d of %d total\n\n", startIdx, endIdx, totalTurns)
	fmt.Fprintf(&sb, "---\n\n")

	for _, turn := range turns {
		fmt.Fprintf(&sb, "## Turn %d\n", turn.Index)
		if chatMode == types.ChatModeGroup {
			turnMeta := parseGroupTurn(turn.UserMsg)
			if turnMeta.Speaker == "unknown" {
				fmt.Fprintf(os.Stderr,
					"[notify] group-parse: turn=%d speaker=unknown raw_prefix=%q\n",
					turn.Index, truncate(turn.UserMsg, 60))
			}
			fmt.Fprintf(&sb, "**Speaker**: %s\n", turnMeta.Speaker)
			fmt.Fprintf(&sb, "**Said**: %s\n\n", turnMeta.Content)
		} else {
			fmt.Fprintf(&sb, "**User**: %s\n\n", turn.UserMsg)
		}
		fmt.Fprintf(&sb, "**Assistant**: %s\n\n", turn.AssistMsg)
		fmt.Fprintf(&sb, "---\n\n")
	}

	fmt.Fprintf(&sb, "Run signal-detector skill to extract entities, originals, and log signals\n")
	fmt.Fprintf(&sb, "from the conversation turns above. Use `knowbase brain put` to write all\n")
	fmt.Fprintf(&sb, "extracted pages. KNOWBASE_BRAIN_UUID is set in the environment.\n")

	if _, err := f.WriteString(sb.String()); err != nil {
		return "", err
	}
	return f.Name(), nil
}

// findDucc 查找 ducc 二进制：先 PATH，再备用路径
func findDucc() (string, error) {
	if p, err := exec.LookPath("ducc"); err == nil {
		return p, nil
	}
	home, err := getHomeDir()
	if err != nil {
		return "", errors.New("ducc binary not found in PATH or ~/.baidu-cc/baidu-cc/bin/ducc")
	}
	backup := filepath.Join(home, ".baidu-cc", "baidu-cc", "bin", "ducc")
	if _, err := os.Stat(backup); err == nil {
		return backup, nil
	}
	return "", errors.New("ducc binary not found in PATH or ~/.baidu-cc/baidu-cc/bin/ducc")
}

// cleanupOldExtractFiles 清理超过 maxAge 的旧提取临时文件
func cleanupOldExtractFiles(dir, prefix string, maxAge time.Duration) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	cutoff := time.Now().Add(-maxAge)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), prefix) {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			os.Remove(filepath.Join(dir, entry.Name()))
		}
	}
}

// Deprecated: resolveDuccModel 从服务端拉取默认模型，失败或未配置时回退到 defaultDuccModel。
// 失败路径仅打印 warning 到 stderr，保证 notify 主流程不被模型接口可用性阻塞。
func resolveDuccModel(authToken, notebookUUID string) string {
	bc := NewBrainClient(authToken, notebookUUID)
	ctx, cancel := context.WithTimeout(context.Background(), defaultModelFetchTimeout)
	defer cancel()
	model, err := bc.GetDefaultModel(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[notify] fetch default model failed: %v, fallback to %s\n", err, defaultDuccModel)
		return defaultDuccModel
	}
	if model == "" {
		fmt.Fprintf(os.Stderr, "[notify] server returned empty default model, fallback to %s\n", defaultDuccModel)
		return defaultDuccModel
	}
	return model
}

// RunNotify 执行 notify 主流程
func RunNotify(cfg NotifyConfig) error {
	if cfg.InputFile != "" {
		return runNotifyWithFile(cfg)
	}

	if cfg.Window <= 0 {
		cfg.Window = 10
	}

	cleanupOldExtractFiles(os.TempDir(), "knowbase-extract-", 24*time.Hour)

	parser, err := pickParser(cfg.SessionFile, cfg.Format)
	if err != nil {
		return err
	}

	cursorPath := resolveCursorPath(cfg.SessionFile, cfg.CursorFile, parser)

	// flock 创建 lock 文件时不会自动建父目录，首次运行需要先 ensure
	if err := os.MkdirAll(filepath.Dir(cursorPath), 0o755); err != nil {
		return fmt.Errorf("create cursor dir: %w", err)
	}

	// 对游标文件加排他锁（非阻塞），防止上次提取还未完成时重复触发
	lock := flock.New(cursorPath + ".lock")
	locked, err := lock.TryLock()
	if err != nil {
		return fmt.Errorf("cursor lock error: %w", err)
	}
	if !locked {
		fmt.Fprintln(os.Stderr, "[notify] previous extraction still running, skipping")
		return nil
	}

	cursor, err := readCursor(cursorPath)
	if err != nil {
		lock.Unlock()
		return fmt.Errorf("read cursor: %w", err)
	}
	if cursor.Format != "" && SessionFormat(cursor.Format) != parser.Format() {
		fmt.Fprintf(os.Stderr,
			"[notify] cursor format %q != session format %q, resetting cursor\n",
			cursor.Format, parser.Format())
		cursor = Cursor{}
	}

	f, err := os.Open(cfg.SessionFile)
	if err != nil {
		lock.Unlock()
		return fmt.Errorf("open session file: %w", err)
	}
	defer f.Close()

	turns, newCursor, err := parser.Parse(f, cursor)
	if err != nil {
		lock.Unlock()
		return fmt.Errorf("parse session: %w", err)
	}
	if len(turns) == 0 {
		lock.Unlock()
		return nil
	}

	// 在提交游标之前，先校验派生子进程的前提条件
	// 避免游标已推进但子进程无法启动导致 turns 被永久跳过
	duccBin := cfg.DuccBin
	if duccBin == "" {
		duccBin, err = findDucc()
		if err != nil {
			lock.Unlock()
			return err // 游标未提交，下次可重试
		}
	}
	if err := writeCursor(cursorPath, newCursor); err != nil {
		lock.Unlock()
		return fmt.Errorf("write cursor: %w", err)
	}
	lock.Unlock()

	window := turns
	if len(window) > cfg.Window {
		window = window[len(window)-cfg.Window:]
	}

	sessionUUID := parser.SessionUUID(cfg.SessionFile)
	groupMeta := GroupTurnMeta{}
	if cfg.ChatMode == types.ChatModeGroup {
		groupMeta = findWindowGroupHeader(window)
	}
	tmpFile, err := buildExtractFile(sessionUUID, cfg.NotebookUUID, window, newCursor.TurnCount,
		cfg.ChatMode, groupMeta)
	if err != nil {
		return fmt.Errorf("build extract file: %w", err)
	}
	removeTmpFile := true
	defer func() {
		if removeTmpFile {
			os.Remove(tmpFile)
		}
	}()

	allowedTools := cfg.AllowedTools
	if allowedTools == "" {
		allowedTools = types.DefaultSignalDetectorAllowedTools
	}

	if err := spawnDucc(duccBin, signalDetectorSystemPrompt, tmpFile,
		cfg.NotebookUUID, cfg.AuthToken, allowedTools, cfg.Model, false, cfg.ChatMode); err != nil {
		return err
	}
	removeTmpFile = false
	return nil
}

// runNotifyWithFile 处理 InputFile 模式：跳过 cursor / JSONL 解析，直接把外部预格式化的
// markdown 文件交给 ducc 子进程。调用方源文件不会被 sidecar 触碰，内部先复制到
// /tmp 副本再喂给 ducc，以免下游 brain put --file 误删原文件。
func runNotifyWithFile(cfg NotifyConfig) error {
	fi, err := os.Stat(cfg.InputFile)
	if err != nil {
		return fmt.Errorf("stat input file: %w", err)
	}
	if fi.IsDir() {
		return fmt.Errorf("input file is a directory: %s", cfg.InputFile)
	}
	if fi.Size() == 0 {
		return fmt.Errorf("input file is empty: %s", cfg.InputFile)
	}

	cleanupOldExtractFiles(os.TempDir(), "knowbase-extract-", 24*time.Hour)

	stagedFile, err := stageInputFile(cfg.InputFile)
	if err != nil {
		return fmt.Errorf("stage input file: %w", err)
	}
	// 默认结束时删除 staged 副本；仅 Sync=false（异步派发）下成功路径会改回 false，
	// 把文件留给后台 ducc 读取，再由 cleanupOldExtractFiles 走 24h TTL 兜底回收。
	removeStaged := true
	defer func() {
		if removeStaged {
			os.Remove(stagedFile)
		}
	}()

	duccBin := cfg.DuccBin
	if duccBin == "" {
		duccBin, err = findDucc()
		if err != nil {
			return err
		}
	}

	allowedTools := cfg.AllowedTools
	if allowedTools == "" {
		allowedTools = types.DefaultSignalDetectorAllowedTools
	}

	if err := spawnDucc(duccBin, signalDetectorSystemPrompt, stagedFile, cfg.NotebookUUID, cfg.AuthToken, allowedTools, cfg.Model, cfg.Sync, cfg.ChatMode); err != nil {
		return err
	}
	// 异步派发：ducc 子进程在后台读 stagedFile，必须保留文件直到子进程退出；
	// 同步派发（Sync=true）：spawnDucc 已 Wait，文件可立即删。
	if !cfg.Sync {
		removeStaged = false
	}
	return nil
}

// stageInputFile 把原始 input-file 拷到 /tmp/knowbase-extract-input-<ts>-<base>，
// 返回副本路径。副本由 cleanupOldExtractFiles 走 24h TTL 自然回收，无需主动删。
func stageInputFile(src string) (string, error) {
	data, err := os.ReadFile(src)
	if err != nil {
		return "", err
	}
	base := filepath.Base(src)
	ts := time.Now().UnixNano()
	dst := filepath.Join(os.TempDir(), fmt.Sprintf("knowbase-extract-input-%d-%s", ts, base))
	if err := os.WriteFile(dst, data, 0o600); err != nil {
		return "", err
	}
	return dst, nil
}

// looksLikeGroupUUID 判断 UUID 是否符合群 ID 形态：仅纯数字。
// 其他形态（字母、字母数字混合、含下划线/大写/连字符/空白等）均视为非群 ID（个人邮箱前缀）。
// 如未来调度器使用带前缀的群 UUID（如 "kb-12345"），只需扩展此函数一处即可。
func looksLikeGroupUUID(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// GroupTurnMeta 是 parseGroupTurn 从含群聊前缀的 user message 中拆出的结构化元数据。
type GroupTurnMeta struct {
	GroupID      string   // 数字字符串；解析不到时为空
	Participants []string // 从 "群「...」" 提取，按 "、" 拆分
	Speaker      string   // email 前缀；找不到 [当前用户] 锚点时为 "unknown"
	Content      string   // 剥离群头后的正文；fallback 时为 raw
}

const currentUserMarker = "[当前用户]"

var (
	reGroupID      = regexp.MustCompile(`群ID[:：]\s*(\d+)`)
	reGroupMembers = regexp.MustCompile(`群「([^」]+)」`)
	// reSpeakerNote 取 speaker 中第一个中/英文括号之前的部分，剥离 "liangfu（梁富）" 注释。
	reSpeakerNote = regexp.MustCompile(`^([^（(]+)`)
)

// findWindowGroupHeader 倒序遍历 window，返回第一条能解析出群头的 turn 的元数据。
// 避免 window 末条是 skill 激活 / IDE 注入等非群消息时丢群 GroupID/Participants 元数据。
func findWindowGroupHeader(window []Turn) GroupTurnMeta {
	var meta GroupTurnMeta
	for i := len(window) - 1; i >= 0; i-- {
		gid, parts := parseGroupHeader(window[i].UserMsg)
		if gid != "" || len(parts) > 0 {
			meta.GroupID = gid
			meta.Participants = parts
			break
		}
	}
	return meta
}

// parseGroupHeader 只提取群头元数据（GroupID + Participants），不查找发言人锚点。
func parseGroupHeader(raw string) (groupID string, participants []string) {
	if m := reGroupID.FindStringSubmatch(raw); len(m) == 2 {
		groupID = m[1]
	}
	if m := reGroupMembers.FindStringSubmatch(raw); len(m) == 2 {
		participants = strings.Split(m[1], "、")
	}
	return
}

// parseGroupTurn 从含群聊前缀的 user message 中拆出元数据（纯函数，不打日志）。
// 找不到 [当前用户] 锚点时返回 Speaker="unknown" + Content=raw 兜底；调用方自行决定是否打日志。
func parseGroupTurn(raw string) GroupTurnMeta {
	meta := GroupTurnMeta{Speaker: "unknown", Content: raw}
	meta.GroupID, meta.Participants = parseGroupHeader(raw)

	anchorIdx := strings.LastIndex(raw, currentUserMarker)
	if anchorIdx == -1 {
		return meta
	}

	beforeAnchor := strings.TrimRight(raw[:anchorIdx], " \t\r\n")
	spaceIdx := strings.LastIndexAny(beforeAnchor, " \t\r\n")
	speakerRaw := beforeAnchor[spaceIdx+1:]

	afterAnchor := raw[anchorIdx+len(currentUserMarker):]
	afterAnchor = strings.TrimLeft(afterAnchor, " \t\r\n")
	afterAnchor = strings.TrimPrefix(afterAnchor, ":")
	afterAnchor = strings.TrimPrefix(afterAnchor, "：")

	speaker := stripSpeakerNote(speakerRaw)
	if speaker == "" {
		// [当前用户] 出现在 raw 起始（或仅前置空白）时 speakerRaw 为空，
		// 保留入口处的 "unknown" 兜底，避免覆写成空串绕过下游 unknown 判定与 prompt 铁律。
		speaker = "unknown"
	}
	meta.Speaker = speaker
	meta.Content = strings.TrimSpace(afterAnchor)
	return meta
}

// stripSpeakerNote 剥离 speaker 后中/英文括号注释：
//   - "liangfu（梁富）" → "liangfu"
//   - "liangfu(liangfu)" → "liangfu"
//   - "liangfu" → "liangfu"
func stripSpeakerNote(s string) string {
	if m := reSpeakerNote.FindStringSubmatch(s); len(m) == 2 {
		return strings.TrimSpace(m[1])
	}
	return strings.TrimSpace(s)
}

// DetectChatMode 解析最终生效的 chat-mode。
//   - explicit 取值（"personal"/"group"）直接生效；与 auto 判定不一致时 stderr warn
//   - 空字符串走 auto：looksLikeGroupUUID(kbUUID) 为 true → group，否则 personal
func DetectChatMode(explicit, kbUUID string) string {
	autoMode := types.ChatModePersonal
	if looksLikeGroupUUID(kbUUID) {
		autoMode = types.ChatModeGroup
	}
	switch explicit {
	case types.ChatModePersonal, types.ChatModeGroup:
		if explicit != autoMode {
			fmt.Fprintf(os.Stderr,
				"[notify] chat-mode override: explicit=%s but UUID %q looks %s\n",
				explicit, kbUUID, autoMode)
		}
		return explicit
	}
	return autoMode
}
