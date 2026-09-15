---
name: brain-warmup
description: >
  Brain 冷启动 Skill（链路 D / Warmup）。在 brain 为空或显式要求时，
  专注从用户全量数据源（Session 对话日志 / 长期记忆 / 知识方舟 / iCafe /
  iCode / Ku / 周报 / 会议记录等）批量采集与按主题整理素材，
  然后通过 `knowbase brain notify --input-file` sidecar 通道交给
  ducc 子进程（运行 signal-detector）抽取知识、写入 brain 并维护反链。
  跨源洞察同样走 sidecar，统一抽取链路。
  触发关键词：warmup、冷启动 brain、初始化外脑、填充 brain、
  构建外脑、给 brain 补知识、init brain。
---

# brain-warmup — Brain 冷启动

从用户全量数据源批量采集素材并按主题整理为 markdown 文件，通过 sidecar （`knowbase brain notify --input-file`）派发给 ducc 子进程抽取与写入 brain。

---

## 环境依赖（执行前确认）

| 变量 / 路径 | 说明                                              |
|------------|-------------------------------------------------|
| `USER_NAME`（首选）/ `BAIDU_CC_USERNAME`（次选）| 当前用户（自动读取，下文用 `${owner}` 指代）                    |
| `BAIDU_CC_USERNAME` | 目标存储 UUID（sidecar 派发必需，缺失时整个流程不能开始）             |
| `KNOWBASE_BRAIN_UUID` | 目标存储 UUID, 值为BAIDU_CC_USERNAME                  |
| `GROUP_ID` | **群模式触发开关**：环境变量存在且为纯数字时进入群模式（`scope=group`），此时 `kb_uuid` 取本值（覆盖 `BAIDU_CC_USERNAME`）；非数字时 phase0 报错退出 |
| `COMATE_AUTH_TOKEN` | 鉴权 token（sidecar 派发必需）                          |
| `~/.claude/projects/<项目目录>/*.jsonl` | Session 对话日志（P0 源）                              |
| `memory_search` / `memory_get` MCP | 长期记忆（P0 源）                                      |
| `knowledge-manage` skill | 知识方舟（P0 源）                                      |
| `icafe-assistant` skill | iCafe 卡片（P0 源）                                  |
| `knowbase` CLI | sidecar 派发通道（`knowbase brain notify --input-file`） |
| 中间产物目录 | `~/ark-brain/warmup/`（自动创建）                     |

---

## 参数解析（自然语言 → 运行配置）

| 参数 | 默认值 | 示例触发 |
|------|--------|---------|
| `sources` | personal 模式无默认值，必须由用户显式选择；group 模式固定为 `infoflow_group_message,knowledge_ark` | "只用 iCafe 和记忆"；`p0` = 仅 P0 源；`all` = P0+P1 全量 |
| `since` | 不限 | "最近 3 个月" → `2026-02-12` |
| `max_items` | 不限 | "最多 100 条" |
| `dry_run` | `false` | "只整理不派发"；仅影响 Phase 4 的 notify 调用 |
| `resume` | `false` | "继续上次"，跳过已存在的 phase1_raw / phase3_clusters |
| `force` | `false` | "强制执行，即使 brain 已有数据" |
| `mode` | `auto` | "强制 cold-start"；可选 `auto` / `cold-start` / `refine` |
| `force_warmup` | `false` | "退役后强制重跑"；绕过 retired 状态临时跑一次 cold-start |
| `max_cache_age` | `7d` | "缓存有效期 14 天"；refine 模式 phase2_normalized.json 新鲜度阈值 |

---

## 群模式 vs 个人模式

brain-warmup 通过 `GROUP_ID` 环境变量自动切换模式：

