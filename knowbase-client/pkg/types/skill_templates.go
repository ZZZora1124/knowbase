// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package types

// DefaultSkillName 默认skill名称
const DefaultSkillName = "knowledge-use"

// DefaultSkillContent SKILL.md的内容
const DefaultSkillContent = `---
name: knowledge-use
description: 用于高效导航和访问知识目录中存储的知识。当用户询问如何查找知识、理解知识目录结构、定位特定知识源(iCafe卡片、iCode Wiki、iCode CR代码变更、API文档、代码仓库、本地知识、知识方舟知识、数据集)或需要帮助访问存储的知识项时使用。触发场景包括:"有哪些知识可用?"、"查找关于X的iCafe卡片"、"API文档在哪里?"、"显示项目文档"或任何关于导航知识库的查询。
allowed-tools: Read, Grep, Glob, Write, Edit
---

# 知识使用指南

## 概述

本 skill 帮助你高效地导航和访问本地文件系统存储的知识。知识目录可能包含各种类型的知识源(iCafe卡片、iCode Wiki文档、iCode CR代码变更、代码仓库、API文档、知识库文档、本地知识、知识方舟知识、数据集),每种类型都组织在专门的子目录中,并有各自的使用说明文档。

## 快速导航策略

### 从这里开始:入口文档

**始终从入口文档开始:**

` + "```" + `
<basePath>/knowledge_usage_readme.md
` + "```" + `

此文档提供:
- 完整的目录结构概览
- 所有知识分类及其描述
- 指向每个子目录使用说明的直接链接
- 可用知识的快速摘要

**默认 basePath:** ` + "`~/knowledge`" + ` 或 ` + "`.knowledge`" + `

### 查找特定知识

一旦知道知识类型,导航到其子目录:

` + "```" + `
<basePath>/<source-type>/*_usage.md
` + "```" + `

此使用说明文件包含:
- 该源中所有知识项的列表
- 每个知识项的完整存储路径
- 每个知识项的描述

## 知识源类型

知识目录支持以下源类型:

1. **icafe_card** - iCafe卡片、Story、Task和Bug
2. **icode_deepwiki** - iCode Wiki文档
3. **icode_repo** - Git代码仓库
4. **iapi_doc** - API接口文档
5. **local_knowledge** - 本地知识文件和文档
6. **ku_doc** - 知识库文档
7. **icode_cr_diff** - iCode代码评审变更
8. **knowledge_book** - 知识方舟知识
9. **idataset** - 数据集

有关每种源类型的详细信息,请参阅 [source-types.md](references/source-types.md)。

## 工作流程

### 步骤1:确定用户需求

理解用户在寻找什么:

- **广泛探索**:"有哪些知识可用?" → 阅读 ` + "`knowledge_usage_readme.md`" + `
- **特定知识类型**:"查找关于认证的iCafe卡片" → 导航到 ` + "`icafe_card/*_usage.md`" + `
- **特定文件/项**:"显示项目X的API文档" → 导航到 ` + "`iapi_doc/*_usage.md`" + `,然后到具体文件

### 步骤2:导航到正确位置

**对于广泛的问题:**
` + "```bash" + `
# 阅读入口文档
Read <basePath>/knowledge_usage_readme.md
` + "```" + `

**对于特定知识类型:**
` + "```bash" + `
# 首先,检查子目录的使用说明文件
Read <basePath>/<source-type>/usage_*.md

# 然后访问具体的知识项
Read <basePath>/<source-type>/<item-path>
` + "```" + `

**对于搜索查询:**
` + "```bash" + `
# 在所有知识中搜索
grep -r "搜索词" <basePath>/

# 或在特定类型中搜索
grep -r "搜索词" <basePath>/<source-type>/
` + "```" + `

### 步骤3:向用户提供上下文

向用户展示知识时,始终包括:

1. **知识类型**:来自哪个源
2. **存储路径**:文件位于何处
3. **内容摘要**:包含什么的简要描述
4. **相关项**:同一类别中的其他相关知识

## 常见使用场景

### 场景1:用户想要概览

**用户查询:** "我们有哪些知识?"

**操作:**
1. 阅读 ` + "`<basePath>/knowledge_usage_readme.md`" + `
2. 展示目录结构和可用的知识分类
3. 根据用户兴趣提供深入特定领域的选项

### 场景2:用户需要特定知识类型

**用户查询:** "显示所有关于登录功能的iCafe卡片"

**操作:**
1. 导航到 ` + "`<basePath>/icafe_card/*_usage.md`" + `
2. 搜索与"登录"相关的卡片
3. 阅读相关的卡片文件
4. 展示发现的内容和文件路径

### 场景3:用户想探索代码

**用户查询:** "认证是如何实现的?"

**操作:**
1. 检查是否有包含相关代码库的 ` + "`icode_repo`" + `
2. 阅读 ` + "`<basePath>/icode_repo/*_usage.md`" + ` 查看可用的仓库
3. 导航到适当的仓库目录
4. 使用 Grep 或代码搜索查找与认证相关的代码
5. 展示带有上下文的代码

### 场景4:用户需要API文档

**用户查询:** "有哪些可用的API端点?"

**操作:**
1. 导航到 ` + "`<basePath>/iapi_doc/*_usage.md`" + `
2. 列出所有可用的API文档文件
3. 如果是Swagger格式,读取并解析JSON
4. 如果是Markdown格式,读取并展示内容

## 文件查找技巧

### 快速定位文件

使用说明文档包含确切路径。当你阅读使用说明文件时,会看到类似这样的条目:

` + "```markdown" + `
- **名称**: 认证API
  **描述**: 用户认证和授权端点
  **路径**: /Users/username/knowledge/iapi_doc/api_366675.json
` + "```" + `

直接使用这些路径访问知识文件。

### 路径模式

常见路径模式:

- iCafe卡片: ` + "`<basePath>/icafe_card/<卡片名称>.md`" + `
- Wiki文档: ` + "`<basePath>/icode_deepwiki/<仓库名>/<文档路径>.md`" + `
- 代码仓库: ` + "`<basePath>/icode_repo/<仓库名>/`" + `
- API文档: ` + "`<basePath>/iapi_doc/<接口名称>.(json|md)`" + `
- 本地知识: ` + "`<basePath>/local_knowledge/<自定义路径>/`" + `
- 知识库文档: ` + "`<basePath>/ku_doc/*_usage.md`" + `
- iCode代码评审变更: ` + "`<basePath>/icode_cr_diff/<变更ID>.md`" + `
- 知识方舟知识: ` + "`<basePath>/knowledge_book/custom/<知识文件>`" + `
- 数据集: ` + "`<basePath>/idataset/*_usage.md`" + `

## 高级功能

### 理解目录结构

要详细了解知识目录的组织方式,包括命名约定和文件结构,请参阅 [knowledge-structure.md](references/knowledge-structure.md)。

### 多个知识源

当多个源可能包含相关信息时:

1. 从最具体的源类型开始
2. 如果未找到,将搜索范围扩大到 ` + "`knowledge_usage_readme.md`" + `
3. 考虑使用 grep 在所有源中搜索

### 处理缺失的知识

如果请求的知识不存在:

1. 检查 ` + "`knowledge_usage_readme.md`" + ` 确认可用内容
2. 告知用户存在哪些知识类型

## 重要提示

- **总是先检查入口文档** 在深入细节之前先获得概览
- **使用说明文件是你的索引** - 它们告诉你什么可用以及存储在哪里
- **路径是绝对路径** - 使用说明文件包含完整路径供直接访问

## 资源文件

### references/

- **knowledge-structure.md** - 目录结构、文件命名约定和路径查找策略的详细说明
- **source-types.md** - 关于每种知识源类型、其格式和典型用例的全面信息
`

