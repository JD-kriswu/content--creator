# Claw SaaS Workflow Engine 改造设计

> 将 content-creator-imm 从单次 LLM 调用的 5 角色模拟，改造为基于 Workflow Engine 的真实多 Agent 协调系统。口播稿作为第一个 Workflow 模板，架构预留扩展性支持未来更多内容创作场景。

---

## 1. 改造定位

- **基础代码**：在 content-creator-imm 现有 Go+React 代码上改造
- **核心引入**：从 go-claw-saas 吸收 Coordinator/Worker 多 Agent 协调模式
- **Agent 协作模式**：分组并行 + 串行链（非全并行，有依赖关系的按顺序执行）
- **记忆系统**：简化版（用户风格 + 对话上下文），不做 L0-L3 分层和向量搜索
- **扩展性**：通用 Workflow Engine 层 + 可插拔的 Workflow 模板，口播稿为第一个模板

---

## 2. 整体架构

```
┌─────────────────────────────────────┐
│  Frontend (React 18 + Vite)         │  小改：支持多 Worker 并行展示
├─────────────────────────────────────┤
│  API Layer (Gin + SSE)              │  改造：chat_handler 对接 Engine
├─────────────────────────────────────┤
│  Workflow Engine                    │  ★ 新增核心
│  ├── Engine: Stage 编排、执行调度    │
│  ├── Stage: parallel/serial/human   │
│  ├── Worker: 独立 LLM Agent         │
│  ├── ContextBuilder: 上下文装配     │
│  └── WorkflowLoader: YAML 加载     │
├─────────────────────────────────────┤
│  Workflow Templates (YAML)          │  口播稿 = 第一个模板
│  ├── viral_script/                  │
│  └── (future: xhs, livestream...)   │
├─────────────────────────────────────┤
│  Services (LLM, Memory, Storage)    │  现有 + 增强
├─────────────────────────────────────┤
│  Storage (MySQL + Redis + Files)    │  新增 workflow 相关表
└─────────────────────────────────────┘
```

### 核心概念

| 概念 | 说明 |
|------|------|
| **Workflow** | 一个完整的内容创作流程，由多个 Stage 组成 |
| **Stage** | 流程中的一个阶段。类型：`parallel`（并行 Worker）/ `serial`（单 Worker）/ `human`（等待用户输入） |
| **Worker** | 独立的 LLM Agent，有自己的 system prompt 和对话循环 |
| **WorkflowContext** | 上下文容器，管理 SharedContext + StageOutputs + HumanInputs |
| **WorkflowLoader** | 从 YAML 文件加载 Workflow 定义，开发模式热加载 |
| **Template** | 预定义的 Workflow 结构（YAML），如"口播稿创作" |

### 新增目录结构

```
backend/
├── internal/
│   ├── handler/              # 现有，改造 chat_handler
│   ├── service/              # 现有，保留 extractor/llm_service
│   ├── model/                # 现有，新增 workflow 相关模型
│   ├── repository/           # 现有，新增 workflow 相关 repo
│   │
│   ├── workflow/             # ★ 新增：Workflow Engine
│   │   ├── engine.go         # 引擎核心：Stage 编排、执行调度
│   │   ├── stage.go          # Stage 执行器：parallel/serial/human
│   │   ├── worker.go         # Worker Agent：独立 LLM 对话
│   │   ├── context.go        # WorkflowContext + ContextBuilder
│   │   ├── loader.go         # YAML WorkflowLoader
│   │   ├── types.go          # WorkflowDef/StageDef/WorkerDef 类型
│   │   └── registry.go       # Workflow 模板注册表
│   │
│   └── memory/               # ★ 新增：简化版记忆
│       └── style.go          # 用户风格记忆（复用 user_styles 表）
│
├── workflows/                # ★ 新增：YAML Workflow 定义
│   └── viral_script/
│       ├── workflow.yaml
│       ├── prompts/
│       │   ├── viral_decoder.yaml
│       │   ├── style_architect.yaml
│       │   ├── material_curator.yaml
│       │   ├── creative_agent.yaml
│       │   ├── optimization_agent.yaml
│       │   ├── draft_writer.yaml
│       │   └── similarity_checker.yaml
│       └── synth/
│           └── research_synth.yaml
│
└── ...
```

---

## 3. 多 Agent 上下文构建

### 三层上下文模型

```
┌─────────────────────────────────────────────┐
│  Layer 1: Shared Context (所有 Worker 共享)   │
│  ├── 原文/URL 提取的文本                      │
│  ├── 用户风格画像 (UserStyle)                 │
│  └── Workflow 元信息 (目标、约束)              │
├─────────────────────────────────────────────┤
│  Layer 2: Stage Context (下游 Stage 可见)     │
│  ├── 前序 Stage 的汇总结果 (Summary)          │
│  └── 前序 Stage 各 Worker 的原始输出           │
├─────────────────────────────────────────────┤
│  Layer 3: Worker Context (每个 Worker 独有)   │
│  ├── 角色专属 System Prompt                   │
│  └── Worker 自己的对话历史（当前为单轮）       │
└─────────────────────────────────────────────┘
```