| 维度 | personal（默认） | group（`GROUP_ID` 非空时） |
|------|-----------------|--------------------------|
| brain UUID | `BAIDU_CC_USERNAME` | `GROUP_ID` |
| owner 字段 | `${USER_NAME}` 或 `${BAIDU_CC_USERNAME}` | `group:${GROUP_ID}` |
| Phase 1 可选源 | 全部 8 个（4 个 P0 + 4 个 P1） | 固定 2 个 P0：`infoflow_group_message / knowledge_ark` |
| 状态文件 | `~/.knowbase/warmup_state.json` | `~/.knowbase/warmup_state.group.${GROUP_ID}.json` |
| 中间产物目录 | `~/ark-brain/warmup/{run_id}/` | `~/ark-brain/warmup/group/${GROUP_ID}/{run_id}/` |
| sidecar chat-mode | personal（auto-detect 字母 UUID） | group（auto-detect 纯数字 UUID） |
| cluster md frontmatter | 不含 chat_mode | 含 `chat_mode: group` / `group_id: ...` |
| cluster md 每 item | 仅 `> author:` 注释 | 额外 `**Speaker**: <name>` 行（供 sidecar 抽 originals 归属） |

### 群模式语义补充

- **群模式不采集 memory**：memory 视为个人源，避免把用户个人记忆灌入群 brain；群上下文长期沉淀统一由群消息与群关联知识方舟覆盖
- **knowledge_ark 仅限于与本群关联的知识方舟**：agent 必须只拉取与本群关联的方舟，**绝对禁止访问用户个人的知识方舟**。如果没有与本群关联的方舟，静默跳过该源。
- **群历史消息抓取**：走 `knowledge-manage` skill 的 `infoflow_group_message` 类型 YAML，落到 `~/knowledge/infoflow_group_message/<群名>_<群id>.md`，再执行 `python3 scripts/parse_infoflow_group_message.py --input <md> --output <run_dir>/phase1_raw/infoflow_group_message.json --group-id ${GROUP_ID}` 解析回 JSON
- **infoflow_group_message 0 条也正常**：群在指定时间窗内可能本就没人发言，phase2 仅 stderr 友好提示，不报错不回退；可继续用 knowledge_ark 推进；若两个群源都为空，则正常产出空报告
- **cluster md Speaker 字段对接** sidecar 群模式 spec：sidecar 按 `Speaker: unknown` 跳过 originals 抽取（仅抽实体），这正是 knowledge_ark 在群 brain 里应有的语义
- **跑群 warmup 不会污染个人 brain**：状态文件与中间产物完全隔离，群 retire 不影响个人 warmup，个人 retire 也不影响新接入的群

---

## 执行流程

```
Phase 0  预检查（脚本：phase0_precheck.py）
   ↓
Phase 1  多源并行采集（Agent 并行调各 source skill）
   ↓
Phase 2  归一化 + 去重（脚本：phase2_normalize.py）
   ↓
Phase 3  主题聚类（Agent 主对话内决策） + 落盘多 cluster md
         （脚本：phase3_write_clusters.py）
   ↓
Phase 4  逐 cluster 串行派发 sidecar（--sync 等待每个完成）
         （脚本：phase4_dispatch_sidecar.py，调 notify --input-file --sync）
   ↓
Phase 5  整合检查：去重合并 + 矛盾检测 + 交叉补全
         （脚本：phase5_consolidate.py，也走 sidecar --sync）
   ↓
Phase 6  完成报告（脚本：phase6_report.py）
```

### 模式与状态

brain-warmup 维护一个三档状态机，状态持久化到 `~/.knowbase/warmup_state.json`：

- **cold-start**：跑全部 Phase 1-6，重新采集所有数据源。首跑、brain 很小（< 5）、或上次 outcome=high 时进入。
- **refine**：跳过 Phase 1，复用上次的 `phase2_normalized.json`，重新组织视角以发掘新关联。brain ≥ 30 且 cold-start 收益开始衰减时滑入此档。
- **retired**：拒绝执行并提示用户切日常 signal-detector 维护。两条退役兜底任一触发即进入：
  - refine 连续 2 次 outcome=none（彻底无产出）
  - 任意 mode 连续 3 次 outcome 非 high（性价比衰减，防 cold-start 反复低产或 refine 反复 low 的慢性循环）

