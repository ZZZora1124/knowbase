---
name: query
version: 1.0.0
description: |
  使用大脑知识通过 3 层搜索、综合和引用传播回答问题。
  当用户提问、需要查找或需要大脑中的信息时使用。
triggers:
  - "我们对...了解什么"
  - "告诉我关于"
  - "...是谁"
  - "发生了什么"
  - "搜索"
  - "查找"
  - "...的背景"
  - "...的笔记"
  - "谁认识谁"
  - "...之间的关系"
  - "关联"
  - "图谱查询"
tools:
  - Bash(knowbase brain search *)
  - Bash(knowbase brain query *)
  - Bash(knowbase brain get *)
  - Bash(knowbase brain list *)
  - Bash(knowbase brain backlinks *)
  - Bash(knowbase brain graph-query *)
  - Bash(knowbase brain timeline *)
  - Bash(knowbase brain timeline-add *)
mutating: false
---

# 查询 Skill

使用大脑知识通过 3 层搜索和综合回答问题。

## 契约

本 skill 保证：
- 每个答案都基于 brain 内容（禁止幻觉）
- 每个论断都有追溯到具体页面 slug 的引用
- 明确标注空白（"大脑中没有关于 X 的信息"）
- 遵循来源优先级（用户陈述 > compiled_truth > 时间线 > 外部）
- 来源冲突时两处都标注

## 各阶段

1. **分解问题**为搜索策略：
   - 关键词搜索：具体姓名、日期、术语
   - 语义查询：概念性问题
   - 结构化查询：列表、反向链接等关系性问题
2. **执行搜索：**
   - `knowbase brain search --timeout 15` 全文检索
   - `knowbase brain query --timeout 15` 混合语义+关键词检索（含扩展）
   - `knowbase brain list --timeout 10` 按类型列出页面（不传 `--limit` 时客户端自动翻页拉全量，避免被服务端默认 100/页截断；只想看前 N 条加 `--limit N`），或 `knowbase brain backlinks --timeout 10` 查结构关系
3. **精读 top 结果。** 读取 top 3-5 页全文获取完整上下文。
4. **综合答案**并附引用。每条论断追溯到具体页面 slug。
5. **标注空白。** 大脑没有相关信息时，说明"大脑中没有关于 X 的信息"，不要幻觉填充。

## 反模式

- 大脑有相关内容时仍用通用知识回答
- 生成大脑中没有的事实
- 来源冲突时静默选一
- 搜索 chunk 已够用时仍加载完整页面
- 忽略来源优先级（用户陈述是最高权威）

## 输出格式

答案应包含：
- 对问题的直接回应
- 引用："据 [来源：people/zhang-san，compiled truth]……"
- 空白标注："大脑中没有关于 X 的信息"
- 来源冲突时标注两处

## 质量规则

- 禁止幻觉。只从 brain 内容回答。
- 引用来源："据 concepts/do-things-that-dont-scale……"
- 标注过期结果：搜索结果显示 [STALE] 时，说明信息可能已过期
- "谁"类问题：用反向链接和有向链接查找关联
- "发生了什么"类问题：使用时间线条目
- "我们了解什么"类问题：直接读取 compiled_truth

## Token 预算意识

搜索返回的是 **chunk**，不是完整页面。先读摘要，再决定是否加载完整页面。

- `knowbase brain search --timeout 15` / `knowbase brain query --timeout 15` 返回带上下文片段的排名 chunk，
  通常足以直接回答问题。
- 只有当 chunk 确认页面相关且需要更多上下文时，才用 `knowbase brain get <slug> --timeout 10` 加载完整页面。
- **"告诉我关于 X"** — 加载完整页面（用户要完整信息）。
- **"有人提过 Y 吗？"** — 搜索结果已够（用户要是/否 + 证据）。

## 来源优先级

多个来源信息冲突时，按以下优先级处理：

1. **用户直接陈述**（最高权威——用户亲口告知的内容）
2. **Compiled truth**（大脑综合、有引用的理解）
3. **时间线条目**（原始证据，逆时间序）
4. **外部来源**（网络搜索、API 丰富——最低权威）

来源冲突时，标注矛盾并附两处引用，不静默选一。

## 答案中的引用

在答案中引用 brain 页面时，传播内联引用：
- 引用页面："据 [来源：people/zhang-san，compiled truth]……"
- brain 页面有内联 `[来源：...]` 引用时，向上传播，让用户追溯事实源头
- 综合多个页面时，引用所有来源

## 图谱遍历（v0.10.1+）

关系类问题（"X 公司谁认识谁？"、"A 和 B 的关联"、"谁在 Acme 工作？"、
"谁参加了站会？"），用图谱层而非全文搜索：

```bash
knowbase brain graph-query <slug> --type <link_type> --depth N --direction in|out|both --timeout 15
```

可用链接类型：`attended`、`works_at`、`invested_in`、`founded`、`advises`、`mentions`、`source`

- `--direction in`：回答"谁指向 X？"（例如谁在 X 公司工作）
- `--direction out`：回答"X 指向什么？"（默认）
- `--depth N`：控制多跳遍历（默认 5）

示例：
```bash
# "谁在 Acme 工作？"
knowbase brain graph-query companies/acme --type works_at --direction in --timeout 15

# "Demo Day W26 有谁参加？"
knowbase brain graph-query meetings/demo-day-w26 --type attended --direction out --timeout 15

# "Emily 建议过哪些公司？"
knowbase brain graph-query people/emily --type advises --direction out --timeout 15

# "Alice 通过会议认识了谁？"
knowbase brain graph-query people/alice --type attended --depth 2 --timeout 15
```