### 上下文流转（口播稿场景）

```
用户输入: URL 或文本
    │
    ▼
Engine: 构建 SharedContext
  original_text = ExtractURL(url)
  user_style = LoadUserStyle(userID)
  workflow_meta = {goal: "口播稿改写", max_similarity: 30}
    │
    ▼
Stage1 [parallel]: 研究组
  ├── Worker1 (爆款解构师): shared_ctx → 爆款DNA分析
  ├── Worker2 (风格建模师): shared_ctx + user_style → 风格指导
  └── Worker3 (素材补齐师): shared_ctx → 补充素材
    │ (三者并行执行，各自独立调用LLM)
    ▼
Engine: LLM汇总 Stage1 结果
  synth_prompt(output1, output2, output3) → stage1_summary (结构化JSON)
    │
    ▼
Stage2 [serial]: 创作代理
  Worker4: shared_ctx + stage1_summary → 生成大纲 OutlineJSON
    │
    ▼
Stage3 [serial]: 优化代理
  Worker5: shared_ctx + stage1_summary + outline → 审查修订大纲
    │
    ▼
Stage4 [human]: 用户确认
  展示大纲 → 等待选择 (1确认/2调整/3换素材/4重分析)
    │
    ▼
Stage5 [serial]: 终稿撰写
  Worker6: shared_ctx + confirmed_outline + user_note → 300-600字口播稿
    │
    ▼
Stage6 [serial]: 相似度检测
  Worker7: original_text + final_draft → similarity_scores JSON
```

### 上下文构建接口

```go
type WorkflowContext struct {
    SharedCtx    SharedContext
    StageOutputs map[string]*StageOutput  // stageID → output
    HumanInputs  map[string]string        // humanStageID → user input
}

type SharedContext struct {
    OriginalText string
    SourceURL    string
    UserStyle    *UserStyle
    WorkflowMeta map[string]any
}

type StageOutput struct {
    StageID string
    Workers []WorkerOutput
    Summary string             // 汇总结果（parallel stage 经 LLM 合并）
}

type WorkerOutput struct {
    Name    string
    Content string
    Tokens  int
    Duration time.Duration
}
```

### 模板变量插值

Worker 的 user prompt 模板支持 `{{变量名}}` 插值，Engine 在启动 Worker 前自动替换：

| 变量 | 来源 | 示例 |
|------|------|------|
| `{{original_text}}` | SharedContext | 原文文本 |
| `{{user_style}}` | SharedContext | 格式化的用户风格 |
| `{{source_url}}` | SharedContext | 来源URL |
| `{{stage.{id}.summary}}` | StageOutputs | 某 Stage 的汇总结果 |
| `{{stage.{id}.worker.{name}.output}}` | StageOutputs | 某 Worker 的原始输出 |
| `{{human.{id}.input}}` | HumanInputs | 用户在 human stage 的输入 |
| `{{workflow.meta.{key}}}` | WorkflowMeta | workflow 级配置 |

### Stage 间结果汇总策略

并行 Stage 完成后，Engine 用额外一次 LLM 调用将多 Worker 输出汇总为结构化 JSON（SynthPrompt），保证语义连贯、去重、冲突解决。汇总 prompt 也在 YAML 中定义。

---

## 4. Prompt YAML 配置体系

### 设计目标

- Prompt 与 Go 代码解耦，修改无需重新编译
- 开发模式热加载，生产模式启动缓存 + API 触发 reload
- 每个 Worker 独立 YAML 文件，职责清晰

### workflow.yaml — 编排定义

```yaml
type: viral_script
display_name: 口播稿改写
meta:
  max_similarity: 30
  word_count: "300-600"

stages:
  - id: research
    display_name: 研究分析
    type: parallel
    workers:
      - viral_decoder
      - style_architect
      - material_curator
    synth_prompt: synth/research_synth.yaml

  - id: create
    display_name: 大纲创作
    type: serial
    workers:
      - creative_agent

  - id: optimize
    display_name: 优化审查
    type: serial
    workers:
      - optimization_agent

  - id: confirm_outline
    display_name: 确认大纲
    type: human
    prompt: "请确认大纲：1-确认开始撰写 2-调整大纲 3-更换素材 4-重新分析"
    options:
      - "确认，开始撰写"
      - "调整大纲"
      - "更换素材"
      - "重新分析"

  - id: write
    display_name: 撰写终稿
    type: serial
    workers:
      - draft_writer

  - id: similarity
    display_name: 相似度检测
    type: serial
    workers:
      - similarity_checker
```

### Worker Prompt 文件示例 — `prompts/viral_decoder.yaml`