切换全自动，用户可用 `--mode` 或 `--force-warmup` 覆盖。

### Phase 0：预检查

运行 `python3 scripts/phase0_precheck.py`（无变化）：

1. 读 `BAIDU_CC_USERNAME`（必需，缺失时立即报错退出，不进入后续阶段）；读 `${owner}`（USER_NAME → BAIDU_CC_USERNAME）
2. 检查 brain 当前条目数（< 10 条才继续，或 `--force`）
3. 创建 `~/ark-brain/warmup/{run_id}/`
4. 把所有运行参数写到 `run_config.json`，后续阶段从该文件读

### Phase 1：多源并行采集

Agent 在同一条消息内**并行**调各源 skill 抓取，落盘到 `phase1_raw/{source}.json`。

> **入口约定**：Phase 1 第一步读 `run_config.json`，按其中 `sources / since / max_items / resume / owner` 驱动；不再从用户消息重新推断。

#### 🚨 强制：个人模式采集前必须让用户显式选择源

进入 Phase 1 实际抓取之前，personal 模式**禁止使用任何默认值（如 `all` / `p0`）静默兜底**。Agent 必须先列出所有可选源，请用户显式选择或自定义后再继续。

> **群模式（`GROUP_ID` 已设置）**：不询问用户选择，`phase0_precheck.py` 固定写入 `sources=infoflow_group_message,knowledge_ark`。群模式只采集群历史消息与本群关联知识方舟，明确排除 memory 和所有个人源。

执行步骤：

1. **展示可选源清单**：必须**逐项明确列出**全部 8 个源的名称、优先级与契约摘要，禁止只用 `p0` / `p1` / `all` 等代号代替——用户无法据此判断要选什么。展示内容至少包含：
   - **P0 源**（核心数据）：
     - `session`：本地 Claude 对话日志（`~/.claude/projects/<项目目录>/*.jsonl`）
     - `memory`：长期记忆（memory_search / memory_get MCP）
     - `knowledge_ark`：知识方舟 entry 正文（knowledge-manage skill）
     - `icafe`：iCafe 卡片 title + description + 评论（icafe-assistant skill）
   - **P1 源**（扩展数据）：
     - `weekly`：周报正文（weekly skill）
     - `ku`：Ku 文档完整 markdown（ku-doc-manage skill）
     - `meeting`：会议主题 / 参会人 / ASR 或 AIGC 总结（calendar-meeting skill）
     - `icode`：commit 完整 message（icode skill）
   - **群模式特化**（`run_config.scope == "group"`）：跳过源选择询问，直接使用以下 2 个 P0 源，并提示"群模式不接收个人源；如需灌入个人数据请 unset GROUP_ID 后重跑"：
     - `infoflow_group_message`：群 `${GROUP_ID}` 的历史消息（走 knowledge-manage skill 拉取，落 `~/knowledge/infoflow_group_message/...`）
     - `knowledge_ark`：与本群关联的知识方舟（仅拉取群关联方舟，禁止访问用户个人方舟；无关联方舟时静默跳过）
   - personal 模式同时提示用户两种选择方式：
     - **预设**：`all`（全部 8 个源）或 `p0`（仅 4 个 P0 源）——但必须先把上面的具体源名展示给用户，再说"也可以用 all/p0 简写"
     - **自定义**：列出想抓取的具体源名（如 `session,memory,icafe`）
