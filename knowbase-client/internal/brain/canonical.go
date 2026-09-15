// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package brain

import (
	"strings"
	"unicode"

	"github.com/mozillazg/go-pinyin"
)

// CanonicalKey 把名称归一化为查重 key。
// 规则：
//  1. 中文 → 拼音（无声调、连续拼写）
//  2. 全部小写
//  3. 去除所有非字母数字字符（连字符 / 下划线 / 空格 / 标点等）
//
// 例：
//
//	CanonicalKey("Agent-Brain")    == "agentbrain"
//	CanonicalKey("deep_research")  == "deepresearch"
//	CanonicalKey("张三")            == "zhangsan"
//	CanonicalKey("张-三")           == "zhangsan"
//	CanonicalKey("Liang Fu")       == "liangfu"
func CanonicalKey(name string) string {
	if name == "" {
		return ""
	}
	// 第一步：中文转拼音（不破坏已有的英文 / 数字）
	args := pinyin.NewArgs()
	args.Style = pinyin.Normal // 无声调
	args.Heteronym = false     // 多音字仅取最常见读音

	var sb strings.Builder
	for _, r := range name {
		if unicode.Is(unicode.Han, r) {
			py := pinyin.SinglePinyin(r, args)
			if len(py) > 0 {
				sb.WriteString(py[0])
			}
			continue
		}
		// 非中文字符原样保留（后续 normalize 再清理）
		sb.WriteRune(r)
	}

	// 第二步：小写 + 仅保留字母数字
	var out strings.Builder
	for _, r := range sb.String() {
		switch {
		case r >= 'a' && r <= 'z':
			out.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			out.WriteRune(r + ('a' - 'A'))
		case r >= '0' && r <= '9':
			out.WriteRune(r)
		}
	}
	return out.String()
}

// CanonicalKeyFromSlug 从已有 slug（可能是 "people/zhang-san" 或 "zhang-san"）提取归一化 key。
// 自动剥离 "<type>/" 前缀。
func CanonicalKeyFromSlug(slug string) string {
	name := slug
	if i := strings.LastIndex(slug, "/"); i >= 0 {
		name = slug[i+1:]
	}
	return CanonicalKey(name)
}