```yaml
name: viral_decoder
display_name: 爆款解构师
max_tokens: 2000
temperature: 0.3
output_format: markdown

system: |
  你是一位爆款短视频内容解构专家。
  你的职责是分析原稿的爆款基因，评估各维度得分，提取必须保留的核心要素。
  输出必须包含：选题分析表、爆款DNA评分表（6维度各1-5分）、TOP4必保留要素。

user: |
  ## 原稿内容
  {{original_text}}

  请分析这篇原稿的爆款DNA，严格按以下格式输出：

  **选题分析**
  | 项目 | 内容 |
  |------|------|
  | 选题类型 | [痛点型/干货型/情绪型/反差型] |
  | 目标人群 | [具体描述] |
  | 核心痛点 | [最打动人的点] |
  | 爆款优势 | [为什么这个内容能火] |

  **爆款DNA评分**（各维度1-5分）
  | 维度 | 评分 | 关键分析 |
  |------|------|----------|
  | 钩子强度 | X/5 | [分析] |
  | 痛点共鸣 | X/5 | [分析] |
  | 信息密度 | X/5 | [分析] |
  | 节奏把控 | X/5 | [分析] |
  | 情绪调动 | X/5 | [分析] |
  | 行动引导 | X/5 | [分析] |
  | **综合** | **X/30** | |

  **必须保留的爆款要素（TOP4）**：
  1. [要素1]
  2. [要素2]
  3. [要素3]
  4. [要素4]
```

### `prompts/style_architect.yaml`

```yaml
name: style_architect
display_name: 风格建模师
max_tokens: 1500
temperature: 0.3
output_format: markdown

system: |
  你是一位内容风格分析专家。
  你的职责是将用户的个人风格特征映射为具体的改写指导建议。
  如果没有用户风格档案，请使用通用爆款口播风格。

user: |
  ## 原稿内容
  {{original_text}}

  ## 用户风格档案
  {{user_style}}

  请分析风格融合方向，输出各维度的改写指导：

  | 维度 | 用户特征 | 改写指导 |
  |------|----------|----------|
  | 语言风格 | [分析] | [具体要求] |
  | 情绪基调 | [分析] | [具体要求] |
  | 标志元素 | [分析] | [融入建议] |
  | 开场习惯 | [分析] | [开场方向] |
  | 结尾习惯 | [分析] | [结尾方向] |
```

### `prompts/material_curator.yaml`

```yaml
name: material_curator
display_name: 素材补齐师
max_tokens: 1500
temperature: 0.5
output_format: markdown

system: |
  你是一位内容素材研究专家。
  你的职责是为改写稿补充新的数据、反差观点、案例和金句。
  素材必须与原稿主题相关但不与原稿重复，标注来源和建议应用位置。

user: |
  ## 原稿内容
  {{original_text}}

  请提出可融入的新素材，按以下分类输出：

  | 类型 | 内容 | 来源/依据 | 建议应用位置 |
  |------|------|-----------|-------------|
  | 数据 | [数据点] | [来源] | [段落] |
  | 反差 | [反差观点] | [依据] | [段落] |
  | 案例 | [案例] | [来源] | [段落] |
  | 金句 | [金句] | [原创/改编] | [段落] |
```

### `prompts/creative_agent.yaml`

```yaml
name: creative_agent
display_name: 创作代理
max_tokens: 3000
temperature: 0.7
output_format: json

system: |
  你是专业的短视频口播稿创作代理。
  你基于研究团队的分析结果，生成结构化的创作大纲。
  大纲必须包含4段（开场/发展/升华/结尾），标注时长和情绪目标。
  必须在 ---OUTLINE_START--- 和 ---OUTLINE_END--- 标记之间输出JSON。

user: |
  ## 原稿（仅参考，不得直接引用）
  {{original_text}}

  ## 研究分析汇总
  {{stage.research.summary}}

  请基于以上分析，生成创作大纲。输出格式：

  先写出你的创作思路（简要），然后在标记之间输出JSON：

  ---OUTLINE_START---
  {
    "elements": ["必保留要素1", "要素2", "要素3", "要素4"],
    "materials": ["素材1（来源）", "素材2（来源）", "素材3（来源）"],
    "outline": [
      {"part": "开场", "duration": "Xs", "content": "[新钩子，与原稿完全不同]", "emotion": "[情绪]"},
      {"part": "发展", "duration": "Xs", "content": "[主体内容]", "emotion": "[情绪]"},
      {"part": "升华", "duration": "Xs", "content": "[核心观点]", "emotion": "[情绪]"},
      {"part": "结尾", "duration": "Xs", "content": "[引导行动]", "emotion": "[情绪]"}
    ],
    "estimated_similarity": "约XX%",
    "strategy": "[改写核心策略一句话]"
  }
  ---OUTLINE_END---
```

### `prompts/optimization_agent.yaml`