2. personal 模式**调用 `AskUserQuestion` 工具**询问用户选择，且 **不提供默认勾选项**；提问选项中必须把上述源名直接写出来，**不允许只写"P0 全部"/"所有源"而不展开**；用户未作答前 Phase 1 不得开始抓取。group 模式跳过该步骤。
3. personal 模式下用户给出选择后，将最终 `sources` 列表回写到 `run_config.json`；group 模式直接复用 Phase 0 写入的固定 `sources`，再按下文契约并行抓取。
4. personal 模式若 `run_config.json` 中已有非空 `sources`（例如 `--resume` 续跑或用户在 Phase 0 已明确指定），可跳过本次询问，直接复用；否则一律必须询问。group 模式永远使用固定 `sources=infoflow_group_message,knowledge_ark`。

> **目的**：避免 brain-warmup 在用户无感知的情况下大规模拉取敏感源（如 session 对话日志、ku 内部文档），保证数据采集范围始终在用户授权范围内。

#### 🚨 铁律：content 必须是"详情正文"，不是"列表摘要"

sidecar 端 signal-detector 是从 `content` 中提取原创思维 / 实体 / 事实，**列表接口返回的 title / description 摘要远远不够**。

| # | 源 | 输出文件 | 依赖 skill / 数据来源                                                                    | content 应包含 |
|---|---|---|------------------------------------------------------------------------------------|---|
| P0(group) | infoflow_group_message | `infoflow_group_message.json` | `knowledge-manage` skill（type=`infoflow_group_message`，groupId=`${GROUP_ID}`）；落 `~/knowledge/infoflow_group_message/<群名>_<群id>.md` 后由 `scripts/parse_infoflow_group_message.py` 解析回 JSON | 单条群消息正文 + speaker（透传 `userName`）+ msg_time_ms；多条按时间排序；支持 `消息内容:` 后多行正文；0 条也正常不报错 |
| P0 | session | `session.json` | `~/.claude/projects/<项目目录>/*.jsonl` 整文件                                            | 用户提问 + assistant 回复正文（保留原文，不做单句裁剪）|
| P0 | memory | `memory.json` | `memory_search` / `memory_get` MCP                                                 | memory 条目正文 |
| P0 | knowledge_ark | `knowledge_ark.json` | `knowledge-manage` skill（排除公开类型方舟,查询方舟列表时，必须指定--visibility,可选值private,personal,比如`knowbase notebook list --visibility private`） | 知识方舟 entry 的正文 markdown。**群模式**：仅限于与本群关联的方舟，禁止拉取用户个人方舟；无关联方舟时静默跳过 |
| P0 | icafe | `icafe.json` | `icafe-assistant` skill                                                            | title + description + 关键评论 + 状态变更（不是单纯 title）|
| P1 | weekly | `weekly.json` | weekly skill                                                                       | 整篇周报正文 |
| P1 | ku | `ku.json` | `ku-doc-manage` skill                                                              | Ku 文档的完整 markdown 正文（不是列表 description）|
| P1 | meeting | `meeting.json` | `calendar-meeting` skill                                                           | 必含：会议主题；起止时间（北京时间）；组织者，写作 `姓名 <邮箱>`；参会人列表，每人写作 `姓名 <邮箱>`，**缺邮箱视为不合格**；ASR 全文 **或** AIGC 总结/决议/待办，二者至少一项有正文。ASR 与 AIGC 都为空时**丢弃该条**|
| P1 | icode | `icode.json` | icode skill                                                                        | commit 完整 message（含 body），如可控可附变更摘要 |

> 上表只规定**契约**：用哪个 skill、content 里要拿到什么。具体调用哪个子命令/接口由各源 skill 自行决定，本 skill 不约束实现路径

**Phase 1 自检（落盘前 Agent 必须执行）**：

按源统计 `len(content)` 的均值与最小值，若任一 P0 源出现 `mean(content) < 200` 或半数以上 item `len(content) < 100`，**回退**重新走详情接口；并在 `errors.log` 记一行 `phase1.<source>: avg_content=NN, suspected list-only fetch`。

