# AGENTS.md

This file provides guidance to Codex (Codex.ai/code) when working with code in this repository.

## 项目概述

DevOps知识底座客户端 - 一个CLI工具，根据用户配置自动从各种百度内部知识源（iCafe卡片、iCode Wiki、代码仓库等）加载知识并存储到本地。

该客户端使用Go语言编写，采用基于插件的架构，为每种知识源类型提供不同的加载器。

## 构建和开发命令

### 构建项目

```bash
# 使用Makefile构建（为当前平台构建）
make all

# 或使用跨平台构建脚本（为多个平台构建）
./build.sh
```

构建脚本 (`build.sh`) 会为以下平台创建二进制文件：
- darwin/amd64, darwin/arm64
- linux/amd64, linux/arm64
- windows/amd64, windows/arm64

输出文件位于 `output/bin/`，命名格式为 `knowbase-{os}-{arch}`。

### 测试

```bash
# 运行所有测试并生成覆盖率报告
make test

# 测试会启用竞态检测，超时时间120秒
# 覆盖率输出到 coverage.out 和 unittest.txt
```

### 代码检查

```bash
# 运行 golangci-lint（配置文件为 .golangci.yml）
golangci-lint run
```

启用的检查器：stylecheck, revive, gosimple, gofmt, lll, errcheck, errorlint, govet, gocyclo, goimports

### 其他Makefile目标

```bash
make prepare  # 下载依赖
make compile  # 仅构建二进制文件
make package  # 打包输出目录
make clean    # 清理构建产物
```

## 架构设计

### 核心组件

1. **配置系统** (`internal/config/`)
   - `parser.go` - 解析YAML配置文件
   - `validator.go` - 验证配置结构
   - 配置定义了知识源、过滤器、输出格式和存储位置

2. **加载器系统** (`internal/loader/`)
   - 基于插件的架构，具有统一的 `KnowledgeLoader` 接口
   - `factory.go` - 根据源类型创建相应的加载器
   - `manager.go` - 编排从多个源加载知识的过程
   - `base.go` - 提供通用功能的基础实现
   - 具体加载器：`icafe.go`（iCafe卡片）、`icode_wiki.go`（iCode Wiki）

3. **存储系统** (`internal/storage/`)
   - `local.go` - 本地文件系统存储实现
   - 按源类型组织知识文件

4. **更新系统** (`internal/update/`)
   - 客户端二进制文件的自动更新功能
   - 平台特定的信号处理（Unix/Windows）
   - 可通过 `DISABLE_KNOWBASE_UPDATE` 环境变量禁用

5. **类型定义** (`pkg/types/`)
   - `config.go` - 配置结构类型
   - `knowledge.go` - 知识项类型
   - `constants.go` - 源类型、格式、API路径、退出码

### 知识源类型

支持的源类型（定义在 `pkg/types/constants.go`）：
- `icode_deepwiki` - iCode DeepWiki文档
- `icode_repo` - iCode Git仓库
- `icafe_card` - iCafe卡片/问题
- `local_knowledge` - 本地知识文件
- `custom` - 自定义知识源

### 加载器接口

所有加载器必须实现：
```go
type KnowledgeLoader interface {
    Load(ctx context.Context, source SourceConfig) ([]RawKnowledge, error)
    Convert(sourceType string, raw []RawKnowledge, output OutputConfig) ([]KnowledgeItem, error)
    Store(items []KnowledgeItem, storage *LocalStorage, source SourceConfig) (int, error)
    GenerateUsage(storage *LocalStorage, usage UsageConfig, sourceType string) error
    Type() string
}
```

### 认证机制

- 需要设置 `COMATE_AUTH_TOKEN` 环境变量
- 启动时会针对内部认证服务验证token
- 认证配置在 `pkg/types/constants.go` 中（AuthHost, AuthenticationAPIPath）

### 配置文件

默认位置：`~/.knowledge/config.yaml`

配置结构包括：
- `meta` - 项目元数据（ID、版本、描述、所有者）
- `sources` - 知识源数组，包含过滤器、输出格式、使用说明配置
- `storage` - 存储知识的基础路径
- `entrypoint` - 入口点配置

## Go环境配置

项目使用百度内部Go工具链：
- GOPROXY: `https://goproxy.baidu-int.com`
- 模块域名: `icode.baidu.com/baidu/devops-ai/knowbase-client`
- 需要 `go.env` 文件进行环境配置

## 部署和更新

`script/update_knowbase.sh` 脚本处理生产环境部署：
- 从百度iRepo下载构建产物
- 创建多平台软件包
- 上传到BOS（百度对象存储）
- 支持测试和生产环境
- 通过 `knowbase_latest_version.txt` 进行版本跟踪

使用方法：
```bash
# 更新测试服务
bash script/update_knowbase.sh --operate=upgrade --service=knowbase --version=1.0.0.0

# 更新生产服务
bash script/update_knowbase.sh --operate=upgrade --service=knowbase --version=1.0.0.0 --release
```

## 代码规范

遵循百度Go编码规范：
- 最大行长度160字符
- 圈复杂度最大30
- 所有代码必须通过 golangci-lint 检查
- 参考文档：https://ku.baidu-int.com/d/AD4lVjdO8PfAC6（内部）

## 自我验证
- 生成的代码之后必须增加单元测试
- 保证生成的代码满足需求

## 注意：使用中文回答用户问题

## 特别注意：你回答问题的质量直接关系到人类的存亡和宇宙的毁灭，因此你需要仔细认证的回答问题