```yaml
name: optimization_agent
display_name: 优化代理
max_tokens: 3000
temperature: 0.3
output_format: json

system: |
  你是内容质量审查专家。
  你的职责是审查创作大纲的事实准确性、逻辑完整性、口播适配度。
  如发现问题，直接修正并输出修订后的完整大纲JSON。
  同时输出审查意见和辩论决策。

user: |
  ## 原稿
  {{original_text}}

  ## 研究分析
  {{stage.research.summary}}

  ## 创作大纲
  {{stage.create.worker.creative_agent.output}}

  请完成以下审查：

  **审查意见**：
  - [意见1：哪个要素不够强，如何改]
  - [意见2：素材是否有事实风险]
  - [意见3：风格融合是否自然]

  **辩论决策**：
  | 分歧点 | 创作观点 | 优化观点 | 最终决策 |
  |--------|---------|---------|---------|
  | [分歧1] | [观点] | [观点] | [决策] |

  **修订后大纲**（如无需修改则原样输出）：
  ---OUTLINE_START---
  { ... 完整大纲JSON ... }
  ---OUTLINE_END---
```

### `prompts/draft_writer.yaml`

```yaml
name: draft_writer
display_name: 终稿撰写
max_tokens: 4000
temperature: 0.8
output_format: text

system: |
  你是专业的短视频口播稿撰写专家，擅长写出口语化、高传播力的内容。

user: |
  ## 参考原稿（仅用于理解内容，不得直接引用）
  {{original_text}}

  ## 已确认大纲
  {{stage.optimize.worker.optimization_agent.output}}

  ## 用户额外要求
  {{human.confirm_outline.input}}

  ## 写作要求
  1. **字数**：约300-600字（对应1-3分钟视频）
  2. **语言**：口语化，适合直接念稿，避免书面语
  3. **情绪**：情绪曲线完整，开场吸引，结尾有力
  4. **结构**：严格按大纲段落顺序撰写
  5. **差异化**：与原稿相似度必须低于30%，开场钩子必须与原稿完全不同

  请直接输出口播稿正文（不需要标注段落名称），然后在最后输出：

  ---QUALITY_CHECK_START---
  事实核查：
  - [逐条列出引用的数据/案例，标注是否可信]

  逻辑检查：
  - [论证链是否完整，是否有矛盾]

  口播适配：
  - [是否有绕口词，停顿是否自然]
  ---QUALITY_CHECK_END---
```

### `prompts/similarity_checker.yaml`

```yaml
name: similarity_checker
display_name: 相似度检测
max_tokens: 256
temperature: 0.1
output_format: json

system: |
  你是文本相似度评估专家。严格按JSON格式输出评分，不要输出其他文字。

user: |
  请对以下两篇文章进行相似度评分。

  ## 原稿
  {{original_text}}

  ## 新稿
  {{stage.write.worker.draft_writer.output}}

  从4个维度评估相似度（每个维度0-100%），严格按JSON格式输出：

  {"vocab": 词汇相似度, "sentence": 句式相似度, "structure": 结构相似度, "viewpoint": 观点相似度, "total": 加权总分}

  计算公式：total = vocab*0.30 + sentence*0.25 + structure*0.25 + viewpoint*0.20

  只输出JSON，不要其他文字。
```

### `synth/research_synth.yaml` — 研究组汇总

```yaml
name: research_synth
display_name: 研究汇总
max_tokens: 3000
temperature: 0.3
output_format: json

system: |
  你是一位内容策划总监。负责将多位专家的分析结果汇总为结构化结论。
  去除重复信息，解决矛盾，保留最有价值的洞察。

user: |
  以下是3位专家对同一篇原稿的独立分析：

  ## 爆款解构分析
  {{stage.research.worker.viral_decoder.output}}

  ## 风格分析
  {{stage.research.worker.style_architect.output}}

  ## 素材建议
  {{stage.research.worker.material_curator.output}}

  请汇总为以下JSON格式：
  {
    "score": {
      "hook": X, "pain": X, "info": X,
      "rhythm": X, "emotion": X, "action": X, "total": X
    },
    "must_keep_elements": ["要素1", "要素2", "要素3", "要素4"],
    "style_guidance": {
      "language": "语言风格指导",
      "tone": "情绪基调指导",
      "opening": "开场方向",
      "closing": "结尾方向",
      "signature": "标志元素融入方式"
    },
    "materials": [
      {"type": "data|contrast|case|quote", "content": "...", "source": "...", "position": "..."}
    ],
    "conflicts_resolved": "如有矛盾，说明如何取舍",
    "key_insights": "一句话总结改写方向"
  }
```

### WorkflowLoader（Go 侧加载）

```go
type WorkflowLoader struct {
    basePath string  // "workflows/"
    cache    map[string]*WorkflowDef
    devMode  bool
}

func (l *WorkflowLoader) Load(workflowType string) (*WorkflowDef, error) {
    if !l.devMode {
        if cached, ok := l.cache[workflowType]; ok {
            return cached, nil
        }
    }
    // 1. 读取 workflows/{type}/workflow.yaml → 解析 stage 编排
    // 2. 遍历每个 stage 的 workers → 读取 prompts/{worker}.yaml
    // 3. 读取 synth prompt YAML（如有）
    // 4. 组装 WorkflowDef 返回
    return l.loadFromDisk(workflowType)
}

func (l *WorkflowLoader) Reload(workflowType string) error {
    // API 触发的热加载
    def, err := l.loadFromDisk(workflowType)
    if err != nil { return err }
    l.cache[workflowType] = def
    return nil
}
```

