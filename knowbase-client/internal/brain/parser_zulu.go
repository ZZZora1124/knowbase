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
)

const zuluMaxFileSize = 32 * 1024 * 1024 // 32MB

type zuluParser struct{}

func (p *zuluParser) Format() SessionFormat { return FormatZuluJSON }

func (p *zuluParser) SessionUUID(sessionFile string) string {
	return strings.TrimPrefix(filepath.Base(sessionFile), "chat_session_")
}

type zuluDoc struct {
	SessionUUID string        `json:"sessionUuid"`
	Messages    []zuluMessage `json:"messages"`
}

type zuluMessage struct {
	ID       string          `json:"id"`
	Role     string          `json:"role"`
	Status   string          `json:"status"`
	Content  json.RawMessage `json:"content"`
	Elements []zuluElement   `json:"elements"`
}

type zuluElement struct {
	ID       string        `json:"id"`
	Type     string        `json:"type"`
	Discard  bool          `json:"discard"`
	Content  string        `json:"content"`
	ToolName string        `json:"toolName"`
	Children []zuluElement `json:"children"`
}

func (p *zuluParser) Parse(f *os.File, prev Cursor) ([]Turn, Cursor, error) {
	fi, err := f.Stat()
	if err != nil {
		return nil, prev, fmt.Errorf("stat zulu session: %w", err)
	}
	if fi.Size() > zuluMaxFileSize {
		return nil, prev, fmt.Errorf("zulu session file too large: %d > %d", fi.Size(), zuluMaxFileSize)
	}

	var doc zuluDoc
	if err := json.NewDecoder(f).Decode(&doc); err != nil {
		return nil, prev, fmt.Errorf("decode zulu session: %w", err)
	}

	start := prev.MessageCount
	if start > len(doc.Messages) {
		fmt.Fprintf(os.Stderr,
			"[notify] zulu session shrunk (cursor=%d, file=%d), resetting cursor\n",
			start, len(doc.Messages))
		start = 0
	}
	newMsgs := doc.Messages[start:]
	turns := pairZuluTurns(newMsgs, prev.TurnCount)

	next := Cursor{
		Format:       string(FormatZuluJSON),
		MessageCount: len(doc.Messages),
		TurnCount:    prev.TurnCount + len(turns),
	}
	return turns, next, nil
}

// pairZuluTurns 按 user→assistant 配对。规则：
//   - user 来时先 commit() 上一对，避免连续 user（或 user → 空 assistant → user）时前一个 user 被覆盖丢失
//   - 非空 assistant text 累积到 pendingAssistant（zulu 实测严格 1:1 交替，累积多见于将来扩展）
//   - 空 assistant（status=cancelled / elements=[]）：跳过累积但保留 pendingUser
//   - 末尾 commit() 兜底，保证最后一个 user 不丢
func pairZuluTurns(msgs []zuluMessage, startIdx int) []Turn {
	var (
		turns            []Turn
		pendingUser      string
		pendingAssistant strings.Builder
		idx              = startIdx
	)
	commit := func() {
		if pendingUser == "" {
			return
		}
		idx++
		turns = append(turns, Turn{
			Index:     idx,
			UserMsg:   pendingUser,
			AssistMsg: strings.TrimSpace(pendingAssistant.String()),
		})
		pendingUser = ""
		pendingAssistant.Reset()
	}
	for _, m := range msgs {
		switch m.Role {
		case "user":
			content := extractZuluUserContent(m.Content)
			if content == "" {
				continue
			}
			commit()
			pendingUser = content
		case "assistant":
			text := extractZuluAssistantText(m.Elements)
			if text == "" {
				continue
			}
			if pendingUser == "" {
				continue
			}
			if pendingAssistant.Len() > 0 {
				pendingAssistant.WriteByte('\n')
			}
			pendingAssistant.WriteString(text)
		}
	}
	commit()
	return turns
}

func extractZuluUserContent(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	return ""
}

func extractZuluAssistantText(elements []zuluElement) string {
	var parts []string
	var walk func(e zuluElement)
	walk = func(e zuluElement) {
		if e.Discard {
			return
		}
		switch e.Type {
		case "TEXT":
			if e.Content != "" {
				parts = append(parts, e.Content)
			}
		case "TOOL":
			if e.ToolName != "" {
				parts = append(parts, fmt.Sprintf("[tool: %s]", e.ToolName))
			}
		}
		for _, c := range e.Children {
			walk(c)
		}
	}
	for _, e := range elements {
		walk(e)
	}
	return strings.Join(parts, "\n")
}
