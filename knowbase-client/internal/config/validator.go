// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package config

import (
	"fmt"
	"strconv"
	"strings"

	"icode.baidu.com/baidu/devops-ai/knowbase-client/pkg/types"
)

// Validator 配置验证器
type Validator struct {
	errors []string
}

// NewValidator 创建验证器
func NewValidator() *Validator {
	return &Validator{
		errors: make([]string, 0),
	}
}

// Validate 验证配置
func (v *Validator) Validate(config *types.KnowledgeConfig) error {
	// 验证版本号
	if config.Version == "" {
		v.addError("version is required")
	}

	// 验证meta
	v.validateMeta(&config.Meta)

	// 验证primaryRepo
	v.validatePrimaryRepo(config.PrimaryRepo, &config.Storage)

	// 验证sources
	if len(config.Sources) == 0 {
		v.addError("at least one source is required")
	} else {
		for i, source := range config.Sources {
			v.validateSource(&source, i)
		}
	}

	// 验证storage
	v.validateStorage(&config.Storage, config.PrimaryRepo)

	// 验证knowledgeSkill配置
	v.validateKnowledgeSkill(config.KnowledgeSkill)

	// 如果有错误，返回
	if len(v.errors) > 0 {
		return fmt.Errorf("validation errors:\n- %s", strings.Join(v.errors, "\n- "))
	}

	return nil
}

// validateMeta 验证meta配置
func (v *Validator) validateMeta(meta *types.MetaConfig) {
	if meta.ProjectID == "" {
		v.addError("meta.projectId is required")
	}
	if meta.Version == "" {
		v.addError("meta.version is required")
	}
}

// validateSource 验证源配置
func (v *Validator) validateSource(source *types.SourceConfig, index int) {
	prefix := fmt.Sprintf("sources[%d]", index)

	// 验证type
	if source.Type == "" {
		v.addError(fmt.Sprintf("%s.type is required", prefix))
		return
	}

	// 验证type是否在支持列表中
	if !contains(types.GetAllSourceTypes(), source.Type) {
		v.addError(fmt.Sprintf("%s.type '%s' is not supported", prefix, source.Type))
		return
	}

	// 验证filters（根据不同的知识源类型）
	if len(source.Filters) == 0 {
		v.addError(fmt.Sprintf("%s.filters is required", prefix))
	} else {
		v.validateFilters(source.Type, source.Filters, prefix)
	}

	// 验证output（根据数据源类型）
	v.validateOutput(source.Type, &source.Output, prefix)
}

// validateFilters 验证filters配置
func (v *Validator) validateFilters(sourceType string, filters []types.FilterConfig, prefix string) {
	for i, filter := range filters {
		filterPrefix := fmt.Sprintf("%s.filters[%d]", prefix, i)

		switch sourceType {
		case types.SourceTypeICodeDeepWiki:
			// iCode DeepWiki需要repo和branch
			if filter.Repo == "" {
				v.addError(fmt.Sprintf("%s.repo is required for %s", filterPrefix, sourceType))
			}
			if filter.Branch == "" {
				v.addError(fmt.Sprintf("%s.branch is required for %s", filterPrefix, sourceType))
			}

		case types.SourceTypeICodeRepo:
			// iCode仓库需要repo、branch和depth
			if filter.Repo == "" {
				v.addError(fmt.Sprintf("%s.repo is required for %s", filterPrefix, sourceType))
			}
			if filter.Branch == "" {
				v.addError(fmt.Sprintf("%s.branch is required for %s", filterPrefix, sourceType))
			}
			if filter.Depth == "" {
				v.addError(fmt.Sprintf("%s.depth is required for %s", filterPrefix, sourceType))
			}

		case types.SourceTypeICafeCard:
			// 如果指定了issueIds，其他条件可选；否则至少需要一个查询条件
			if len(filter.IssueIDs) == 0 {
				hasCondition := len(filter.Status) > 0 || len(filter.Types) > 0 ||
					filter.Keyword != "" || filter.TimeRange != "" || filter.Space != "" || len(filter.Owner) > 0
				if !hasCondition {
					v.addError(fmt.Sprintf("%s: must specify either issueIds or at least one query condition (status/types/keyword/timeRange/space)", filterPrefix))
				}
			}

		case types.SourceTypeIApiDoc:
			// iApi文档需要apiId或者projectId
			if filter.ProjectID == "" && len(filter.ApiIDs) == 0 {
				v.addError(fmt.Sprintf("%s: must specify either ProjectID or ApiIDs", filterPrefix))
			}

		case types.SourceTypeLocalKnowledge:
			// 本地知识需要srcPath和destPath
			if filter.SrcPath == "" {
				v.addError(fmt.Sprintf("%s.srcPath is required for %s", filterPrefix, sourceType))
			}
			if filter.DestPath == "" {
				v.addError(fmt.Sprintf("%s.destPath is required for %s", filterPrefix, sourceType))
			}

		case types.SourceTypeCustom:
			// 自定义知识需要knowledgeIds
			if len(filter.KnowledgeIDs) == 0 {
				v.addError(fmt.Sprintf("%s.knowledgeIds is required for %s", filterPrefix, sourceType))
			}

		case types.SourceTypeInfoflowGroupMessage:
			// 如流群消息需要groupId
			if filter.GroupID == "" {
				v.addError(fmt.Sprintf("%s.groupId is required for %s", filterPrefix, sourceType))
			} else {
				// 验证groupId格式(应该是数字)
				if _, err := strconv.ParseInt(filter.GroupID, 10, 64); err != nil {
					v.addError(fmt.Sprintf("%s.groupId must be a valid number: %s", filterPrefix, filter.GroupID))
				}
			}

			// 验证时间范围
			if filter.StartTimestamp > 0 && filter.EndTimestamp > 0 {
				if filter.StartTimestamp >= filter.EndTimestamp {
					v.addError(fmt.Sprintf("%s: startTime must be less than endTime", filterPrefix))
				}
			}

			// 验证maxMessages
			if filter.MaxMessages < 0 {
				v.addError(fmt.Sprintf("%s.maxMessages cannot be negative", filterPrefix))
			}
		}
	}
}