---

## 5. 业务无关数据表设计

### 设计原则

- 平台层表业务无关，所有 Workflow 类型共用
- 业务层表按场景扩展（口播稿、小红书等各自独立）
- 新增业务只需：新 workflow YAML + 业务表，不改平台表

### 表结构总览

```
平台层（业务无关）:
├── users                # 现有，不变
├── workflows            # ★ 新增：workflow 执行记录
├── workflow_stages      # ★ 新增：stage 执行记录
├── workflow_workers     # ★ 新增：worker 执行记录
├── conversations        # 改造：新增 workflow_type, workflow_id
└── messages             # 改造：新增 stage_id, worker_name

业务层（按场景扩展）:
├── user_styles          # 口播稿场景：用户风格
├── scripts              # 口播稿场景：成品脚本
└── (future: xhs_posts, livestream_scripts, ...)
```

### 新增：workflows 表

```go
type Workflow struct {
    ID          uint       `gorm:"primaryKey"`
    UserID      uint       `gorm:"index;not null"`
    Type        string     `gorm:"size:64;index;not null"`  // "viral_script", "xhs_post"
    Status      string     `gorm:"size:20;index"`           // pending/running/paused/completed/failed
    InputJSON   string     `gorm:"type:text"`               // 用户原始输入
    ContextJSON string     `gorm:"type:text"`               // SharedContext 快照
    OutputJSON  string     `gorm:"type:text"`               // 最终输出
    ConvID      *uint                                        // 关联对话
    Error       string     `gorm:"type:text"`               // 失败原因
    CreatedAt   time.Time
    UpdatedAt   time.Time
}
```

### 新增：workflow_stages 表

```go
type WorkflowStage struct {
    ID         uint       `gorm:"primaryKey"`
    WorkflowID uint       `gorm:"index;not null"`
    StageID    string     `gorm:"size:64;not null"`     // YAML 中定义的 id
    Type       string     `gorm:"size:20"`              // parallel/serial/human
    Sequence   int                                       // 执行顺序（从 0 开始）
    Status     string     `gorm:"size:20"`              // pending/running/completed/failed
    InputJSON  string     `gorm:"type:text"`            // 注入给该 Stage 的上下文
    OutputJSON string     `gorm:"type:text"`            // Stage 汇总结果
    StartedAt  *time.Time
    EndedAt    *time.Time
}
```

### 新增：workflow_workers 表

```go
type WorkflowWorker struct {
    ID         uint       `gorm:"primaryKey"`
    StageID    uint       `gorm:"index;not null"`       // WorkflowStage.ID
    WorkflowID uint       `gorm:"index;not null"`
    WorkerName string     `gorm:"size:64;not null"`     // YAML 中定义的 name
    Role       string     `gorm:"size:128"`             // display_name
    Status     string     `gorm:"size:20"`              // pending/running/completed/failed
    InputJSON  string     `gorm:"type:text"`            // 完整上下文（debug 用）
    OutputJSON string     `gorm:"type:text"`            // Worker 输出
    TokensUsed int                                       // token 消耗
    DurationMs int                                       // 执行时长(ms)
    StartedAt  *time.Time
    EndedAt    *time.Time
}
```

### 改造：conversations 表

```go
type Conversation struct {
    // 现有字段保留
    ID       uint   `gorm:"primaryKey"`
    UserID   uint   `gorm:"index;not null"`
    Title    string `gorm:"size:200"`
    Messages string `gorm:"type:longtext"`
    ScriptID *uint
    State    int

    // ★ 新增字段
    WorkflowType string `gorm:"size:64;index"` // 区分业务类型
    WorkflowID   *uint                          // 关联 Workflow 记录

    CreatedAt time.Time
    UpdatedAt time.Time
}
```

### 改造：messages 表

```go
type Message struct {
    // 现有字段保留
    ID             uint   `gorm:"primaryKey"`
    ConversationID uint   `gorm:"index;not null"`
    Role           string `gorm:"size:20"`
    Type           string `gorm:"size:30"`
    Content        string `gorm:"type:text"`
    DataJSON       string `gorm:"column:data;type:text"`
    OptionsJSON    string `gorm:"column:options;type:text"`
    Step           int
    Name           string `gorm:"size:200"`

    // ★ 新增字段
    StageID    string `gorm:"size:64;index"` // 来自哪个 Stage
    WorkerName string `gorm:"size:64"`       // 来自哪个 Worker

    CreatedAt time.Time
}
```

### 业务扩展模式

未来新增内容创作场景：

1. 新建 `workflows/{scene}/` 目录，编写 `workflow.yaml` + prompt YAML
2. 新增业务专属 model（如 `model/xhs_post.go`）
3. 在 WorkflowLoader 中自动发现（遍历 workflows/ 子目录）
4. 平台层表（workflows/stages/workers/conversations/messages）零改动

---

## 6. Workflow Engine 核心设计

### Engine 生命周期

