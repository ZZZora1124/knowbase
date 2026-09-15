// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package constants

// 知识类型常量
const (
	KnowledgeTypeContent   = "content"    // 短文本
	KnowledgeTypeReadme    = "readme"     // README文本（与content相同逻辑）
	KnowledgeTypeTextFile  = "text_file"  // 文本文件
	KnowledgeTypeAudioFile = "audio_file" // 音频文件
	KnowledgeTypeImageFile = "image_file" // 图片文件
	KnowledgeTypeVideoFile = "video_file" // 视频文件
	KnowledgeTypeConfig    = "config"     // 配置类型（非自定义知识源使用）
)

// 知识源类型常量
const (
	SourceTypeICodeDeepWiki = "icode_deepwiki" // deepwiki
	SourceTypeICodeRepo     = "icode_repo"     // 代码库
	SourceTypeICafeCard     = "icafe_card"     // 需求卡片
	SourceTypeInfoFlowGroup = "infoflow_group" // 如流群消息
	SourceTypeIApiDoc       = "iapi_doc"       // iapi接口文档
	SourceTypeGithubRepo    = "github_repo"    // github代码库
	SourceTypeKuDoc         = "ku_doc"         // 知识库文档
	SourceTypeKnowledgeBook = "knowledge_book" // 知识方舟知识本
	SourceTypeCustom        = "custom"         // 自定义
	SourceTypeBrain         = "ark_brain"      // brain put_page 写入来源
)

// OpenAPI README 追加常量
const (
	OpenAPIReadmeAppendLimitCustom = 10  // OpenAPI 场景下自定义知识追加限制
	OpenAPIReadmeAppendLimitSystem = 100 // OpenAPI 场景下系统知识追加限制
)

// 知识本成员类型常量
const (
	KnowledgebaseMemberTypeUsername   = "username"    // 用户名
	KnowledgebaseMemberTypeEmailGroup = "email_group" // 邮件组
	KnowledgebaseMemberTypeDepartment = "department"  // 部门
	KnowledgebaseMemberTypeChatGroup  = "chat_group"  // 群（ 如流群）
)

// 知识本成员角色常量
const (
	KnowledgebaseMemberRoleAdmin  = "admin"  // 管理员
	KnowledgebaseMemberRoleEditor = "editor" // 编辑者
	KnowledgebaseMemberRoleViewer = "viewer" // 查看者
)

// 知识本操作类型常量
const (
	OperationCreateKB         = "create_kb"
	OperationUpdateKB         = "update_kb"
	OperationDeleteKB         = "delete_kb"
	OperationCreateDir        = "create_dir"
	OperationUpdateDir        = "update_dir"
	OperationDeleteDir        = "delete_dir"
	OperationMoveDir          = "move_dir"
	OperationCreateKnowledge  = "create_knowledge"
	OperationUpdateKnowledge  = "update_knowledge"
	OperationDeleteKnowledge  = "delete_knowledge"
	OperationMoveKnowledge    = "move_knowledge"
	OperationAddMember        = "add_member"
	OperationRemoveMember     = "remove_member"
	OperationUpdateMemberRole = "update_member_role"
	OperationCreateTag        = "create_tag"
	OperationUpdateTag        = "update_tag"
	OperationDeleteTag        = "delete_tag"
	OperationBindTag          = "bind_tag"
	OperationUnbindTag        = "unbind_tag"
)

// 知识本状态常量
const (
	KnowledgebaseStatusDisabled = 0 // 禁用
	KnowledgebaseStatusEnabled  = 1 // 启用
)

// 可见性常量
const (
	VisibilityPrivate  = "private"  // 私密 - 只有成员可见
	VisibilityPublic   = "public"   // 公开 - 所有人可见
	VisibilityPersonal = "personal" // 个人 - 只有owner可见
)

// 资源类型常量
const (
	ResourceTypeKnowledge     = "knowledge"     // 知识资源
	ResourceTypeKnowledgebase = "knowledgebase" // 知识本资源
	ResourceTypeDirectory     = "directory"     // 目录资源
	ResourceTypeMember        = "member"        // 知识本成员资源
	ResourceTypeTag           = "tag"           // 知识本标签资源
)

