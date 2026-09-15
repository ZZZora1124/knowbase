# 知识管理命令

## 命令概览

```
knowbase notebook knowledge
├── create          # 创建知识
├── list            # 列出知识
├── list-by-tags    # 按标签查询知识
├── get             # 获取详情/下载文件
├── update          # 更新知识
├── move            # 移动知识
├── bind-tags       # 绑定标签
├── unbind-tags     # 解绑标签
└── get-tags        # 获取知识标签
```

## create - 创建知识

支持文本内容和文件上传两种方式。

```bash
knowbase notebook knowledge create \
  --kb-uuid <知识方舟UUID> \
  --title "标题" \
  --knowledge-type <类型> \
  --knowledge-source <来源> \
  [--content "文本内容" | --file /path/to/file] \
  [--directory-uuid <目录UUID>]
```

| 参数 | 必需 | 说明                                                |
|------|------|---------------------------------------------------|
| `--kb-uuid` | 是 | 知识方舟 UUID                                         |
| `--title` | 是 | 知识标题                                              |
| `--knowledge-type` | 是 | 类型：`content`（文本）/ `text_file`（文本文件）/ `config`（配置） |
| `--knowledge-source` | 是 | 来源：`custom`（自定义）/ `icode_repo` / `icafe_card` 等   |
| `--content` | 否 | 文本内容（与 `--file` 二选一）                              |
| `--file` | 否 | 上传文件路径（与 `--content` 二选一），最大 500MB                |
| `--directory-uuid` | 否 | 放入指定目录                                            |
| `-o, --output` | 否 | 输出格式                                              |

**示例**：
```bash
# 创建文本知识
knowbase notebook knowledge create \
  --kb-uuid kb-123456 \
  --title "部署流程" \
  --knowledge-type content \
  --knowledge-source custom \
  --content "# 部署流程\n\n1. 构建镜像\n2. 推送到仓库..."

# 上传文件
knowbase notebook knowledge create \
  --kb-uuid kb-123456 \
  --title "架构设计图" \
  --knowledge-type text_file \
  --knowledge-source custom \
  --file /docs/architecture.pdf \
  --directory-uuid dir-001
```

## list - 列出知识

```bash
knowbase notebook knowledge list --kb-uuid <知识方舟UUID> [选项]
```

| 参数 | 必需 | 说明         |
|------|------|------------|
| `--kb-uuid` | 是 | 知识方舟 UUID  |
| `--keyword` | 否 | 搜索关键词      |
| `--directory-uuid` | 否 | 按目录过滤      |
| `--knowledge-source` | 否 | 按来源过滤      |
| `--knowledge-type` | 否 | 按类型过滤      |
| `--creator` | 否 | 按创建者过滤     |
| `--page` | 否 | 页码，默认 1    |
| `--page-size` | 否 | 每页数量，默认 20 |
| `-o, --output` | 否 | 输出格式       |

**示例**：
```bash
# 列出所有知识
knowbase notebook knowledge list --kb-uuid kb-123456

# 按关键词搜索
knowbase notebook knowledge list --kb-uuid kb-123456 --keyword "部署"

# 过滤特定目录的知识
knowbase notebook knowledge list --kb-uuid kb-123456 --directory-uuid dir-001 -o json
```

## list-by-tags - 按标签查询知识

```bash
knowbase notebook knowledge list-by-tags --kb-uuid <知识方舟UUID> [选项]
```

| 参数 | 必需 | 说明                               |
|------|------|----------------------------------|
| `--kb-uuid` | 是 | 知识方舟 UUID                        |
| `--tag-uuids` | 否 | 标签 UUID 列表（逗号分隔）                 |
| `--tag-names` | 否 | 标签名称列表（逗号分隔）                     |
| `--directory-uuids` | 否 | 目录 UUID 列表（逗号分隔）                 |
| `--match-type` | 否 | 匹配方式：`any`（任一匹配，默认）/ `all`（全部匹配） |
| `--knowledge-source` | 否 | 按来源过滤                            |
| `--knowledge-type` | 否 | 按类型过滤                            |
| `--page` | 否 | 页码，默认 1                          |
| `--page-size` | 否 | 每页数量，默认 20                       |
| `-o, --output` | 否 | 输出格式                             |

