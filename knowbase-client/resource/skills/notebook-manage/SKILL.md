---
name: notebook-manage
description: 管理方舟平台的技能。支持方舟的创建、查询、更新，成员管理，标签管理，目录管理，内容管理等完整功能。使用场景：当用户需要管理方舟平台相关内容时使用此skill。
allowed-tools: Read, Grep, Glob, Bash(knowbase notebook *), Bash(which knowbase), Bash(/bin/bash -c *)
metadata:
  version: "2.0"
---

# 方舟管理

管理方舟平台的技能，通过 `knowbase notebook` 命令实现方舟全生命周期管理。

## 前置检查（必须）

**执行任何命令前，先检查并安装客户端：**

```bash
# 检查是否已安装
which knowbase

# 未安装则执行安装
/bin/bash -c "$(curl -fsSL http://knowbase-client.bj.bcebos.com/knowbase/install.sh)"
```

**检查认证状态：**

```bash
knowbase login status
```

knowbase 支持两种认证模式：`COMATE_AUTH_TOKEN` 环境变量 和 UGate 本地登录。

- 平台内置 token 场景：通常使用 `COMATE_AUTH_TOKEN`，保持原有链路不变。
- 个人用户场景：同时支持两种模式，其中 UGate 登录通过 `knowbase login` 登录一次后本地持久化 token。
- 如果 `COMATE_AUTH_TOKEN` 已设置，会完全优先使用环境变量；如需使用 UGate，先执行 `unset COMATE_AUTH_TOKEN`。

UGate 登录：

```bash
# 访问 https://uuap.baidu.com/agent/token 获取 token
# 只保留真实 JWT token，不要带 “ugate token:” 前缀或复制说明文字
knowbase login <username> '<ugate-token>'
```

常用认证命令：

```bash
knowbase login status
knowbase logout
```

## 快速入门

执行 notebook 命令前，先确认 `knowbase login status` 已显示已认证；如果未登录，先完成 `COMATE_AUTH_TOKEN` 设置或 UGate 登录。

### 1. 创建方舟

```bash
knowbase notebook create --name "我的方舟" --visibility private
```

### 2. 添加内容

```bash
# 添加文本内容
knowbase notebook knowledge create \
  --kb-uuid <方舟UUID> \
  --title "部署流程" \
  --knowledge-type content \
  --knowledge-source custom \
  --content "内容..."

# 上传文件
knowbase notebook knowledge create \
  --kb-uuid <方舟UUID> \
  --title "架构图" \
  --knowledge-type text_file \
  --knowledge-source custom \
  --file /path/to/file
```

### 3. 查询内容

```bash
# 列出方舟中的内容
knowbase notebook knowledge list --kb-uuid <方舟UUID>

# 获取详情并下载文件
knowbase notebook knowledge get <内容UUID> /保存目录
```

### 4. JSON 输出（Agent 友好）

所有命令支持 `-o json` 获取结构化输出：

```bash
knowbase notebook list -o json
```

## 核心场景

### 场景 A：搭建团队方舟

```bash
# 1. 创建方舟
knowbase notebook create --name "DevOps最佳实践" --visibility private -o json

# 2. 创建目录分类（支持最多5层嵌套）
knowbase notebook dir create --kb-uuid <kb-uuid> --name "部署文档"
knowbase notebook dir create --kb-uuid <kb-uuid> --name "Docker部署" --parent-uuid <部署文档uuid>
knowbase notebook dir create --kb-uuid <kb-uuid> --name "Dockerfile规范" --parent-uuid <Docker部署uuid>
knowbase notebook dir create --kb-uuid <kb-uuid> --name "监控告警"

# 查看目录树结构
knowbase notebook dir tree --kb-uuid <kb-uuid>

# 3. 添加标签
knowbase notebook tag create --kb-uuid <kb-uuid> --name "CICD"

# 4. 邀请成员
knowbase notebook member add --kb-uuid <kb-uuid> \
  --member-type username --member-value lisi --role editor
```

### 场景 B：按标签检索内容

```bash
# 按标签名查询
knowbase notebook knowledge list-by-tags \
  --kb-uuid <kb-uuid> \
  --tag-names CICD,Deploy \
  --match-type any \
  -o json
```

### 场景 C：内容整理与归档

```bash
# 移动内容到目录
knowbase notebook knowledge move <内容UUID> --directory-uuid <目录UUID>

# 给内容打标签
knowbase notebook knowledge bind-tags <内容UUID> --tag-uuids tag-001,tag-002

# 批量下载内容文件
knowbase notebook knowledge get <内容UUID> /backup/knowledge
```

## 命令速查

| 模块   | 常用命令 | 说明      |
|------|---------|---------|
| 方舟 | `notebook create` | 创建方舟  |
|      | `notebook list` | 列出方舟  |
|      | `notebook get <uuid>` | 获取详情    |
| 内容   | `notebook knowledge create` | 创建内容    |
|      | `notebook knowledge list` | 列出内容    |
|      | `notebook knowledge get` | 获取详情/下载 |
| 目录   | `notebook dir create` | 创建目录（支持多级） |
|      | `notebook dir list` | 列出目录    |
|      | `notebook dir tree` | 获取目录树   |
|      | `notebook dir move` | 移动目录    |
| 标签   | `notebook tag create` | 创建标签    |
|      | `notebook tag list` | 列出标签    |
| 成员   | `notebook member add` | 添加成员    |
|      | `notebook member list` | 列出成员    |

## 详细参考

需要完整命令参数说明，请参考：

- [方舟管理命令](references/commands-knowledgebase.md)
- [内容管理命令](references/commands-knowledge.md)
- [目录管理命令](references/commands-directory.md)
- [标签管理命令](references/commands-tag.md)
- [成员管理命令](references/commands-member.md)
- [错误处理指南](references/error-handling.md)
