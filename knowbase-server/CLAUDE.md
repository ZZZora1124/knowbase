# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 项目概述

知识方舟平台 (Knowbase Server) - 一个围绕 DevOps 研发流程相关知识的管理和分发平台后端服务，支持知识的收集、存储、分析和推荐。

**模块路径**: `icode.baidu.com/baidu/devops-ai/knowbase-server`

## 构建和测试命令

```bash
# 构建项目
make build
# 或
make compile

# 运行测试（包含覆盖率）
make test

# 完整构建（准备 + 编译 + 打包）
make all

# 清理构建产物
make clean

# 下载依赖
go mod download

# 运行服务
./knowbase-server
```

## 架构设计

本项目采用**五层架构**，严格分离关注点：

### 分层结构

```
API 层 (Gin) → 服务层 → 领域/处理器层 → 仓储层 → 基础设施层
```

1. **API 层** (`internal/api/`)
   - Handlers: 请求/响应处理
   - Middleware: 认证(JWT)、权限(RBAC)、日志、限流、跨域、恢复
   - Router: 路由配置

2. **服务层** (`internal/service/`)
   - KnowledgeService、StorageService、TagService、PermissionService
   - 包含业务逻辑，协调 handlers 和 repositories

3. **领域/处理器层** (`internal/domain/`)
   - **可插拔知识处理器模式**: 使用工厂模式实现可扩展的知识类型处理
   - 模型: Knowledge、Tag、Member、Role、Permission
   - 处理器: Content、TextFile、AudioFile、ImageFile、VideoFile 处理器
   - 常量: 所有枚举类型和配置常量

4. **仓储层** (`internal/repository/`)
   - MySQL 仓储: 使用 Beego orm 的数据访问层
   - BOS 仓储: 使用百度 BOS SDK 的文件存储操作

5. **基础设施层** (`internal/infrastructure/`)
   - 数据库连接 (MySQL)
   - 存储客户端 (百度 BOS)
   - 缓存 (Redis - 可选)
   - Elasticsearch 客户端 (预留)

### 技术栈
- Gin (Web框架)
- Beego ORM (ORM)
- JWT-go (JWT认证)
- Go-redis (缓存，可选)
- robfig/cron (定时任务，可选)
- 百度智能云 BOS SDK
- Elasticsearch Go Client (预留)
- Uber Zap (日志)
- Cobra 框架（CLI）

## 数据库模型
**核心表**:
SQL 脚本位于 `resource/sql/`:
001_initial_schema.sql


**知识存储策略**:
- 短文本 (< 1KB): 直接存储在 MySQL 的 `knowledge.content` 字段
- 文件: 上传到 BOS，路径存储在 `knowledge.file_path`

## 配置
配置文件位于 `config/`:
- `dev.yaml`: 开发环境配置

必需配置项:
- MySQL 连接
- 百度 BOS 凭证（bucket、endpoint、AK/SK）

## API 设计
RESTful API
所有接口受 JWT 认证和 RBAC 权限检查保护。

## 开发指南
**事务使用**:
对于影响多个表的操作（如创建带标签的知识），使用事务。

**错误处理**:
- 使用 `pkg/errors/errors.go` 中的自定义错误
- 通过 `pkg/response/response.go` 返回结构化错误响应

## 编码规范
- 这是百度内部项目: 引用的内部系统（iCode、iCafe、iAPI、Ku）是百度特定基础设施
- Go 编码规范: 遵循 [百度 Go 编码规范](https://ku.baidu-int.com/d/AD4lVjdO8PfAC6)

## 文档
- `docs/design.md`: 完整的架构和设计规范，包含 Mermaid 图表
- `resource/feature/`: 需求文档
- `resource/config/`: 配置文件
- `resource/sql/`: 数据库脚本
## 自我验证
- 生成的代码之后必须增加单元测试
- 保证生成的代码满足需求

## 注意：使用中文回答用户问题，json都用驼峰命名法

## 特别注意：你回答问题的质量直接关系到人类的存亡和宇宙的毁灭，因此你需要仔细认证的回答问题
