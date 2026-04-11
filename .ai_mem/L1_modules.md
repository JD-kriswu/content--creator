# L1 — 模块功能 / 接口说明

> ← [L0_overview.md](L0_overview.md) | 细节 → [L2_details.md](L2_details.md)

---

## 后端模块

### 1. config (`backend/config/config.go`)

加载优先级：**环境变量 > config.json > 默认值**

| 字段 | 环境变量 | 说明 |
|------|----------|------|
| `port` | `PORT` | 监听端口，默认 3004 |
| `jwt_secret` | `JWT_SECRET` | JWT签名密钥 |
| `anthropic_api_key` | `ANTHROPIC_API_KEY` | LLM API Key |
| `llm_base_url` | `LLM_BASE_URL` | LLM API地址，默认 api.anthropic.com |
| `db_*` | `DB_HOST/DB_PASSWORD` | MySQL连接参数 |
| `cors_origins` | `CORS_ORIGINS` | 允许的跨域来源 |
| `base_path` | `BASE_PATH` | URL前缀，如 `/creator` |
| `storage_path` | `STORAGE_PATH` | 稿件本地存储目录 |
| `feishu_enabled` | `FEISHU_ENABLED` | 是否启用飞书集成 |
| `feishu_ws_reconnect_max` | `FEISHU_WS_RECONNECT_MAX` | WS 重连最大次数 |
| `feishu_ws_heartbeat_sec` | - | WS 心跳间隔（秒） |
| `feishu_card_throttle_ms` | - | Card 更新节流（毫秒） |

---

### 2. middleware (`backend/middleware/auth.go`)

- `Auth()` gin中间件：从 `Authorization: Bearer <token>` 提取JWT，验证后将 `userID(uint)` 写入 gin.Context

---

### 3. model (`backend/internal/model/`)

#### User / UserStyle
```
User: id, username(unique), email(unique), password_hash, role(admin/user), active
UserStyle: id, user_id(unique), language_style, emotion_tone, opening_style, closing_style, catchphrases
```

#### Conversation
```
id, user_id(index), title(200), messages(longtext JSON←legacy), script_id(*uint), state(0=进行中/1=完成), created_at, updated_at
```

#### Message（核心消息实体，实时落库）
```
id, conversation_id(index), role(user/assistant), type, content(text), data(text←JSON), options(text←JSON数组), step(int), name(200), created_at
```

type枚举：`text | step | info | outline | action | similarity | complete | error`

#### Workflow（workflow执行实例）
```
id, user_id(index), type, status(pending/running/paused/completed/failed), input_json, context_json, output_json, conv_id(*uint←关联Conversation), error, paused_at, created_at
```
Workflow.ConvID 用于关联 Conversation，确保页面刷新后能恢复会话状态。

#### Script
```
id, user_id(index), title(256), source_url(1024), platform(32), content_path(1024), similarity_score(float64), viral_score(float64), tags(text←JSON), created_at
```

#### FeishuBot（飞书机器人实例）
```
id, user_id(index), app_id(unique), app_secret, tenant_key, bot_name, ws_connected, last_heartbeat, created_at
```
关联 Web 用户，存储飞书 App 凭证和 WebSocket 连接状态。

#### FeishuUser（飞书用户身份）
```
id, feishu_id(unique), open_id(unique), union_id, user_id(index), bind_status(independent/merged), created_at
```
飞书用户身份映射，可关联 Web 用户（合并后 bind_status=merged）。

#### FeishuConversation（飞书会话映射）
```
id, bot_id(index), conv_id(index), feishu_chat_id(unique), created_at
```
飞书聊天窗口 → 内部 Conversation 记录的映射。

---

### 4. repository (`backend/internal/repository/`)

| 文件 | 函数 |
|------|------|
| `user_repo.go` | `GetUserByEmail`, `GetUserByID`, `CreateUser`, `GetStyleByUserID`, `UpsertStyle` |
| `script_repo.go` | `CreateScript`, `ListScripts(userID,page,limit)→([]Script,total,err)`, `GetScript(id,userID)` |
| `conversation_repo.go` | `CreateConversation`, `UpdateConversation`, `UpdateConversationTitle(id,title)`, `UpdateConversationMeta(id,map)`, `ListConversations(userID,limit)`, `GetConversation(id,userID)` |
| `message_repo.go` | `CreateMessage`, `ListMessagesByConvID(convID)→[]Message` |
| `feishu_bot_repo.go` | `CreateFeishuBot`, `GetFeishuBotByAppID`, `GetFeishuBotsByUserID`, `GetConnectedFeishuBots`, `UpdateFeishuBotWSStatus`, `DeleteFeishuBot` |
| `feishu_user_repo.go` | `CreateFeishuUser`, `GetFeishuUserByOpenID`, `GetOrCreateFeishuUserByOpenID`, `UpdateFeishuUserBind` |
| `feishu_conv_repo.go` | `CreateFeishuConv`, `GetFeishuConvByChatID`, `GetOrCreateFeishuConv(botID,chatID,userID)→(FeishuConv,convID,err)`, `DeleteFeishuConvsByBotID` |

