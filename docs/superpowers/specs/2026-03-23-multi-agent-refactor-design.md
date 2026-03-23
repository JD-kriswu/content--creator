# 口播稿助手 Multi-Agent 全量重构设计文档

**日期**: 2026-03-23
**状态**: 已确认，待实施
**参考**: `assert/refactor.tx`

---

## 背景与目标

当前系统将5个"角色"内嵌在单一大 Prompt 中（伪并行），缺乏真正的 Multi-Agent 架构、质量审核闭环和反馈学习机制。

本次重构目标：
1. 实现真正并行的 3-Agent 分析
2. 引入 Multi-Agent 辩论协调层（1轮）
3. 建立质量审核闭环（事实 + 逻辑 + 表达 + 相似度）
4. 支持用户风格建模（从历史稿提取8D量化向量）
5. 反馈学习机制（用户输入 → 更新各 agent 可进化 prompt）

---

## 新流程总览

```
[用户首次]
发送历史口播稿（聊天界面内） → 风格建模师 → 生成《人设风格说明书》(8D向量)
→ UserStyle.is_initialized = true

[后续每次会话]
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
    爆款解构师 × 素材补齐师 × 风格建模师(只读参与) → 辩论消息流 → 融合大纲
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
    ↓
Step8: 最终输出
    → 保存稿件（Script）+ 质量报告（QualityReport）
    → 收集用户最终反馈 → feedback_processor → 更新 AgentConfig
```

---

## 数据模型变化

### UserStyle（扩展现有表）

新增字段：
```sql
style_vector        TEXT        -- JSON, 8维向量 {"authority":0.8,"affinity":0.6,...}
style_doc           LONGTEXT    -- 《人设风格说明书》全文
historical_scripts  LONGTEXT    -- JSON数组，存用户输入的历史稿原文
is_initialized      BOOL        -- default false，控制 Onboarding 流程
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
> 若用户无自定义版本，使用系统默认 prompt（`user_id=0` 或 hardcoded）。

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
+ debate_log    LONGTEXT    -- 辩论过程消息JSON，可供历史回溯
```

---

## 后端服务层重构

### 目录结构

```
backend/internal/service/
├── pipeline.go              会话状态机主干（调用下列模块）
├── agent_runner.go          并行执行多个 agent，goroutine + channel 汇聚结果
├── style_modeler.go         风格建模师（初始化建模 + 反馈更新）
├── viral_analyzer.go        爆款解构师（分析目标稿 DNA）
├── material_enricher.go     素材补齐师（补充新素材数据）
├── debate_coordinator.go    辩论协调（1轮3方 LLM 对话 → 融合大纲）
├── creator_agent.go         创作代理（生成初稿）
├── quality_gate.go          质量审核闭环（4项检查 + 重试逻辑）
├── feedback_processor.go    反馈整理 → AgentConfig 版本化更新
├── llm_service.go           （保留）CallClaude / StreamClaude
├── extractor.go             （保留）URL 提取
└── prompts.go               （重构）各 agent 默认 prompt，改为函数返回，支持 AgentConfig 覆盖
```

### 状态机（新增2个状态）

```
StateIdle
    → StateAnalyzing   (Step3: 并行分析)
    → StateDebating    (Step4: 辩论协调)  ← 新增
    → StateAwaiting    (Step5: 等待用户确认大纲)
    → StateWriting     (Step6: 创作初稿)
    → StateReviewing   (Step7: 质量审核)  ← 新增
    → StateComplete
```

重入恢复机制扩展：
- `StateAnalyzing` 超时3分钟 → 降级 `StateIdle`
- `StateDebating` 超时3分钟 → 降级 `StateIdle`
- `StateWriting` 超时3分钟 → 降级 `StateAwaiting`
- `StateReviewing` 超时5分钟 → 降级 `StateAwaiting`

### agent_runner.go（并行执行）

```go
type AgentResult struct {
    AgentName string
    Content   string
    Error     error
}

// RunParallel 并发执行多个 agent，通过 SSE callback 流式推送各自输出
func RunParallel(agents []AgentTask, sseCallback func(AgentResult)) []AgentResult
```

### debate_coordinator.go（辩论协调）

1轮辩论流程：
1. 汇总3份报告（爆款DNA + 素材包 + 人设风格）
2. `爆款解构师` 发言（StreamClaude）：基于DNA，主张保留哪些核心结构
3. `素材补齐师` 发言（StreamClaude）：基于素材包，主张插入哪些新内容
4. `协调者` 综合发言（StreamClaude）：基于人设风格，给出融合方案 → 输出 OutlineData JSON

