// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package brain

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SessionFormat 标识 session 日志文件格式
type SessionFormat string

const (
	FormatCCJSONL  SessionFormat = "cc-jsonl"
	FormatZuluJSON SessionFormat = "zulu-json"
)

// SessionParser 按格式封装 session 文件的解析与游标推进
type SessionParser interface {
	Format() SessionFormat
	// SessionUUID 从文件路径中提取 session uuid，用于命名 cursor 文件
	SessionUUID(sessionFile string) string
	// Parse 基于 prev 游标做增量解析，返回新 turns 与下一次游标
	Parse(f *os.File, prev Cursor) ([]Turn, Cursor, error)
}

// pickParser 按 hint > 文件名 > 路径关键字 顺序识别 session 格式
func pickParser(sessionFile string, hint SessionFormat) (SessionParser, error) {
	if hint != "" {
		switch hint {
		case FormatCCJSONL:
			return &ccJSONLParser{}, nil
		case FormatZuluJSON:
			return &zuluParser{}, nil
		default:
			return nil, fmt.Errorf("unknown session format hint: %q", hint)
		}
	}
	base := filepath.Base(sessionFile)
	if strings.HasSuffix(base, ".jsonl") {
		return &ccJSONLParser{}, nil
	}
	if strings.HasPrefix(base, "chat_session_") {
		return &zuluParser{}, nil
	}
	normalized := filepath.ToSlash(sessionFile)
	if strings.Contains(normalized, "comate-engine/store") {
		return &zuluParser{}, nil
	}
	if strings.Contains(normalized, "/.claude/") {
		return &ccJSONLParser{}, nil
	}
	return nil, fmt.Errorf("cannot detect session format for %q; pass --format explicitly", sessionFile)
}