---

### 5. service (`backend/internal/service/`)

#### llm_service.go
- `CallClaude(system, prompt string, maxTokens ...int) (string, error)` — 非流式调用
- `StreamClaude(system, prompt string, cb StreamCallback) (string, error)` — 流式调用，每个 token 触发回调
- HTTP client 超时 300s，scanner buffer 64KB
- 当前模型：`glm-5`，接口协议：Anthropic `/v1/messages`

#### extractor.go
- `ExtractURL(url) (string, error)` — 抓取URL并提取正文文本（最大1MB，截断至5000字）
- `IsURL(s) bool` — 判断是否为 http/https URL

#### prompts.go
- `BuildAnalysisPrompt(text, style)` — 5角色分析Prompt，内嵌 `---OUTLINE_START---..---OUTLINE_END---` 标记
- `BuildFinalDraftPrompt(text, outline, note)` — 终稿写作Prompt，内嵌 `---QUALITY_CHECK_START---` 标记
- `BuildSimilarityCheckPrompt(original, new)` — 相似度评分Prompt，输出纯JSON

#### pipeline.go（会话状态机核心）
- `SessionState`: `StateIdle / StateAnalyzing / StateAwaiting / StateWriting / StateComplete`
- `ChatSession`: 内存中的会话对象（含锁），每个用户一个
- `GetOrCreateSession(userID)`, `ResetSession(userID)` — 会话管理
- `EnsureConversation(sess, title)` — 创建DB会话记录（幂等）
- `FlushConversation(sess, state, scriptID)` — 更新会话状态/scriptID（不再写messages JSON）
- `PersistMsg(convID, StoredMsg)` — 单条消息立即写入Message表
- `SaveScript(userID, sess, simScore, viralScore)` — 保存稿件文件+DB记录，调用FlushConversation(state=1)
- `ParseOutlineFromAnalysis(text)` — 从流式输出中提取 OUTLINE JSON
- `stripQualityCheck(text)` — 移除QUALITY_CHECK块

#### feishu_api.go（飞书 API 封装）
- `FeishuAPI` 结构体：AppID, AppSecret, Token, TokenExp
- `NewFeishuAPI(appID, appSecret)` — 创建 API 客户端
- `GetToken()` — 获取 tenant_access_token（自动缓存，60秒过期缓冲）
- `CreateCard(chatID, cardJSON)` — 创建 Card 消息，返回 messageID
- `UpdateCard(messageID, cardJSON)` — 更新已发送的 Card

#### feishu_session.go（飞书会话状态）
- `FeishuState`: `FeishuIdle / FeishuAnalyzing / FeishuAwaiting / FeishuWriting`
- `FeishuSession`: ChatID, BotID, UserID, ConvID, WorkflowID, State, lock
- `FeishuSessionMgr`（单例）: GetOrCreate, Get, SetState, SetWorkflowID, SetConvID, IsBusy, Clear

---

### 6. workflow (`backend/internal/workflow/`)

#### types.go
- `StageType`: `parallel | serial | human`
- `StageDef`: `id, display_name, type, workers, skip_if`（skip_if 支持条件跳过）
- `WorkflowDef`: `type, display_name, meta, stages[]`
- `WorkflowContext`: `SharedContext, StageOutputs, HumanInputs`

#### engine.go
- `Engine.Start(workflowType, input)` — 加载定义、创建DB记录、开始执行
- `Engine.Resume(workflowID, humanInput)` — 恢复暂停的工作流
- `runStages(startIdx)` — 迭代执行阶段，检查 skip_if 条件
- `evaluateSkipCondition(expr)` — 解析条件表达式如 `{{stage.X.field}} == false`
- `resolveResumeStage(humanInput)` — 用户选择映射到恢复阶段

#### context.go
- `buildVarsMap(ctx)` — 构建变量映射，支持 JSON 字段提取（如 `{{stage.X.worker.Y.output.field}}`）
- `interpolate(tpl, vars)` — 变量插值

#### loader.go
- 从 `workflows/<type>/workflow.yaml` 加载定义
- 从 `prompts/<name>.yaml` 加载 worker 定义

---

### 7. handler (`backend/internal/handler/`)

#### auth_handler.go
- `POST /api/auth/register` — 注册，返回 `{token, user}`
- `POST /api/auth/login` — 登录，返回 `{token, user}`

