// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package converter

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"sort"
	"strings"

	"icode.baidu.com/baidu/devops-ai/knowbase-client/pkg/types"
)

// Converter 格式转换器
type Converter struct{}

// NewConverter 创建转换器
func NewConverter() *Converter {
	return &Converter{}
}

// Convert 转换知识数据
func (c *Converter) Convert(sourceType string, raw []types.RawKnowledge, output types.OutputConfig) ([]types.KnowledgeItem, error) {
	items := make([]types.KnowledgeItem, 0, len(raw))

	for _, r := range raw {
		item, err := c.convertOne(sourceType, r, output)
		if err != nil {
			return nil, fmt.Errorf("failed to convert knowledge item: %w", err)
		}
		items = append(items, item)
	}

	return items, nil
}

// convertOne 转换单个知识项
func (c *Converter) convertOne(sourceType string, raw types.RawKnowledge, output types.OutputConfig) (types.KnowledgeItem, error) {
	// 提取配置的字段
	extracted := make(map[string]interface{})
	if sourceType == types.SourceTypeICafeCard {
		output.Fields = append(output.Fields,
			types.FieldConfig{
				KeyPath:     "createdTime",
				Description: "创建时间",
			},
			types.FieldConfig{
				KeyPath:     "lastModifiedTime",
				Description: "更新时间",
			},
			types.FieldConfig{
				KeyPath:     "status",
				Description: "状态",
			},
		)
	}
	for _, field := range output.Fields {
		value := c.extractField(raw.Data, field.KeyPath)
		if value != nil {
			extracted[field.KeyPath] = value
		}
	}

	// 根据格式转换
	var content string
	var err error

	switch output.Format {
	case types.OutputFormatMarkdown:
		content = c.toMarkdown(extracted, output.Fields)
	case types.OutputFormatJSON:
		content, err = c.toJSON(extracted)
	case types.OutputFormatXML:
		content, err = c.toXML(extracted)
	default:
		return types.KnowledgeItem{}, fmt.Errorf("unsupported format: %s", output.Format)
	}

	if err != nil {
		return types.KnowledgeItem{}, err
	}

	// 构建知识项
	item := types.KnowledgeItem{
		SourceType: raw.SourceType,
		Content:    content,
		Metadata:   extracted,
	}

	// 尝试提取ID和标题
	if id, ok := raw.Data["id"]; ok {
		item.ID = fmt.Sprintf("%v", id)
	}
	if title, ok := extracted["title"]; ok {
		item.Title = fmt.Sprintf("%v", title)
	}

	return item, nil
}

// extractField 从数据中提取字段值（支持嵌套路径，如 "user.name"）
func (c *Converter) extractField(data map[string]interface{}, keyPath string) interface{} {
	keys := strings.Split(keyPath, ".")
	var current interface{} = data

	for _, key := range keys {
		if m, ok := current.(map[string]interface{}); ok {
			current = m[key]
		} else {
			return nil
		}
	}

	return current
}

// toMarkdown 转换为Markdown格式
func (c *Converter) toMarkdown(data map[string]interface{}, fields []types.FieldConfig) string {
	var sb strings.Builder

	for _, field := range fields {
		value := data[field.KeyPath]
		if value == nil {
			continue
		}

		// 使用description作为标题
		if field.Description != "" {
			sb.WriteString(fmt.Sprintf("## %s\n\n", field.Description))
		} else {
			sb.WriteString(fmt.Sprintf("## %s\n\n", field.KeyPath))
		}

		// 写入值
		sb.WriteString(fmt.Sprintf("%v\n\n", value))
	}

	return sb.String()
}

// toJSON 转换为JSON格式
func (c *Converter) toJSON(data map[string]interface{}) (string, error) {
	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %w", err)
	}
	return string(bytes), nil
}

// toXML 将知识数据转换为XML格式字符串
//
// 该函数会接收一个包含知识数据的map，将其转换为格式良好的XML字符串。
// XML结构以<knowledge>为根元素，内部元素按map的键值对进行递归转换。
//
// 参数:
//
//	data - 包含知识数据的map，键为字符串，值为任意类型
//
// 返回:
//
//	string - 转换后的XML格式字符串
//	error - 转换过程中发生的错误，成功时为nil
func (c *Converter) toXML(data map[string]interface{}) (string, error) {
	var buf bytes.Buffer
	buf.WriteString(xml.Header)
	buf.WriteString("<knowledge>\n")

	if err := c.mapToXML(&buf, data, "  "); err != nil {
		return "", fmt.Errorf("failed to marshal XML: %w", err)
	}

	buf.WriteString("</knowledge>\n")
	return buf.String(), nil
}

// mapToXML 递归将map转换为XML
func (c *Converter) mapToXML(buf *bytes.Buffer, data map[string]interface{}, indent string) error {
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		if err := c.valueToXML(buf, key, data[key], indent); err != nil {
			return err
		}
	}
	return nil
}

// valueToXML 将单个值转换为XML
func (c *Converter) valueToXML(buf *bytes.Buffer, key string, value interface{}, indent string) error {
	switch v := value.(type) {
	case nil:
		fmt.Fprintf(buf, "%s<%s/>\n", indent, key)

	case string:
		fmt.Fprintf(buf, "%s<%s>%s</%s>\n", indent, key, c.escapeXML(v), key)

	case int, int64, int32, float64, float32, bool:
		fmt.Fprintf(buf, "%s<%s>%v</%s>\n", indent, key, v, key)

	case map[string]interface{}:
		fmt.Fprintf(buf, "%s<%s>\n", indent, key)
		if err := c.mapToXML(buf, v, indent+"  "); err != nil {
			return err
		}
		fmt.Fprintf(buf, "%s</%s>\n", indent, key)

	case []interface{}:
		for _, item := range v {
			if err := c.valueToXML(buf, key, item, indent); err != nil {
				return err
			}
		}

	default:
		fmt.Fprintf(buf, "%s<%s>%v</%s>\n", indent, key, v, key)
	}
	return nil
}

// escapeXML 转义XML特殊字符
func (c *Converter) escapeXML(s string) string {
	replacer := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		"\"", "&quot;",
		"'", "&apos;",
	)
	return replacer.Replace(s)
}
