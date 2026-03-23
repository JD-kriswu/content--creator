# 口播稿助手 Multi-Agent 全量重构设计文档

**日期**: 2026-03-23
**状态**: 已确认，待实施
**参考**: `assert/refactor.tx`

---

## 背景与目标

当前系统将5个"角色"内嵌在单一大 Prompt 中（伪并行），缺乏真正的 Multi-Agent 架构、质量审核闭环和反馈学习机制。

本次重构目标：
1. 实现真正并行的 2-Agent 分析（爆款解构师 + 素材补齐师），风格建模师只读参与辩论
2. 引入 Multi-Agent 辩论协调层（1轮）
3. 建立质量审核闭环（事实 + 逻辑 + 表达 + 相似度）
4. 支持用户风格建模（从历史稿提取8D量化向量，一次性初始化，独立于会话状态机）
5. 反馈学习机制（用户输入 → 更新各 agent 可进化 prompt）

---

## 新流程总览

```
[用户首次 — 独立于会话状态机]
POST /api/user/style/init（独立 SSE 流）
    → 风格建模师 → 生成《人设风格说明书》(8D向量)
    → UserStyle.is_initialized = true

[后续每次会话 — 走 chat/message 状态机]
发送链接/文案
    ↓
Step1: 提取文本（URL → HTML解析）
    ↓
Step2: 读取《人设风格说明书》（UserStyle）
    ↓
Step3: 2 Agent 真并行分析（流式输出各自结果）
    ① 爆款解构师 — 分析目标稿 DNA（结构/内容/情感/表达）
    ② 素材补齐师 — 补充新素材和数据
    ↓
Step4: Multi-Agent 辩论协调（1轮）
    爆款解构师发言 × 素材补齐师发言 × 协调者综合（输入含人设风格只读数据）
    → 辩论消息流（SSE debate 事件）→ 融合大纲（OutlineData JSON）
    ↓
Step5: 大纲展示，等待用户确认
    用户可输入: 1(确认) / 2(调整备注) / 3(更换素材) / 4(重新分析)
    ↓
Step6: 创作代理生成初稿（流式）
    输入: 大纲 + 爆款DNA报告 + 人设风格说明书 + 素材包
    ↓
Step7: 质量审核闭环（串行，最多重试2次）
    → 相似度检测（算法，阈值 <30%）
    → 事实核查（LLM，阈值 >90%准确）
    → 逻辑检查（LLM，阈值 >85%连贯）
    → 表达评分（LLM，阈值 >75分）
    任一不通过 → 将问题反馈给创作代理 → 重生成（最多2次）
    若创作代理在重试中报错 → 中止，使用上次成功草稿，标记 Passed=false
    ↓
Step8: 最终输出
    → 保存稿件（Script）+ 质量报告（QualityReport）
    → 收集用户最终反馈（文字评价） → feedback_processor → 更新 AgentConfig
```

---

## 数据模型变化

### UserStyle（扩展现有表）

新增字段：
```sql
style_vector        TEXT        -- JSON, 8维向量 {"authority":0.8,"affinity":0.6,...}
style_doc           LONGTEXT    -- 《人设风格说明书》全文
historical_scripts  LONGTEXT    -- JSON数组，最多保留10篇历史稿原文，超出时滚动替换最旧一篇
is_initialized      BOOL        -- default false，控制 Onboarding 引导
style_version       INT         -- 每次反馈更新后自增
```

8维向量字段：`authority`（权威度）、`affinity`（亲和力）、`expertise`（专业深度）、`humor`（幽默感）、`risk`（风险偏好）、`emotion`（情感强度）、`interaction`（互动性）、`storytelling`（叙事性）

### AgentConfig（新增表）

```go
type AgentConfig struct {
    ID             uint      `gorm:"primaryKey"`
    UserID         uint      `gorm:"index"`
    AgentName      string    `gorm:"size:64"` // style_modeler/viral_analyzer/material_enricher/creator/reviewer
    PromptTemplate string    `gorm:"type:longtext"`
    Version        int
    CreatedAt      time.Time
}
```

