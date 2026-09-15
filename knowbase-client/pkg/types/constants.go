// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package types

import "fmt"

// 知识源类型常量
const (
	SourceTypeICodeDeepWiki        = "icode_deepwiki"
	SourceTypeICodeRepo            = "icode_repo"
	SourceTypeICafeCard            = "icafe_card"
	SourceTypeLocalKnowledge       = "local_knowledge"
	SourceTypeCustom               = "custom"
	SourceTypeIApiDoc              = "iapi_doc"
	SourceTypeGithubRepo           = "github_repo"
	SourceTypeKuDoc                = "ku_doc"
	SourceTypeInfoflowGroupMessage = "infoflow_group_message"
	SourceTypeICodeCRDiff          = "icode_cr_diff"
	SourceTypeKnowledgeBook        = "knowledge_book"
	SourceTypeIDataset             = "idataset"
)

// 输出格式常量
const (
	OutputFormatMarkdown = "md"
	OutputFormatJSON     = "json"
	OutputFormatXML      = "xml"
)

// iApi导出类型常量
const (
	ExportTypeMd      = "md"
	ExportTypeSwagger = "swagger"
)

// 文件后缀常量
const (
	FileSuffixMarkdown = ".md"
	FileSuffixJSON     = ".json"
	FileSuffixXML      = ".xml"
)

// 使用说明类型常量
const (
	UsageTypeRule  = "rule"
	UsageTypeSkill = "skill"
)

// 环境变量常量
const (
	EnvComateAuthToken       = "COMATE_AUTH_TOKEN"
	EnvDisableKnowbaseUpdate = "DISABLE_KNOWBASE_UPDATE"
	EnvKnowbaseEnv           = "KNOWBASE_ENV"
	EnvKnowbaseBrainUUID     = "KNOWBASE_BRAIN_UUID"
)

// 退出码
const (
	ExitSuccess        int = 0 // success
	ExitUpdateFailed   int = 1 // fail to update client
	ExitRunSubCMDError int = 3 // run sub comamnd error
)

