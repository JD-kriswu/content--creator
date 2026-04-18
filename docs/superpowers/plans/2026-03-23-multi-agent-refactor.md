# Multi-Agent 全量重构 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将现有单体 pipeline 重构为真正的 Multi-Agent 架构，包含并行分析、辩论协调、质量审核闭环和反馈学习。

**Architecture:** 后端拆解为 8 个独立 agent service 模块（agent_runner/style_modeler/viral_analyzer/material_enricher/debate_coordinator/creator_agent/quality_gate/feedback_processor），通过新状态机（+StateDebating/StateReviewing）串联；风格建模独立于会话状态机，通过专用 SSE handler 完成初始化。

**Tech Stack:** Go 1.22 + Gin, GORM + MySQL, Anthropic-compatible LLM API（StreamClaude/CallClaude），React 18 + TypeScript + Vite，Tailwind CSS v4，Radix UI，React Context + useReducer，原生 fetch SSE

**Spec:** `docs/superpowers/specs/2026-03-23-multi-agent-refactor-design.md`

---

## 文件变更地图

### 后端 — 新建

| 文件 | 职责 |
|------|------|
| `backend/internal/model/agent_config.go` | AgentConfig 数据模型 |
| `backend/internal/model/quality_report.go` | QualityReport 数据模型 |
| `backend/internal/repository/agent_config_repo.go` | AgentConfig CRUD |
| `backend/internal/repository/quality_report_repo.go` | QualityReport CRUD |
| `backend/internal/service/agent_runner.go` | goroutine 并行执行多 agent |
| `backend/internal/service/style_modeler.go` | 风格建模（初始化 + 反馈更新） |
| `backend/internal/service/viral_analyzer.go` | 爆款解构师 |
| `backend/internal/service/material_enricher.go` | 素材补齐师 |
| `backend/internal/service/debate_coordinator.go` | 1轮辩论协调 → OutlineData |
| `backend/internal/service/creator_agent.go` | 初稿创作 |
| `backend/internal/service/quality_gate.go` | 4项质量审核 + 重试闭环 |
| `backend/internal/service/feedback_processor.go` | 用户反馈 → AgentConfig 版本化更新 |
| `backend/internal/handler/style_handler.go` | POST /api/user/style/init SSE + GET /api/user/style/doc |
| `backend/internal/service/agent_runner_test.go` | agent_runner 单元测试 |
| `backend/internal/service/quality_gate_test.go` | quality_gate 纯函数测试 |

### 后端 — 修改

| 文件 | 变更 |
|------|------|
| `backend/internal/model/user.go` | UserStyle 添加 8 个新字段 |
| `backend/internal/model/conversation.go` | 添加 DebateLog 字段 |
| `backend/internal/db/db.go` | AutoMigrate 添加 AgentConfig、QualityReport |
| `backend/internal/repository/user_repo.go` | 添加 GetAgentConfigPrompt、UpsertAgentConfig |
| `backend/internal/service/prompts.go` | 重构为每 agent 独立函数，支持 AgentConfig override |
| `backend/internal/service/pipeline.go` | 新增 StateDebating/StateReviewing，ChatSession 扩展字段，更新状态机 |
| `backend/internal/handler/chat_handler.go` | 新增 sendDebate/sendQuality/sendRetry，handleIdle 加 is_initialized 检查，更新超时恢复 |
| `backend/main.go` | 注册 style 和 feedback 路由 |

### 前端 — 新建

| 文件 | 职责 |
|------|------|
| `frontend/src/api/style.ts` | style init SSE + get style doc |
| `frontend/src/components/StyleInitBanner.tsx` | 首次引导横幅（含历史稿输入 Dialog）|
| `frontend/src/components/create/DebateBubble.tsx` | 辩论气泡（3色：蓝/绿/橙）|
| `frontend/src/components/create/QualityCard.tsx` | 质量报告卡片（4项进度条）|

### 前端 — 修改

| 文件 | 变更 |
|------|------|
| `frontend/src/contexts/AuthContext.tsx` | 添加 isStyleInitialized, styleVersion, setStyleStatus |
| `frontend/src/lib/sse.ts` | 添加 debate/quality/retry/style_init 事件类型 |
| `frontend/src/pages/Dashboard.tsx` | 集成 StyleInitBanner，添加 debate/quality/retry reducer action，SSE handler |
| `frontend/src/components/create/MessageList.tsx` | 渲染 debate/quality/retry 新消息类型 |

---

## Phase 1：数据层

### Task 1：UserStyle 和 Conversation 模型扩展

**Files:**
- Modify: `backend/internal/model/user.go`
- Modify: `backend/internal/model/conversation.go`

- [ ] **Step 1: 扩展 UserStyle**

编辑 `backend/internal/model/user.go`，在 `UserStyle` 结构体末尾添加：

```go
// Multi-Agent 扩展字段
StyleVector       string `gorm:"type:text" json:"style_vector"`          // JSON 8D向量
StyleDoc          string `gorm:"type:longtext" json:"style_doc"`          // 《人设风格说明书》
HistoricalScripts string `gorm:"type:longtext" json:"historical_scripts"` // JSON[]，最多10篇
IsInitialized     bool   `gorm:"default:false" json:"is_initialized"`
StyleVersion      int    `gorm:"default:0" json:"style_version"`
```

- [ ] **Step 2: 扩展 Conversation**

编辑 `backend/internal/model/conversation.go`，在 `UpdatedAt` 前添加：

```go
DebateLog string `gorm:"type:longtext" json:"-"` // 辩论结果JSON，不对外暴露
```

- [ ] **Step 3: 手动验证结构体编译通过**

```bash
cd backend && go build ./internal/model/...
```
Expected: 无错误输出

- [ ] **Step 4: commit**

```bash
git add backend/internal/model/user.go backend/internal/model/conversation.go
git commit -m "feat: extend UserStyle and Conversation models for multi-agent"
```

---

### Task 2：新增 AgentConfig 和 QualityReport 模型

**Files:**
- Create: `backend/internal/model/agent_config.go`
- Create: `backend/internal/model/quality_report.go`

- [ ] **Step 1: 创建 AgentConfig 模型**

```go
// backend/internal/model/agent_config.go
package model

import "time"

// AgentConfig stores versioned prompt templates per user per agent.
// user_id=0 rows serve as system defaults.
type AgentConfig struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	UserID         uint      `gorm:"index;not null" json:"user_id"`
	AgentName      string    `gorm:"size:64;not null" json:"agent_name"` // style_modeler/viral_analyzer/material_enricher/creator/reviewer
	PromptTemplate string    `gorm:"type:longtext" json:"prompt_template"`
	Version        int       `gorm:"default:1" json:"version"`
	CreatedAt      time.Time `json:"created_at"`
}
```

- [ ] **Step 2: 创建 QualityReport 模型**

```go
// backend/internal/model/quality_report.go
package model

import "time"

// QualityReport stores the result of the 4-step quality gate for a script.
type QualityReport struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	ScriptID        uint      `gorm:"index;not null" json:"script_id"`
	SimilarityScore float64   `json:"similarity_score"`
	FactScore       float64   `json:"fact_score"`
	LogicScore      float64   `json:"logic_score"`
	ExpressionScore float64   `json:"expression_score"`
	Passed          bool      `json:"passed"`
	RetryCount      int       `json:"retry_count"`
	Issues          string    `gorm:"type:text" json:"issues"` // JSON []string
	CreatedAt       time.Time `json:"created_at"`
}
```

- [ ] **Step 3: 编译验证**

```bash
cd backend && go build ./internal/model/...
```
Expected: 无错误

- [ ] **Step 4: commit**

```bash
git add backend/internal/model/agent_config.go backend/internal/model/quality_report.go
git commit -m "feat: add AgentConfig and QualityReport models"
```

---

### Task 3：更新 AutoMigrate

**Files:**
- Modify: `backend/internal/db/db.go`

- [ ] **Step 1: 在 AutoMigrate 中添加新模型**

将 `db.go` 中 `return DB.AutoMigrate(...)` 替换为：

```go
return DB.AutoMigrate(
    &model.User{},
    &model.UserStyle{},
    &model.Script{},
    &model.Conversation{},
    &model.Message{},
    &model.AgentConfig{},
    &model.QualityReport{},
)
```

- [ ] **Step 2: 编译整个后端验证**

```bash
cd backend && go build .
```
Expected: 无错误

- [ ] **Step 3: commit**

```bash
git add backend/internal/db/db.go
git commit -m "feat: auto-migrate AgentConfig and QualityReport tables"
```

---

### Task 4：新增 Repository 函数

**Files:**
- Create: `backend/internal/repository/agent_config_repo.go`
- Create: `backend/internal/repository/quality_report_repo.go`
- Modify: `backend/internal/repository/user_repo.go`

- [ ] **Step 1: 创建 agent_config_repo.go**

```go
// backend/internal/repository/agent_config_repo.go
package repository

import (
	"content-creator-imm/internal/db"
	"content-creator-imm/internal/model"
)

// GetAgentPrompt returns the latest prompt template for a given user+agent.
// Falls back to system default (user_id=0) if no user-specific version exists.
func GetAgentPrompt(userID uint, agentName string) (string, error) {
	var cfg model.AgentConfig
	// Try user-specific first
	err := db.DB.Where("user_id = ? AND agent_name = ?", userID, agentName).
		Order("version DESC").First(&cfg).Error
	if err == nil {
		return cfg.PromptTemplate, nil
	}
	// Fall back to system default
	err = db.DB.Where("user_id = 0 AND agent_name = ?", agentName).
		Order("version DESC").First(&cfg).Error
	if err != nil {
		return "", err
	}
	return cfg.PromptTemplate, nil
}

// CreateAgentConfig saves a new versioned prompt (increments version from latest).
func CreateAgentConfig(userID uint, agentName, prompt string) error {
	var latest model.AgentConfig
	version := 1
	if err := db.DB.Where("user_id = ? AND agent_name = ?", userID, agentName).
		Order("version DESC").First(&latest).Error; err == nil {
		version = latest.Version + 1
	}
	cfg := &model.AgentConfig{
		UserID:         userID,
		AgentName:      agentName,
		PromptTemplate: prompt,
		Version:        version,
	}
	return db.DB.Create(cfg).Error
}
```

- [ ] **Step 2: 创建 quality_report_repo.go**

```go
// backend/internal/repository/quality_report_repo.go
package repository

import (
	"content-creator-imm/internal/db"
	"content-creator-imm/internal/model"
)

func CreateQualityReport(r *model.QualityReport) error {
	return db.DB.Create(r).Error
}

func GetQualityReportByScriptID(scriptID uint) (*model.QualityReport, error) {
	var r model.QualityReport
	err := db.DB.Where("script_id = ?", scriptID).First(&r).Error
	return &r, err
}
```

- [ ] **Step 3: 在 user_repo.go 添加 UpdateStyleInitialized**

在 `backend/internal/repository/user_repo.go` 末尾添加：

```go
// UpdateStyleFields updates style-related fields after modeling.
func UpdateStyleFields(userID uint, styleDoc, styleVector, historicalScripts string, version int) error {
	return db.DB.Model(&model.UserStyle{}).Where("user_id = ?", userID).Updates(map[string]interface{}{
		"style_doc":           styleDoc,
		"style_vector":        styleVector,
		"historical_scripts":  historicalScripts,
		"is_initialized":      true,
		"style_version":       version,
	}).Error
}
```

- [ ] **Step 4: 编译验证**

```bash
cd backend && go build ./internal/repository/...
```
Expected: 无错误

- [ ] **Step 5: commit**

```bash
git add backend/internal/repository/
git commit -m "feat: add AgentConfig and QualityReport repositories"
```

---

## Phase 2：Prompt 层重构

### Task 5：重构 prompts.go

**Files:**
- Modify: `backend/internal/service/prompts.go`

当前 `prompts.go` 有 `BuildAnalysisPrompt`、`BuildFinalDraftPrompt`、`BuildSimilarityCheckPrompt`。重构为每个 agent 独立函数，并添加新 agent 的 prompt builder。

- [ ] **Step 1: 重构文件结构**

将 `backend/internal/service/prompts.go` 替换为以下内容（保留原有3个函数，新增其余）：