> 每个用户拥有独立的 agent prompt 版本。反馈触发时生成新版本，旧版本保留（可回滚）。
> 若用户无自定义版本，使用系统默认 prompt（`user_id=0` 或 hardcoded fallback）。

### QualityReport（新增表）

```go
type QualityReport struct {
    ID              uint    `gorm:"primaryKey"`
    ScriptID        uint    `gorm:"index"`
    SimilarityScore float64
    FactScore       float64
    LogicScore      float64
    ExpressionScore float64
    Passed          bool
    RetryCount      int
    Issues          string  `gorm:"type:text"` // JSON数组，各项问题描述
    CreatedAt       time.Time
}
```

### Conversation（新增字段）

```sql
+ debate_log    LONGTEXT    -- 辩论过程最终结果JSON（非流式中间数据），可供历史回溯
```

---

## 后端服务层重构

### 目录结构

```
backend/internal/service/
├── pipeline.go              会话状态机主干（调用下列模块）
├── agent_runner.go          并行执行多个 agent，goroutine + channel 汇聚结果
├── style_modeler.go         风格建模师（初始化建模 + 反馈更新，独立于状态机）
├── viral_analyzer.go        爆款解构师（分析目标稿 DNA）
├── material_enricher.go     素材补齐师（补充新素材数据）
├── debate_coordinator.go    辩论协调（1轮3方 LLM 对话 → 融合大纲）
├── creator_agent.go         创作代理（生成初稿）
├── quality_gate.go          质量审核闭环（4项检查 + 重试逻辑）
├── feedback_processor.go    反馈整理 → AgentConfig 版本化更新
├── llm_service.go           （保留）CallClaude / StreamClaude
├── extractor.go             （保留）URL 提取
└── prompts.go               （重构）各 agent 默认 prompt，函数返回，支持 AgentConfig 覆盖
```

新增 handler：
```
backend/internal/handler/style_handler.go   处理 POST /api/user/style/init（独立 SSE 流，不走 chat session）
```

### 状态机（新增2个状态）

**风格建模初始化不经过 chat session 状态机**，由独立 `style_handler.go` 处理。

会话状态机状态（仅用于改写流程）：
```
StateIdle
    → StateAnalyzing   (Step3: 并行分析)
    → StateDebating    (Step4: 辩论协调)  ← 新增
    → StateAwaiting    (Step5: 等待用户确认大纲)
    → StateWriting     (Step6: 创作初稿)
    → StateReviewing   (Step7: 质量审核)  ← 新增
    → StateComplete
```

`handleIdle` 入口判断：若 `UserStyle.is_initialized == false`，直接返回 SSE error 消息，提示用户先完成风格建模（调用 `/api/user/style/init`），不进入 Analyzing 流程。

重入恢复机制扩展：
- `StateAnalyzing` 超时3分钟 → 降级 `StateIdle`（重新提交 URL 重跑）
- `StateDebating` 超时3分钟 → 降级 `StateIdle`（辩论阶段失败代价低于并行分析，一并重跑更简单）
- `StateWriting` 超时3分钟 → 降级 `StateAwaiting`
- `StateReviewing` 超时5分钟 → 降级 `StateAwaiting`

### agent_runner.go（并行执行）

```go
type AgentTask struct {
    Name   string // agent 标识，用于 SSE event agent 字段
    System string // system prompt
    Prompt string // user prompt
}

type AgentResult struct {
    AgentName string
    Content   string
    Error     error
}

// RunParallel 并发执行多个 agent，每个 token 通过 sseCallback 推送
// 各 agent goroutine 独立调用 StreamClaude，结果汇聚后按 AgentName 返回
// 若某 agent 超时（60s）或报错，其 AgentResult.Error != nil，Content 为空字符串
// pipeline 调用方应检查 Error 字段：debate_coordinator 对空 Content 的 agent 以"无报告"占位，不中止流程
func RunParallel(agents []AgentTask, sseCallback func(AgentResult)) []AgentResult
```

