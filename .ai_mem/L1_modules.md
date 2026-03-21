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

#### Script
```
id, user_id(index), title(256), source_url(1024), platform(32), content_path(1024), similarity_score(float64), viral_score(float64), tags(text←JSON), created_at
```

---

### 4. repository (`backend/internal/repository/`)

| 文件 | 函数 |
|------|------|
| `user_repo.go` | `GetUserByEmail`, `GetUserByID`, `CreateUser`, `GetStyleByUserID`, `UpsertStyle` |
| `script_repo.go` | `CreateScript`, `ListScripts(userID,page,limit)→([]Script,total,err)`, `GetScript(id,userID)` |
| `conversation_repo.go` | `CreateConversation`, `UpdateConversation`, `UpdateConversationTitle(id,title)`, `UpdateConversationMeta(id,map)`, `ListConversations(userID,limit)`, `GetConversation(id,userID)` |
| `message_repo.go` | `CreateMessage`, `ListMessagesByConvID(convID)→[]Message` |

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

---

### 6. handler (`backend/internal/handler/`)

#### auth_handler.go
- `POST /api/auth/register` — 注册，返回 `{token, user}`
- `POST /api/auth/login` — 登录，返回 `{token, user}`

#### chat_handler.go（核心）
- `GET /api/chat/session` — 返回当前session ID和状态
- `POST /api/chat/reset` — 刷新旧会话→重置session→创建新"新会话"记录，返回 `{conv_id}`
- `POST /api/chat/message` — SSE流式响应，根据 session 状态路由到 handleIdle/handleAwaiting
  - `handleIdle`: 创建会话→更新标题→执行5角色分析→等待确认
  - `handleAwaiting`: 处理 1/2/3/4 用户选择
  - `writeFinalDraft`: 撰写终稿→相似度检测→保存
  - `addMsg(sess, msg)` helper: 同时写内存和Message表
- `GET /api/scripts` — 稿件列表
- `GET /api/scripts/:id` — 稿件详情+内容（读文件）
- `GET /api/conversations` — 会话列表（最近50条）
- `GET /api/conversations/:id` — 会话详情，从Message表组装消息JSON

#### 重入恢复机制
若 session 停在中间状态超过 **3分钟**：
- `StateAnalyzing` → 降级到 `StateIdle`
- `StateWriting` → 降级到 `StateAwaiting`

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
| `token` | `content` | 追加到当前流式消息 |
| `step` | `step(int), name` | 新增步骤badge消息 |
| `info` | `content` | 新增info badge消息 |
| `outline` | `data(OutlineData)` | 新增outline卡片消息 |
| `action` | `options([]string)` | 新增action按钮组消息 |
| `similarity` | `data(SimilarityData)` | 新增similarity卡片消息 |
| `complete` | `scriptId` | justCompleted++ → 触发刷新 |
| `error` | `message` | 新增可重试error消息 |
