// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package types

// KnowledgeConfig 知识配置文件的顶层结构
type KnowledgeConfig struct {
	Version        string                `yaml:"version" json:"version"`
	Meta           MetaConfig            `yaml:"meta" json:"meta"`
	PrimaryRepo    *PrimaryRepoConfig    `yaml:"primaryRepo,omitempty" json:"primaryRepo,omitempty"`
	Sources        []SourceConfig        `yaml:"sources" json:"sources"`
	Storage        StorageConfig         `yaml:"storage" json:"storage"`
	Entrypoint     EntrypointConfig      `yaml:"entrypoint" json:"entrypoint"`
	KnowledgeSkill *KnowledgeSkillConfig `yaml:"knowledgeSkill,omitempty" json:"knowledgeSkill,omitempty"`
}

// MetaConfig 元信息配置
type MetaConfig struct {
	ProjectID   string `yaml:"projectId" json:"projectId"`
	Version     string `yaml:"version" json:"version"`
	Description string `yaml:"description" json:"description"`
	Owner       string `yaml:"owner" json:"owner"`
}

// PrimaryRepoConfig 主代码库配置
type PrimaryRepoConfig struct {
	Enabled         bool   `yaml:"enabled" json:"enabled"`                                     // 是否启用集成模式
	Type            string `yaml:"type" json:"type"`                                           // local(本地已有) 或 remote(即时拉取)
	Path            string `yaml:"path,omitempty" json:"path,omitempty"`                       // 本地路径(type=local时必填)
	Repo            string `yaml:"repo,omitempty" json:"repo,omitempty"`                       // 远程仓库地址(type=remote时必填)
	Branch          string `yaml:"branch,omitempty" json:"branch,omitempty"`                   // 分支名(type=remote时必填)
	Depth           string `yaml:"depth,omitempty" json:"depth,omitempty"`                     // 克隆深度(type=remote时,默认1)
	KnowledgeSubDir string `yaml:"knowledgeSubDir,omitempty" json:"knowledgeSubDir,omitempty"` // 知识存储子目录,默认.knowledge
}

// SourceConfig 知识源配置
type SourceConfig struct {
	Type    string         `yaml:"type" json:"type"`
	Enabled bool           `yaml:"enabled" json:"enabled"`
	Filters []FilterConfig `yaml:"filters" json:"filters"`
	Output  OutputConfig   `yaml:"output" json:"output"`
	Usage   UsageConfig    `yaml:"usage" json:"usage"`
}