### debate_coordinator.go（辩论协调）

1轮辩论流程（每步均以 `StreamClaude` 流式输出，通过 SSE `debate` 事件推送）：

1. **输入汇总**：爆款DNA报告（viral_analyzer输出）+ 素材包（material_enricher输出）+ 人设风格说明书（UserStyle.style_doc，只读）
2. **爆款解构师发言**：system=爆款解构师角色，prompt=DNA报告+主张保留核心结构的指令
3. **素材补齐师发言**：system=素材补齐师角色，prompt=素材包+爆款解构师发言+主张插入新内容的指令
4. **协调者综合发言**：system=协调者角色，prompt=三方全部报告+辩论发言+人设风格，要求输出融合方案，**末尾必须输出 `---OUTLINE_START---{JSON}---OUTLINE_END---` 标记**

OutlineData JSON Schema（与现有保持一致，无变化）：
```json
{
  "elements": ["要素1"...],
  "materials": ["素材1"...],
  "outline": [{"part":"开场","duration":"3s","content":"...","emotion":"..."}...],
  "estimated_similarity": "约15%",
  "strategy": "改写核心策略"
}
```

辩论结束后，将3条发言的完整文本 JSON 存入 `Conversation.debate_log`（非中间流式数据）。

### quality_gate.go（质量审核闭环）

```go
type QualityCheckResult struct {
    SimilarityScore float64
    FactScore       float64
    LogicScore      float64
    ExpressionScore float64
    Passed          bool
    Issues          []string
}

// RunQualityGate 执行4项检查，返回最终稿和综合结果
// 若未通过且 retry < maxRetry，调用 creator_agent 重生成
// 若 creator_agent 在重试中报错，中止重试，返回上次成功草稿，Passed=false
func RunQualityGate(draft string, original string, maxRetry int, sseCallback SSECallback) (finalDraft string, result QualityCheckResult, err error)
```

检查顺序：相似度（最快，不通过直接重试）→ 事实核查 → 逻辑检查 → 表达评分

### feedback_processor.go（反馈学习）

触发时机：
- 用户在 StateAwaiting 中输入非数字内容（大纲修改意见）
- 用户通过 `POST /api/scripts/:id/feedback` 提交对终稿的文字评价

"手动修改终稿"说明：MVP 阶段不提供在线编辑器，用户将自行修改后的稿件文本通过 feedback 接口以 `content` 字段提交，系统将其作为优化参考而非替换稿件内容。

处理逻辑：
```go
// ProcessFeedback 汇总用户反馈，更新对应 agent 的 AgentConfig
// 生成新版本 prompt，旧版本保留（可回滚到任意历史版本）
func ProcessFeedback(userID uint, feedbackType string, content string) error
```

---

## SSE 新增消息类型

| type | 字段 | 前端处理 |
|------|------|---------|
| `debate` | `agent(string), content(string)` | 按 agent 分 3 色气泡（爆款=蓝/素材=绿/协调=橙）；每条 `debate` 事件携带单个 token，前端按 agent 字段路由追加到对应气泡 |
| `quality` | `data(QualityReport)` | 质量报告卡片（4项评分 + 通过/重试状态），`Issues` 字段为 `[]string`，每条为人类可读的中文问题描述（如"第3段数据引用年份有误"），不含结构化子字段 |
| `style_init` | `content(string)` | 来自 `/api/user/style/init` SSE 流的**进度提示** badge（如"正在分析第1篇…"）；LLM 生成风格说明书的流式 token 使用标准 `token` 类型，不复用 `style_init` |
| `retry` | `count(int), reason(string)` | 质量未通过提示 badge |

---

## 前端变化

### 首次引导（Home.vue）