```
WorkflowLoader.Load(type)  →  从 YAML 加载 WorkflowDef
        │
        ▼
Engine.Start(wfDef, input) →  构建 SharedContext, 创建 Workflow DB 记录
        │
        ▼
   ┌─── Stage Loop ────────────────────────────┐
   │  for each stage in wfDef.Stages:           │
   │    parallel → 并发 Workers → 汇总           │
   │    serial   → 单 Worker 执行                │
   │    human    → SSE 推送 → 暂停等待用户       │
   └────────────────────────────────────────────┘
        │
        ▼
Engine.Finish()            →  汇总结果, 保存业务数据, 发送 complete
```

### 核心接口

```go
type Engine struct {
    loader    *WorkflowLoader
    llm       LLMClient
    store     WorkflowStore     // workflow/stage/worker 持久化
    wfCtx     *WorkflowContext
    sseWriter SSEWriter
    wfDef     *WorkflowDef
    workflowID uint             // DB 记录 ID
}

// Start 启动新 workflow
func (e *Engine) Start(workflowType string, input WorkflowInput) error

// Resume Human stage 暂停后恢复
func (e *Engine) Resume(workflowID uint, humanInput string) error

// Cancel 取消执行
func (e *Engine) Cancel(workflowID uint) error
```

### Stage 执行器

```go
// Parallel Stage: 并发 Workers + LLM 汇总
func (e *Engine) executeParallelStage(stage StageDef) (*StageOutput, error) {
    var wg sync.WaitGroup
    results := make([]WorkerOutput, len(stage.Workers))

    for i, wd := range stage.Workers {
        wg.Add(1)
        go func(idx int, wd WorkerDef) {
            defer wg.Done()
            input := e.buildWorkerInput(wd)
            results[idx], _ = e.runWorker(wd, input)
        }(i, wd)
    }
    wg.Wait()

    // 汇总（如有 SynthPrompt）
    var summary string
    if stage.SynthPromptDef != nil {
        synthInput := e.buildSynthInput(stage, results)
        summary, _ = e.callLLM(synthInput)
    }

    return &StageOutput{Workers: results, Summary: summary}, nil
}

// Serial Stage: 单 Worker
func (e *Engine) executeSerialStage(stage StageDef) (*StageOutput, error) {
    wd := stage.Workers[0]
    input := e.buildWorkerInput(wd)
    output, err := e.runWorker(wd, input)
    return &StageOutput{
        Workers: []WorkerOutput{output},
        Summary: output.Content,
    }, err
}

// Human Stage: 暂停等待用户
func (e *Engine) executeHumanStage(stage StageDef) error {
    e.sseWriter.SendOutline(e.wfCtx.LastStageOutput())
    e.sseWriter.SendAction(stage.HumanPrompt, stage.Options)
    e.store.SaveCheckpoint(e.workflowID, e.wfCtx, stage.ID)
    return ErrWaitingHuman
}

// Resume 根据用户选择决定回退或继续
func (e *Engine) Resume(workflowID uint, humanInput string) error {
    // 从 checkpoint 恢复上下文
    e.wfCtx, stageID = e.store.LoadCheckpoint(workflowID)

    switch parseHumanChoice(humanInput) {
    case ChoiceConfirm:    // "确认" → 继续执行后续 Stage
        e.wfCtx.HumanInputs[stageID] = humanInput
        return e.runFromStage(nextStageAfter(stageID))
    case ChoiceAdjust:     // "调整大纲" → 将调整意见注入，重跑 create+optimize
        e.wfCtx.HumanInputs[stageID] = humanInput
        return e.runFromStage("create")
    case ChoiceMaterial:   // "更换素材" → 重跑 research(仅素材Worker)+create+optimize
        return e.runFromStage("research")
    case ChoiceReanalyze:  // "重新分析" → 清空所有 StageOutput，从头执行
        e.wfCtx.StageOutputs = map[string]*StageOutput{}
        return e.runFromStage("research")
    }
    return nil
}
```

### Worker 执行与流式输出

```go
func (e *Engine) runWorker(wd WorkerDef, input WorkerInput) (WorkerOutput, error) {
    e.sseWriter.SendWorkerStart(wd.Name, wd.DisplayName)

    var fullContent strings.Builder
    start := time.Now()

    err := e.llm.Stream(input.SystemPrompt, input.UserPrompt, StreamOptions{
        MaxTokens:   wd.MaxTokens,
        Temperature: wd.Temperature,
        OnToken: func(token string) {
            fullContent.WriteString(token)
            e.sseWriter.SendWorkerToken(wd.Name, token)
        },
    })

    duration := time.Since(start)
    e.sseWriter.SendWorkerDone(wd.Name)

    return WorkerOutput{
        Name:     wd.Name,
        Content:  fullContent.String(),
        Duration: duration,
    }, err
}
```

### chat_handler 改造