// KnowledgeStructureContent references/knowledge-structure.md的内容
const KnowledgeStructureContent = `# 知识目录结构说明

本文档说明知识底座的标准目录结构和文件组织方式。

## 标准目录结构

` + "```" + `
<basePath>/                           # 知识根目录（默认 ~/knowledge）
├── knowledge_usage_readme.md         # 整体知识使用说明（入口文档）
├── icode_deepwiki/                   # iCode Wiki 文档
│   ├── *_usage.md                    # 该源的使用说明
│   └── [wiki文档文件]
├── idataset/                         # 数据集
│   ├── *_usage.md                    # 该源的使用说明
│   └── [数据集文件]
├── icode_repo/                       # iCode 代码仓库
│   ├── *_usage.md                    # 该源的使用说明
│   └── [代码仓库目录]
├── icafe_card/                       # iCafe 卡片
│   ├── *_usage.md                    # 该源的使用说明
│   └── [卡片文件，如 card_123.md]
├── iapi_doc/                         # iAPI 接口文档
│   ├── *_usage.md                    # 该源的使用说明
│   └── [API文档文件]
├── local_knowledge/                  # 本地知识
│   ├── *_usage.md                    # 该源的使用说明
│   └── [本地知识文件或目录]
├── infoflow_group_message/           # 如流群消息记录
│   ├── *_usage.md                    # 该源的使用说明
│   └── [群聊记录文件]
├── knowledge_book/                   # 知识方舟知识
│   ├── *_usage.md                    # 该源的使用说明
│   └── custom/                       # 自定义知识方舟目录
│       └── [知识方舟文件]
└── icode_cr_diff/                    # iCode代码评审变更
    ├── *_usage.md                    # 该源的使用说明
    └── [代码评审变更文件]
` + "```" + `

## 关键文件说明

### 1. knowledge_usage_readme.md

**位置:** ` + "`<basePath>/knowledge_usage_readme.md`" + `

**内容:**
- 项目基本信息
- 完整的目录结构树
- 各知识分类的说明
- 每个子目录包含的内容概览

**作用:** 这是整个知识目录的入口文档,提供全局视图。

**何时查看:**
- 首次了解知识目录结构时
- 需要快速定位某类知识时
- 想了解整体知识内容时

### 2. 子目录 *_usage.md

**位置:** ` + "`<basePath>/<source-type>/*_usage.md`" + `

**内容:**
- 该知识源的详细说明
- 所有知识项的列表
- 每个知识项的存储路径
- 知识项的描述信息

**作用:** 提供特定知识源的详细索引。

**何时查看:**
- 已知需要某类知识时
- 需要查找具体知识项的路径时
- 了解某知识源包含哪些内容时

## 文件命名规范

### icode_deepwiki
- 文件保持原 Wiki 的目录结构
- Markdown 文件命名与 Wiki 页面标题对应

### icode_repo
- 目录名为仓库名
- 保持原代码仓库的完整结构

### icafe_card
- 文件命名格式:` + "`<卡片名称>.md`" + ` 或 ` + "`card_<卡片名称>.json`" + `
- 示例:` + "`添加按钮.md`" + `

### iapi_doc
- Swagger 格式:` + "`查询项目列表.json`" + `
- Markdown 格式:` + "`查询项目.md`" + `

### github_repo
- 目录名为仓库名
- 保持原代码仓库的完整结构

### local_knowledge
- 保持源文件的原始结构
- 或按配置的 ` + "`destPath`" + ` 相对路径组织 + 

### icode_cr_diff
- 文件命名格式：*diff_summary.md 表示代码变更汇总；*diff_detail*.md表示代码变更详细
- 示例：baidu_devops_hello_diff_detail_119.md

## 路径查找策略

### 1. 从整体到局部

` + "```" + `
1. 先查看 knowledge_usage_readme.md 了解整体结构
2. 根据需求定位到具体的子目录
3. 查看子目录的 *_usage.md 获取详细信息
4. 访问具体的知识文件
` + "```" + `

### 2. 直接查找

如果已知知识类型:

` + "```bash" + `
# 查看某类知识的使用说明
cat <basePath>/<source-type>/*_usage.md

# 查找具体文件
ls <basePath>/<source-type>/
` + "```" + `

### 3. 搜索查找

如果不确定位置:

` + "```bash" + `
# 在整个知识目录中搜索
grep -r "关键词" <basePath>/

# 查找特定类型的文件
find <basePath> -name "*.md" | grep "关键词"
` + "```" + `
`