// validateOutput 验证输出配置（根据数据源类型）
func (v *Validator) validateOutput(sourceType string, output *types.OutputConfig, prefix string) {
	switch sourceType {
	case types.SourceTypeICafeCard:
		// 如果需要转换
		if output.NeedConvert {
			// 验证fields
			if len(output.Fields) == 0 {
				v.addError(fmt.Sprintf("%s.output.fields is required when needConvert is true", prefix))
			} else {
				// 验证每个field的配置
				for j, field := range output.Fields {
					if field.KeyPath == "" {
						v.addError(fmt.Sprintf("%s.output.fields[%d].keyPath is required", prefix, j))
					}
				}
			}
		}
		// 验证format
		if output.Format == "" {
			v.addError(fmt.Sprintf("%s.output.format is required", prefix))
		} else {
			if !contains(types.GetAllOutputFormats(), output.Format) {
				v.addError(fmt.Sprintf("%s.output.format '%s' is not supported (valid: %s)",
					prefix, output.Format, strings.Join(types.GetAllOutputFormats(), ", ")))
			}
		}

	case types.SourceTypeIApiDoc:
		// iApi导出需要format
		if output.ExportType != "" {
			if !contains(types.GetAllApiExportTypes(), output.ExportType) {
				v.addError(fmt.Sprintf("%s.output.exportType '%s' is not supported (valid: %s)",
					prefix, output.ExportType, strings.Join(types.GetAllApiExportTypes(), ", ")))
			}
		}

	case types.SourceTypeInfoflowGroupMessage:
		// 如流群消息只支持md格式
		if output.ExportType != "" && output.ExportType != "md" {
			v.addError(fmt.Sprintf("%s.output.exportType: only 'md' is supported for %s, got: %s",
				prefix, sourceType, output.ExportType))
		}
		// 不支持转换
		if output.NeedConvert {
			v.addError(fmt.Sprintf("%s: source type '%s' does not support needConvert=true",
				prefix, sourceType))
		}

	default:
		// 其他数据源类型不支持转换
		if output.NeedConvert {
			v.addError(fmt.Sprintf("%s: source type '%s' does not support needConvert=true, only %s supports data conversion",
				prefix, sourceType, types.SourceTypeICafeCard))
		}
	}
}

// validateStorage 验证存储配置
func (v *Validator) validateStorage(storage *types.StorageConfig, primaryRepo *types.PrimaryRepoConfig) {
	// 如果启用了primaryRepo且为local类型，basePath可以为空
	if primaryRepo != nil && primaryRepo.Enabled && primaryRepo.Type == "local" {
		return
	}

	// 其他情况下basePath必填
	if storage.BasePath == "" {
		v.addError("storage.basePath is required")
	}
}

// validatePrimaryRepo 验证主代码库配置
func (v *Validator) validatePrimaryRepo(primaryRepo *types.PrimaryRepoConfig, storage *types.StorageConfig) {
	// 如果没有配置或未启用，跳过验证
	if primaryRepo == nil || !primaryRepo.Enabled {
		return
	}

	// 验证type字段
	if primaryRepo.Type == "" {
		v.addError("primaryRepo.type is required when primaryRepo is enabled")
		return
	}

	// type只能是local或remote
	if primaryRepo.Type != "local" && primaryRepo.Type != "remote" {
		v.addError("primaryRepo.type must be 'local' or 'remote'")
		return
	}

	// 根据type验证不同的字段
	switch primaryRepo.Type {
	case "local":
		// local类型必须有path
		if primaryRepo.Path == "" {
			v.addError("primaryRepo.path is required when type is 'local'")
		}

	case "remote":
		// remote类型必须有repo和branch
		if primaryRepo.Repo == "" {
			v.addError("primaryRepo.repo is required when type is 'remote'")
		}
		if primaryRepo.Branch == "" {
			v.addError("primaryRepo.branch is required when type is 'remote'")
		}
		// remote类型必须有basePath作为克隆工作目录
		if storage.BasePath == "" {
			v.addError("storage.basePath is required when primaryRepo.type is 'remote'")
		}
	}

	// knowledgeSubDir有默认值，不验证
}

// validateKnowledgeSkill 验证knowledgeSkill配置
func (v *Validator) validateKnowledgeSkill(skillConfig *types.KnowledgeSkillConfig) {
	// knowledgeSkill为可选配置,如果为nil直接返回
	if skillConfig == nil {
		return
	}

	// 如果启用了skill,设置默认值
	if skillConfig.Enable {
		// title如果为空,设置默认值
		if skillConfig.Title == "" {
			skillConfig.Title = types.DefaultSkillName
		}
	}
}

// addError 添加错误
func (v *Validator) addError(msg string) {
	v.errors = append(v.errors, msg)
}

// contains 检查字符串是否在切片中
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
