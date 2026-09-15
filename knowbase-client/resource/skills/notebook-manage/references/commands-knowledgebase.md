# 知识方舟管理命令

## 命令概览

```
knowbase notebook
├── create              # 创建知识方舟
├── list                # 列出知识方舟
├── get <uuid>          # 按UUID获取详情
├── get-by-name <name>  # 按名称获取详情
└── update <uuid>       # 更新知识方舟
```

## create - 创建知识方舟

```bash
knowbase notebook create --name "名称" --visibility private [--description "描述"]
```

| 参数 | 必需 | 说明 |
|------|------|------|
| `--name` | 是 | 知识方舟名称 |
| `--visibility` | 是 | 可见性：`private`（私有）/ `public`（公开）/ `personal`（个人） |
| `--description` | 否 | 知识方舟描述 |
| `-o, --output` | 否 | 输出格式：`json` / `table`（默认） |

**示例**：
```bash
# 创建私有知识方舟
knowbase notebook create --name "DevOps最佳实践" --visibility private

# 创建并获取JSON输出
knowbase notebook create --name "公共文档" --visibility public --description "团队共享文档" -o json
```

## list - 列出知识方舟

```bash
knowbase notebook list [选项]
```

| 参数 | 必需 | 说明 |
|------|------|------|
| `--keyword` | 否 | 搜索关键词 |
| `--visibility` | 否 | 按可见性过滤 |
| `--owner` | 否 | 按所有者过滤 |
| `--page` | 否 | 页码，默认 1 |
| `--page-size` | 否 | 每页数量，默认 20，范围 1-100 |
| `-o, --output` | 否 | 输出格式：`json` / `table`（默认） |

**示例**：
```bash
# 列出所有知识方舟
knowbase notebook list

# 按关键词搜索
knowbase notebook list --keyword DevOps

# 分页查询私有知识方舟
knowbase notebook list --visibility private --page 1 --page-size 50 -o json
```

**输出字段**（table 模式）：
- UUID - 知识方舟唯一标识
- NAME - 名称
- VISIBILITY - 可见性
- OWNER - 所有者
- KNOWLEDGE - 知识数量
- MEMBERS - 成员数量
- ROLE - 当前用户角色

## get - 按UUID获取详情

```bash
knowbase notebook get <uuid>
```

| 参数 | 必需 | 说明 |
|------|------|------|
| `<uuid>` | 是 | 知识方舟 UUID（位置参数） |
| `-o, --output` | 否 | 输出格式 |

**示例**：
```bash
knowbase notebook get kb-123456-abcd-7890
```

## get-by-name - 按名称获取详情

```bash
knowbase notebook get-by-name <name>
```

| 参数 | 必需 | 说明 |
|------|------|------|
| `<name>` | 是 | 知识方舟名称（位置参数） |
| `-o, --output` | 否 | 输出格式 |

**示例**：
```bash
knowbase notebook get-by-name "DevOps最佳实践"
```

## update - 更新知识方舟

```bash
knowbase notebook update <uuid> [选项]
```

| 参数 | 必需 | 说明 |
|------|------|------|
| `<uuid>` | 是 | 知识方舟 UUID（位置参数） |
| `--name` | 否 | 新名称 |
| `--description` | 否 | 新描述 |
| `--visibility` | 否 | 新可见性 |
| `-o, --output` | 否 | 输出格式 |

**示例**：
```bash
# 更新名称
knowbase notebook update kb-123456 --name "新名称"

# 更新多个字段
knowbase notebook update kb-123456 --name "新名称" --description "新描述" --visibility public
```
