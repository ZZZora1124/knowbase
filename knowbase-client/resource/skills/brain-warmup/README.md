# brain-warmup

Brain 冷启动 Skill —— 从用户全量数据源采集与按主题整理素材，通过 sidecar（`knowbase brain notify --input-file`）派发给 ducc 子进程抽取并写入 brain。

## 概述

当 brain 为空或用户显式要求时，本 skill 完成两件事：

1. **多源采集**：并行从 Session 对话日志 / 长期记忆 / 知识方舟 / iCafe / 周报 / Ku / 会议 / iCode 等源拉数据，统一归一化与去重
2. **按主题聚类整理**：把跨源材料按主题切成多个 markdown 文件（每个文件内按时间线排序），交给 sidecar 异步处理

支持 **个人模式**（默认）和 **群模式**（设置 `GROUP_ID` 环境变量为纯数字时自动切换）。群模式下的差异：
- 群模式固定使用 2 个 P0 源（如流群消息 / 与本群关联的知识方舟），无需用户二次确认
- 状态文件、中间产物、brain UUID 完全隔离，不污染个人 brain
- cluster md 含 `chat_mode: group` 标识和 `**Speaker**:` 行，供 sidecar 群模式 prompt 处理

抽取语义（提原创思维 / 实体 / 反向链接 / 时间线写入）由 sidecar 端 `signal-detector` 系统提示词承担——本 skill **不再**亲自跑 LLM 抽取或写 brain。

触发方式：`warmup`、`冷启动 brain`、`初始化外脑`、`填充 brain`、`构建外脑`、`给 brain 补知识`、`init brain`

## 执行流程

```
Phase 0  预检查（环境 + 参数落盘）
   ↓
Phase 1  多源并行采集（P0 + P1 数据源）
   ↓
Phase 2  归一化 + 去重（脚本）
   ↓
Phase 3  ★主题聚类（Agent 主对话内决策） + 落盘多 cluster md（脚本）
   ↓
Phase 4  ★逐 cluster 串行派发 sidecar（脚本，--sync）
         knowbase brain notify --input-file <cluster.md> --sync
   ↓
Phase 5  ★跨源洞察 + 整合检查（仍走 sidecar）
         5.1 全局摘要 → notify --input-file phase5_global_summary.md --sync
         5.2 整合检查 → phase5_consolidate.py（去重 / 矛盾 / 补全）
   ↓
Phase 6  完成报告（脚本）
```

> **串行派发语义**：`phase4_dispatch_sidecar.py` 对每个 cluster md 用 `--sync` 调用 `knowbase brain notify`，等 ducc 子进程跑完才派发下一个，避免并发 sidecar 创建重复实体。重复 dispatch 的兜底由三层防御覆盖：Phase 4 在每次派发前清 `~/.knowbase/canonical_index.json` 强制下一个 sidecar 重拉 brain list；`canonical-slug` 子命令 + put 增量缓存让 sidecar 看到前序写入；Phase 5.2 整合检查兜底合并漏网重复。

## 数据源

**P0（核心，默认必抓）**：Session 对话日志 / 长期记忆 / 知识方舟 / iCafe 卡片

**P1（扩展，按需）**：周报 / 知识库（Ku）/ 会议记录 / iCode 提交

> **群模式**（`GROUP_ID` 已设置）：固定使用 2 个 P0 源——如流群消息（infoflow_group_message）/ 与本群关联的知识方舟（knowledge_ark），不支持 P1 源、memory 或其他个人源。

## 目录结构

```
brain-warmup/
├── SKILL.md                         # 主流程定义
├── README.md                        # 本文件
├── references/
│   └── cluster_format.md            # cluster md 文件格式规范
└── scripts/
    ├── run_config.py                # 跨阶段配置读取（共享模块）
    ├── phase0_precheck.py           # Phase 0 预检查
    ├── parse_infoflow_group_message.py # 群消息 md → phase1_raw JSON
    ├── phase2_normalize.py          # Phase 2 归一化 + 去重
    ├── phase3_write_clusters.py     # Phase 3 聚类 JSON → 多 cluster md
    ├── phase4_dispatch_sidecar.py   # Phase 4 批量调 notify --input-file
    ├── phase5_consolidate.py        # Phase 5.2 整合检查
    └── phase6_report.py             # Phase 6 完成报告
```

运行时中间产物：

**个人模式**：
```
~/ark-brain/warmup/{run_id}/
├── run_config.json                  # 运行参数（所有阶段共享）
├── phase1_raw/                      # 各源原始数据
├── phase2_normalized.json           # 归一化后数据
├── phase3_clusters.json             # Agent 决策的聚类 JSON
├── phase3_clusters/                 # 落盘的 cluster md（每主题一个）
│   ├── c001-brain-warmup-refactor.md
│   └── c002-...
├── phase5_global_summary.md         # 跨源洞察输入
├── dispatch.log                     # Phase 4 sidecar 派发记录
└── errors.log
```

**群模式**（`GROUP_ID=12931387`）：
```
~/ark-brain/warmup/group/12931387/{run_id}/
└── ...（同上）
```

## 常用参数

| 参数 | 说明 | 示例 |
|------|------|------|
| `sources` | 指定数据源范围 | `"p0"` / `"icafe,memory"` / `"all"` |
| `since` | 时间过滤 | `"最近 3 个月"` |
| `dry_run` | 只整理不派发，仅影响 Phase 4 的 notify 调用 | `"只整理不派发"` |
| `resume` | 从上次中断处继续 | `"继续上次"` |
| `force` | brain 不为空时强制追加 | `"强制执行"` |

## 环境依赖

| 变量 | 说明 |
|----|------|
| `BAIDU_CC_USERNAME` | sidecar 派发目标方舟 UUID（个人模式必需）|
| `KNOWBASE_BRAIN_UUID` | 目标存储 UUID, 值为BAIDU_CC_USERNAME                  |
| `GROUP_ID` | 群模式开关：存在且为纯数字时进入群模式，`kb_uuid` 取本值 |
| `COMATE_AUTH_TOKEN` | 鉴权 token（必需）|
| `USER_NAME` / `BAIDU_CC_USERNAME` | 用户名（自动读取，下文 `${owner}`）|

## 架构背景

详细设计文档见 `docs/brain-warmup-refactor-spec.md`：
- 为什么把抽取从 skill 移到 sidecar（Agent 上下文压力 / 复用 signal-detector / 复用 notify 通道）
- cluster md 文件格式与时间线分隔约定
- fire-and-forget 派发与 Phase 5 启动时机的取舍