**字段约定**：每条至少含 `content / source_id / date`；源特有字段放顶层（不要硬塞进 content），Phase 2 会自动收纳到 `extra` 子对象。

**邮箱字段必须保留**：源数据中包含的邮箱信息必须保留（icafe 的 creator/assignee 邮箱、icode 的 author 邮箱、meeting 的 attendees 邮箱列表等），禁止丢弃 email 只保留 name。邮箱前缀是下游人名去重的唯一可靠标识。

**敏感数据双重防线**：Agent 在 Phase 1 过滤一次（`password=` / 私钥块等）；Phase 2 脚本再做正则兜底，命中即丢弃并写 `errors.log`。

### Phase 2：归一化 + 去重

运行 `python3 scripts/phase2_normalize.py --run-id {run_id}`（无变化）。

输出 `phase2_normalized.json`，每条统一为：

```json
{
  "source": "session|memory|knowledge_ark|icafe|...",
  "source_id": "唯一 ID",
  "content": "正文",
  "date": "YYYY-MM-DD",
  "author": "${owner}",
  "url": "可选",
  "tags": [],
  "content_hash": "sha256(content)",
  "extra": { "source 特有字段..." }
}
```

去重：`content_hash` 跨源去重。slug 级去重在 sidecar 端由 canonical_index 完成；语义级合并由 Phase 5.2 整合检查兜底。

### Phase 3：主题聚类 + 落盘多 cluster md

> **聚类决策由 Agent 在主对话内完成**（语义相似度只能 LLM 判断）；
> **机械落盘由脚本完成**（保证格式稳定）。

#### 3.1 Agent 步骤

1. 读 `~/ark-brain/warmup/{run_id}/phase2_normalized.json`
2. 扫描每条 item 的 `content`（必要时参考 `extra`），按主题/项目/人物/问题做聚类划分
3. 输出 `phase3_clusters.json`（落到 run 目录），结构：

```json
[
  {
    "cluster_id": "c001",
    "topic": "brain-warmup 架构重构",
    "topic_slug": "brain-warmup-refactor",
    "source_ids": ["icafe:DEV-3945", "session:abc.jsonl#42", "memory:m-1234", "weekly:w-2026w19"]
  },
  {
    "cluster_id": "c002",
    "topic": "...",
    "topic_slug": "...",
    "source_ids": ["..."]
  }
]
```

**聚类原则**：
- 单个 cluster item 数 ≤ 30，超过则按子主题切
- 单条 item 自成主题允许（孤立但有价值）
- 实在无法分类 → 归为 `c-misc`

#### 3.2 脚本落盘

```bash
python3 scripts/phase3_write_clusters.py \
  --run-id {run_id} \
  --clusters-json phase3_clusters.json
```

脚本会：
1. 读 `phase3_clusters.json` + `phase2_normalized.json`
2. 按 `source_id` 关联到原文，按 `date` 升序排
3. 每个 cluster 输出 `phase3_clusters/{cluster_id}-{topic_slug}.md`
4. 检查重复 source_id、未引用 item、缺失引用，并打印警告

文件格式见 [`references/cluster_format.md`](references/cluster_format.md)。要点：
- frontmatter 含 `type: warmup-cluster / cluster_id / topic / sources / date_range / item_count / run_id`
- 正文以 `# 主题：...` 开头并提示 sidecar"按 signal-detector 规则处理"
- 时间线段落固定 `### YYYY-MM-DD [source:source_id]` 分隔，原文不转述、不截断

**群模式特化**（`run_config.scope == "group"`）：
- frontmatter 增加 `chat_mode: group` 与 `group_id: ${GROUP_ID}` 两行
- 顶部提示扩展为"按 signal-detector 群模式规则处理：Speaker: unknown 的 item 不抽取 originals，仅抽实体；originals 必须含 `author: <speaker>` frontmatter，timeline 每条带 `by <speaker>`"
- 每个 item 段头紧贴 `### YYYY-MM-DD [source:source_id]` 之后增加一行 `**Speaker**: <name>`
  - infoflow_group_message：speaker = 原 userName（含 `_Bot` 后缀、邮箱前缀、中文括号等所有形态，原样透传）
  - knowledge_ark：speaker = `unknown`（sidecar 自动跳过 originals 抽取，仅抽实体）