```go
package service

import "fmt"

// ─────────────────────────────────────────────
// StyleProfile: 统一的风格数据结构，用于传递给各 agent
// ─────────────────────────────────────────────

// StyleProfile holds the user's style data passed to agents.
// Populated from UserStyle DB record.
type StyleProfile struct {
	LanguageStyle     string
	EmotionTone       string
	OpeningStyle      string
	ClosingStyle      string
	Catchphrases      string
	StyleDoc          string // Full 《人设风格说明书》, used by multi-agent flow
	IsInitialized     bool
}

// ─────────────────────────────────────────────
// 风格建模师
// ─────────────────────────────────────────────

// BuildStyleModelPrompt returns the style modeler system prompt and user prompt.
// override: if non-empty, replaces the default system prompt (from AgentConfig).
func BuildStyleModelPrompt(scripts []string, override string) (system, user string) {
	system = override
	if system == "" {
		system = `你是一个专业的风格建模师。请分析用户的历史口播稿，量化用户的人设风格特征。

请从以下维度进行分析：
1. 语言风格：句式复杂度、词汇特征、口语化程度
2. 内容偏好：高频话题、内容深度、论证风格
3. 人设特征：权威度、亲和力、专业深度、风险偏好、情感强度、互动性、叙事性、幽默感（0-1量化评分）
4. 表达习惯：修辞手法、节奏控制、开场结尾方式

请输出：
1. 结构化的《人设风格说明书》全文
2. 末尾输出 JSON 格式的8维向量，用如下标记包裹：
---STYLE_VECTOR_START---
{"authority":0.8,"affinity":0.6,"expertise":0.7,"humor":0.3,"risk":0.5,"emotion":0.6,"interaction":0.7,"storytelling":0.5}
---STYLE_VECTOR_END---`
	}
	combined := ""
	for i, s := range scripts {
		combined += fmt.Sprintf("\n\n===第%d篇===\n%s", i+1, s)
	}
	user = "请分析以下历史口播稿：" + combined
	return
}

// ─────────────────────────────────────────────
// 爆款解构师
// ─────────────────────────────────────────────

func BuildViralAnalyzerPrompt(originalText, override string) (system, user string) {
	system = override
	if system == "" {
		system = `你是一个专业的爆款口播稿解构师。请分析口播稿的DNA，从4个维度进行深度分析：

1. 结构分析：识别段落分布、逻辑结构、节奏控制点
2. 内容分析：提取核心观点、关键词、专业术语、数据引用
3. 情感分析：绘制情感曲线、识别共鸣点和情感转折点
4. 表达分析：分析句式特征、修辞手法、口语化程度、互动元素

请输出结构化的《爆款DNA分析报告》，包含：
- 成功要素总结（3-5个关键要素）
- 可复制模式（2-3个可复用的模式）
- 具体的数据支撑和量化指标`
	}
	user = "请分析以下口播稿：\n\n" + originalText
	return
}

// ─────────────────────────────────────────────
// 素材补齐师
// ─────────────────────────────────────────────

func BuildMaterialEnricherPrompt(originalText, override string) (system, user string) {
	system = override
	if system == "" {
		system = `你是一个专业的素材补齐师。请分析口播稿的核心观点，补充相关的新素材和数据。

请完成以下任务：
1. 核心观点提取：提取5个核心观点
2. 补充素材：基于核心观点补充相关案例、数据、故事（不依赖实时搜索）
3. 反差问答生成：生成2-3个挑战性问题与回答
4. 素材建议：提供关联素材和插入位置建议

请输出《新素材包+素材补充建议报告》（结构化格式）`
	}
	user = "请基于以下口播稿补充素材：\n\n" + originalText
	return
}

// ─────────────────────────────────────────────
// 辩论协调
// ─────────────────────────────────────────────

// BuildDebateViralSpeechPrompt: 爆款解构师在辩论中的发言 prompt
func BuildDebateViralSpeechPrompt(dnaReport, materialsReport, styleDoc string) (system, user string) {
	system = `你是爆款解构师。正在参与一场关于新口播稿大纲的辩论。
基于你对原稿爆款基因的分析，简要阐述：你主张在新大纲中保留哪些核心结构和爆款要素？为什么？（200字以内）`
	user = fmt.Sprintf("爆款DNA报告：\n%s\n\n素材补充报告：\n%s\n\n用户风格：\n%s",
		dnaReport, materialsReport, styleDoc)
	return
}

// BuildDebateMaterialSpeechPrompt: 素材补齐师在辩论中的发言 prompt
func BuildDebateMaterialSpeechPrompt(dnaReport, materialsReport, viralSpeech, styleDoc string) (system, user string) {
	system = `你是素材补齐师。正在参与一场关于新口播稿大纲的辩论。
基于你收集的新素材，简要阐述：你主张在新大纲中融入哪些新素材？如何与爆款解构师的意见相结合？（200字以内）`
	user = fmt.Sprintf("爆款解构师的发言：\n%s\n\n素材补充报告：\n%s\n\n爆款DNA报告：\n%s\n\n用户风格：\n%s",
		viralSpeech, materialsReport, dnaReport, styleDoc)
	return
}

// BuildDebateCoordinatorPrompt: 协调者综合发言，必须输出 OutlineData JSON
func BuildDebateCoordinatorPrompt(dnaReport, materialsReport, viralSpeech, materialSpeech, styleDoc string) (system, user string) {
	system = `你是一个专业的内容协调者。你需要综合爆款解构师和素材补齐师的观点，结合用户人设风格，输出最优融合方案。

要求：
1. 简要总结融合策略（100字以内）
2. 必须在末尾输出以下格式的大纲JSON：

---OUTLINE_START---
{
  "elements": ["成功要素1", "成功要素2"],
  "materials": ["新素材1", "新素材2"],
  "outline": [
    {"part":"开场","duration":"15s","content":"具体内容","emotion":"情感基调"},
    {"part":"主体","duration":"60s","content":"具体内容","emotion":"情感基调"},
    {"part":"结尾","duration":"15s","content":"具体内容","emotion":"情感基调"}
  ],
  "estimated_similarity": "约15%",
  "strategy": "改写核心策略一句话"
}
---OUTLINE_END---`
	user = fmt.Sprintf(`爆款解构师发言：%s

素材补齐师发言：%s

爆款DNA报告摘要：%s

新素材包摘要：%s

用户人设风格说明书：%s`,
		viralSpeech, materialSpeech, dnaReport, materialsReport, styleDoc)
	return
}

// ─────────────────────────────────────────────
// 创作代理
// ─────────────────────────────────────────────

// BuildCreatorPrompt builds the final draft writing prompt.
// issues: feedback from failed quality check (empty on first attempt).
func BuildCreatorPrompt(outlineJSON, dnaReport, materialsReport, styleDoc, issues, override string) (system, user string) {
	system = override
	if system == "" {
		system = `你是一个专业的口播稿创作代理。基于提供的大纲、分析报告和用户风格，生成高质量的口播稿。

创作要求：
1. 严格遵循大纲结构，确保逻辑连贯
2. 应用用户的人设风格特征
3. 融合新素材包中的相关素材
4. 控制节奏和情感曲线
5. 确保逻辑连贯性和表达流畅性

输出要求：
- 吸引人的标题
- 清晰的段落结构（钩子→分析→案例→总结）
- 口语化表达（适合口播的节奏）
- 有力的结尾（总结要点+呼吁行动）

末尾输出质量自检块（将被程序截断，不展示给用户）：
---QUALITY_CHECK_START---
自检内容
---QUALITY_CHECK_END---`
	}
	issuesSection := ""
	if issues != "" {
		issuesSection = fmt.Sprintf("\n\n⚠️ 上一版本质量检查未通过，请修正以下问题：\n%s", issues)
	}
	user = fmt.Sprintf(`大纲：
%s

爆款DNA报告：
%s

新素材包：
%s

用户人设风格说明书：
%s%s`, outlineJSON, dnaReport, materialsReport, styleDoc, issuesSection)
	return
}

// ─────────────────────────────────────────────
// 质量审核（保留旧函数签名，内部重新实现）
// ─────────────────────────────────────────────

func BuildFactCheckPrompt(draft string) string {
	return fmt.Sprintf(`你是一个专业的事实核查专家。请对以下口播稿进行事实核查。

核查要求：
1. 数据准确性：检查所有数字、百分比、日期、统计数据
2. 引用正确性：验证所有专家观点、研究报告的准确性
3. 概念准确性：确保专业术语使用正确
4. 时效性：标注可能过时的数据

以JSON格式输出：
{"score": 92.5, "passed": true, "issues": ["问题描述1", "问题描述2"]}

阈值：score >= 90 则 passed=true

口播稿内容：
%s`, draft)
}

func BuildLogicCheckPrompt(draft string) string {
	return fmt.Sprintf(`你是一个逻辑分析专家。请检查以下口播稿的逻辑连贯性。

检查维度：
1. 论点一致性：所有论点是否相互支持，无矛盾
2. 推理链条：从前提→推理→结论是否完整
3. 证据支撑：每个论点是否有足够证据支撑
4. 结构逻辑：段落间逻辑关系是否清晰

以JSON格式输出：
{"score": 88.0, "passed": true, "issues": ["问题描述1"]}

阈值：score >= 85 则 passed=true

口播稿内容：
%s`, draft)
}

func BuildExpressionCheckPrompt(draft string) string {
	return fmt.Sprintf(`你是一个专业的表达质量评估专家。请评估以下口播稿的表达质量。

评估维度（权重）：
1. 语言流畅性（30%%）
2. 节奏控制（25%%）
3. 情感表达（20%%）
4. 口语化程度（15%%）
5. 互动设计（10%%）

以JSON格式输出：
{"score": 78.0, "passed": true, "issues": ["问题描述1"]}

阈值：score >= 75 则 passed=true

注意：只评估不修改。

口播稿内容：
%s`, draft)
}

// ─────────────────────────────────────────────
// 相似度检测（保留原有函数）
// ─────────────────────────────────────────────

func BuildSimilarityCheckPrompt(originalText, newScript string) string {
	return fmt.Sprintf(`请分析以下两篇文章的相似度，从4个维度评分（0-100，100为完全相同）：

1. vocab: 词汇相似度（权重30%%）
2. sentence: 句式相似度（权重25%%）
3. structure: 结构相似度（权重25%%）
4. viewpoint: 观点相似度（权重20%%）

total = vocab*0.30 + sentence*0.25 + structure*0.25 + viewpoint*0.20

只输出JSON，不要其他内容：
{"vocab":20,"sentence":15,"structure":18,"viewpoint":10,"total":16.25}

原文：
%s

新文：
%s`, originalText, newScript)
}

// ─────────────────────────────────────────────
// 反馈处理
// ─────────────────────────────────────────────

func BuildFeedbackProcessPrompt(agentName, currentPrompt, feedbackContent string) string {
	return fmt.Sprintf(`你是一个 AI prompt 优化专家。请根据用户反馈，优化以下 agent 的 prompt。

Agent 名称：%s

当前 Prompt：
%s

用户反馈内容：
%s

请输出：
1. 分析用户反馈涉及到 prompt 的哪个方面
2. 输出优化后的完整 prompt（保持原有格式和结构，只针对性改进）

---OPTIMIZED_PROMPT_START---
（优化后的完整 prompt 内容）
---OPTIMIZED_PROMPT_END---`, agentName, currentPrompt, feedbackContent)
}

// ─────────────────────────────────────────────
// 兼容层：保留旧函数供现有代码使用（将在后续 Task 中逐步替换）
// ─────────────────────────────────────────────

// BuildAnalysisPrompt is kept for backward compatibility.
// New code should use BuildViralAnalyzerPrompt + BuildMaterialEnricherPrompt separately.
func BuildAnalysisPrompt(originalText string, style *StyleProfile) string {
	_, user := BuildViralAnalyzerPrompt(originalText, "")
	return user // simplified fallback
}

// BuildFinalDraftPrompt is kept for backward compatibility.
func BuildFinalDraftPrompt(originalText, outlineJSON, userNote string) string {
	_, user := BuildCreatorPrompt(outlineJSON, "", "", "", userNote, "")
	return user
}

// ParseOutlineFromAnalysis extracts OutlineData JSON from LLM output.
// Looks for ---OUTLINE_START---...---OUTLINE_END--- markers.
func ParseOutlineFromAnalysis(text string) (*OutlineData, string) {
	const startMark = "---OUTLINE_START---"
	const endMark = "---OUTLINE_END---"
	start := indexOf(text, startMark)
	if start < 0 {
		return nil, ""
	}
	start += len(startMark)
	end := indexOf(text[start:], endMark)
	if end < 0 {
		return nil, ""
	}
	jsonStr := text[start : start+end]
	var od OutlineData
	if err := unmarshalJSON([]byte(jsonStr), &od); err != nil {
		return nil, jsonStr
	}
	return &od, jsonStr
}

// ParseStyleVectorFromDoc extracts the 8D style vector JSON from LLM output.
func ParseStyleVectorFromDoc(text string) string {
	const startMark = "---STYLE_VECTOR_START---"
	const endMark = "---STYLE_VECTOR_END---"
	start := indexOf(text, startMark)
	if start < 0 {
		return ""
	}
	start += len(startMark)
	end := indexOf(text[start:], endMark)
	if end < 0 {
		return ""
	}
	return text[start : start+end]
}

// ParseOptimizedPromptFromFeedback extracts the optimized prompt from LLM output.
func ParseOptimizedPromptFromFeedback(text string) string {
	const startMark = "---OPTIMIZED_PROMPT_START---"
	const endMark = "---OPTIMIZED_PROMPT_END---"
	start := indexOf(text, startMark)
	if start < 0 {
		return ""
	}
	start += len(startMark)
	end := indexOf(text[start:], endMark)
	if end < 0 {
		return ""
	}
	return text[start : start+end]
}

// StripQualityCheck removes the QUALITY_CHECK block from draft text before saving.
// Exported so chat_handler.go can use it.
func StripQualityCheck(text string) string {
	const marker = "---QUALITY_CHECK_START---"
	if idx := indexOf(text, marker); idx >= 0 {
		return text[:idx]
	}
	return text
}

// ─────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────

import (
	"encoding/json"
	"strings"
)

func indexOf(s, substr string) int {
	return strings.Index(s, substr)
}

func unmarshalJSON(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}
```