// 知识源调用路径
const (
	AuthHost            = "http://10.11.152.208:8701"
	AuthClientKey       = "uwly7TYhHuPQd2Ts"
	AuthClientValue     = "6D6d1D0WF8y19uWXiBfR1oC24ED05dwB"
	AuthHostTest        = "http://10.11.157.225:8061"
	AuthClientKeyTest   = "V8JIvRV2IcaaofaP"
	AuthClientValueTest = "Llb16sJy8cDNqA1LTZPwW6VQmCcZtNjD"

	AuthenticationAPIPath = "/api/rest/v1/auth/validate"

	IApiHost     = "http://10.11.57.61:8100"
	IApiHostTest = "http://code-apidoc.test.bapi.appspace.baidu.com"

	ICodeHost     = "icode.baidu.com:8235"
	ICodeHostTest = "icodeqa.baidu.com:8235"

	KnowbaseHost     = "http://10.11.153.100:8101"
	KnowbaseHostTest = "http://10.11.157.225:8081"

	PathIcafeCard             = "/api/process/icafe/api"
	PathIcodeWiki             = "/api/process/icode/rest/ai/wiki/api/wikiManagement/downloadWikiZip"
	PathIApiList              = "/api/v1/iapi/api/list"
	PathIApiDetail            = "/api/v1/iapi/api/detail"
	PathKuDocPageByRepo       = "/api/process/ku/ku/openapi/queryRepo"
	PathKuDocQueryContent     = "/api/process/ku/ku/openapi/queryContent"
	UgateKuOpenAPIPrefix      = "/api/process/ku/ku/openapi/"
	UgateKuProxyPrefix        = "/wiki/so/ku/openapi/"
	UgateKuAccessKeyHeader    = "access-key"
	UgateKuAccessKey          = "EAmLKTO1X6AUJalpK7+LrQ=="
	PathInfoflowGroupMessages = "/api/v1/infoflow/group/messages"
	PathMetricsDownload       = "/api/v1/metrics/download"

	// KnowledgeBook 相关路径
	PathKnowledgeBookList   = "/open-api/v1/knowledgebases/%s/knowledge"
	PathKnowledgeBookByTags = "/open-api/v1/knowledgebases/%s/knowledge/by-tags"
	PathKnowledgeBatch      = "/open-api/v1/knowledge/batch"

	// Notebook 管理相关路径
	PathNotebookCreate    = "/open-api/v1/knowledgebases"
	PathNotebookList      = "/open-api/v1/knowledgebases/my"
	PathNotebookGetByName = "/open-api/v1/knowledgebases/name/%s"
	PathNotebookGetByUUID = "/open-api/v1/knowledgebases/%s"
	PathNotebookUpdate    = "/open-api/v1/knowledgebases/%s"

	// Notebook 成员管理路径
	PathNotebookMemberAdd        = "/open-api/v1/knowledgebases/%s/members"
	PathNotebookMemberList       = "/open-api/v1/knowledgebases/%s/members"
	PathNotebookMemberUpdateRole = "/open-api/v1/knowledgebases/%s/members/%s"

	// Notebook 标签管理路径
	PathNotebookTagCreate = "/open-api/v1/knowledgebases/%s/tags"
	PathNotebookTagList   = "/open-api/v1/knowledgebases/%s/tags"
	PathNotebookTagUpdate = "/open-api/v1/knowledgebases/%s/tags/%s"

	// Notebook 目录管理路径
	PathNotebookDirCreate = "/open-api/v1/knowledgebases/%s/directories"
	PathNotebookDirList   = "/open-api/v1/knowledgebases/%s/directories"
	PathNotebookDirGet    = "/open-api/v1/knowledgebases/%s/directories/%s"
	PathNotebookDirUpdate = "/open-api/v1/knowledgebases/%s/directories/%s"
	PathNotebookDirDelete = "/open-api/v1/knowledgebases/%s/directories/%s"
	PathNotebookDirTree   = "/open-api/v1/knowledgebases/%s/directories/tree"
	PathNotebookDirMove   = "/open-api/v1/knowledgebases/%s/directories/%s/move"

	// Notebook 知识管理路径
	PathNotebookKnowledgeCreate     = "/open-api/v1/knowledgebases/%s/knowledge"
	PathNotebookKnowledgeList       = "/open-api/v1/knowledgebases/%s/knowledge"
	PathNotebookKnowledgeByTags     = "/open-api/v1/knowledgebases/%s/knowledge/by-tags"
	PathNotebookKnowledgeGet        = "/open-api/v1/knowledge/%s"
	PathNotebookKnowledgeUpdate     = "/open-api/v1/knowledge/%s"
	PathNotebookKnowledgeMove       = "/open-api/v1/knowledge/%s/move"
	PathNotebookKnowledgeBindTags   = "/open-api/v1/knowledge/%s/tags"
	PathNotebookKnowledgeUnbindTags = "/open-api/v1/knowledge/%s/tags"
	PathNotebookKnowledgeGetTags    = "/open-api/v1/knowledge/%s/tags"
	PathNotebookKnowledgeUpdateFile = "/open-api/v1/knowledge/%s/file"

	// 文件上传大小限制
	MaxFileUploadSize = 500 * 1024 * 1024 // 500MB

	// iCode CR Diff 相关路径
	PathICodeChangeReviewInfo   = "/api/process/icode/rest/review/api/change/reviewinfo/get"
	PathICodePatchBaseRevision  = "/api/process/icode/rest/review/api/change/patch/base/revision"
	PathICodeCompareFilesDetail = "/api/process/icode/rest/git/api/compare/files/detail"
	PathICodeDiffByCommit       = "/api/process/icode/rest/review/api/diff/getDiffByCommit"

	// iDataset 相关配置
	IDatasetHost         = "http://icode.baidu.com"
	PathIDatasetTree     = "/files/api/tree/getTree"
	PathIDatasetDownload = "/files/api/v2/blob/download/fileraw"
	IDatasetClientID     = "b7dc6ff16c68799c"
	IDatasetClientSecret = "08e088f4c448f8fdd59578b27b0ea137"

	KuJsonToMdHost = "http://10.11.153.100:8001"
	PathKuJsonToMd = "/jsonToMd"
)

// Brain PG 接口路径
const (
	PathBrainPage      = "/open-api/v1/brain/page"
	PathBrainSearch    = "/open-api/v1/brain/search"
	PathBrainQuery     = "/open-api/v1/brain/query"
	PathBrainGraph     = "/open-api/v1/brain/graph/%s"
	PathBrainTimeline  = "/open-api/v1/brain/timeline/%s"
	PathBrainBacklinks = "/open-api/v1/brain/backlinks/%s"

	// PathBrainConfigDefaultModel 服务端下发的默认模型（供 notify 等子进程场景使用）
	PathBrainConfigDefaultModel = "/open-api/v1/brain/config/default-model"
)

