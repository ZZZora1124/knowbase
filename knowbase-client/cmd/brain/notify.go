// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package brain

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	br "icode.baidu.com/baidu/devops-ai/knowbase-client/internal/brain"
	"icode.baidu.com/baidu/devops-ai/knowbase-client/internal/utils"
	"icode.baidu.com/baidu/devops-ai/knowbase-client/pkg/types"
)

// stopData 兼容 CC/Ducc 和 Zulu/Comate 两套 Stop hook payload
type stopData struct {
	HookEventName string `json:"hook_event_name"`
	Cwd           string `json:"cwd"`

	// CC/Ducc
	TranscriptPath string `json:"transcript_path,omitempty"`
	// SessionID: cc 是字符串（UUID），zulu 是数字；cmd 层不消费该字段，仅做兼容反序列化
	SessionID      json.RawMessage `json:"session_id,omitempty"`
	PermissionMode string          `json:"permission_mode,omitempty"`
	StopHookActive bool            `json:"stop_hook_active,omitempty"`

	// Zulu/Comate
	ConversationID string `json:"conversation_id,omitempty"`
	Status         string `json:"status,omitempty"`
	LoopCount      int    `json:"loop_count,omitempty"`
}

// resolveEffectiveFormat 在 hook 模式下解析最终生效的 session 文件格式：
// 用户显式通过 --format 传入的值优先于 payload 推断出的 hintFormat，
// 二者都为空时返回空字符串，交由下游 pickParser 自动检测。
func resolveEffectiveFormat(userFormat string, hintFormat br.SessionFormat) br.SessionFormat {
	if userFormat != "" {
		return br.SessionFormat(userFormat)
	}
	return hintFormat
}

// resolveHookSessionFile 按优先级解析 hook payload 中 session 文件路径与格式 hint：
//  1. transcript_path 非空（cc 协议）
//  2. conversation_id 非空（zulu 协议） → ~/.comate-engine/store/chat_session_<id>
//  3. 都缺 → 报错
func resolveHookSessionFile(sd stopData) (string, br.SessionFormat, error) {
	if sd.TranscriptPath != "" {
		return sd.TranscriptPath, br.SessionFormat(""), nil
	}
	if sd.ConversationID != "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", "", fmt.Errorf("get home dir: %w", err)
		}
		path := filepath.Join(home, ".comate-engine", "store",
			"chat_session_"+sd.ConversationID)
		return path, br.FormatZuluJSON, nil
	}
	return "", "", fmt.Errorf("hook payload missing both transcript_path and conversation_id")
}

// resolveNotebookUUID resolves the notebook UUID with priority: --kb-uuid > --notebook-uuid > env.
func resolveNotebookUUID() (string, error) {
	if brainKbUUID != "" {
		return brainKbUUID, nil
	}
	if notifyNotebookUUID != "" {
		return notifyNotebookUUID, nil
	}
	if v := os.Getenv(types.EnvKnowbaseBrainUUID); v != "" {
		return v, nil
	}
	return "", fmt.Errorf("notebook-uuid is required (via --kb-uuid, --notebook-uuid, or env %s)", types.EnvKnowbaseBrainUUID)
}

var notifyCmd = &cobra.Command{
	Use:   "notify",
	Short: "Notify knowbase of a new session turn for knowledge extraction (sidecar mode)",
	Long: `Incrementally read a ducc session JSONL file and spawn a background ducc
subprocess to extract knowledge via signal-detector skill.

This command is stateless and self-contained: it reads the cursor, parses new
turns, advances the cursor, and detaches a ducc process — then exits immediately.

Required environment variables:
  COMATE_AUTH_TOKEN    Authentication token (forwarded to the ducc subprocess)

Modes:
  CLI mode (default):  Pass --session and --notebook-uuid as flags.
  Hook mode (--hook):  Read Stop hook event JSON from stdin, extract transcript_path
                       as the session file. Useful when configured as a Claude Code
                       Stop hook in .claude/settings.json.

Examples:
  knowbase brain notify --session ~/.claude/sessions/abc.jsonl --notebook-uuid <uuid>
  knowbase brain notify --session ~/.claude/sessions/abc.jsonl --notebook-uuid <uuid> --model GLM-5.1
  knowbase brain notify --session ~/.comate-engine/store/chat_session_xyz --notebook-uuid <uuid> --format zulu-json
  echo '{"transcript_path":"/path/to/session.jsonl"}' | knowbase brain notify --hook --notebook-uuid <uuid>
  echo '{"conversation_id":"xyz","hook_event_name":"Stop"}' | knowbase brain notify --hook --notebook-uuid <uuid>`,
	Run: runNotify,
}

var notifySession string
var notifyNotebookUUID string
var notifyCursor string
var notifyWindow int
var notifyAllowedTools string
var notifyHook bool
var notifyInputFile string
var notifySync bool
var notifyFormat string
var notifyChatMode string
var notifyModel string