// 虚拟默认目录常量（用于表示未归属任何自定义目录的知识集合）
const (
	DirectoryRootUUID = "/"         // 虚拟根目录 UUID
	DirectoryRootName = "默认目录"      // 虚拟根目录显示名称
	DirectoryMaxDepth = 5           // 目录最大层级深度
	ArkBrainRootDir   = "ark-brain" // brain put_page 根目录名称
)

// 环境变量常量
const (
	EnvConfigPath = "CONFIG_PATH" // 配置文件路径环境变量
	EnvPort       = "PORT"        // 服务端口环境变量
)

// 认证头常量
const (
	HeaderAuthClientKey = "x-auth-client-token"
	HeaderAuthACKey     = "x-ac-Authorization"
	// OpenAPI认证头
	HeaderOpenAPIClientToken = "X-Auth-Client-Token"
)

// 缓存key常量
const (
	CacheKeyACUserPlat = "ac:userplat:"
	// 用户角色缓存前缀: user_role:{kb_uuid}:{username}
	CacheKeyUserRole = "user_role:"
	// 知识本UUID缓存前缀: kb:uuid:{uuid}
	CacheKeyKBUUID = "kb:uuid:"
	// 知识本名称缓存前缀: kb:name:{name}
	CacheKeyKBName = "kb:name:"
	// 知识本列表缓存前缀: kb:list:{hash}
	CacheKeyKBList = "kb:list:"
	// 知识本成员数量缓存前缀: kb:member_count:{kb_uuid}
	CacheKeyKBMemberCount = "kb:member_count:"
	// 标签列表缓存前缀: kb:tag_list:{kb_uuid}
	CacheKeyTagList = "kb:tag_list:"
	// 标签数量缓存前缀: kb:tag_count:{kb_uuid}
	CacheKeyTagCount = "kb:tag_count:"
	// 知识数量缓存前缀: kb:knowledge_count:{kb_uuid}
	CacheKeyKnowledgeCount = "kb:knowledge_count:"
	// UUAP缓存前缀
	CacheKeyUUAPReult = "uuap:"
	// 如流缓存前缀
	CacheKeyInfoFlowResult = "infoflow:"
	// OpenAPI客户端缓存前缀: openapi:client:{token}
	CacheKeyOpenAPIClient = "openapi:client:"
	// CacheKeyBrainDefaultModel Brain 模块全局默认模型 redis key（对应客户端 --model 参数）
	CacheKeyBrainDefaultModel = "brain:default_model"
	// CacheKeyGroupFailStreak 如流群成员刷新连续失败次数前缀: infoflow:group_fail_streak:{groupId}
	CacheKeyGroupFailStreak = "infoflow:group_fail_streak:"
	// CacheKeyGroupMembersCache 如流群成员缓存前缀: infoflow:group_members:{groupId}
	CacheKeyGroupMembersCache = "infoflow:group_members:"
	// CacheKeyUserMatchedGroups 用户匹配的群聊缓存前缀: infoflow:user_matched_groups:{uid}
	CacheKeyUserMatchedGroups = "infoflow:user_matched_groups:"
	// CacheKeyGroupCacheRefreshLock 群成员缓存刷新任务分布式锁 key
	CacheKeyGroupCacheRefreshLock = "scheduler:group_cache_refresh:lock"
)

// 如流事件类型常量
const (
	InfoFlowEventGroupMemberAdd = "GROUP_MEMBER_ADD" // 群成员新增
	InfoFlowEventGroupMemberDel = "GROUP_MEMBER_DEL" // 群成员删除
)

// Brain 模块相关常量
const (
	// BrainDefaultModelFallback redis 未配置或不可达时的默认模型兜底值
	BrainDefaultModelFallback = "GLM-5.1"
)

// 鉴权相关常量
const (
	AuthUsername      = "username"        // 认证用户名
	AuthPlatform      = "platform"        // 认证平台
	AuthType          = "auth_type"       // 认证类型
	AuthClientName    = "client_name"     // 客户端名称
	AuthClientUUID    = "client_uuid"     // 客户端UUID
	AuthClientIsAdmin = "client_is_admin" // 客户端Admin权限标识
)

// 认证类型常量
const (
	AuthTypeJWT     = "jwt"     // JWT认证
	AuthTypeHeader  = "header"  // Header认证
	AuthTypeOpenAPI = "openapi" // OpenAPI认证
	AuthTypeCloud   = "cloud"   // 云上百度认证
)