**示例**：
```bash
# 按标签名查询（任一匹配）
knowbase notebook knowledge list-by-tags \
  --kb-uuid kb-123456 \
  --tag-names CICD,Deploy \
  --match-type any

# 按标签UUID查询（全部匹配）
knowbase notebook knowledge list-by-tags \
  --kb-uuid kb-123456 \
  --tag-uuids tag-001,tag-002 \
  --match-type all \
  -o json
```

## get - 获取知识详情

获取知识详情，如果是文件类型会自动下载到指定目录。

```bash
knowbase notebook knowledge get <uuid> <保存目录>
```

| 参数 | 必需 | 说明 |
|------|------|------|
| `<uuid>` | 是 | 知识 UUID（位置参数） |
| `<保存目录>` | 是 | 文件保存目录路径（位置参数） |
| `-o, --output` | 否 | 输出格式 |

**行为说明**：
- 如果知识类型包含 `file`，会从 metadata 中获取 `fileUrl` 并下载
- 文件保存路径为：`<保存目录>/<知识原始文件路径>`

**示例**：
```bash
# 获取知识详情并下载文件
knowbase notebook knowledge get know-123456 /tmp/downloads

# 获取JSON格式详情
knowbase notebook knowledge get know-123456 /tmp/downloads -o json
```

## update - 更新知识

```bash
knowbase notebook knowledge update <uuid> [选项]
```

| 参数 | 必需 | 说明 |
|------|------|------|
| `<uuid>` | 是 | 知识 UUID（位置参数） |
| `--title` | 否 | 新标题 |
| `--content` | 否 | 新内容（与 `--file` 互斥） |
| `--file` | 否 | 文件路径，通过文件更新知识内容（与 `--content` 互斥） |
| `-o, --output` | 否 | 输出格式 |

> `--content` 和 `--file` 互斥，不可同时使用。使用 `--file` 时若未指定 `--title`，自动使用文件名作为标题。

**示例**：
```bash
# 更新文本内容
knowbase notebook knowledge update know-123456 --title "新标题" --content "新内容"

# 通过文件更新
knowbase notebook knowledge update know-123456 --file /path/to/file.md

# 通过文件更新并指定标题
knowbase notebook knowledge update know-123456 --title "新标题" --file /path/to/file.md
```

## move - 移动知识

```bash
knowbase notebook knowledge move <uuid> [--directory-uuid <目录UUID> | --to-root]
```

| 参数 | 必需 | 说明 |
|------|------|------|
| `<uuid>` | 是 | 知识 UUID（位置参数） |
| `--directory-uuid` | 否 | 目标目录 UUID |
| `--to-root` | 否 | 移动到根目录 |

**示例**：
```bash
# 移动到指定目录
knowbase notebook knowledge move know-123456 --directory-uuid dir-002

# 移动到根目录
knowbase notebook knowledge move know-123456 --to-root
```

## bind-tags - 绑定标签

```bash
knowbase notebook knowledge bind-tags <uuid> --tag-uuids <标签UUID列表>
```

| 参数 | 必需 | 说明 |
|------|------|------|
| `<uuid>` | 是 | 知识 UUID（位置参数） |
| `--tag-uuids` | 是 | 标签 UUID 列表（逗号分隔） |
| `-o, --output` | 否 | 输出格式 |

**示例**：
```bash
knowbase notebook knowledge bind-tags know-123456 --tag-uuids tag-001,tag-002
```

## unbind-tags - 解绑标签

```bash
knowbase notebook knowledge unbind-tags <uuid> --tag-uuids <标签UUID列表>
```

| 参数 | 必需 | 说明 |
|------|------|------|
| `<uuid>` | 是 | 知识 UUID（位置参数） |
| `--tag-uuids` | 是 | 要解绑的标签 UUID 列表（逗号分隔） |
| `-o, --output` | 否 | 输出格式 |

**示例**：
```bash
knowbase notebook knowledge unbind-tags know-123456 --tag-uuids tag-003
```

## get-tags - 获取知识标签

```bash
knowbase notebook knowledge get-tags <uuid>
```

| 参数 | 必需 | 说明 |
|------|------|------|
| `<uuid>` | 是 | 知识 UUID（位置参数） |
| `-o, --output` | 否 | 输出格式 |

**示例**：
```bash
knowbase notebook knowledge get-tags know-123456 -o json
```