// SourceTypesContent references/source-types.md的内容
const SourceTypesContent = `# 知识源类型详细说明

本文档详细描述了知识底座支持的各种知识源类型及其特点。

## 支持的知识源类型

### 1. icode_deepwiki

**描述:** iCode DeepWiki 文档知识源

**存储内容:**
- Wiki 文档的 Markdown 文件
- 文档结构按照 Wiki 的目录结构组织

**典型使用场景:**
- 查找技术文档
- 了解项目设计方案
- 查看开发指南

### 2. icode_repo

**描述:** iCode Git 代码仓库

**存储内容:**
- 完整的 Git 仓库克隆
- 包含所有代码文件和目录结构

**典型使用场景:**
- 查看源代码实现
- 分析代码架构
- 查找特定功能的实现

### 3. icafe_card

**描述:** iCafe 卡片/需求/任务

**存储内容:**
- 卡片内容导出为 Markdown 或 JSON 格式
- 包含卡片标题、详情、状态等信息

**典型使用场景:**
- 查看当前迭代的需求
- 了解 Bug 修复情况
- 跟踪任务进度

### 4. iapi_doc

**描述:** iAPI 接口文档

**存储内容:**
- API 接口文档,支持 Swagger 或 Markdown 格式
- 包含接口定义、参数说明、返回值等

**典型使用场景:**
- 查看 API 接口定义
- 了解接口参数和返回值
- 集成第三方服务

### 5. local_knowledge

**描述:** 本地知识文件

**存储内容:**
- 通过软链接或复制的本地文件
- 保持原有的文件结构

**典型使用场景:**
- 引用本地文档
- 集成自定义知识库
- 管理团队共享资料

### 6. ku_doc

**描述:** 知识库知识源

**存储内容:**
- 知识库文档

**典型使用场景:**
- 查看知识库文档
- 自定义知识集成

### 7. icode_cr_diff

**描述：** iCode代码评审变更

**存储内容：**
- 代码评审变更内容导出为 Markdown 格式
- 包含变更前后的代码 diff

**典型使用场景：**
- 查看代码变更详情
- 了解修复问题
- 审查代码

### 8. knowledge_book

**描述：** 知识方舟知识源

**存储内容：**
- 包含知识方舟的所有笔记或文档

**典型使用场景：**
- 集成知识方舟内容


### 9. idataset

**描述：** 数据集知识源

**存储内容：**
- 从 iCode 数据集服务下载的文件
- 保持原始目录结构
- 支持各种文件格式（.jsonl、.xlsx、.md 等）

**典型使用场景：**
- 获取数据飞轮产出的数据集
- 下载训练数据或评测数据
- 获取数据分析结果


## 输出格式

知识内容支持以下输出格式:

- **md (Markdown)**: 适用于文档类知识,易读性强
- **json**: 适用于结构化数据,便于程序处理
- **xml**: 适用于需要 XML 格式的场景
- **swagger**: 专用于 iAPI 文档的 Swagger 格式

## 使用说明文件

每个知识源子目录下都会自动生成或包含一个使用说明文件(*_usage.md),内容包括:

1. **知识项列表**: 该源下所有下载的知识项
2. **存储路径**: 每个知识项的完整存储路径
3. **知识描述**: 知识项的简要说明

**查找使用说明的方法:**
` + "```bash" + `
# 在知识源子目录中查找
ls <basePath>/<source-type>/*_usage.md

# 例如
ls ~/knowledge/icafe_card/*_usage.md
` + "```" + `
`