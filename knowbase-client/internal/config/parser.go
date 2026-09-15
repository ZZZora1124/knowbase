// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
	"icode.baidu.com/baidu/devops-ai/knowbase-client/internal/utils"
	"icode.baidu.com/baidu/devops-ai/knowbase-client/pkg/types"
)

// Parser 配置解析器
type Parser struct {
	configPath string
}

// NewParser 创建配置解析器
func NewParser(configPath string) (*Parser, error) {
	// 展开路径
	expandedPath, err := utils.ExpandPath(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to expand config path: %w", err)
	}

	// 检查文件是否存在
	if !utils.FileExists(expandedPath) {
		return nil, fmt.Errorf("config file not found: %s", expandedPath)
	}

	return &Parser{
		configPath: expandedPath,
	}, nil
}

// Parse 解析配置文件
func (p *Parser) Parse() (*types.KnowledgeConfig, error) {
	// 读取文件
	data, err := os.ReadFile(p.configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// 检测配置文件格式
	format, err := detectConfigFormat(p.configPath, data)
	if err != nil {
		return nil, err
	}

	// 根据格式解析配置
	var config types.KnowledgeConfig
	switch format {
	case "json":
		if err := json.Unmarshal(data, &config); err != nil {
			return nil, fmt.Errorf("failed to parse JSON: %w", err)
		}
	case "yaml":
		if err := yaml.Unmarshal(data, &config); err != nil {
			return nil, fmt.Errorf("failed to parse YAML: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported config format: %s", format)
	}

	// 验证配置
	if err := p.validate(&config); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &config, nil
}

// validate 验证配置
func (p *Parser) validate(config *types.KnowledgeConfig) error {
	validator := NewValidator()
	return validator.Validate(config)
}

// detectConfigFormat 检测配置文件格式
// 参数: filePath - 配置文件路径, data - 文件内容
// 返回: "json" 或 "yaml"
func detectConfigFormat(filePath string, data []byte) (string, error) {
	// 1. 首先根据文件扩展名判断
	ext := strings.ToLower(filepath.Ext(filePath))

	// 2. 如果是.json扩展名,尝试JSON解析
	if ext == ".json" {
		var test interface{}
		if err := json.Unmarshal(data, &test); err == nil {
			return "json", nil
		}
		return "", fmt.Errorf("failed to parse config file as JSON or YAML")
	}

	// 3. 如果是.yaml或.yml扩展名,先尝试YAML解析
	if ext == ".yaml" || ext == ".yml" {
		var test interface{}
		if err := yaml.Unmarshal(data, &test); err == nil {
			return "yaml", nil
		}
		return "", fmt.Errorf("failed to parse config file as YAML or JSON")
	}

	return "", fmt.Errorf("unsupported config file format: %s", ext)
}