- 与 sidecar 群模式 spec 的 group prompt fragment 对接：sidecar `DetectChatMode` 对纯数字 `--kb-uuid` auto-detect 为 group，自动拼接 group prompt fragment 处理 cluster md

### Phase 4：sidecar 派发（默认有限并发）

```bash
python3 scripts/phase4_dispatch_sidecar.py --run-id {run_id} [--concurrency N] [--dry-run]
```

脚本对每个 `phase3_clusters/*.md` 调用：

```bash
knowbase brain notify --input-file <path> --kb-uuid $BAIDU_CC_USERNAME --sync
```

**并发模型**：默认 `--concurrency 3`，用 ThreadPoolExecutor 同时跑 3 个 sidecar；可调大以加速，可设 `--concurrency 1` 退化为完全串行。每个 sidecar 内部 `--sync` 等 ducc 完成才退出，所以单个 sidecar 的 brain put 是顺序的。

**关键语义（--sync）**：使 `knowbase brain notify` 等待 ducc 子进程完成才返回。同一个 sidecar 自己的连续写入有序，跨 sidecar 之间靠 canonical-slug 缓存协作。

**Canonical 缓存生命周期**：Phase 4 入口处一次性 `unlink ~/.knowbase/canonical_index.json`，让所有 sidecar 从同一 brain 基线开始查重；之后每个 sidecar 第一次调 `canonical-slug` 触发一次 `brain list` 重建缓存，后续 put 增量追加。

**并发的代价**：同时启动的几个 sidecar 看不到对方刚写入的实体，可能各自创建同一实体的页面（小概率）。Phase 5 整合检查（任务 1 去重合并）兜底处理。需要绝对零重复时用 `--concurrency 1`。

每行写到 `dispatch.log`：`[2026-05-13T14:32] dispatched cluster=c001 file=... rc=0`，失败写 `errors.log`。

`--dry-run` 仅打印命令，不实际派发。

### Phase 5：跨源洞察 + 整合检查

Phase 4 使用 `--sync` 串行派发，所有 cluster sidecar 已完成后才进入 Phase 5。

#### 5.1 跨源洞察（Agent 步骤）

1. 汇总每个 cluster 的 `topic / 主要源 / 时间跨度 / 关键事件` 到 `phase5_global_summary.md`：

```markdown
---
type: warmup-global-summary
run_id: 20260513-1430
cluster_count: 8
purpose: insight-synthesis
---

# Warmup 全局摘要 — 跨源综合洞察素材

请基于横向综合视角，从下方 cluster 概要中识别：
- 核心关注域、协作网络、决策倾向、知识盲点、矛盾张力、时间线趋势
- 输出写入 `insights/warmup-{topic}-{date}` 类型页面，按 signal-detector 规则维护反向链接

## Cluster 概要

### c001 brain-warmup 架构重构
- 时间跨度：2026-04-15 ~ 2026-05-13
- 来源：icafe, session, memory, weekly
- 关键实体：DEV-3945, signal-detector, ducc
- 关键事件：...

### c002 ...
...
```

2. 调一次 sidecar：

```bash
knowbase brain notify --input-file phase5_global_summary.md --kb-uuid $BAIDU_CC_USERNAME --sync
```

#### 5.2 整合检查（去重合并 + 矛盾检测 + 交叉补全）

```bash
python3 scripts/phase5_consolidate.py --run-id {run_id} [--dry-run]
```

跨源洞察完成后，运行整合检查作为兜底 + 增值：

