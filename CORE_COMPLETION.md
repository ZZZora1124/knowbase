# 五类核心业务补全说明

本次仅补全已确认的五类内容，目标是连接现有业务调用链，便于阅读架构。新增代码依据现有调用方、仓储接口和 SQL 重建，**不是从原始 TXT 找回的源码**。

## 补全范围与阅读入口

| 类别 | 新增文件（相对于对应项目目录） | 业务作用 |
| --- | --- | --- |
| 知识版本管理 | 服务端 `internal/service/knowledge_version_service.go`、`version_helper.go`；`internal/domain/knowledge_version.go`；`internal/api/request/knowledge_version_request.go`；`internal/api/response/knowledge_version_response.go` | 接通已有版本 Handler 和 Repository，支持历史列表、指定版本、批量查询及回滚 |
| 操作审计 | 服务端 `internal/domain/knowledgebase_operation_log.go`；`internal/repository/knowledgebase_operation_log_repository.go`；`internal/api/response/operation_log_response.go` | 连接现有业务写入和日志查询，支持事务写入、条件过滤、分页 |
| 标签与成员 | 服务端 `internal/domain/knowledgebase_tag.go`；`internal/api/response/tag_response.go`、`knowledgebase_member_response.go` | 补齐已有标签、成员服务使用的数据结构，不另写一套业务 |
| 业务错误 | 服务端 `pkg/errors/errors.go` | 补齐现有调用使用的错误码、预定义错误、包装及错误链；同步补充 `pkg/response/response.go` 的 HTTP 状态映射 |
| 客户端 CLI 入口 | 客户端 `cmd/root.go` | 连接 `main → cmd.Execute`、已有 notebook/brain 子命令，以及配置加载主流程 |

共新增 **13 个 Go 文件**，调整 **1 个现有 Go 文件**。数据库模型沿用 `001_initial_schema.sql` 和 `009_knowledge_version_and_apply.sql`，未新增表或迁移。

## 核心流程

### 知识版本

阅读顺序：[Handler](knowbase-server/internal/api/handler/knowledge_version_handler.go) → [Service](knowbase-server/internal/service/knowledge_version_service.go) → [Repository](knowbase-server/internal/repository/knowledge_version_repository.go)。

- 查询先检查知识、所属知识本是否存在，再复用现有权限服务检查可读权限。
- 批量查询限制为 1–100 项，每条知识都校验权限；任一知识或快照不存在、无权限时，整批返回错误。
- 回滚先检查编辑权限和同目录标题冲突，再把历史快照中的标题、内容、文件路径/大小/格式及元数据写回知识，生成一个新版本。
- 知识更新与新快照创建使用同一事务。保留所有历史版本，不删除 BOS 文件；知识类型、模板、目录、标签等没有进入快照的字段保持当前值。
- 回滚行为记录在新快照的 `operation=rollback`、操作者和备注中；本次未增加独立的回滚审计日志写入。
- 批量详情按请求顺序返回。`knowledge` 是当前知识基本信息，`version` 是所选历史快照，标签和目录名称取当前关系。

### 操作审计、标签和成员

现有知识、目录、标签、成员等服务已包含业务规则。本次补上它们依赖的模型、返回结构和审计仓储。审计查询按时间及 ID 倒序分页；写入沿用调用方传入的事务。

### 客户端入口

[cmd/root.go](knowbase-client/cmd/root.go) 注册已有 `notebook`、`brain` 命令。根命令执行链为：初始化环境 → 解析和校验配置 → 获取已有认证凭据 → 初始化本地存储 → `Manager.LoadAll`。

配置通过 `-c/--config` 指定，默认 `~/.knowledge/config.yaml`。环境读取 `KNOWBASE_ENV`，为空时使用 `prod`。帮助命令不触发配置加载或认证；子命令继续使用自身已有的认证流程。

## 原文缺失时采用的约定

以下内容无法证明与原版完全相同，后续拿到原始源码或接口约定时应核对：

1. 默认版本 `v1` 有 SQL 依据；自动递增规则为末尾数字加一，例如 `v1 → v2`、`v1.0.9 → v1.0.10`；无数字后缀则追加 `.1`。日期样式作为普通标签，不做日期计算。
2. 版本号允许 1–50 个 ASCII 字符，首字符为字母或数字，其余允许字母、数字、`.`、`_`、`-`。生成结果过长或与已有版本冲突时不覆盖历史；本次回滚会显式返回错误。
3. 回滚请求体为 `{"version":"v1","remark":"说明"}`；知识 UUID 只取路由。批量历史请求体为 `{"knowledgeUuids":["知识UUID"]}`。
4. 批量详情请求体为 `{"items":[{"knowledgeUuid":"知识UUID","version":"v1"}]}`；省略版本时读取主记录的 `current_version` 对应快照。缺失快照时返回错误，不伪造快照。
5. 业务错误名称和调用签名与现有代码对应；**错误码数值是重建约定**，不能据此宣称兼容原服务的数字协议。

## 核验与边界

- 对本次全部 14 个 Go 文件执行 `gofmt`，完成语法解析和格式检查。
- 对业务错误包、版本辅助函数、新增模型和请求/响应类型进行局部编译检查。
- 静态核对服务端业务错误引用与相关结构体字面量字段是否有定义。
- 在临时目录检查版本号边界、大数字递增、错误链和路由 UUID 不被请求体覆盖；检查通过，未向项目加入测试文件。
- 未新增项目测试文件，未启动服务、连接数据库或执行迁移。版本服务的数据库事务、权限查询和客户端外部认证未做运行验证。
- 保留此前明确排除的缺口：如流集成及调度器、多源加载器、冷启动脚本、下载统计等。整个项目仍不具备完整编译和运行条件，本次不以此为验收目标。

原 [TXT 复原报告](knowbase-server/RESTORATION_REPORT.md) 作为历史记录保留；其中的缺失清单应结合本说明阅读。