// FilterConfig 过滤条件配置(不同类型的知识源有不同的字段)
type FilterConfig struct {
	// iCode相关
	Repo     string `yaml:"repo,omitempty" json:"repo,omitempty"`
	Branch   string `yaml:"branch,omitempty" json:"branch,omitempty"`
	CommitID string `yaml:"commitId,omitempty" json:"commitId,omitempty"`
	Depth    string `yaml:"depth,omitempty" json:"depth,omitempty"`

	// GitHub相关
	RepoHttpUrl string `yaml:"repoHttpUrl,omitempty" json:"repoHttpUrl,omitempty"`

	// iCafe相关
	Space         string   `yaml:"space,omitempty" json:"space,omitempty"`
	Owner         []string `yaml:"owner,omitempty" json:"owner,omitempty"`
	Status        []string `yaml:"status,omitempty" json:"status,omitempty"`
	Types         []string `yaml:"types,omitempty" json:"types,omitempty"`
	Keyword       string   `yaml:"keyword,omitempty" json:"keyword,omitempty"`
	TimeRange     string   `yaml:"timeRange,omitempty" json:"timeRange,omitempty"`
	StartTimeDate string   `yaml:"startTimeDate,omitempty" json:"startTimeDate,omitempty"` // 开始时间(YYYY-MM-DD)
	EndTimeDate   string   `yaml:"endTimeDate,omitempty" json:"endTimeDate,omitempty"`     // 结束时间(YYYY-MM-DD)
	IssueIDs      []string `yaml:"issueIds,omitempty" json:"issueIds,omitempty"`

	// iApi相关
	ProjectID string   `yaml:"projectId,omitempty" json:"projectId,omitempty"`
	ApiIDs    []string `yaml:"apiId,omitempty" json:"apiId,omitempty"`

	// ku_doc相关
	RepositoryGuid string   `yaml:"repositoryGuid,omitempty" json:"repositoryGuid,omitempty"` // 知识库ID
	ParentDocGuid  string   `yaml:"parentDocGuid,omitempty" json:"parentDocGuid,omitempty"`   // 父文档ID
	Urls           []string `yaml:"urls,omitempty" json:"urls,omitempty"`                     // 文档URL列表；不为空时忽略 repositoryGuid/parentDocGuid，同时 users 过滤也不生效
	Users          []string `yaml:"users,omitempty" json:"users,omitempty"`                   // 按用户名过滤，只拉取创建者或发布者在列表中的文档；为空时不过滤；仅在 repositoryGuid/parentDocGuid 模式下生效

	// local_knowledge相关
	SrcPath  string `yaml:"srcPath,omitempty" json:"srcPath,omitempty"`
	DestPath string `yaml:"destPath,omitempty" json:"destPath,omitempty"`
	Editable bool   `yaml:"editable,omitempty" json:"editable,omitempty"`

	// infoflow_group_message相关
	GroupID        string `yaml:"groupId,omitempty" json:"groupId,omitempty"`
	StartTimestamp int64  `yaml:"startTimeStamp,omitempty" json:"startTimeStamp,omitempty"`
	EndTimestamp   int64  `yaml:"endTimeStamp,omitempty" json:"endTimeStamp,omitempty"`
	MaxMessages    int    `yaml:"maxMessages,omitempty" json:"maxMessages,omitempty"`

	// custom相关
	KnowledgeIDs []string `yaml:"knowledgeIds,omitempty" json:"knowledgeIds,omitempty"`

	// knowledge_book、idataset相关
	KnowledgebookUuid string   `yaml:"knowledgebookUuid,omitempty" json:"knowledgebookUuid,omitempty"`
	TagNames          []string `yaml:"tagNames,omitempty" json:"tagNames,omitempty"`
	DirectoryNames    []string `yaml:"directoryNames,omitempty" json:"directoryNames,omitempty"`
	KnowledgeUuids    []string `yaml:"knowledgeUuids,omitempty" json:"knowledgeUuids,omitempty"`
	KnowledgeType     string   `yaml:"knowledgeType,omitempty" json:"knowledgeType,omitempty"` // 知识类型(readme等)

	// idataset相关
	CommitTimeStart string `yaml:"commitTimeStart,omitempty" json:"commitTimeStart,omitempty"` // 提交时间起点(北京时间)
	CommitTimeEnd   string `yaml:"commitTimeEnd,omitempty" json:"commitTimeEnd,omitempty"`     // 提交时间终点(北京时间)
}

// OutputConfig 输出配置
type OutputConfig struct {
	NeedConvert bool          `yaml:"needConvert" json:"needConvert"`
	Fields      []FieldConfig `yaml:"fields,omitempty" json:"fields,omitempty"`
	Format      string        `yaml:"format,omitempty" json:"format,omitempty"`         // md, json, xml
	ExportType  string        `yaml:"exportType,omitempty" json:"exportType,omitempty"` // md, swagger (for iapi_doc)
}

// FieldConfig 字段配置
type FieldConfig struct {
	KeyPath     string `yaml:"keyPath" json:"keyPath"`
	Description string `yaml:"description" json:"description"`
}

// UsageConfig 使用说明配置
type UsageConfig struct {
	Type    string `yaml:"type" json:"type"` // rule, skill
	Content string `yaml:"content" json:"content"`
	Path    string `yaml:"path,omitempty" json:"path,omitempty"`
}

// StorageConfig 存储配置
type StorageConfig struct {
	BasePath string `yaml:"basePath" json:"basePath"`
}

// EntrypointConfig 入口配置
type EntrypointConfig struct {
	Content string `yaml:"content" json:"content"`
	Path    string `yaml:"path,omitempty" json:"path,omitempty"`
}

// KnowledgeSkillConfig skill自动生成配置
type KnowledgeSkillConfig struct {
	Enable  bool   `yaml:"enable" json:"enable"`
	Title   string `yaml:"title" json:"title"`
	Content string `yaml:"content" json:"content"`
}