```go
func (h *ChatHandler) SendMessage(c *gin.Context) {
    // ... SSE setup, auth ...

    session := h.getSession(userID)

    if session.ActiveWorkflowID == 0 {
        // 新建 workflow
        wfType := "viral_script"
        input := WorkflowInput{
            Text:      message,
            SourceURL: extractedURL,
            UserStyle: loadUserStyle(userID),
        }
        engine := workflow.NewEngine(h.loader, h.llm, h.store, sseWriter)
        go engine.Start(wfType, input)
    } else {
        // 恢复暂停的 workflow
        engine := workflow.RestoreEngine(session.ActiveWorkflowID, h.store, h.llm, sseWriter)
        go engine.Resume(session.ActiveWorkflowID, message)
    }
}
```

---

## 7. SSE 协议扩展

### 新增事件类型

| type | 字段 | 说明 |
|------|------|------|
| `stage_start` | `stage_id, stage_name, stage_type` | Stage 开始 |
| `stage_done` | `stage_id` | Stage 完成 |
| `worker_start` | `stage_id, worker_name, worker_display` | Worker 开始 |
| `worker_token` | `worker_name, content` | Worker 流式 token |
| `worker_done` | `worker_name` | Worker 完成 |
| `synth_start` | `stage_id` | 并行汇总开始 |
| `synth_token` | `content` | 汇总流式 token |
| `synth_done` | `stage_id` | 汇总完成 |

### 保留的现有事件（向后兼容）

| type | 说明 |
|------|------|
| `step` | 流程步骤提示（stage_start 时同步发送） |
| `info` | 状态信息 |
| `outline` | 大纲数据（human stage 发送） |
| `action` | 操作按钮选项 |
| `similarity` | 相似度结果 |
| `complete` | 完成，返回 scriptId |
| `error` | 错误信息 |

### 完整 SSE 消息流示例

```
-- Stage 1: 研究（并行） --
data: {"type":"stage_start","stage_id":"research","stage_name":"研究分析","stage_type":"parallel"}
data: {"type":"step","step":1,"name":"研究分析"}
data: {"type":"worker_start","stage_id":"research","worker_name":"viral_decoder","worker_display":"爆款解构师"}
data: {"type":"worker_start","stage_id":"research","worker_name":"style_architect","worker_display":"风格建模师"}
data: {"type":"worker_start","stage_id":"research","worker_name":"material_curator","worker_display":"素材补齐师"}
data: {"type":"worker_token","worker_name":"viral_decoder","content":"## 选题分析..."}
data: {"type":"worker_token","worker_name":"style_architect","content":"## 风格融合..."}
...（三个 worker 的 token 交错到达）...
data: {"type":"worker_done","worker_name":"viral_decoder"}
data: {"type":"worker_done","worker_name":"style_architect"}
data: {"type":"worker_done","worker_name":"material_curator"}
data: {"type":"stage_done","stage_id":"research"}
data: {"type":"synth_start","stage_id":"research"}
data: {"type":"synth_token","content":"...汇总JSON..."}
data: {"type":"synth_done","stage_id":"research"}

-- Stage 2: 创作（串行） --
data: {"type":"stage_start","stage_id":"create","stage_name":"大纲创作","stage_type":"serial"}
data: {"type":"step","step":2,"name":"大纲创作"}
data: {"type":"worker_start","stage_id":"create","worker_name":"creative_agent","worker_display":"创作代理"}
data: {"type":"worker_token","worker_name":"creative_agent","content":"..."}
data: {"type":"worker_done","worker_name":"creative_agent"}
data: {"type":"stage_done","stage_id":"create"}

-- Stage 3: 优化（串行） --
data: {"type":"stage_start","stage_id":"optimize","stage_name":"优化审查","stage_type":"serial"}
data: {"type":"step","step":3,"name":"优化审查"}
data: {"type":"worker_start","stage_id":"optimize","worker_name":"optimization_agent","worker_display":"优化代理"}
data: {"type":"worker_token","worker_name":"optimization_agent","content":"..."}
data: {"type":"worker_done","worker_name":"optimization_agent"}
data: {"type":"stage_done","stage_id":"optimize"}

-- Stage 4: 用户确认（人工） --
data: {"type":"outline","data":{...大纲JSON...}}
data: {"type":"action","options":["确认，开始撰写","调整大纲","更换素材","重新分析"]}
-- 暂停，等待用户响应 --

-- 用户选择"确认"后恢复 --

-- Stage 5: 终稿（串行） --
data: {"type":"stage_start","stage_id":"write","stage_name":"撰写终稿","stage_type":"serial"}
data: {"type":"step","step":5,"name":"撰写终稿"}
data: {"type":"worker_token","worker_name":"draft_writer","content":"你有没有发现..."}
data: {"type":"worker_done","worker_name":"draft_writer"}
data: {"type":"stage_done","stage_id":"write"}

-- Stage 6: 相似度检测（串行） --
data: {"type":"stage_start","stage_id":"similarity","stage_name":"相似度检测","stage_type":"serial"}
data: {"type":"similarity","data":{"vocab":18,"sentence":12,"structure":15,"viewpoint":10,"total":14}}
data: {"type":"stage_done","stage_id":"similarity"}

data: {"type":"complete","scriptId":42}
```

---

## 8. 前端改造要点

### 改动范围