- 登录后检测 `UserStyle.is_initialized`（从 `GET /api/user/profile` 获取）
- 若为 `false`，聊天区顶部显示固定 Banner："请先初始化您的个人风格档案，点击此处开始"
- 点击 Banner → 触发 `POST /api/user/style/init`（独立 SSE 流），前端展示进度
- 完成后 Banner 消失，`is_initialized` 写入 userStore，刷新状态

### 辩论消息渲染（ChatPanel.vue）

- 新增 `debate` 消息类型渲染
- 前端维护3个 agent 的消息 buffer，按 `agent` 字段路由：`爆款解构师`（蓝）、`素材补齐师`（绿）、`协调者`（橙）
- 每个 agent 发言为独立气泡，`debate` token 事件流式追加到对应气泡

### 质量报告卡片（ChatPanel.vue）

- 展示4项评分（进度条形式）
- 通过：绿色 ✅ badge
- 未通过：橙色 ⚠️ + 显示具体问题 + 重试次数

### 风格档案页（新增，Profile 页或 Settings Tab）

- 8D风格向量雷达图（SVG/canvas）
- 《人设风格说明书》全文展示
- "重新建模"按钮（触发 `POST /api/user/style/init`）
- 历史稿列表（已输入的历史稿，最多展示10篇）

---

## API 变化

### 新增接口

```
POST /api/user/style/init       触发风格建模，独立 SSE 流（handler: style_handler.go）
                                Request: { scripts: string[] }（历史稿文本数组）
                                Response: SSE text/event-stream
                                  - token: LLM 生成风格说明书的流式字符
                                  - style_init: 进度提示（如"正在分析第1篇…"）
                                  - complete: { style_version, style_vector, is_initialized: true }（前端直接更新 userStore，无需额外请求）
                                  - error: { message }

GET  /api/user/style/doc        获取《人设风格说明书》全文（JSON: {style_doc, style_vector, style_version}）

POST /api/scripts/:id/feedback  提交稿件反馈（触发 feedback_processor）
                                Request: { content: string }（文字评价或用户修改版全文）
                                Response: { ok: true }
```

### 现有接口变化

- `GET /api/user/profile` — 返回结果增加 `is_initialized, style_version, style_vector` 字段
- `GET /api/scripts/:id` — 返回结果增加 `quality_report` 嵌套对象

### Post-MVP（本次不实现）

```
GET  /api/agent-configs         获取当前用户的 agent prompt 配置列表
PUT  /api/agent-configs/:agent  手动更新某个 agent 的 prompt
```

---

## 迁移策略

1. **数据库迁移**：GORM AutoMigrate 自动添加新字段（不破坏现有数据）
2. **存量用户**：`is_initialized=false`，进入应用时触发引导
3. **Agent Prompts**：系统默认 prompt 作为全局默认（`user_id=0`），用户自定义版本覆盖
4. **前后向兼容**：旧的5角色大 Prompt 保留作 fallback，新架构稳定后删除

---

## 不包含在本次重构中

- 外部搜索 API 集成（素材补齐师目前仍用 LLM 生成，非实时搜索）
- Agent prompt 管理 API（`/api/agent-configs`，Post-MVP）
- 多语言支持
- 团队/协作功能
- 移动端 App

---

## 风险与缓解

| 风险 | 缓解措施 |
|------|---------|
| LLM 调用次数增加3-5倍，成本上升 | quality_gate 检查顺序优化（相似度最先），失败早退；辩论轮次固定1轮 |
| 并行 agent 响应时间不一致 | agent_runner 设置单个 agent 超时（60s），超时则用空结果继续 |
| 总 pipeline 耗时可能超过 nginx 300s proxy_read_timeout | 部署时将 `proxy_read_timeout` 提升至 600s；后端 HTTP client 超时同步调整 |
| debate_log 存储量大 | 仅存最终辩论结果 JSON（非流式中间数据） |
| 反馈学习 prompt 变化导致质量下降 | AgentConfig 版本化，可一键回滚到任意历史版本 |