#### chat_handler.go（核心）
- `GET /api/chat/session` — 返回当前session ID和状态
- `POST /api/chat/reset` — 刷新旧会话→重置session→创建新"新会话"记录，返回 `{conv_id}`
- `POST /api/chat/message` — SSE流式响应，根据 session 状态路由到 handleIdle/handleAwaiting
  - `handleIdle`: 创建会话→更新标题→执行workflow→等待确认
  - `handleAwaiting`: 处理 1/2/3/4 用户选择
  - `writeFinalDraft`: 撰写终稿→相似度检测→保存
  - `addMsg(sess, msg)` helper: 同时写内存和Message表
- `GET /api/scripts` — 稿件列表
- `GET /api/scripts/:id` — 稿件详情+内容（读文件）
- `GET /api/conversations` — 会话列表（最近50条）
- `GET /api/conversations/:id` — 会话详情，从Message表组装消息JSON

#### 重入恢复机制

**Session 恢复（SendMessage 开始时）**：
1. 检查数据库是否有 `status=paused` 的 Workflow 记录
2. 若有，恢复 `sess.ActiveWorkflowID` 和 `sess.ConvID`（从 Workflow.ConvID 或 InputJSON）
3. 设置 session 状态为 `StateAwaiting`

**Workflow-Conversation 关联**：
- Workflow 创建时设置 `ConvID` 字段，确保刷新页面后能恢复
- Resume 分支有 fallback：从 Workflow 记录获取 ConvID

**超时降级**：
- `StateAnalyzing` → 降级到 `StateIdle`
- `StateWriting` → 降级到 `StateAwaiting`

#### feishu_handler.go（飞书绑定 API）
- `GET /api/feishu/bots` — 获取用户绑定的飞书机器人列表
- `DELETE /api/feishu/bots/:id` — 解绑飞书机器人（同时删除关联会话）

---

### 8. feishu (`backend/internal/feishu/`)

飞书集成模块，实现 WebSocket 消息接收和 Card 流式输出。

#### types.go
- `WSEvent`: WebSocket 推送事件（type, app_id, tenant_key, event）
- `MessageEvent`: 消息接收事件（sender.open_id, message.chat_id, message.content）
- `CardActionEvent`: Card 按钮点击事件
- `Card`, `CardHeader`, `CardElement`, `CardAction`: Card 消息结构
- `WSStatus`: 连接状态常量（connected/disconnected/reconnecting）

#### ws_client.go
- `WSConnection`: 单个 WebSocket 连接（心跳、重连、消息接收）
- `Connect()`, `Disconnect()`, `heartbeatLoop()`, `receiveLoop()`
- 重连策略：指数退避（5s × attempt），最大重连次数可配置

#### ws_pool.go
- `WSConnectionPool`（单例）: 连接池管理
- `GetWSPool()`, `Connect(appID, appSecret, handler)`, `Disconnect(appID)`, `Status(appID)`
- 服务启动时自动连接所有已绑定的 Bot

#### card_sse.go
- `FeishuSSEWriter`: 实现 `workflow.SSEWriter` 接口
- 将 SSE 事件转换为飞书 Card 更新（throttled）
- `Init()`: 创建初始 Card
- `SendStageStart/SendWorkerToken/SendComplete/SendError` 等：更新 Card 内容
- Card 颜色：blue（进行中）、green（完成）、red（错误）

#### router.go
- `Router`: 消息路由，处理飞书 WebSocket 事件
- `HandleEvent`: 事件分发（im.message.receive_v1 → handleMessage）
- `handleMessage`: 查找 Bot/User/Conv → 路由到 workflow
- `handleIdle`: 启动新 workflow
- `handleAwaiting`: 恢复暂停的 workflow
- `handleCardAction`: 处理 Card 按钮点击

---

## Workflow 定义 (`workflows/viral_script/`)

### 文件结构
```
workflows/viral_script/
├── _charter.yaml      # 系统宪章（注入到所有 worker 的 system prompt 开头）
├── workflow.yaml      # 阶段定义
└── prompts/           # worker prompts
    ├── viral_decoder.yaml
    ├── material_check.yaml
    ├── material_curator.yaml
    ├── creative_agent.yaml
    ├── optimization_agent.yaml
    ├── draft_writer.yaml
    └── similarity_checker.yaml
```

### 系统宪章 (`_charter.yaml`)
- **作用**：注入到每个 worker 的 system prompt 开头，定义不可违反的规则
- **编辑**：通过 `/api/prompts` API 或前端 `/creator/prompts` 页面在线编辑
- **内容**：相似度约束、内容质量要求、输出规范、禁止事项

### workflow.yaml 阶段结构（8阶段）