// DefaultBaseDir 默认基础目录常量
const (
	DefaultBaseDir = "~/knowledge"
)

// DefaultSignalDetectorAllowedTools 限定 signal-detector 子进程可用的工具集（默认值，可通过 --allowed-tools 覆盖）
// Bash(KNOWBASE_BRAIN_UUID=* knowbase brain *) 作为兜底，防止 agent 仍然添加环境变量前缀
// Bash(echo *) 允许 agent 构造临时内容写入文件后再通过 --file 参数写入
// Bash(cat *) 允许 agent 在 bash 中查看临时文件内容以校验写入结果
// Bash(which*)/Bash(env*)/Bash(printenv*)/Bash(ls*) 放开常用诊断命令
const DefaultSignalDetectorAllowedTools = "Read,Grep,Glob,Write,Edit," +
	"Bash(knowbase*),Bash(KNOWBASE_BRAIN_UUID=* knowbase brain *)," +
	"Bash(echo *),Bash(cat *)," +
	"Bash(which*),Bash(env*),Bash(printenv*),Bash(ls*)"

// GetAllSourceTypes 获取所有支持的知识源类型
func GetAllSourceTypes() []string {
	return []string{
		SourceTypeICodeDeepWiki,
		SourceTypeICodeRepo,
		SourceTypeICafeCard,
		SourceTypeLocalKnowledge,
		SourceTypeCustom,
		SourceTypeIApiDoc,
		SourceTypeGithubRepo,
		SourceTypeKuDoc,
		SourceTypeInfoflowGroupMessage,
		SourceTypeICodeCRDiff,
		SourceTypeKnowledgeBook,
		SourceTypeIDataset,
	}
}

// GetAllOutputFormats 获取所有支持的输出格式
func GetAllOutputFormats() []string {
	return []string{
		OutputFormatMarkdown,
		OutputFormatJSON,
		OutputFormatXML,
	}
}

// GetAllApiExportTypes 获取所有支持的iApi导出类型
func GetAllApiExportTypes() []string {
	return []string{
		ExportTypeMd,
		ExportTypeSwagger,
	}
}

// 环境配置管理
var currentEnv = "prod" // 默认为生产环境

// InitEnv 初始化环境配置
func InitEnv(env string) error {
	if env != "prod" && env != "test" {
		return fmt.Errorf("invalid environment: %s, must be 'prod' or 'test'", env)
	}
	currentEnv = env
	return nil
}

// GetCurrentEnv 获取当前环境
func GetCurrentEnv() string {
	return currentEnv
}

// GetAuthHost 获取认证服务地址
func GetAuthHost() string {
	if currentEnv == "test" {
		return AuthHostTest
	}
	return AuthHost
}

// GetAuthClientKey 获取认证客户端Key
func GetAuthClientKey() string {
	if currentEnv == "test" {
		return AuthClientKeyTest
	}
	return AuthClientKey
}

// GetAuthClientValue 获取认证客户端Value
func GetAuthClientValue() string {
	if currentEnv == "test" {
		return AuthClientValueTest
	}
	return AuthClientValue
}

// GetIApiHost 获取iApi服务地址
func GetIApiHost() string {
	if currentEnv == "test" {
		return IApiHostTest
	}
	return IApiHost
}

// GetICodeHost 获取iCode服务地址
func GetICodeHost() string {
	if currentEnv == "test" {
		return ICodeHostTest
	}
	return ICodeHost
}

// GetKuHost 获取知识库服务地址
func GetKuHost() string {
	if currentEnv == "test" {
		return AuthHostTest
	}
	return AuthHost
}

// GetKnowbaseHost 获取如流服务地址
func GetKnowbaseHost() string {
	if currentEnv == "test" {
		return KnowbaseHostTest
	}
	return KnowbaseHost
}

// ChatMode 取值：knowbase brain notify --chat-mode flag 与 NotifyConfig.ChatMode 字段的合法值。
const (
	ChatModePersonal = "personal"
	ChatModeGroup    = "group"
)