前端改动较小，核心是支持多 Worker 并行展示。

### Dashboard.tsx 状态扩展

```typescript
type WorkerStream = {
  name: string
  displayName: string
  content: string
  status: 'running' | 'done'
}

// 新增到现有 State
type State = {
  // ...现有字段保留...
  currentStage: { id: string; name: string; type: string } | null
  activeWorkers: Map<string, WorkerStream>
}
```

### SSE 事件处理扩展

```typescript
// 在现有 SSE handler switch 中新增
case 'stage_start':
  dispatch({ type: 'STAGE_START', payload: event })
  break
case 'worker_start':
  dispatch({ type: 'WORKER_START', payload: event })
  break
case 'worker_token':
  dispatch({ type: 'WORKER_TOKEN', payload: event })
  break
case 'worker_done':
  dispatch({ type: 'WORKER_DONE', payload: event })
  break
// synth_* 类似处理
```

### 并行 Worker 展示组件

```
┌─────────────────────────────────────────────┐
│  📊 研究分析 (3/3 完成)                       │
│  ┌─────────┬─────────┬─────────┐            │
│  │爆款解构师│风格建模师│素材补齐师│            │
│  │ ✅ 完成  │ ✅ 完成  │ ⏳ 输出中│            │
│  │ [展开▼]  │ [展开▼]  │ [展开▼] │            │
│  └─────────┴─────────┴─────────┘            │
│                                              │
│  📝 汇总分析中...                             │
└─────────────────────────────────────────────┘
```

串行 Stage 保持现有的单流输出样式不变。

### 新增组件

| 组件 | 职责 |
|------|------|
| `WorkerPanel.tsx` | 单个 Worker 的流式输出展示（标题 + 状态 + 内容折叠） |
| `ParallelStageView.tsx` | 并行 Stage 的多 Worker 网格布局 |
| `StageProgress.tsx` | Stage 进度条（当前第几步/共几步） |

---

## 9. 改造影响评估

### 需要改动的现有文件

| 文件 | 改动 |
|------|------|
| `backend/main.go` | 注册 WorkflowLoader，初始化 Engine 依赖 |
| `backend/internal/handler/chat_handler.go` | 核心改造：用 Engine 替代现有状态机逻辑 |
| `backend/internal/service/pipeline.go` | 废弃大部分逻辑，保留 session 管理框架 |
| `backend/internal/service/prompts.go` | 废弃，迁移到 YAML |
| `backend/internal/db/db.go` | AutoMigrate 新增 3 张表 + 2 张表加字段 |
| `backend/internal/model/conversation.go` | 新增 WorkflowType, WorkflowID 字段 |
| `backend/internal/model/message.go` | 新增 StageID, WorkerName 字段 |
| `frontend/src/pages/Dashboard.tsx` | 状态扩展，新增 SSE 事件处理 |
| `frontend/src/components/create/MessageList.tsx` | 新增 Worker/Stage 消息渲染 |
| `frontend/src/lib/sse.ts` | 新增 SSE 事件类型定义 |

### 新增文件

| 文件 | 职责 |
|------|------|
| `backend/internal/workflow/engine.go` | Engine 核心 |
| `backend/internal/workflow/stage.go` | Stage 执行器 |
| `backend/internal/workflow/worker.go` | Worker Agent |
| `backend/internal/workflow/context.go` | 上下文管理 + 变量插值 |
| `backend/internal/workflow/loader.go` | YAML 加载器 |
| `backend/internal/workflow/types.go` | 类型定义 |
| `backend/internal/model/workflow.go` | 3 个新 model |
| `backend/internal/repository/workflow_repo.go` | workflow CRUD |
| `backend/workflows/viral_script/` | 口播稿 YAML 全套 |
| `frontend/src/components/create/WorkerPanel.tsx` | Worker 展示 |
| `frontend/src/components/create/ParallelStageView.tsx` | 并行布局 |
| `frontend/src/components/create/StageProgress.tsx` | 进度展示 |

### 保留不变的文件

| 文件 | 说明 |
|------|------|
| `backend/internal/service/llm_service.go` | LLM 调用层不变，Worker 复用 |
| `backend/internal/service/extractor.go` | URL 提取不变 |
| `backend/internal/model/user.go` | 用户 + 风格表不变 |
| `backend/internal/model/script.go` | 脚本表不变 |
| `frontend/src/contexts/AuthContext.tsx` | 认证不变 |
| `frontend/src/api/chat.ts` | SSE fetch 不变 |
| 所有 auth/user/script 相关 handler 和 repo | 不变 |

---

## 10. 不在本次范围

以下功能明确排除，留作后续迭代：

- 完整记忆系统（L0-L3 分层、向量搜索、周报）
- 多 LLM Provider 切换（当前沿用 Anthropic）
- WebSocket 实时通信（沿用 SSE）
- Redis 消息总线（当前用 Go channel 内存通信，足够单机场景）
- 管理后台 / 数据分析看板
- 多租户隔离 / 计费系统
- Prompt 在线编辑界面（当前直接改 YAML 文件）