脚本自动完成：
1. 调 `knowbase brain list` 获取当前 brain 全量页面（CLI 不传 `--limit` 时客户端自动按 offset 翻页，不会被服务端默认 100/页截断）
2. 读取本次 run 的 cluster topics
3. 生成 `phase5_consolidation.md` 整合检查指令文件
4. 通过 `notify --input-file --sync` 派发给 ducc 执行

Sidecar 执行的三项任务：

**任务 1 — 去重合并**：
- 识别指向同一实体的重复页面（slug 不同但实际同一人/项目/概念）
- 合并 timeline 和 compiled_truth 到更完整的页面
- 删除冗余页面

**任务 2 — 矛盾检测**：
- 不同页面间的信息冲突（角色/职责/时间线矛盾）
- 根据来源可信度和时间新旧修正

**任务 3 — 交叉补全**：
- 识别 compiled_truth 过于稀薄的页面
- 从其他页面 timeline 中提取相关信息整合

### Phase 6：完成报告

运行 `python3 scripts/phase6_report.py --run-id {run_id}`。

输出：

```
🧠 Brain Warmup 完成报告（{stage_label} 阶段）
═══════════════════════════════════════
🆔 run_id：{run_id}

📂 数据源（原始条目）：
  Session 对话日志：N 条
  ...
  归一化 + 去重后：N 条有效

🧩 主题聚类与 sidecar 派发：
  Phase 3 聚类：N 个 cluster md
  Phase 4 派发：成功 X / 失败 Y / 总计 Z

📈 本次发散收益（A=实体增量，B=信号增量）：
  ...

💡 Phase 5 洞察：...

⚠️ 错误记录：...

📁 中间产物：~/ark-brain/warmup/{run_id}/
```

> 本次发散收益由 `phase6_report.py` 通过 `knowbase brain list` 对比 + `dispatch.log` 中的 `signals=N` 聚合得到，并写入 `~/.knowbase/warmup_state.json` 的 `recent_runs` 数组（FIFO 10）。

---

## 错误处理

- **Phase 0 brain 不为空**：提示用户确认追加/覆盖
- **单个源抓取失败**：写 `errors.log`，继续其他源
- **`phase3_clusters.json` 缺失或非数组**：脚本立即退出（exit 2），提示重新生成
- **某个 cluster sidecar 派发失败**：记录到 `dispatch.log` + `errors.log`，继续其他 cluster
- **`--resume`**：跳过已存在的 `phase1_raw/{source}.json` 与 `phase3_clusters/`；sidecar 端 `knowbase brain put` 本身幂等

---

## 文件结构

```
scripts/
├── phase0_precheck.py             # Phase 0
├── parse_infoflow_group_message.py # 群消息 md → phase1_raw JSON
├── phase2_normalize.py            # Phase 2
├── phase3_write_clusters.py       # Phase 3 落盘
├── phase4_dispatch_sidecar.py     # Phase 4 串行派发（--sync）
├── phase5_consolidate.py          # Phase 5.2 整合检查
└── phase6_report.py               # Phase 6

references/
├── cluster_format.md              # cluster md 格式规范

~/ark-brain/warmup/{run_id}/
├── run_config.json                # Phase 0 落盘
├── phase1_raw/                    # Agent 抓取
│   ├── session.json
│   ├── memory.json
│   ├── knowledge_ark.json
│   ├── icafe.json
│   └── ...（P1 源按需）
├── phase2_normalized.json         # Phase 2
├── phase3_clusters.json           # Agent 决策的聚类 JSON
├── phase3_clusters/               # 落盘的 cluster md
│   ├── c001-brain-warmup-refactor.md
│   ├── c002-...
│   └── c-misc-others.md
├── phase5_global_summary.md       # Phase 5.1 跨源洞察输入
├── phase5_consolidation.md        # Phase 5.2 整合检查输入
├── dispatch.log                   # Phase 4 派发记录
└── errors.log
```