每次发言通过 SSE `debate` 事件推送前端，记入 `debate_log`。

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

// RunQualityGate 执行4项检查，返回综合结果
// 若未通过，调用 creator_agent 重试（最多 maxRetry 次）
func RunQualityGate(draft string, original string, maxRetry int, sseCallback) (string, QualityCheckResult, error)
```

检查顺序：相似度（最快，不通过直接重试）→ 事实核查 → 逻辑检查 → 表达评分

### feedback_processor.go（反馈学习）

触发时机：
- 用户手动修改终稿后保存
- 用户对大纲提出修改意见（StateAwaiting 中输入非数字反馈）
- 用户在完成后的对话中继续发送评价

处理逻辑：
```go
// ProcessFeedback 汇总用户反馈，更新对应 agent 的 AgentConfig
// 生成新版本 prompt，旧版本保留
func ProcessFeedback(userID uint, feedbackType string, content string) error
```

---

## SSE 新增消息类型

| type | 字段 | 前端处理 |
|------|------|---------|
| `debate` | `agent(string), content(string)` | 3色气泡（爆款=蓝/素材=绿/协调=橙），流式追加 |
| `quality` | `data(QualityReport)` | 质量报告卡片（4项评分 + 通过/重试状态） |
| `style_init` | `content(string)` | 风格建模进度 info badge |
| `retry` | `count(int), reason(string)` | 质量未通过提示 badge |

---

## 前端变化

### 首次引导（Home.vue）

- 登录后检测 `UserStyle.is_initialized`
- 若为 `false`，聊天区顶部显示固定 Banner："请先发送您的历史口播稿（建议3篇），系统将为您建立个人风格档案"
- Banner 包含"了解更多"展开说明
- 用户在输入框发送历史稿后，Banner 消失，系统进入 StyleInit 流程

### 辩论消息渲染（ChatPanel.vue）

- 新增 `debate` 消息类型
- 3个 agent 分色标注：`爆款解构师`（蓝）、`素材补齐师`（绿）、`协调者`（橙）
- 每个 agent 发言为独立气泡，支持流式追加

### 质量报告卡片（ChatPanel.vue）

- 展示4项评分（进度条形式）
- 通过：绿色 ✅ badge
- 未通过：橙色 ⚠️ + 显示具体问题 + 重试次数

### 风格档案页（新增，Settings 或 Profile 页）

- 8D风格向量雷达图（可用 canvas 或 SVG 绘制）
- 《人设风格说明书》全文展示
- "重新建模"按钮（触发 StyleInit 流程）
- 历史稿列表（已输入的稿件）

---

## API 变化

### 新增接口

```
POST /api/user/style/init       触发风格建模（SSE，返回建模进度）
GET  /api/user/style/doc        获取《人设风格说明书》全文
POST /api/scripts/:id/feedback  提交稿件反馈（触发 feedback_processor）

GET  /api/agent-configs         获取当前用户的 agent prompt 配置列表
PUT  /api/agent-configs/:agent  手动更新某个 agent 的 prompt（高级功能，可后期开放）
```

### 现有接口变化

- `GET /api/user/profile` — 返回结果增加 `is_initialized, style_version, style_vector` 字段
- `GET /api/scripts/:id` — 返回结果增加 `quality_report` 嵌套对象

---

## 迁移策略

1. **数据库迁移**：GORM AutoMigrate 自动添加新字段（不破坏现有数据）
2. **存量用户**：`is_initialized=false`，进入应用时触发引导
3. **Agent Prompts**：系统默认 prompt 作为全局默认（`user_id=0`），用户自定义版本覆盖
4. **前后向兼容**：旧的5角色大 Prompt 保留作 fallback，新架构稳定后删除

---

## 不包含在本次重构中

- 外部搜索 API 集成（素材补齐师目前仍用 LLM 生成，非实时搜索）
- 多语言支持
- 团队/协作功能
- 移动端 App

---

## 风险与缓解

| 风险 | 缓解措施 |
|------|---------|
| LLM 调用次数增加3-5倍，成本上升 | quality_gate 检查顺序优化（相似度最先），失败早退；辩论轮次固定1轮 |
| 并行 agent 响应时间不一致 | agent_runner 设置单个 agent 超时（60s），超时则用空结果继续 |
| debate_log 存储量大 | 仅存最终辩论结果 JSON，不存中间流式数据 |
| 反馈学习 prompt 变化导致质量下降 | AgentConfig 版本化，可一键回滚到上一版本 |
