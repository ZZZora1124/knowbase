# Notebook-Manage Skill 使用说明

## 概述

`notebook-manage` 是一个用于管理[方舟平台](https://console.cloud.baidu-int.com/onetool/knowledge)的 Agent Skill。通过自然语言与 AI 交互，可以轻松完成方舟的创建、内容管理、目录组织、标签分类、成员协作等操作。

## 功能特性

| 功能模块 | 支持的操作 |
|---------|-----------|
| 方舟管理 | 创建、列出、查询、更新方舟 |
| 内容管理 | 创建、列出、获取、更新、移动内容，支持文件上传 |
| 目录管理 | 创建、列出、获取、更新、移动目录，支持多级目录（最多5层） |
| 标签管理 | 创建、列出、更新标签，为内容绑定/解绑标签 |
| 成员管理 | 添加成员、列出成员、更新成员角色 |

## 前置条件

### 1. 身份认证

`knowbase notebook` 支持两种认证方式：

1. `COMATE_AUTH_TOKEN` 环境变量认证
2. UGate 本地登录认证


优先使用 `COMATE_AUTH_TOKEN`环境变量认证，未设置 `COMATE_AUTH_TOKEN` 时，才会读取本地 UGate 登录态。

#### 方式一：COMATE_AUTH_TOKEN 环境变量

适用于平台内置 token 或已有 Comate token 的场景。

```bash
export COMATE_AUTH_TOKEN="your-auth-token"
knowbase login status
```

如果认证成功，会显示：

```text
认证模式: comate (环境变量)
```

#### 方式二：UGate 本地登录

适用于个人用户。先访问以下页面获取 UGate token：

```text
https://uuap.baidu.com/agent/token
```

复制 token 后，只保留真实 JWT token，不要带 `ugate token:` 前缀及其他文字，然后执行：

```bash
knowbase login <username> '<ugate-token>'
knowbase login status
```

如果认证成功，会显示：

```text
认证模式: ugate (本地 Token)
```

登出本地 UGate 登录态：

```bash
knowbase logout
```

如需从 Comate 环境变量模式切换到 UGate 模式，先执行：

```bash
unset COMATE_AUTH_TOKEN
```

### 2. 客户端安装

Skill 会自动检测并安装 knowbase 客户端。如需手动安装：

```bash
/bin/bash -c "$(curl -fsSL http://knowbase-client.bj.bcebos.com/knowbase/install.sh)"
```

## 使用方法

### 触发方式

在Ducc中使用以下方式触发此 Skill：

```
/notebook-manage
```

或在对话中描述方舟管理相关需求，AI 会自动识别并使用此 Skill。

### 使用示例

#### 示例 1：创建方舟并添加内容

**用户输入：**
> 帮我创建一个名为"DevOps最佳实践"的私有方舟，并添加一条关于 CI/CD 流程的内容

**AI 执行：**
1. 创建方舟
2. 添加内容
3. 返回执行结果

#### 示例 2：组织内容结构

**用户输入：**
> 在方舟中创建"部署文档"和"监控告警"两个目录，并创建 CICD、K8S、Docker 三个标签

**AI 执行：**
1. 创建目录
2. 创建标签
3. 汇总创建结果

#### 示例 3：团队协作

**用户输入：**
> 把用户 lisi 添加为方舟 kb-123456 的编辑者，把 devops-team@baidu.com 邮件组添加为查看者

**AI 执行：**
1. 添加个人用户为编辑者
2. 添加邮件组为查看者
3. 返回成员列表

#### 示例 4：内容检索

**用户输入：**
> 查找方舟中所有带有 CICD 或 Deploy 标签的内容

**AI 执行：**
1. 按标签查询内容
2. 返回匹配的内容列表

#### 示例 5：内容归档

**用户输入：**
> 把内容 know-123456 移动到"部署文档"目录，并打上 K8S 和 Docker 标签

**AI 执行：**
1. 移动内容到指定目录
2. 绑定标签
3. 确认操作结果

## 命令速查表

### 方舟操作

| 操作        | 命令 |
|-----------|------|
| 创建方舟    | `knowbase notebook create --name "名称" --visibility private` |
| 列出方舟    | `knowbase notebook list` |
| 按 UUID 获取 | `knowbase notebook get <uuid>` |
| 按名称获取     | `knowbase notebook get-by-name "名称"` |
| 更新方舟    | `knowbase notebook update <uuid> --name "新名称"` |

### 内容操作

| 操作 | 命令 |
|------|------|
| 创建文本内容 | `knowbase notebook knowledge create --kb-uuid <kb-uuid> --title "标题" --knowledge-type content --knowledge-source custom --content "内容"` |
| 上传文件 | `knowbase notebook knowledge create --kb-uuid <kb-uuid> --title "标题" --knowledge-type text_file --knowledge-source custom --file /path/to/file` |
| 列出内容 | `knowbase notebook knowledge list --kb-uuid <kb-uuid>` |
| 按标签查询 | `knowbase notebook knowledge list-by-tags --kb-uuid <kb-uuid> --tag-names tag1,tag2` |
| 获取内容/下载 | `knowbase notebook knowledge get <uuid> /保存目录` |
| 移动内容 | `knowbase notebook knowledge move <uuid> --directory-uuid <dir-uuid>` |
| 绑定标签 | `knowbase notebook knowledge bind-tags <uuid> --tag-uuids tag1,tag2` |

### 目录操作

| 操作 | 命令 |
|------|------|
| 创建目录 | `knowbase notebook dir create --kb-uuid <kb-uuid> --name "目录名"` |
| 创建子目录 | `knowbase notebook dir create --kb-uuid <kb-uuid> --name "子目录" --parent-uuid <父目录UUID>` |
| 列出目录 | `knowbase notebook dir list --kb-uuid <kb-uuid>` |
| 获取目录树 | `knowbase notebook dir tree --kb-uuid <kb-uuid>` |
| 移动目录 | `knowbase notebook dir move --kb-uuid <kb-uuid> --dir-uuid <uuid> --parent-uuid <目标目录UUID>` |
| 更新目录 | `knowbase notebook dir update --kb-uuid <kb-uuid> --dir-uuid <uuid> --name "新名称"` |

### 标签操作

| 操作 | 命令 |
|------|------|
| 创建标签 | `knowbase notebook tag create --kb-uuid <kb-uuid> --name "标签名"` |
| 列出标签 | `knowbase notebook tag list --kb-uuid <kb-uuid>` |
| 更新标签 | `knowbase notebook tag update --kb-uuid <kb-uuid> --tag-uuid <uuid> --name "新名称"` |

### 成员操作

| 操作 | 命令 |
|------|------|
| 添加成员 | `knowbase notebook member add --kb-uuid <kb-uuid> --member-type username --member-value <用户名> --role editor` |
| 列出成员 | `knowbase notebook member list --kb-uuid <kb-uuid>` |
| 更新角色 | `knowbase notebook member update-role --kb-uuid <kb-uuid> --member-uuid <uuid> --role admin` |

## 角色权限说明

| 角色 | 说明 | 查看 | 编辑 | 管理成员 | 删除方舟 |
|------|------|:----:|:----:|:--------:|:----------:|
| viewer | 查看者 | ✓ | - | - | - |
| editor | 编辑者 | ✓ | ✓ | - | - |
| admin | 管理员 | ✓ | ✓ | ✓ | ✓ |

## 内容类型说明

| 类型 | 说明 | 使用场景 |
|------|------|---------|
| content | 文本内容 | 直接输入的文本 |
| text_file | 文本文件 | 上传文档、代码等文本文件 |
| config | 配置 | 配置文件类内容 |

## 可见性说明

| 可见性 | 说明 |
|--------|------|
| private | 私有，仅成员可见 |
| public | 公开，所有人可见 |
| personal | 个人，仅创建者可见 |

## 常见问题

### Q1: 如何获取方舟的 UUID？

使用 `knowbase notebook list --keyword 关键字 -o json` 命令查看名称关键字匹配的方舟及其 UUID。

### Q2: 文件上传有大小限制吗？

是的，单个文件最大支持 50MB。

### Q3: 如何批量导入内容？

可以编写脚本循环调用 `knowbase notebook knowledge create` 命令，或使用 AI 辅助批量操作。

### Q4: 标签查询支持哪些匹配模式？

- `any`：任一匹配，内容包含任意一个指定标签即返回
- `all`：全部匹配，内容必须包含所有指定标签

### Q5: 如何导出内容？

使用 `knowbase notebook knowledge get <uuid> /保存目录` 命令下载内容文件到本地。

### Q6: 目录最多支持几层嵌套？

最多支持 5 层嵌套。可以使用 `knowbase notebook dir tree` 命令查看当前目录树结构。


## 错误排查

遇到问题时，可以：

1. **检查认证**：执行 `knowbase login status`，确认当前是 `comate (环境变量)` 还是 `ugate (本地 Token)`
2. **切换认证模式**：如需使用 UGate，先执行 `unset COMATE_AUTH_TOKEN`
3. **使用 JSON 输出**：添加 `-o json` 参数查看详细返回信息
4. **查看命令帮助**：`knowbase notebook --help` 或 `knowbase notebook <子命令> --help`
5. **分步执行**：将复杂操作拆分为单步执行，定位具体问题环节