> **注意**：上面的 `import` 块放在文件顶部 `package service` 之后，不能嵌套在函数内。实际写文件时按标准 Go 文件格式组织 import。

- [ ] **Step 2: 写单元测试验证 parse 函数**

创建 `backend/internal/service/prompts_test.go`：

```go
package service

import (
	"strings"
	"testing"
)

func TestParseStyleVectorFromDoc(t *testing.T) {
	input := `这是风格说明书内容...
---STYLE_VECTOR_START---
{"authority":0.8,"affinity":0.6,"expertise":0.7,"humor":0.3,"risk":0.5,"emotion":0.6,"interaction":0.7,"storytelling":0.5}
---STYLE_VECTOR_END---
`
	got := ParseStyleVectorFromDoc(input)
	if got == "" {
		t.Fatal("expected non-empty style vector JSON")
	}
	if !strings.Contains(got, "authority") {
		t.Errorf("expected authority field, got: %s", got)
	}
}

func TestParseOutlineFromAnalysis(t *testing.T) {
	input := `辩论结论...
---OUTLINE_START---
{"elements":["要素1"],"materials":["素材1"],"outline":[{"part":"开场","duration":"15s","content":"内容","emotion":"兴奋"}],"estimated_similarity":"约15%","strategy":"改写策略"}
---OUTLINE_END---
`
	od, _ := ParseOutlineFromAnalysis(input)
	if od == nil {
		t.Fatal("expected non-nil OutlineData")
	}
	if len(od.Elements) == 0 {
		t.Error("expected elements to be populated")
	}
}

func TestParseOptimizedPromptFromFeedback(t *testing.T) {
	input := `分析：用户反馈表明...
---OPTIMIZED_PROMPT_START---
你是一个优化后的 agent。
---OPTIMIZED_PROMPT_END---`
	got := ParseOptimizedPromptFromFeedback(input)
	if got == "" {
		t.Fatal("expected non-empty optimized prompt")
	}
}
```

- [ ] **Step 3: 运行测试**

```bash
cd backend && go test ./internal/service/ -run TestParse -v
```
Expected: 3 tests PASS

- [ ] **Step 4: 编译整个后端**

```bash
cd backend && go build .
```
Expected: 无错误（可能有 `BuildAnalysisPrompt`/`BuildFinalDraftPrompt` 相关警告，正常）

- [ ] **Step 5: commit**

```bash
git add backend/internal/service/prompts.go backend/internal/service/prompts_test.go
git commit -m "feat: refactor prompts.go for multi-agent, add parse helpers + tests"
```

---

## Phase 3：Agent Service 层

### Task 6：agent_runner.go（并行执行框架）

**Files:**
- Create: `backend/internal/service/agent_runner.go`
- Create: `backend/internal/service/agent_runner_test.go`

- [ ] **Step 1: 写测试（先写失败的测试）**

```go
// backend/internal/service/agent_runner_test.go
package service

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestRunParallelBothSucceed(t *testing.T) {
	tasks := []AgentTask{
		{Name: "agent1", System: "s1", Prompt: "p1"},
		{Name: "agent2", System: "s2", Prompt: "p2"},
	}

	// Stub: replace StreamClaude with a synchronous version for testing
	// We test the concurrency logic by injecting a fake runner
	var callCount int64
	fakeRun := func(task AgentTask, cb func(AgentResult)) AgentResult {
		atomic.AddInt64(&callCount, 1)
		time.Sleep(10 * time.Millisecond)
		cb(AgentResult{AgentName: task.Name, Content: "result:" + task.Name})
		return AgentResult{AgentName: task.Name, Content: "result:" + task.Name}
	}

	results := runParallelWithRunner(tasks, func(r AgentResult) {}, fakeRun)

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if atomic.LoadInt64(&callCount) != 2 {
		t.Error("expected both agents to be called")
	}
}

func TestRunParallelOneTimeout(t *testing.T) {
	tasks := []AgentTask{
		{Name: "fast", System: "s", Prompt: "p"},
		{Name: "slow", System: "s", Prompt: "p"},
	}

	fakeRun := func(task AgentTask, cb func(AgentResult)) AgentResult {
		if task.Name == "slow" {
			time.Sleep(200 * time.Millisecond) // will exceed 50ms test timeout
			return AgentResult{AgentName: task.Name, Content: "late"}
		}
		return AgentResult{AgentName: task.Name, Content: "fast-result"}
	}

	// Use a short timeout for testing
	results := runParallelWithTimeout(tasks, func(r AgentResult) {}, fakeRun, 50*time.Millisecond)

	var slowResult *AgentResult
	for i := range results {
		if results[i].AgentName == "slow" {
			slowResult = &results[i]
		}
	}
	if slowResult == nil {
		t.Fatal("expected slow agent result in output")
	}
	if slowResult.Error == nil {
		t.Error("expected slow agent to have timeout error")
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

```bash
cd backend && go test ./internal/service/ -run TestRunParallel -v
```
Expected: 编译错误（`AgentTask`, `runParallelWithRunner` 未定义）

- [ ] **Step 3: 实现 agent_runner.go**

```go
// backend/internal/service/agent_runner.go
package service

import (
	"fmt"
	"sync"
	"time"
)

// AgentTask defines a single agent invocation.
type AgentTask struct {
	Name   string // agent identifier, used in SSE event's agent field
	System string // system prompt
	Prompt string // user prompt
}

// AgentResult holds the output of a single agent run.
type AgentResult struct {
	AgentName string
	Content   string
	Error     error
}

// agentRunnerFunc is the underlying function that executes one agent.
// In production this calls StreamClaude; in tests it's replaced with a stub.
type agentRunnerFunc func(task AgentTask, tokenCb func(AgentResult)) AgentResult

const defaultAgentTimeout = 60 * time.Second

// RunParallel concurrently runs all agents, streaming tokens via sseCallback.
// If an agent times out (60s), its result has Error set; pipeline continues with empty Content.
func RunParallel(tasks []AgentTask, sseCallback func(AgentResult)) []AgentResult {
	return runParallelWithTimeout(tasks, sseCallback, llmAgentRunner, defaultAgentTimeout)
}

// llmAgentRunner is the production runner that calls StreamClaude.
func llmAgentRunner(task AgentTask, tokenCb func(AgentResult)) AgentResult {
	var sb strings.Builder
	err := StreamClaude(task.System, task.Prompt, func(token string) bool {
		sb.WriteString(token)
		tokenCb(AgentResult{AgentName: task.Name, Content: token})
		return true
	})
	return AgentResult{AgentName: task.Name, Content: sb.String(), Error: err}
}

// runParallelWithRunner allows injecting a custom runner (for testing).
func runParallelWithRunner(tasks []AgentTask, sseCallback func(AgentResult), runner agentRunnerFunc) []AgentResult {
	return runParallelWithTimeout(tasks, sseCallback, runner, defaultAgentTimeout)
}

// runParallelWithTimeout is the core implementation.
func runParallelWithTimeout(tasks []AgentTask, sseCallback func(AgentResult), runner agentRunnerFunc, timeout time.Duration) []AgentResult {
	results := make([]AgentResult, len(tasks))
	var wg sync.WaitGroup

	for i, task := range tasks {
		wg.Add(1)
		go func(idx int, t AgentTask) {
			defer wg.Done()
			done := make(chan AgentResult, 1)
			go func() {
				done <- runner(t, sseCallback)
			}()
			select {
			case r := <-done:
				results[idx] = r
			case <-time.After(timeout):
				results[idx] = AgentResult{
					AgentName: t.Name,
					Content:   "",
					Error:     fmt.Errorf("agent %s timed out after %v", t.Name, timeout),
				}
			}
		}(i, task)
	}

	wg.Wait()
	return results
}
```

> 注意：`strings` import 需添加到文件顶部的 import 块中。

- [ ] **Step 4: 运行测试确认通过**

```bash
cd backend && go test ./internal/service/ -run TestRunParallel -v
```
Expected: 2 tests PASS

- [ ] **Step 5: commit**

```bash
git add backend/internal/service/agent_runner.go backend/internal/service/agent_runner_test.go
git commit -m "feat: add AgentRunner with parallel execution and timeout support"
```

---

### Task 7：style_modeler.go

**Files:**
- Create: `backend/internal/service/style_modeler.go`

- [ ] **Step 1: 实现**

```go
// backend/internal/service/style_modeler.go
package service

import (
	"encoding/json"
	"strings"

	"content-creator-imm/internal/repository"
)

// StyleVector represents the 8-dimensional persona vector.
type StyleVector struct {
	Authority    float64 `json:"authority"`
	Affinity     float64 `json:"affinity"`
	Expertise    float64 `json:"expertise"`
	Humor        float64 `json:"humor"`
	Risk         float64 `json:"risk"`
	Emotion      float64 `json:"emotion"`
	Interaction  float64 `json:"interaction"`
	Storytelling float64 `json:"storytelling"`
}

// SSEStyleCallback is called during style modeling to stream progress/tokens.
// eventType: "style_init" for progress badges, "token" for LLM text tokens.
type SSEStyleCallback func(eventType, content string)

// InitializeStyle runs the style modeler on historical scripts and persists results.
// scripts: raw text of user's historical scripts (1-10 items).
func InitializeStyle(userID uint, scripts []string, callback SSEStyleCallback) error {
	callback("style_init", "正在分析您的口播稿风格，请稍候...")

	// Get user-specific or default prompt override
	systemPrompt, _ := repository.GetAgentPrompt(userID, "style_modeler")

	system, userPrompt := BuildStyleModelPrompt(scripts, systemPrompt)

	var sb strings.Builder
	err := StreamClaude(system, userPrompt, func(token string) bool {
		sb.WriteString(token)
		callback("token", token)
		return true
	})
	if err != nil {
		return err
	}

	fullDoc := sb.String()

	// Extract 8D vector from doc
	vectorJSON := ParseStyleVectorFromDoc(fullDoc)
	if vectorJSON == "" {
		vectorJSON = "{}" // fallback: empty vector, doc still saved
	}

	// Persist historical scripts (cap at 10, rolling)
	historicalJSON := marshalHistoricalScripts(userID, scripts)

	// Get current style version
	existingStyle, _ := repository.GetStyleByUserID(userID)
	version := 1
	if existingStyle != nil {
		version = existingStyle.StyleVersion + 1
	}

	callback("style_init", "正在保存风格档案...")

	return repository.UpdateStyleFields(userID, fullDoc, vectorJSON, historicalJSON, version)
}