| 阶段 | ID | 类型 | Workers | 说明 |
|------|-----|------|---------|------|
| 研究分析 | `research` | parallel | `viral_decoder` | 仅爆款解构，无 synth |
| 素材需求判断 | `material_check` | serial | `material_check` | 输出 `need_material: true/false` |
| 素材补齐 | `material_curator` | serial | `material_curator` | **skip_if**：`need_material == false` 时跳过 |
| 大纲创作 | `create` | serial | `creative_agent` | 基于 viral_decoder + 素材生成大纲 |
| 优化审查 | `optimize` | serial | `optimization_agent` | 审查大纲，辩论决策 |
| 确认大纲 | `confirm_outline` | human | - | 用户选择：1-确认 2-调整 3-更换素材 4-重新 |
| 撰写终稿 | `write` | serial | `draft_writer` | 按大纲撰写口播稿 |
| 相似度检测 | `similarity` | serial | `similarity_checker` | 输出相似度评分 JSON |

### 用户选择映射（resume 逻辑）

| 输入 | 恢复阶段 |
|------|---------|
| 1 / 确认 | `write`（下一阶段） |
| 2 / 调整 | `create`（重新创作大纲） |
| 3 / 更换素材 | `material_check`（重新判断素材需求） |
| 4 / 重新 | `research`（完全重启） |

---

## 前端模块

### api/
| 文件 | 功能 |
|------|------|
| `request.ts` | Axios实例，baseURL=`/creator/api`，自动附加JWT Header |
| `auth.ts` | `login(email,pwd)`, `register(username,email,pwd)` |
| `chat.ts` | `getSession()`, `resetSession()→{conv_id}`, `sendMessage(text)→fetch Response（SSE）` |
| `scripts.ts` | `getScripts()`, `getScript(id)` |
| `user.ts` | `getProfile()`, `updateStyle(style)` |
| `conversations.ts` | `listConversations()`, `getConversation(id)→{conversation, messages:string}` |
| `feishu.ts` | `getFeishuBots()→{bots}`, `unbindFeishuBot(botId)` |

> ⚠️ SSE 使用原生 `fetch` 而非 Axios（需要手动附加 token），baseURL 硬编码为 `/creator/api/chat/message`

### stores/
#### chat.ts（核心状态）
```
messages: ChatMessage[]       渲染消息列表
sending: boolean              是否正在发送/流式中
justCompleted: number         complete事件计数器（watch触发刷新）
messagesUpdated: number       每次send()结束后自增（watch触发刷新会话列表）
currentConvId: number         当前活跃会话DB ID（reset时从后端获取）
lastSentText: string          最后发送的文本（用于重试）
```
关键方法：`send(text)`, `retry()`, `reset()`, `restoreMessages(jsonStr)`

#### user.ts
token + user 存 localStorage，`login/register/logout/isLoggedIn`

### views/
- `Login.vue` — 登录/注册表单，粉紫渐变主题
- `Home.vue` — 主布局：header + 左侧sidebar(会话Tab/稿件Tab) + 右侧ChatPanel
  - `newChat()`: reset() → 创建新会话 → 刷新列表
  - `loadConversation(conv)`: sending中且是当前会话→保留内存；否则从DB加载
  - watch `messagesUpdated` → 自动刷新会话列表

### components/
- `ChatPanel.vue` — 消息渲染（outline卡片/action按钮/similarity卡片/普通气泡）+ 输入框；retryable错误消息显示重试按钮
- `ConversationList.vue` — 历史会话列表，高亮当前会话，进行中/已完成badge
- `ScriptList.vue` — 历史稿件列表

---

## SSE 消息协议

所有消息格式：`data: <JSON>\n\n`

| type | 字段 | 前端处理 |
|------|------|---------|
| `stage_start` | `stage_id, stage_name, stage_type` | 显示阶段开始 |
| `stage_done` | `stage_id` | 阶段完成 |
| `worker_start` | `stage_id, worker_name, worker_display` | worker 开始 |
| `worker_token` | `worker_name, content` | 流式 token |
| `worker_done` | `worker_name` | worker 完成 |
| `synth_start` | `stage_id` | synth 开始 |
| `synth_token` | `content` | synth 流式 token |
| `synth_done` | `stage_id` | synth 完成 |
| `info` | `content` | 信息提示（如"跳过阶段"） |
| `outline` | `data(OutlineData)` | outline卡片消息 |
| `action` | `options([]string)` | action按钮组消息 |
| `similarity` | `data(SimilarityData)` | similarity卡片消息 |
| `final_draft` | `content` | 终稿内容（用于右侧展示） |
| `complete` | `scriptId` | justCompleted++ → 触发刷新 |
| `error` | `message` | 可重试error消息 |