func init() {
	notifyCmd.Flags().StringVar(&notifySession, "session", "", "Path to CC/Ducc session JSONL file (required in CLI mode)")
	notifyCmd.Flags().StringVar(&notifyNotebookUUID, "notebook-uuid", "", "Target knowledge base UUID (required; in hook mode, falls back to env KNOWBASE_BRAIN_UUID)")
	notifyCmd.Flags().StringVar(&notifyCursor, "cursor", "", "Cursor file path (default: ~/.knowbase/cursors/<session-uuid>.cursor)")
	notifyCmd.Flags().IntVar(&notifyWindow, "window", 10, "Number of recent turns to include in extraction window")
	notifyCmd.Flags().StringVar(&notifyAllowedTools, "allowed-tools", "", "Override allowed tools for signal-detector subprocess (default: built-in safe set)")
	notifyCmd.Flags().BoolVar(&notifyHook, "hook", false, "Hook mode: read Stop event JSON from stdin instead of --session flag")
	notifyCmd.Flags().StringVar(&notifyInputFile, "input-file", "", "Path to a pre-formatted markdown file (mutually exclusive with --session/--hook). Skips JSONL parsing and cursor logic; ducc receives the file as-is.")
	notifyCmd.Flags().BoolVar(&notifySync, "sync", true, "Wait for ducc subprocess to complete before returning (only effective with --input-file; default true to ensure serial dispatch is safe by default; pass --sync=false to opt-in to fire-and-forget).")
	notifyCmd.Flags().StringVar(&notifyFormat, "format", "",
		"Session file format (leave empty for auto-detect; explicit values: cc-jsonl or zulu-json)")
	notifyCmd.Flags().StringVar(&notifyChatMode, "chat-mode", "",
		"Chat mode: personal | group (auto-detected from --kb-uuid format if empty)")
	notifyCmd.Flags().StringVar(&notifyModel, "model", "",
		"Model to use for knowledge extraction (optional, defaults to the model used in the session)")
}

func runNotify(cmd *cobra.Command, args []string) {
	envVal := os.Getenv(types.EnvKnowbaseEnv)
	if envVal == "" {
		envVal = "prod"
	}
	if err := types.InitEnv(envVal); err != nil {
		utils.PrintlnError("Error: %v", err)
		os.Exit(types.ExitRunSubCMDError)
	}

	authToken := os.Getenv(types.EnvComateAuthToken)
	if authToken == "" {
		utils.PrintlnError("Env: %s is not set", types.EnvComateAuthToken)
		os.Exit(types.ExitRunSubCMDError)
	}

	// input-file 模式：与 --session/--hook 互斥，跳过 cursor / JSONL 解析，直接派给 ducc
	if notifyInputFile != "" {
		if notifySession != "" || notifyHook {
			utils.PrintlnError("--input-file is mutually exclusive with --session and --hook")
			os.Exit(types.ExitRunSubCMDError)
		}
		notebookUUID, err := resolveNotebookUUID()
		if err != nil {
			utils.PrintlnError("%v", err)
			os.Exit(types.ExitRunSubCMDError)
		}
		cfg := br.NotifyConfig{
			InputFile:    notifyInputFile,
			NotebookUUID: notebookUUID,
			AuthToken:    authToken,
			AllowedTools: notifyAllowedTools,
			Sync:         notifySync,
			ChatMode:     br.DetectChatMode(notifyChatMode, notebookUUID),
			Model:        notifyModel,
		}
		if err := br.RunNotify(cfg); err != nil {
			utils.PrintlnError("notify failed: %v", err)
			os.Exit(types.ExitRunSubCMDError)
		}
		return
	}

	if notifyHook {
		defer func() {
			// 无论成败都向 stdout 写 "{}"，确保 zulu BLOCKING hook fail-open
			fmt.Fprintln(os.Stdout, "{}")
		}()

		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			utils.PrintlnError("Error reading stdin: %v", err)
			return
		}
		var sd stopData
		if err := json.Unmarshal(data, &sd); err != nil {
			utils.PrintlnError("Error parsing hook data: %v", err)
			return
		}
		sessionFile, hintFormat, err := resolveHookSessionFile(sd)
		if err != nil {
			utils.PrintlnError("%v", err)
			return
		}
		if _, err := os.Stat(sessionFile); os.IsNotExist(err) {
			utils.PrintlnError("Session file not found: %s", sessionFile)
			return
		}

		notebookUUID, err := resolveNotebookUUID()
		if err != nil {
			utils.PrintlnError("%v", err)
			return
		}

		cfg := br.NotifyConfig{
			SessionFile:  sessionFile,
			NotebookUUID: notebookUUID,
			CursorFile:   notifyCursor,
			Window:       notifyWindow,
			AuthToken:    authToken,
			AllowedTools: notifyAllowedTools,
			Format:       resolveEffectiveFormat(notifyFormat, hintFormat),
			ChatMode:     br.DetectChatMode(notifyChatMode, notebookUUID),
			Model:        notifyModel,
		}
		if err := br.RunNotify(cfg); err != nil {
			utils.PrintlnError("notify failed: %v", err)
			return
		}
		fmt.Fprintln(os.Stderr, "brain notify: hook mode completed")
		return
	}

	// CLI 模式：使用 --session flag
	if notifySession == "" {
		utils.PrintlnError("--session is required (or use --hook mode)")
		os.Exit(types.ExitRunSubCMDError)
	}
	notebookUUID, err := resolveNotebookUUID()
	if err != nil {
		utils.PrintlnError("%v", err)
		os.Exit(types.ExitRunSubCMDError)
	}

	cfg := br.NotifyConfig{
		SessionFile:  notifySession,
		NotebookUUID: notebookUUID,
		CursorFile:   notifyCursor,
		Window:       notifyWindow,
		AuthToken:    authToken,
		AllowedTools: notifyAllowedTools,
		Format:       br.SessionFormat(notifyFormat),
		ChatMode:     br.DetectChatMode(notifyChatMode, notebookUUID),
		Model:        notifyModel,
	}
	if err := br.RunNotify(cfg); err != nil {
		utils.PrintlnError("notify failed: %v", err)
		os.Exit(types.ExitRunSubCMDError)
	}
}
