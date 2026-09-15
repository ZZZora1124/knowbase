// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package brain

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type ccJSONLParser struct{}

func (p *ccJSONLParser) Format() SessionFormat { return FormatCCJSONL }

func (p *ccJSONLParser) SessionUUID(sessionFile string) string {
	return strings.TrimSuffix(filepath.Base(sessionFile), ".jsonl")
}

func (p *ccJSONLParser) Parse(f *os.File, prev Cursor) ([]Turn, Cursor, error) {
	fi, err := f.Stat()
	if err != nil {
		return nil, prev, fmt.Errorf("stat session file: %w", err)
	}
	if fi.Size() <= prev.ByteOffset {
		return nil, Cursor{
			Format:       string(FormatCCJSONL),
			ByteOffset:   prev.ByteOffset,
			MessageCount: prev.MessageCount,
			TurnCount:    prev.TurnCount,
		}, nil
	}
	if _, err := f.Seek(prev.ByteOffset, io.SeekStart); err != nil {
		return nil, prev, fmt.Errorf("seek session file: %w", err)
	}
	turns, bytesRead, err := parseSessionTurns(f)
	if err != nil {
		return nil, prev, fmt.Errorf("parse session: %w", err)
	}
	// Index 在配对时从 1 起，按 prev.TurnCount 偏移以保持全局序号
	for i := range turns {
		turns[i].Index += prev.TurnCount
	}
	next := Cursor{
		Format:     string(FormatCCJSONL),
		ByteOffset: prev.ByteOffset + bytesRead,
		TurnCount:  prev.TurnCount + len(turns),
	}
	return turns, next, nil
}

// parseSessionTurns 把 jsonl reader 按 role+block-type 配对成 (UserMsg, AssistMsg) Turn 序列。
// 配对规则：
//   - role=user 且 extractText 非空 → 提交上一对 Turn，开始新 Turn 的 UserMsg
//   - role=user 且 extractText 为空（tool_result / 空 string）→ 跳过，不影响 pendingUser
//   - role=assistant 且 extractText 非空 → 累积到当前 Turn 的 AssistMsg（多 text block 用 \n 拼接）
//   - role=assistant 且 extractText 为空（thinking / tool_use）→ 跳过累积
//   - role=assistant 出现时 pendingUser 为空 → 孤儿 assistant，跳过
//   - 文件结尾：提交最后一对未完成 Turn（即使 AssistMsg 为空也保留，避免丢用户输入）
func parseSessionTurns(r io.Reader) ([]Turn, int64, error) {
	const maxLineBytes = 4 * 1024 * 1024 // 4MB

	reader := bufio.NewReader(r)
	var (
		pendingUser      string
		pendingAssistant strings.Builder
		turns            []Turn
		idx              int
		totalBytes       int64
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

	for {
		line, err := reader.ReadBytes('\n')
		totalBytes += int64(len(line))

		if len(line) > maxLineBytes {
			fmt.Fprintf(os.Stderr, "[notify] skipping oversized line (%d bytes)\n", len(line))
			if err == io.EOF {
				break
			}
			if err != nil {
				return nil, totalBytes, err
			}
			continue
		}

		line = bytes.TrimRight(line, "\r\n")
		if len(line) == 0 {
			if err == io.EOF {
				break
			}
			if err != nil {
				return nil, totalBytes, err
			}
			continue
		}

		var sl sessionLine
		if jsonErr := json.Unmarshal(line, &sl); jsonErr != nil {
			if err == io.EOF {
				break
			}
			if err != nil {
				return nil, totalBytes, err
			}
			continue
		}
		if sl.Message == nil {
			if err == io.EOF {
				break
			}
			if err != nil {
				return nil, totalBytes, err
			}
			continue
		}
		text := extractText(sl.Message.Content)

		switch sl.Message.Role {
		case "user":
			if text == "" {
				// tool_result 等空 user，不覆盖 pendingUser
				break
			}
			commit()
			pendingUser = text
		case "assistant":
			if text == "" {
				// thinking / tool_use 单独行，无文本可累积
				break
			}
			if pendingUser == "" {
				// 孤儿 assistant（文件开头就出现），跳过
				break
			}
			if pendingAssistant.Len() > 0 {
				pendingAssistant.WriteByte('\n')
			}
			pendingAssistant.WriteString(text)
		}

		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, totalBytes, err
		}
	}
	commit() // 文件结尾：提交最后一对
	return turns, totalBytes, nil
}

type sessionLine struct {
	Type    string          `json:"type"`
	Message *sessionMessage `json:"message,omitempty"`
}

type sessionMessage struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}

func extractText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	var blocks []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &blocks); err == nil {
		var parts []string
		for _, b := range blocks {
			if b.Type == "text" && b.Text != "" {
				parts = append(parts, b.Text)
			}
		}
		return strings.Join(parts, "\n")
	}
	return ""
}