// marshalHistoricalScripts loads existing scripts, appends new ones, caps at 10.
func marshalHistoricalScripts(userID uint, newScripts []string) string {
	existing, _ := repository.GetStyleByUserID(userID)
	var all []string
	if existing != nil && existing.HistoricalScripts != "" {
		_ = json.Unmarshal([]byte(existing.HistoricalScripts), &all)
	}
	all = append(all, newScripts...)
	if len(all) > 10 {
		all = all[len(all)-10:] // keep most recent 10
	}
	b, _ := json.Marshal(all)
	return string(b)
}
```

- [ ] **Step 2: 编译验证**

```bash
cd backend && go build ./internal/service/...
```
Expected: 无错误

- [ ] **Step 3: commit**

```bash
git add backend/internal/service/style_modeler.go
git commit -m "feat: add StyleModeler service for 8D persona initialization"
```

---

### Task 8：viral_analyzer.go 和 material_enricher.go

**Files:**
- Create: `backend/internal/service/viral_analyzer.go`
- Create: `backend/internal/service/material_enricher.go`

- [ ] **Step 1: 创建 viral_analyzer.go**

```go
// backend/internal/service/viral_analyzer.go
package service

import (
	"strings"

	"content-creator-imm/internal/repository"
)

// RunViralAnalyzer streams the viral DNA analysis for the given script text.
// Returns the full analysis report text.
func RunViralAnalyzer(userID uint, originalText string, tokenCb func(string)) (string, error) {
	override, _ := repository.GetAgentPrompt(userID, "viral_analyzer")
	system, userPrompt := BuildViralAnalyzerPrompt(originalText, override)

	var sb strings.Builder
	err := StreamClaude(system, userPrompt, func(token string) bool {
		sb.WriteString(token)
		tokenCb(token)
		return true
	})
	return sb.String(), err
}
```

- [ ] **Step 2: 创建 material_enricher.go**

```go
// backend/internal/service/material_enricher.go
package service

import (
	"strings"

	"content-creator-imm/internal/repository"
)

// RunMaterialEnricher streams material supplementation for the given script text.
// Returns the full materials report text.
func RunMaterialEnricher(userID uint, originalText string, tokenCb func(string)) (string, error) {
	override, _ := repository.GetAgentPrompt(userID, "material_enricher")
	system, userPrompt := BuildMaterialEnricherPrompt(originalText, override)

	var sb strings.Builder
	err := StreamClaude(system, userPrompt, func(token string) bool {
		sb.WriteString(token)
		tokenCb(token)
		return true
	})
	return sb.String(), err
}
```

- [ ] **Step 3: 编译验证**

```bash
cd backend && go build ./internal/service/...
```
Expected: 无错误

- [ ] **Step 4: commit**

```bash
git add backend/internal/service/viral_analyzer.go backend/internal/service/material_enricher.go
git commit -m "feat: add ViralAnalyzer and MaterialEnricher agent services"
```

---

### Task 9：debate_coordinator.go

**Files:**
- Create: `backend/internal/service/debate_coordinator.go`

- [ ] **Step 1: 实现**

```go
// backend/internal/service/debate_coordinator.go
package service

import (
	"encoding/json"
	"strings"
)

// DebateSpeech represents one turn in the debate.
type DebateSpeech struct {
	AgentName string `json:"agent_name"`
	Content   string `json:"content"`
}

// DebateLog is stored in conversation.debate_log.
type DebateLog struct {
	Speeches []DebateSpeech `json:"speeches"`
}

// SSEDebateCallback streams individual debate tokens to the frontend.
// agentName identifies which agent is speaking for color-routing.
type SSEDebateCallback func(agentName, token string)

// RunDebate executes one round of multi-agent debate and returns the OutlineData + debate log.
// dnaReport: viral analyzer output
// materialsReport: material enricher output
// styleDoc: user's style profile doc (read-only)
func RunDebate(dnaReport, materialsReport, styleDoc string, debateCb SSEDebateCallback) (*OutlineData, string, error) {
	var speeches []DebateSpeech

	// Turn 1: Viral analyzer speaks
	viralSpeech, err := streamDebateTurn("爆款解构师",
		BuildDebateViralSpeechPrompt(dnaReport, materialsReport, styleDoc),
		debateCb)
	if err != nil {
		return nil, "", err
	}
	speeches = append(speeches, DebateSpeech{AgentName: "爆款解构师", Content: viralSpeech})

	// Turn 2: Material enricher speaks
	materialSpeech, err := streamDebateTurn("素材补齐师",
		BuildDebateMaterialSpeechPrompt(dnaReport, materialsReport, viralSpeech, styleDoc),
		debateCb)
	if err != nil {
		return nil, "", err
	}
	speeches = append(speeches, DebateSpeech{AgentName: "素材补齐师", Content: materialSpeech})

	// Turn 3: Coordinator synthesizes → must output OutlineData
	coordinatorOut, err := streamDebateTurn("协调者",
		BuildDebateCoordinatorPrompt(dnaReport, materialsReport, viralSpeech, materialSpeech, styleDoc),
		debateCb)
	if err != nil {
		return nil, "", err
	}
	speeches = append(speeches, DebateSpeech{AgentName: "协调者", Content: coordinatorOut})

	// Extract outline from coordinator output
	outlineData, _ := ParseOutlineFromAnalysis(coordinatorOut)

	// Serialize debate log for persistence
	logData, _ := json.Marshal(DebateLog{Speeches: speeches})

	return outlineData, string(logData), nil
}

// streamDebateTurn runs one StreamClaude call, routing tokens to the SSE callback.
func streamDebateTurn(agentName string, prompts [2]string, cb SSEDebateCallback) (string, error) {
	var sb strings.Builder
	err := StreamClaude(prompts[0], prompts[1], func(token string) bool {
		sb.WriteString(token)
		cb(agentName, token)
		return true
	})
	return sb.String(), err
}
```

> **注意：** `BuildDebateViralSpeechPrompt` 等函数返回 `(system, user string)`，而 `streamDebateTurn` 接受 `[2]string`。调用处需要调整：

```go
// 调用示例（在 RunDebate 内部正确写法）:
viralSpeech, err := streamDebateTurnPrompts("爆款解构师",
    BuildDebateViralSpeechPrompt(dnaReport, materialsReport, styleDoc),
    debateCb)
```

将 `streamDebateTurn` 签名改为接受两个字符串：

```go
func streamDebateTurn(agentName, system, userPrompt string, cb SSEDebateCallback) (string, error) {
    var sb strings.Builder
    err := StreamClaude(system, userPrompt, func(token string) bool {
        sb.WriteString(token)
        cb(agentName, token)
        return true
    })
    return sb.String(), err
}
```

调用处展开元组：
```go
sys, usr := BuildDebateViralSpeechPrompt(dnaReport, materialsReport, styleDoc)
viralSpeech, err := streamDebateTurn("爆款解构师", sys, usr, debateCb)
```

- [ ] **Step 2: 编译验证**

```bash
cd backend && go build ./internal/service/...
```
Expected: 无错误

- [ ] **Step 3: commit**

```bash
git add backend/internal/service/debate_coordinator.go
git commit -m "feat: add DebateCoordinator - 1-round multi-agent debate producing OutlineData"
```

---

### Task 10：creator_agent.go 和 quality_gate.go

**Files:**
- Create: `backend/internal/service/creator_agent.go`
- Create: `backend/internal/service/quality_gate.go`
- Create: `backend/internal/service/quality_gate_test.go`

- [ ] **Step 1: 创建 creator_agent.go**

```go
// backend/internal/service/creator_agent.go
package service

import (
	"strings"

	"content-creator-imm/internal/repository"
)

// RunCreatorAgent generates a draft script from the given inputs.
// issues: pass empty string on first attempt; pass quality check failures on retry.
func RunCreatorAgent(userID uint, outlineJSON, dnaReport, materialsReport, styleDoc, issues string, tokenCb func(string)) (string, error) {
	override, _ := repository.GetAgentPrompt(userID, "creator")
	system, userPrompt := BuildCreatorPrompt(outlineJSON, dnaReport, materialsReport, styleDoc, issues, override)

	var sb strings.Builder
	err := StreamClaude(system, userPrompt, func(token string) bool {
		sb.WriteString(token)
		tokenCb(token)
		return true
	})
	return sb.String(), err
}
```

- [ ] **Step 2: 写 quality_gate 测试（先写失败）**

```go
// backend/internal/service/quality_gate_test.go
package service

import (
	"encoding/json"
	"testing"
)

func TestParseQualityScore(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		wantPass bool
		wantScore float64
	}{
		{"passed", `{"score": 92.5, "passed": true, "issues": []}`, true, 92.5},
		{"failed", `{"score": 72.0, "passed": false, "issues": ["数据过时"]}`, false, 72.0},
		{"invalid json", `not json`, false, 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			score, passed, _ := parseQualityScore(tc.input)
			if passed != tc.wantPass {
				t.Errorf("want passed=%v, got %v", tc.wantPass, passed)
			}
			if tc.wantScore > 0 && score != tc.wantScore {
				t.Errorf("want score=%v, got %v", tc.wantScore, score)
			}
		})
	}
}

func TestCollectIssues(t *testing.T) {
	type qr struct {
		Issues []string `json:"issues"`
	}
	r := qr{Issues: []string{"问题A", "问题B"}}
	b, _ := json.Marshal(r)

	issues := collectIssuesFromJSON(string(b))
	if len(issues) != 2 {
		t.Fatalf("expected 2 issues, got %d", len(issues))
	}
	if issues[0] != "问题A" {
		t.Errorf("expected 问题A, got %s", issues[0])
	}
}
```

- [ ] **Step 3: 运行测试确认失败**

```bash
cd backend && go test ./internal/service/ -run TestParse -v
```
Expected: `parseQualityScore` undefined 错误

- [ ] **Step 4: 创建 quality_gate.go**

```go
// backend/internal/service/quality_gate.go
package service

import (
	"encoding/json"
	"fmt"
	"strings"
)

// QualityCheckResult holds the result of all 4 quality checks.
type QualityCheckResult struct {
	SimilarityScore float64  `json:"similarity_score"`
	FactScore       float64  `json:"fact_score"`
	LogicScore      float64  `json:"logic_score"`
	ExpressionScore float64  `json:"expression_score"`
	Passed          bool     `json:"passed"`
	RetryCount      int      `json:"retry_count"`
	Issues          []string `json:"issues"`
}

// SSEQualityCallback is called when a quality check step completes.
type SSEQualityCallback func(eventType string, data interface{})

// RunQualityGate runs all 4 quality checks and retries up to maxRetry times if failed.
// On creator_agent error during retry: returns last successful draft with Passed=false.
func RunQualityGate(
	userID uint,
	draft, originalText string,
	outlineJSON, dnaReport, materialsReport, styleDoc string,
	maxRetry int,
	cb SSEQualityCallback,
) (finalDraft string, result QualityCheckResult, err error) {
	finalDraft = draft

	for attempt := 0; attempt <= maxRetry; attempt++ {
		result, err = runChecks(finalDraft, originalText, attempt)
		cb("quality", result)

		if result.Passed {
			return finalDraft, result, nil
		}

		if attempt < maxRetry {
			issueStr := strings.Join(result.Issues, "\n")
			cb("retry", map[string]interface{}{"count": attempt + 1, "reason": issueStr})

			newDraft, retryErr := RunCreatorAgent(
				userID, outlineJSON, dnaReport, materialsReport, styleDoc,
				issueStr,
				func(token string) {
					// tokens during retry are not streamed (silent regeneration)
				},
			)
			if retryErr != nil {
				// creator error: return last good draft, mark failed
				result.Passed = false
				return finalDraft, result, nil
			}
			finalDraft = newDraft
		}
	}

	return finalDraft, result, nil
}

// runChecks runs all 4 checks in order; returns on first failure.
func runChecks(draft, originalText string, retryCount int) (QualityCheckResult, error) {
	result := QualityCheckResult{RetryCount: retryCount}

	// 1. Similarity (fastest, uses algorithm-like LLM call)
	simPrompt := BuildSimilarityCheckPrompt(originalText, draft)
	simRaw, err := CallClaude("", simPrompt, 256)
	if err != nil {
		return result, err
	}
	result.SimilarityScore, _, simIssues := parseSimScore(simRaw)
	result.Issues = append(result.Issues, simIssues...)
	if result.SimilarityScore >= 30 {
		result.Issues = append(result.Issues, fmt.Sprintf("相似度过高：%.1f%%（需低于30%%）", result.SimilarityScore))
		return result, nil // early exit
	}

	// 2. Fact check
	factRaw, err := CallClaude("", BuildFactCheckPrompt(draft))
	if err != nil {
		return result, err
	}
	result.FactScore, result.Passed, factIssues := parseQualityScore(factRaw)
	result.Issues = append(result.Issues, factIssues...)
	if !result.Passed {
		return result, nil
	}

	// 3. Logic check
	logicRaw, err := CallClaude("", BuildLogicCheckPrompt(draft))
	if err != nil {
		return result, err
	}
	result.LogicScore, result.Passed, logicIssues := parseQualityScore(logicRaw)
	result.Issues = append(result.Issues, logicIssues...)
	if !result.Passed {
		return result, nil
	}

	// 4. Expression check
	exprRaw, err := CallClaude("", BuildExpressionCheckPrompt(draft))
	if err != nil {
		return result, err
	}
	result.ExpressionScore, result.Passed, exprIssues := parseQualityScore(exprRaw)
	result.Issues = append(result.Issues, exprIssues...)

	return result, nil
}

// parseQualityScore parses {"score": 92.5, "passed": true, "issues": [...]} from LLM output.
func parseQualityScore(raw string) (score float64, passed bool, issues []string) {
	// Find JSON in potentially noisy LLM output
	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start < 0 || end < start {
		return 0, false, nil
	}
	jsonStr := raw[start : end+1]

	var result struct {
		Score  float64  `json:"score"`
		Passed bool     `json:"passed"`
		Issues []string `json:"issues"`
	}
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return 0, false, nil
	}
	return result.Score, result.Passed, result.Issues
}

// parseSimScore parses the similarity score response.
func parseSimScore(raw string) (total float64, passed bool, issues []string) {
	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start < 0 || end < start {
		return 0, false, nil
	}
	var result struct {
		Total float64 `json:"total"`
	}
	if err := json.Unmarshal([]byte(raw[start:end+1]), &result); err != nil {
		return 0, false, nil
	}
	return result.Total, result.Total < 30, nil
}

// collectIssuesFromJSON extracts []string issues from a JSON string.
func collectIssuesFromJSON(raw string) []string {
	var r struct {
		Issues []string `json:"issues"`
	}
	_ = json.Unmarshal([]byte(raw), &r)
	return r.Issues
}
```

- [ ] **Step 5: 运行测试确认通过**

```bash
cd backend && go test ./internal/service/ -run "TestParseQuality|TestCollectIssues" -v
```
Expected: 3 tests PASS

- [ ] **Step 6: 编译整体**

```bash
cd backend && go build ./internal/service/...
```
Expected: 无错误

- [ ] **Step 7: commit**

```bash
git add backend/internal/service/creator_agent.go backend/internal/service/quality_gate.go backend/internal/service/quality_gate_test.go
git commit -m "feat: add CreatorAgent and QualityGate with retry loop and tests"
```

---

### Task 11：feedback_processor.go

**Files:**
- Create: `backend/internal/service/feedback_processor.go`

- [ ] **Step 1: 实现**

```go
// backend/internal/service/feedback_processor.go
package service

import (
	"content-creator-imm/internal/repository"
)

// ProcessFeedback updates the AgentConfig for the most relevant agent based on feedback.
// feedbackType: "outline_note" (during StateAwaiting) or "script_feedback" (post-completion)
// content: the user's feedback text
func ProcessFeedback(userID uint, feedbackType, content string) error {
	// Determine which agent to update based on feedback type
	agentName := "creator" // default: creation feedback
	if feedbackType == "outline_note" {
		agentName = "viral_analyzer" // outline feedback relates to analysis
	}

	// Get current prompt for this agent
	currentPrompt, err := repository.GetAgentPrompt(userID, agentName)
	if err != nil || currentPrompt == "" {
		// No user-specific prompt yet; get system default to use as base
		currentPrompt, _ = repository.GetAgentPrompt(0, agentName)
	}

	if currentPrompt == "" {
		return nil // nothing to optimize
	}

	// Ask LLM to optimize the prompt based on feedback
	optimizationPrompt := BuildFeedbackProcessPrompt(agentName, currentPrompt, content)
	result, err := CallClaude("", optimizationPrompt)
	if err != nil {
		return err
	}

	// Extract the optimized prompt
	optimized := ParseOptimizedPromptFromFeedback(result)
	if optimized == "" {
		return nil // LLM didn't produce a valid optimized prompt
	}

	// Save new version
	return repository.CreateAgentConfig(userID, agentName, optimized)
}
```

- [ ] **Step 2: 编译验证**

```bash
cd backend && go build ./internal/service/...
```
Expected: 无错误

- [ ] **Step 3: commit**

```bash
git add backend/internal/service/feedback_processor.go
git commit -m "feat: add FeedbackProcessor for versioned agent prompt evolution"
```

---

## Phase 4：状态机与 Handler 层

### Task 12：更新 pipeline.go（新状态 + ChatSession 扩展）

**Files:**
- Modify: `backend/internal/service/pipeline.go`

- [ ] **Step 1: 新增状态常量**

在 `pipeline.go` 的 `const (` 块中，添加新状态：

```go
const (
    StateIdle       SessionState = iota
    StateAnalyzing               // running parallel analysis
    StateDebating                // running debate coordination  ← 新增
    StateAwaiting                // waiting for user outline confirmation
    StateWriting                 // writing final draft
    StateReviewing               // running quality gate          ← 新增
    StateComplete
)
```

- [ ] **Step 2: 扩展 ChatSession 字段**

在 `ChatSession` 结构体中添加：

```go
// Multi-agent fields
DNAReport     string  // viral analyzer output
MaterialsReport string // material enricher output
StyleDoc       string  // user style doc (cached for session)
DebateLog      string  // debate log JSON for persistence
```

- [ ] **Step 3: 在 FlushConversation 中支持 debate_log**

在 `FlushConversation` 函数中，添加 `debate_log` 的持久化：

```go
func FlushConversation(sess *ChatSession, state int, scriptID *uint) {
    if sess.ConvID == 0 {
        return
    }
    updates := map[string]interface{}{"state": state}
    if scriptID != nil {
        updates["script_id"] = *scriptID
    }
    if sess.DebateLog != "" {
        updates["debate_log"] = sess.DebateLog
    }
    _ = repository.UpdateConversationMeta(sess.ConvID, updates)
}
```

- [ ] **Step 4: 删除 pipeline.go 中与 prompts.go 重复的函数**

Task 5 在 `prompts.go` 中重新定义了 `ParseOutlineFromAnalysis`、`OutlineData` 等。`pipeline.go` 中的旧版本需要删除，避免编译时 "redeclared" 错误。

删除 `pipeline.go` 中以下内容（搜索并移除）：
- `OutlineData` 结构体定义（如果存在）
- `ParseOutlineFromAnalysis` 函数定义
- `StyleProfile` 结构体定义（已移至 prompts.go）

保留 `pipeline.go` 中的：`SessionState`, `ChatSession`, `FlushConversation`, `EnsureConversation`, `SaveScript`, `SetState`, `IsURL`, `ExtractURL`

- [ ] **Step 5: 编译验证**

```bash
cd backend && go build .
```
Expected: 无错误

- [ ] **Step 6: commit**

```bash
git add backend/internal/service/pipeline.go
git commit -m "feat: add StateDebating/StateReviewing and extend ChatSession for multi-agent"
```

---

### Task 13：更新 chat_handler.go（新 SSE 事件 + 重构 handleIdle + 超时恢复）

**Files:**
- Modify: `backend/internal/handler/chat_handler.go`

- [ ] **Step 1: 在 sseWriter 中添加新事件方法**

在 `chat_handler.go` 的 `sseWriter` 方法组末尾添加：

```go
func (w *sseWriter) sendDebate(agentName, token string) {
    w.send("msg", map[string]string{"type": "debate", "agent": agentName, "content": token})
}

func (w *sseWriter) sendQuality(data interface{}) {
    w.send("msg", map[string]interface{}{"type": "quality", "data": data})
}

func (w *sseWriter) sendRetry(count int, reason string) {
    w.send("msg", map[string]interface{}{"type": "retry", "count": count, "reason": reason})
}

func (w *sseWriter) sendSimilarity(data interface{}) {
    w.send("msg", map[string]interface{}{"type": "similarity", "data": data})
}
```

- [ ] **Step 2: 在 SendMessage 中扩展超时恢复逻辑**

将现有的超时恢复 switch 扩展为（`StateReviewing` 使用单独的 5 分钟超时）：

```go
const stuckTimeout = 3 * time.Minute
const reviewTimeout = 5 * time.Minute

elapsed := time.Since(sess.StateChangedAt)
if (sess.State == service.StateReviewing && elapsed > reviewTimeout) ||
    (sess.State != service.StateReviewing && elapsed > stuckTimeout) {
    switch sess.State {
    case service.StateAnalyzing, service.StateDebating:
        sess.SetState(service.StateIdle)
        w.sendInfo("⚠️ 上次分析超时，已自动重置，请重新发送内容。")
    case service.StateWriting:
        sess.SetState(service.StateAwaiting)
        w.sendInfo("⚠️ 上次撰写超时，已恢复到大纲确认阶段，发送 \"1\" 重新撰写终稿。")
    case service.StateReviewing:
        sess.SetState(service.StateAwaiting)
        w.sendInfo("⚠️ 质量审核超时，已恢复到大纲确认阶段，发送 \"1\" 重新撰写终稿。")
    }
}
```

- [ ] **Step 3: 重写 handleIdle（加 is_initialized 检查，替换旧并行逻辑）**

用以下内容替换 `handleIdle` 函数（完整替换，不是追加）：

```go
func handleIdle(w *sseWriter, sess *service.ChatSession, userID uint, input string) {
    // Check style initialization
    style, err := repository.GetStyleByUserID(userID)
    if err != nil || !style.IsInitialized {
        w.sendError("请先初始化您的个人风格档案。请在页面顶部点击风格初始化引导，输入您的历史口播稿。")
        sess.SetState(service.StateIdle)
        return
    }
    sess.StyleDoc = style.StyleDoc

    sess.SetState(service.StateAnalyzing)
    input = strings.TrimSpace(input)

    title := input
    runes := []rune(title)
    if len(runes) > 30 {
        title = string(runes[:30]) + "..."
    }
    service.EnsureConversation(sess, title)
    if sess.ConvID != 0 {
        _ = repository.UpdateConversationTitle(sess.ConvID, title)
    }

    addMsg(sess, service.StoredMsg{Role: "user", Type: "text", Content: input})

    // Step 1: Extract text
    w.sendStep(1, "获取原稿内容")
    addMsg(sess, service.StoredMsg{Role: "assistant", Type: "step", Step: 1, Name: "获取原稿内容"})
    if service.IsURL(input) {
        text, err := service.ExtractURL(input)
        if err != nil {
            w.sendError("无法提取URL内容：" + err.Error())
            sess.SetState(service.StateIdle)
            return
        }
        sess.OriginalText = text
        sess.SourceURL = input
        w.sendInfo(fmt.Sprintf("✅ 已提取 %d 字", len([]rune(text))))
        addMsg(sess, service.StoredMsg{Role: "assistant", Type: "info", Content: fmt.Sprintf("已提取 %d 字", len([]rune(text)))})
    } else {
        sess.OriginalText = input
    }

    // Step 2: Load style
    w.sendStep(2, "读取风格档案")
    addMsg(sess, service.StoredMsg{Role: "assistant", Type: "step", Step: 2, Name: "读取风格档案"})

    // Step 3: Parallel analysis
    w.sendStep(3, "并行分析中")
    addMsg(sess, service.StoredMsg{Role: "assistant", Type: "step", Step: 3, Name: "并行分析中"})

    tasks := []service.AgentTask{
        {Name: "爆款解构师", System: "", Prompt: ""}, // prompts built inside runner
        {Name: "素材补齐师", System: "", Prompt: ""},
    }

    var dnaReport, materialsReport string
    var analysisMu sync.Mutex

    // Run in goroutines with SSE token streaming
    var wg sync.WaitGroup
    wg.Add(2)

    go func() {
        defer wg.Done()
        report, err := service.RunViralAnalyzer(userID, sess.OriginalText, func(token string) {
            w.sendDebate("爆款解构师", token)
        })
        if err == nil {
            analysisMu.Lock()
            dnaReport = report
            analysisMu.Unlock()
        }
    }()

    go func() {
        defer wg.Done()
        report, err := service.RunMaterialEnricher(userID, sess.OriginalText, func(token string) {
            w.sendDebate("素材补齐师", token)
        })
        if err == nil {
            analysisMu.Lock()
            materialsReport = report
            analysisMu.Unlock()
        }
    }()

    wg.Wait()
    sess.DNAReport = dnaReport
    sess.MaterialsReport = materialsReport

    // Step 4: Debate coordination
    sess.SetState(service.StateDebating)
    w.sendStep(4, "辩论协调中")
    addMsg(sess, service.StoredMsg{Role: "assistant", Type: "step", Step: 4, Name: "辩论协调中"})

    outlineData, debateLog, err := service.RunDebate(
        sess.DNAReport,
        sess.MaterialsReport,
        sess.StyleDoc,
        func(agentName, token string) {
            w.sendDebate(agentName, token)
        },
    )
    if err != nil || outlineData == nil {
        w.sendError("辩论协调失败，请重试")
        sess.SetState(service.StateIdle)
        return
    }
    sess.DebateLog = debateLog
    sess.OutlineData = outlineData

    // Serialize outline for storage
    outlineBytes, _ := json.Marshal(outlineData)
    sess.OutlineJSON = string(outlineBytes)

    // Step 5: Present outline
    w.sendStep(5, "大纲已生成")
    addMsg(sess, service.StoredMsg{Role: "assistant", Type: "step", Step: 5, Name: "大纲已生成"})

    w.sendOutline(outlineData)
    addMsg(sess, service.StoredMsg{Role: "assistant", Type: "outline", Data: json.RawMessage(outlineBytes)})

    options := []string{"1 - 确认大纲，开始撰写", "2 - 补充备注后确认", "3 - 更换素材重新分析", "4 - 完全重新分析"}
    w.sendAction(options)
    addMsg(sess, service.StoredMsg{Role: "assistant", Type: "action", Options: options})

    sess.SetState(service.StateAwaiting)
    service.FlushConversation(sess, 0, nil)
}
```

> **注意：** 在函数顶部添加所需 import（`sync`, `encoding/json`, `fmt`）。

- [ ] **Step 3b: 在 handleAwaiting 中调用 ProcessFeedback（当用户输入非数字反馈时）**

找到 `handleAwaiting` 函数（`chat_handler.go` 中），在处理用户输入逻辑的末尾，当输入不匹配选项 1/2/3/4 时（即视为自由文本反馈），添加：

```go
// 当用户输入不匹配1/2/3/4时，视为大纲修改意见
// 在 "default" 或等效分支中添加：
default:
    // Treat non-option input as outline feedback → update agent prompts
    go func() {
        _ = service.ProcessFeedback(userID, "outline_note", input)
    }()
    // Then re-present the outline for confirmation
    if sess.OutlineData != nil {
        outlineBytes, _ := json.Marshal(sess.OutlineData)
        w.sendOutline(sess.OutlineData)
        options := []string{"1 - 确认大纲，开始撰写", "2 - 补充备注后确认", "3 - 更换素材重新分析", "4 - 完全重新分析"}
        w.sendAction(options)
    }
```

- [ ] **Step 4: 重写 writeFinalDraft（使用新 QualityGate）**

用以下内容替换 `writeFinalDraft` 函数：

```go
func writeFinalDraft(w *sseWriter, sess *service.ChatSession, userID uint) {
    sess.SetState(service.StateWriting)

    styleProfile, _ := repository.GetStyleByUserID(userID)
    styleDoc := ""
    if styleProfile != nil {
        styleDoc = styleProfile.StyleDoc
    }

    // Step 6: Create initial draft
    w.sendStep(6, "创作初稿")
    addMsg(sess, service.StoredMsg{Role: "assistant", Type: "step", Step: 6, Name: "创作初稿"})

    draft, err := service.RunCreatorAgent(
        userID,
        sess.OutlineJSON,
        sess.DNAReport,
        sess.MaterialsReport,
        styleDoc,
        "", // no issues on first attempt
        func(token string) { w.sendToken(token) },
    )
    if err != nil {
        w.sendError("初稿创作失败：" + err.Error())
        sess.SetState(service.StateAwaiting)
        return
    }
    addMsg(sess, service.StoredMsg{Role: "assistant", Type: "text", Content: draft})

    // Step 7: Quality gate
    sess.SetState(service.StateReviewing)
    w.sendStep(7, "质量审核中")
    addMsg(sess, service.StoredMsg{Role: "assistant", Type: "step", Step: 7, Name: "质量审核中"})

    finalDraft, qResult, err := service.RunQualityGate(
        userID,
        draft,
        sess.OriginalText,
        sess.OutlineJSON,
        sess.DNAReport,
        sess.MaterialsReport,
        styleDoc,
        2, // maxRetry
        func(eventType string, data interface{}) {
            switch eventType {
            case "quality":
                w.sendQuality(data)
            case "retry":
                if m, ok := data.(map[string]interface{}); ok {
                    reason, _ := m["reason"].(string)
                    count, _ := m["count"].(int)
                    w.sendRetry(count, reason)
                    addMsg(sess, service.StoredMsg{Role: "assistant", Type: "info",
                        Content: fmt.Sprintf("质量未通过，第%d次重新创作...", count)})
                }
            }
        },
    )
    if err != nil {
        w.sendError("质量审核异常：" + err.Error())
        sess.SetState(service.StateAwaiting)
        return
    }

    sess.FinalDraft = service.StripQualityCheck(finalDraft)

    // Save quality report to DB after script is saved (done in SaveScript below)
    // Step 8: Save script
    w.sendStep(8, "保存稿件")
    addMsg(sess, service.StoredMsg{Role: "assistant", Type: "step", Step: 8, Name: "保存稿件"})

    script, err := service.SaveScript(userID, sess, qResult.SimilarityScore, 7.5)
    if err != nil {
        w.sendError("保存失败：" + err.Error())
        return
    }

    // Save quality report linked to script
    qReport := &model.QualityReport{
        ScriptID:        script.ID,
        SimilarityScore: qResult.SimilarityScore,
        FactScore:       qResult.FactScore,
        LogicScore:      qResult.LogicScore,
        ExpressionScore: qResult.ExpressionScore,
        Passed:          qResult.Passed,
        RetryCount:      qResult.RetryCount,
    }
    if issueJSON, err2 := json.Marshal(qResult.Issues); err2 == nil {
        qReport.Issues = string(issueJSON)
    }
    _ = repository.CreateQualityReport(qReport)

    w.sendComplete(script.ID)
    addMsg(sess, service.StoredMsg{Role: "assistant", Type: "complete"})
    sess.SetState(service.StateComplete)
}
```

- [ ] **Step 5: 编译整个后端**

```bash
cd backend && go build .
```
Expected: 无错误

- [ ] **Step 6: commit**

```bash
git add backend/internal/handler/chat_handler.go
git commit -m "feat: refactor chat_handler for multi-agent pipeline (debate+quality gate)"
```

---

### Task 14：style_handler.go + feedback 路由

**Files:**
- Create: `backend/internal/handler/style_handler.go`
- Modify: `backend/main.go`

- [ ] **Step 1: 创建 style_handler.go**

```go
// backend/internal/handler/style_handler.go
package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"content-creator-imm/internal/repository"
	"content-creator-imm/internal/service"

	"github.com/gin-gonic/gin"
)

// InitStyle handles POST /api/user/style/init
// Accepts historical scripts, runs style modeler, streams progress via SSE.
func InitStyle(c *gin.Context) {
	userID := c.GetUint("userID")

	var req struct {
		Scripts []string `json:"scripts" binding:"required,min=1"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请至少提供1篇历史口播稿"})
		return
	}

	// Set SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	w := &sseWriter{c}

	err := service.InitializeStyle(userID, req.Scripts, func(eventType, content string) {
		w.send("msg", map[string]string{"type": eventType, "content": content})
	})
	if err != nil {
		w.sendError("风格建模失败：" + err.Error())
		return
	}

	// Fetch updated style to return in complete event
	style, _ := repository.GetStyleByUserID(userID)
	completeData := map[string]interface{}{
		"is_initialized": true,
		"style_version":  0,
		"style_vector":   "{}",
	}
	if style != nil {
		completeData["style_version"] = style.StyleVersion
		completeData["style_vector"] = style.StyleVector
	}

	w.send("msg", map[string]interface{}{"type": "complete", "data": completeData})
}

// GetStyleDoc handles GET /api/user/style/doc
func GetStyleDoc(c *gin.Context) {
	userID := c.GetUint("userID")
	style, err := repository.GetStyleByUserID(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "风格档案不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"style_doc":     style.StyleDoc,
		"style_vector":  style.StyleVector,
		"style_version": style.StyleVersion,
		"is_initialized": style.IsInitialized,
	})
}

// SubmitScriptFeedback handles POST /api/scripts/:id/feedback
func SubmitScriptFeedback(c *gin.Context) {
	userID := c.GetUint("userID")

	var req struct {
		Content string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Process feedback asynchronously (non-blocking for user)
	go func() {
		_ = service.ProcessFeedback(userID, "script_feedback", req.Content)
	}()

	c.JSON(http.StatusOK, gin.H{"ok": true})
}
```

- [ ] **Step 2: 更新 main.go 注册新路由**

在 `main.go` 的 `api` 路由组中添加：

```go
// Style management
api.POST("/user/style/init", handler.InitStyle)
api.GET("/user/style/doc", handler.GetStyleDoc)

// Script feedback
api.POST("/scripts/:id/feedback", handler.SubmitScriptFeedback)
```

- [ ] **Step 3: 更新 GetProfile 返回 is_initialized 字段**

找到 `backend/internal/handler/auth_handler.go` 或 user profile handler，确保 `GET /api/user/profile` 返回的 UserStyle 包含 `is_initialized` 字段。由于 `UserStyle` struct 已经有 `json:"is_initialized"` tag，AutoMigrate 后字段会自动包含在 JSON 响应中。

- [ ] **Step 3b: 更新 GetScript 返回 quality_report 字段**

找到 `GET /api/scripts/:id` 的 handler（`script_handler.go` 或类似文件），在查询 Script 并返回结果时，额外查询关联的 QualityReport 并嵌套返回：

```go
// 在查询 Script 后添加：
qReport, _ := repository.GetQualityReportByScriptID(script.ID)

// 在 c.JSON 响应中添加 quality_report 字段：
c.JSON(http.StatusOK, gin.H{
    // ...现有字段...
    "quality_report": qReport, // nil if not available (older scripts pre-refactor)
})
```

- [ ] **Step 4: 编译整个后端**

```bash
cd backend && go build .
```
Expected: 无错误

- [ ] **Step 5: commit**

```bash
git add backend/internal/handler/style_handler.go backend/main.go
git commit -m "feat: add StyleHandler (init/doc) and feedback endpoint, register routes"
```

---

### Task 15：构建后端并验证基本功能

- [ ] **Step 1: 构建二进制**

```bash
cd backend && go build -o ../content-creator-imm .
```
Expected: 无错误

- [ ] **Step 2: 重启服务**

```bash
./manage.sh restart
```

- [ ] **Step 3: 验证 API 联通（health check）**

```bash
# 登录获取 token
TOKEN=$(curl -s http://localhost/creator/api/auth/login \
  -X POST -H "Content-Type: application/json" \
  -d '{"email":"test2@test.com","password":"Test1234"}' | python3 -c "import sys,json; print(json.load(sys.stdin)['token'])")
echo "TOKEN: $TOKEN"

# 验证 /api/user/profile 包含 is_initialized 字段
curl -s http://localhost/creator/api/user/profile \
  -H "Authorization: Bearer $TOKEN" | python3 -c "import sys,json; d=json.load(sys.stdin); print('is_initialized:', d.get('style',{}).get('is_initialized','MISSING'))"
```
Expected: `is_initialized: False` 或 `is_initialized: false`

- [ ] **Step 4: commit**

```bash
git add .
git commit -m "chore: backend build verified, new routes live"
```

---

## Phase 5：前端（React 18 + TypeScript + Tailwind v4 + Radix UI）

> ⚠️ 前端实际框架为 **React 18**，不是 Vue 3。状态管理用 React Context + useReducer，UI 用 Radix UI + Tailwind，无 Pinia/Element Plus/Vue Router。

### Task 16：扩展 AuthContext + 创建 style API

**Files:**
- Modify: `frontend/src/contexts/AuthContext.tsx`
- Create: `frontend/src/api/style.ts`

- [ ] **Step 1: 在 AuthContext 中添加风格状态**

在 `frontend/src/contexts/AuthContext.tsx` 中扩展接口和 state：

```typescript
// 在 AuthContextValue 接口中添加
interface AuthContextValue {
  // ...现有字段...
  isStyleInitialized: boolean
  styleVersion: number
  setStyleStatus: (initialized: boolean, version: number) => void
}

// 在 AuthProvider 中添加 state
const [isStyleInitialized, setIsStyleInitialized] = useState<boolean>(false)
const [styleVersion, setStyleVersion] = useState<number>(0)

const setStyleStatus = (initialized: boolean, version: number) => {
  setIsStyleInitialized(initialized)
  setStyleVersion(version)
}

// 在 return 的 value 中添加
<AuthContext.Provider value={{ ..., isStyleInitialized, styleVersion, setStyleStatus }}>
```

- [ ] **Step 2: 创建 style API**

```typescript
// frontend/src/api/style.ts

export interface StyleDoc {
  style_doc: string
  style_vector: string
  style_version: number
  is_initialized: boolean
}

export async function getStyleDoc(token: string): Promise<StyleDoc> {
  const res = await fetch('/creator/api/user/style/doc', {
    headers: { Authorization: `Bearer ${token}` },
  })
  if (!res.ok) throw new Error('获取风格档案失败')
  return res.json()
}

export interface StyleInitCompleteData {
  is_initialized: boolean
  style_version: number
  style_vector: string
}

// initStyle streams the style init SSE.
export async function initStyle(
  token: string,
  scripts: string[],
  onToken: (content: string) => void,
  onProgress: (content: string) => void,
  onComplete: (data: StyleInitCompleteData) => void,
  onError: (msg: string) => void,
): Promise<void> {
  const res = await fetch('/creator/api/user/style/init', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify({ scripts }),
  })
  if (!res.ok || !res.body) {
    onError('请求失败')
    return
  }
  const reader = res.body.getReader()
  const decoder = new TextDecoder()
  let buf = ''
  while (true) {
    const { done, value } = await reader.read()
    if (done) break
    buf += decoder.decode(value, { stream: true })
    const lines = buf.split('\n')
    buf = lines.pop() ?? ''
    for (const line of lines) {
      if (!line.startsWith('data: ')) continue
      try {
        const event = JSON.parse(line.slice(6)) as Record<string, unknown>
        if (event.type === 'complete') {
          onComplete(event.data as StyleInitCompleteData)
        } else if (event.type === 'error') {
          onError((event.message as string) || '未知错误')
        } else if (event.type === 'style_init') {
          onProgress((event.content as string) || '')
        } else if (event.type === 'token') {
          onToken((event.content as string) || '')
        }
      } catch { /* ignore */ }
    }
  }
}

export async function submitScriptFeedback(
  token: string,
  scriptId: number,
  content: string,
): Promise<void> {
  await fetch(`/creator/api/scripts/${scriptId}/feedback`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify({ content }),
  })
}
```

- [ ] **Step 3: 编译验证**

```bash
cd frontend && npx tsc --noEmit 2>&1 | head -20
```
Expected: 无 type error

- [ ] **Step 4: commit**

```bash
git add frontend/src/contexts/AuthContext.tsx frontend/src/api/style.ts
git commit -m "feat: add style state to AuthContext and style init API"
```

---

### Task 17：StyleInitBanner 组件 + Dashboard 集成

**Files:**
- Create: `frontend/src/components/StyleInitBanner.tsx`
- Modify: `frontend/src/pages/Dashboard.tsx`

- [ ] **Step 1: 创建 StyleInitBanner.tsx**

```tsx
// frontend/src/components/StyleInitBanner.tsx
import { useState, useCallback } from 'react'
import { toast } from 'sonner'
import { useAuth } from '../contexts/AuthContext'
import { initStyle } from '../api/style'

export function StyleInitBanner() {
  const { token, isStyleInitialized, setStyleStatus } = useAuth()
  const [open, setOpen] = useState(false)
  const [scripts, setScripts] = useState(['', '', ''])
  const [streaming, setStreaming] = useState(false)
  const [progress, setProgress] = useState('')
  const [output, setOutput] = useState('')

  const handleScriptChange = (index: number, value: string) => {
    setScripts((prev) => prev.map((s, i) => (i === index ? value : s)))
  }

  const handleSubmit = useCallback(async () => {
    const filtered = scripts.map((s) => s.trim()).filter((s) => s.length > 50)
    if (filtered.length === 0) {
      toast.error('请至少输入一篇有效口播稿（50字以上）')
      return
    }
    setStreaming(true)
    setOutput('')
    setProgress('')
    try {
      await initStyle(
        token!,
        filtered,
        (content) => setOutput((prev) => prev + content),
        (content) => setProgress(content),
        (data) => {
          setStyleStatus(data.is_initialized, data.style_version)
          toast.success('风格档案初始化完成！')
          setOpen(false)
        },
        (msg) => toast.error(msg),
      )
    } finally {
      setStreaming(false)
    }
  }, [scripts, token, setStyleStatus])

  if (isStyleInitialized) return null

  return (
    <>
      {/* Banner */}
      <div className="mx-4 mt-4 px-4 py-3 rounded-xl bg-amber-50 dark:bg-amber-950/30 border border-amber-200 dark:border-amber-800 flex items-center justify-between gap-3">
        <p className="text-sm text-amber-800 dark:text-amber-300">
          请先初始化您的个人风格档案，AI 将基于您的历史口播稿为您定制专属写作风格
        </p>
        <button
          onClick={() => setOpen(true)}
          className="flex-shrink-0 px-3 py-1.5 text-xs font-medium bg-amber-500 hover:bg-amber-600 text-white rounded-lg transition-colors"
        >
          立即初始化
        </button>
      </div>

      {/* Dialog */}
      {open && (
        <div className="fixed inset-0 z-50 flex items-center justify-center">
          <div className="absolute inset-0 bg-black/50" onClick={() => !streaming && setOpen(false)} />
          <div className="relative z-10 w-full max-w-2xl mx-4 bg-white dark:bg-gray-900 rounded-2xl shadow-2xl overflow-hidden">
            <div className="px-6 py-5 border-b border-gray-200 dark:border-gray-700">
              <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">初始化风格档案</h2>
              <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">粘贴您过往的口播稿（至少1篇，最多3篇），AI 将分析您的写作风格</p>
            </div>
            <div className="px-6 py-4 space-y-3 max-h-[50vh] overflow-y-auto">
              {scripts.map((s, i) => (
                <div key={i}>
                  <label className="block text-xs font-medium text-gray-600 dark:text-gray-400 mb-1">
                    历史口播稿 {i + 1}{i === 0 ? '（必填）' : '（选填）'}
                  </label>
                  <textarea
                    value={s}
                    onChange={(e) => handleScriptChange(i, e.target.value)}
                    disabled={streaming}
                    placeholder={`粘贴第 ${i + 1} 篇口播稿内容...`}
                    className="w-full h-28 p-3 text-sm border border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 text-gray-900 dark:text-gray-100 rounded-lg resize-none focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:opacity-50"
                  />
                </div>
              ))}
              {streaming && (
                <div className="rounded-lg bg-blue-50 dark:bg-blue-950/30 border border-blue-100 dark:border-blue-900 p-3">
                  {progress && <p className="text-xs text-blue-600 dark:text-blue-400 mb-2">{progress}</p>}
                  {output && <p className="text-sm text-gray-700 dark:text-gray-300 whitespace-pre-wrap leading-relaxed">{output}</p>}
                  {!output && !progress && <p className="text-sm text-blue-600 dark:text-blue-400 animate-pulse">正在分析...</p>}
                </div>
              )}
            </div>
            <div className="px-6 py-4 border-t border-gray-200 dark:border-gray-700 flex justify-end gap-3">
              <button
                onClick={() => setOpen(false)}
                disabled={streaming}
                className="px-4 py-2 text-sm text-gray-600 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-200 disabled:opacity-50 transition-colors"
              >
                取消
              </button>
              <button
                onClick={handleSubmit}
                disabled={streaming || scripts.every((s) => s.trim().length < 50)}
                className="px-4 py-2 text-sm font-medium bg-gradient-to-r from-blue-500 to-purple-600 text-white rounded-lg hover:opacity-90 disabled:opacity-40 disabled:cursor-not-allowed transition-opacity"
              >
                {streaming ? '分析中...' : '开始初始化'}
              </button>
            </div>
          </div>
        </div>
      )}
    </>
  )
}
```

- [ ] **Step 2: 在 Dashboard 集成 StyleInitBanner**

在 `frontend/src/pages/Dashboard.tsx` 的 idle 阶段 JSX 中，在 `<div className="max-w-3xl mx-auto px-4 pt-16">` 之前插入 `<StyleInitBanner />`：

```tsx
// 在 import 区域顶部添加
import { StyleInitBanner } from '../components/StyleInitBanner'

// 在 idle 状态渲染的 <div className="flex-1 overflow-y-auto ..."> 内
// 将 <div className="max-w-3xl mx-auto px-4 pt-16"> 改为外层加 Banner：
<div className="flex-1 overflow-y-auto bg-gradient-to-br from-blue-50 via-white to-purple-50 dark:from-gray-950 dark:via-gray-900 dark:to-gray-950">
  <StyleInitBanner />
  <div className="max-w-3xl mx-auto px-4 pt-16">
    {/* ...现有内容不变... */}
  </div>
</div>
```

- [ ] **Step 3: 在 AuthContext 加载 profile 时同步 style 状态**

在 `AuthContext.tsx` 中 `fetchProfile`（或等效函数）读取 `/api/user/profile` 后，将 `is_initialized` 和 `style_version` 写入 context：

```typescript
// 在 fetchProfile 返回数据解析处添加：
if (data.is_initialized !== undefined) {
  setStyleStatus(!!data.is_initialized, data.style_version ?? 0)
}
```

- [ ] **Step 4: 编译验证**

```bash
cd frontend && npx tsc --noEmit 2>&1 | head -20
```
Expected: 无 type error

- [ ] **Step 5: commit**

```bash
git add frontend/src/components/StyleInitBanner.tsx frontend/src/pages/Dashboard.tsx frontend/src/contexts/AuthContext.tsx
git commit -m "feat: add StyleInitBanner component with style init dialog"
```

---

### Task 18：DebateBubble + QualityCard 组件

**Files:**
- Create: `frontend/src/components/create/DebateBubble.tsx`
- Create: `frontend/src/components/create/QualityCard.tsx`

- [ ] **Step 1: 创建 DebateBubble.tsx**

```tsx
// frontend/src/components/create/DebateBubble.tsx

// 每个辩论发言显示为独立气泡
// agent 字段决定颜色：爆款解构师=蓝，素材补齐师=绿，协调者=橙

const AGENT_STYLES: Record<string, { label: string; bg: string; text: string; dot: string }> = {
  '爆款解构师': {
    label: '爆款解构师',
    bg: 'bg-blue-50 dark:bg-blue-950/30 border-blue-100 dark:border-blue-900',
    text: 'text-blue-800 dark:text-blue-200',
    dot: 'bg-blue-400',
  },
  '素材补齐师': {
    label: '素材补齐师',
    bg: 'bg-green-50 dark:bg-green-950/30 border-green-100 dark:border-green-900',
    text: 'text-green-800 dark:text-green-200',
    dot: 'bg-green-400',
  },
  '协调者': {
    label: '协调者',
    bg: 'bg-orange-50 dark:bg-orange-950/30 border-orange-100 dark:border-orange-900',
    text: 'text-orange-800 dark:text-orange-200',
    dot: 'bg-orange-400',
  },
}

const DEFAULT_STYLE = AGENT_STYLES['协调者']

interface DebateBubbleProps {
  agent: string
  content: string
  streaming?: boolean
}

export function DebateBubble({ agent, content, streaming }: DebateBubbleProps) {
  const style = AGENT_STYLES[agent] ?? DEFAULT_STYLE
  return (
    <div className={`rounded-xl border p-3 text-sm ${style.bg}`}>
      <div className="flex items-center gap-2 mb-1.5">
        <span className={`w-2 h-2 rounded-full flex-shrink-0 ${style.dot}`} />
        <span className={`text-xs font-semibold ${style.text}`}>{agent}</span>
      </div>
      <p className={`leading-relaxed whitespace-pre-wrap ${style.text}`}>
        {content}
        {streaming && <span className="inline-block w-0.5 h-4 ml-0.5 animate-pulse align-middle bg-current opacity-60" />}
      </p>
    </div>
  )
}
```

- [ ] **Step 2: 创建 QualityCard.tsx**

```tsx
// frontend/src/components/create/QualityCard.tsx

export interface QualityReportData {
  similarity_score: number
  fact_score: number
  logic_score: number
  expression_score: number
  passed: boolean
  retry_count: number
  issues: string[]
}

interface QualityCardProps {
  data: QualityReportData
}

function ScoreBar({ label, score, threshold }: { label: string; score: number; threshold: number }) {
  const pct = Math.min(Math.round(score), 100)
  const passed = score >= threshold
  return (
    <div className="space-y-1">
      <div className="flex items-center justify-between text-xs">
        <span className="text-gray-600 dark:text-gray-400">{label}</span>
        <span className={passed ? 'text-green-600 dark:text-green-400' : 'text-red-500'}>
          {score.toFixed(1)} {passed ? '✓' : `< ${threshold}`}
        </span>
      </div>
      <div className="h-1.5 rounded-full bg-gray-200 dark:bg-gray-700 overflow-hidden">
        <div
          className={`h-full rounded-full transition-all ${passed ? 'bg-green-400' : 'bg-red-400'}`}
          style={{ width: `${pct}%` }}
        />
      </div>
    </div>
  )
}

export function QualityCard({ data }: QualityCardProps) {
  return (
    <div className={`rounded-xl border p-3 text-sm space-y-3 ${
      data.passed
        ? 'bg-green-50 dark:bg-green-950/30 border-green-100 dark:border-green-900'
        : 'bg-orange-50 dark:bg-orange-950/30 border-orange-100 dark:border-orange-900'
    }`}>
      <div className="flex items-center gap-2">
        <span className="text-base">{data.passed ? '✅' : '⚠️'}</span>
        <span className={`text-xs font-semibold ${data.passed ? 'text-green-700 dark:text-green-300' : 'text-orange-700 dark:text-orange-300'}`}>
          质量审核 {data.passed ? '通过' : `未通过（已重试 ${data.retry_count} 次）`}
        </span>
      </div>
      <div className="space-y-2">
        <ScoreBar label="相似度（越低越好）" score={100 - data.similarity_score} threshold={70} />
        <ScoreBar label="事实准确性" score={data.fact_score} threshold={90} />
        <ScoreBar label="逻辑连贯性" score={data.logic_score} threshold={85} />
        <ScoreBar label="表达质量" score={data.expression_score} threshold={75} />
      </div>
      {data.issues.length > 0 && (
        <div className="space-y-1">
          <p className="text-xs font-medium text-orange-700 dark:text-orange-300">发现问题：</p>
          <ul className="space-y-0.5">
            {data.issues.map((issue, i) => (
              <li key={i} className="text-xs text-orange-600 dark:text-orange-400 flex items-start gap-1">
                <span className="mt-0.5 flex-shrink-0">•</span>
                {issue}
              </li>
            ))}
          </ul>
        </div>
      )}
    </div>
  )
}
```

- [ ] **Step 3: 编译验证**

```bash
cd frontend && npx tsc --noEmit 2>&1 | head -20
```
Expected: 无 type error

- [ ] **Step 4: commit**

```bash
git add frontend/src/components/create/DebateBubble.tsx frontend/src/components/create/QualityCard.tsx
git commit -m "feat: add DebateBubble and QualityCard components"
```

---

### Task 19：SSE 类型扩展 + Dashboard + MessageList 集成

**Files:**
- Modify: `frontend/src/lib/sse.ts`
- Modify: `frontend/src/pages/Dashboard.tsx`
- Modify: `frontend/src/components/create/MessageList.tsx`

- [ ] **Step 1: 扩展 sse.ts 新增事件类型**

将 `frontend/src/lib/sse.ts` 中的 `SSEEvent` union type 替换为：

```typescript
export type SSEEvent =
  | { type: 'token'; content: string }
  | { type: 'step'; step: number; name: string }
  | { type: 'info'; content: string }
  | { type: 'outline'; data: unknown }
  | { type: 'action'; options: string[] }
  | { type: 'similarity'; data: unknown }
  | { type: 'complete'; scriptId: number }
  | { type: 'error'; message: string }
  | { type: 'debate'; agent: string; content: string }
  | { type: 'quality'; data: QualityReportSSE }
  | { type: 'retry'; count: number; reason: string }
  | { type: 'style_init'; content: string }

export interface QualityReportSSE {
  similarity_score: number
  fact_score: number
  logic_score: number
  expression_score: number
  passed: boolean
  retry_count: number
  issues: string[]
}
```

- [ ] **Step 2: 扩展 Dashboard.tsx 的 ChatMsg 和 reducer**

在 `frontend/src/pages/Dashboard.tsx` 中：

**2a. 扩展 ChatMsg 类型**（已在 MessageList.tsx 中定义，需同步修改 `MessageList.tsx` 中的 `ChatMsg`，见 Step 3）

**2b. 添加 reducer action 类型：**

在 `Action` union type 末尾添加：
```typescript
| { type: 'ADD_DEBATE_TOKEN'; agent: string; content: string }
| { type: 'ADD_QUALITY'; data: QualityReportSSE }
| { type: 'ADD_RETRY'; count: number; reason: string }
```

**2c. 添加 reducer case：**

在 `reducer` 函数的 `switch` 中添加（在 `default` 之前）：

```typescript
case 'ADD_DEBATE_TOKEN': {
  const msgs = [...state.messages]
  // Find existing streaming debate bubble for this agent, or create new one
  const lastDebate = [...msgs].reverse().find(
    (m) => m.type === 'debate' && m.debateAgent === action.agent && m.streaming
  )
  if (lastDebate) {
    const idx = msgs.indexOf(lastDebate)
    msgs[idx] = { ...lastDebate, content: (lastDebate.content ?? '') + action.content }
  } else {
    msgs.push({
      id: `${Date.now()}-db-${action.agent}`,
      type: 'debate',
      debateAgent: action.agent,
      content: action.content,
      streaming: true,
    })
  }
  return { ...state, messages: msgs }
}
case 'ADD_QUALITY':
  return {
    ...state,
    messages: [...state.messages, {
      id: `${Date.now()}-q`,
      type: 'quality',
      data: action.data,
    }],
  }
case 'ADD_RETRY':
  return {
    ...state,
    messages: [...state.messages, {
      id: `${Date.now()}-retry`,
      type: 'retry',
      content: `质量审核未通过，正在重试（第 ${action.count} 次）：${action.reason}`,
    }],
  }
```

**2d. 在 `STREAM_DONE` case 中关闭 debate streaming：**

```typescript
case 'STREAM_DONE': {
  const msgs = state.messages.map((m) =>
    m.streaming ? { ...m, streaming: false } : m
  )
  return { ...state, messages: msgs, sending: false }
}
```

（已有此逻辑，无需修改，`streaming: false` 会同时关闭 ai 和 debate 气泡）

**2e. 在 `runSSE` 的 SSE 事件处理 switch 中添加 case：**

```typescript
case 'debate':
  dispatch({ type: 'ADD_DEBATE_TOKEN', agent: event.agent, content: event.content })
  break
case 'quality':
  dispatch({ type: 'ADD_QUALITY', data: event.data })
  break
case 'retry':
  dispatch({ type: 'ADD_RETRY', count: event.count, reason: event.reason })
  break
```

**2f. 在 `handleSelectConversation` 的消息类型映射中添加新类型：**

```typescript
// 在 type 映射处添加：
: m.type === 'debate' ? 'debate'
: m.type === 'quality' ? 'quality'
: m.type === 'retry' ? 'retry'
```

**2g. 添加必要 import：**

```typescript
import type { QualityReportSSE } from '../lib/sse'
```

- [ ] **Step 3: 扩展 MessageList.tsx**

**3a. 更新 ChatMsg 接口，添加新字段：**

```typescript
export interface ChatMsg {
  id: string
  type: 'user' | 'ai' | 'step' | 'info' | 'action' | 'similarity' | 'error' | 'outline'
    | 'debate' | 'quality' | 'retry'
  content?: string
  options?: string[]
  data?: unknown
  streaming?: boolean
  debateAgent?: string  // for debate type
}
```

**3b. 在 MessageList 顶部添加 import：**

```typescript
import { DebateBubble } from './DebateBubble'
import { QualityCard, type QualityReportData } from './QualityCard'
```

**3c. 在 `MessageList` 函数中的消息渲染列表末尾（`similarity` 之后），添加新消息类型的渲染：**

```tsx
{msg.type === 'debate' && msg.debateAgent && (
  <DebateBubble
    agent={msg.debateAgent}
    content={msg.content ?? ''}
    streaming={msg.streaming}
  />
)}

{msg.type === 'quality' && !!msg.data && (
  <QualityCard data={msg.data as QualityReportData} />
)}

{msg.type === 'retry' && (
  <div className="rounded-xl px-3 py-2 bg-orange-50 dark:bg-orange-950/50 text-orange-700 dark:text-orange-400 text-sm">
    ⚠️ {msg.content}
  </div>
)}
```

**3d. 在 `debate` 类型的外层 div 中使用 `justify-start`（和 ai 一样）并显示 Bot 图标**

确认 `msg.type !== 'user'` 条件已覆盖 `debate`/`quality`/`retry` 类型，Bot 图标会自动显示。

- [ ] **Step 4: 编译验证**

```bash
cd frontend && npx tsc --noEmit 2>&1 | head -30
```
Expected: 无 type error

- [ ] **Step 5: 构建验证**

```bash
cd frontend && npm run build 2>&1 | tail -10
```
Expected: `✓ built in` 且无 error

- [ ] **Step 6: commit**

```bash
git add frontend/src/lib/sse.ts frontend/src/pages/Dashboard.tsx frontend/src/components/create/MessageList.tsx
git commit -m "feat: wire debate/quality/retry SSE events through Dashboard and MessageList"
```

---

## Phase 6：集成验证与收尾

### Task 20：全量构建 + 端到端验证

- [ ] **Step 1: 全量构建**

```bash
./build.sh
```
Expected: 无构建错误，`content-creator-imm` 二进制和 `frontend/dist/` 生成

- [ ] **Step 2: 重启服务**

```bash
./manage.sh restart
```

- [ ] **Step 3: 验证前端可访问**

```bash
curl -s -o /dev/null -w "HTTP %{http_code}\n" http://localhost/creator/
```
Expected: `HTTP 200`

- [ ] **Step 4: 验证 API 登录**

```bash
curl -s http://localhost/creator/api/auth/login \
  -X POST -H "Content-Type: application/json" \
  -d '{"email":"test2@test.com","password":"Test1234"}' \
  | python3 -c "import sys,json; d=json.load(sys.stdin); print('OK' if 'token' in d else 'FAIL', d.get('user',{}).get('username',''))"
```
Expected: `OK test2`

- [ ] **Step 5: 验证新 DB 表已创建（AutoMigrate）**

```bash
mysql -u root content_creator -e "SHOW TABLES;" 2>/dev/null | sort
```
Expected: 应包含 `agent_configs` 和 `quality_reports`

- [ ] **Step 6: 验证 style/doc 接口**

```bash
TOKEN=$(curl -s http://localhost/creator/api/auth/login \
  -X POST -H "Content-Type: application/json" \
  -d '{"email":"test2@test.com","password":"Test1234"}' | python3 -c "import sys,json; print(json.load(sys.stdin)['token'])")

curl -s http://localhost/creator/api/user/style/doc \
  -H "Authorization: Bearer $TOKEN"
```
Expected: JSON 响应（可能是 404 if not initialized，或包含 `is_initialized` 字段）

- [ ] **Step 7: 运行全量后端测试**

```bash
cd backend && go test ./... -v 2>&1 | tail -20
```
Expected: 所有测试 PASS（或合理的 SKIP）

- [ ] **Step 8: 更新 .ai_mem 文档**

更新 `.ai_mem/L0_overview.md`、`L1_modules.md`、`L2_details.md` 以反映新架构：
- L0: 更新核心流程（新增辩论/质量审核步骤），更新数据库表（7张）
- L1: 新增所有 service 模块说明，新增 SSE 事件类型，更新前端组件列表
- L2: 更新状态机流程，添加并行执行和辩论流程说明

- [ ] **Step 9: 最终 commit**

```bash
git add .ai_mem/
git commit -m "docs: update .ai_mem indexes to reflect multi-agent architecture"
```

---

## 任务依赖关系

```
Task 1 → Task 2 → Task 3 → Task 4  (数据层，顺序执行)
Task 4 → Task 5                     (prompts 依赖 model)
Task 5 → Task 6 → Task 7 → Task 8 → Task 9 → Task 10 → Task 11  (service 层，顺序)
Task 11 → Task 12 → Task 13 → Task 14 → Task 15  (handler 层，顺序)

Task 15 可与 Task 16-19 并行开始  (前端不依赖后端编译)

Task 16 → Task 17 → Task 18 → Task 19 → Task 20  (前端，顺序)
```

---

## 附：nginx 配置更新提醒

重构后总 pipeline 耗时可能超过 300s。部署时需更新 nginx：

```nginx
location /creator/api/ {
    proxy_read_timeout 600s;  # 从 300s 提升到 600s
    # 其他配置不变
}
```
