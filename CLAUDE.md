# 口播稿助手 (content-creator-imm)

AI 驱动的爆款口播稿改写工具。用户提交短视频链接或文案，系统通过 Multi-Agent 并行分析生成原创改写稿，相似度检测 < 30%。

---

## ✅ 开发完成必做清单

**每次开发任务完成后，在 commit 之前必须执行以下检查：**

### 1. 更新 CLAUDE.md（本文件）

需要同步更新的内容：

| 变更类型 | 需更新的章节 |
|---------|------------|
| 新增 API 接口 | `## API 列表` |
| 新增 SSE 事件类型 | `## API 列表 → SSE 消息格式` |
| 新增/修改数据模型 | `## 核心数据模型` |
| 状态机变更 | `## 会话生命周期` |
| 新增关键文件 | `## 关键文件索引` |
| 架构层面的变化 | `## 架构` |

### 2. 更新 `.ai_mem/` 项目记忆索引

三层索引文件职责不同，按实际变更选择更新：

| 文件 | 何时必须更新 |
|------|------------|
| `.ai_mem/L0_overview.md` | 目录结构变化、技术栈变更、核心流程增删步骤、数据库表增减 |
| `.ai_mem/L1_modules.md` | 新增/删除模块或文件、API 接口变化、SSE 事件变化、前端组件增删 |
| `.ai_mem/L2_details.md` | 状态机流程变化、Prompt 约定变化、关键函数签名变化、扩展指引更新 |

**更新原则：**
- 描述"是什么"，不记录"怎么实现的"（实现细节看代码）
- 删除已不存在的内容，不留过时描述
- 新增内容与对应代码保持一致，可引用具体文件路径

### 3. 验证清单

**⚠️ 重要：每次完成新的开发任务，必须执行以下全部验证步骤：**

#### 基础验证

后端变更：
```bash
cd backend && go build .          # 必须编译通过
cd backend && go test ./...       # 相关测试必须通过
```

前端变更：
```bash
cd frontend && npx tsc --noEmit   # 必须无 type error
cd frontend && npm run build      # 必须构建成功
```

#### E2E 集成测试（必须执行）

每次完成开发任务后，必须执行 Mock SSE E2E 测试验证完整流程：

```bash
# 1. 确保后端服务已启动
./manage.sh status || ./manage.sh start

# 2. 登录获取 token
TOKEN=$(curl -s http://localhost:3004/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@test.com","password":"password"}' | jq -r '.token')

# 3. 执行 Mock SSE 测试
curl -s http://localhost:3004/api/chat/message \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"message":"test","mock":true}'

# 4. 验证响应包含完整的 SSE 流程（stage_start → worker_token → complete）
```

**注意事项**：
- 如果测试用户不存在，先通过 `/api/auth/register` 创建
- Mock SSE 不调用真实 LLM，快速返回模拟流程
- 必须验证响应格式正确（`data: <JSON>\n\n` 格式）

---

## 测试

### 后端单元测试

```bash
cd backend && go test ./... -v
```

测试文件位置：
- `backend/internal/workflow/*_test.go` — Workflow 引擎测试
- 包含 loader、context、engine 等模块的单元测试

### 前端单元测试

```bash
cd frontend && npm test
# 或监听模式
cd frontend && npm run test:watch
```

测试文件位置：
- `frontend/src/contexts/__tests__/AuthContext.test.tsx` — 认证 Context 测试
- `frontend/src/lib/__tests__/sse.test.ts` — SSE 解析测试

测试框架：Vitest + Testing Library

### E2E 集成测试（Mock SSE）

后端提供 Mock SSE 功能，用于测试完整 SSE 流程（无需调用真实 LLM）：

```bash
# 1. 先登录获取 token
TOKEN=$(curl -s http://localhost:3004/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@test.com","password":"password"}' | jq -r '.token')

# 2. 调用 Mock SSE 端点
curl -s http://localhost:3004/api/chat/message \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"message":"test","mock":true}'
```

Mock SSE 会返回完整的模拟流程：
1. 分析爆款元素
2. 生成大纲
3. 撰写终稿
4. 相似度检测
5. 完成

**触发条件**：请求 body 中传入 `"mock": true`

---

## ⚠️ 开发前必读：项目记忆索引

**任何功能开发前，必须先阅读 `.ai_mem/` 目录下的索引文件：**

| 文件 | 内容 |
|------|------|
| `.ai_mem/L0_overview.md` | 项目整体框架、技术栈、目录结构、核心流程 |
| `.ai_mem/L1_modules.md` | 各模块功能、所有API接口、SSE协议、前端组件说明 |
| `.ai_mem/L2_details.md` | 代码实现细节、状态机流程、Prompt约定、扩展指引 |

---

## 架构

Monorepo 前后端分离：

```
content_creator_imm/
├── backend/        Go 1.22 + Gin API 服务（默认端口 3004）
├── frontend/       React 18 + Vite + Tailwind v4 SPA（开发端口 5173）
├── build.sh        一键构建前后端
├── manage.sh       服务管理脚本（启停、用户管理）
└── docs/           架构文档与实施计划
```

**后端**：REST API + SSE 流式响应，JWT 认证，MySQL + GORM，配置文件 `backend/config.json`

**前端**：React 18 + TypeScript SPA，React Context（AuthContext）+ useReducer 状态管理，React Router v7，Radix UI + shadcn/ui 风格组件，Tailwind CSS v4，原生 `fetch + ReadableStream` 处理 SSE 流

## 核心数据模型

| 模型 | 说明 |
|------|------|
| `User` | 用户账户 |
| `UserStyle` | 用户风格档案（语言风格、情绪基调、口头禅等；multi-agent 重构后新增 style_doc/style_vector/is_initialized 等字段） |
| `Conversation` | 会话记录：`id, user_id, title, messages(longtext JSON), script_id, state(0=进行中/1=完成), created_at` |
| `Script` | 生成的稿件：`id, user_id, title, source_url, content_path, similarity_score, viral_score` |

**Conversation.messages** 存储 `[]StoredMsg` JSON，每条 StoredMsg 包含：`role(user/assistant), type, content, data, options, step, name`

## 会话生命周期

```
StateIdle → StateAnalyzing → StateAwaiting → StateWriting → StateComplete
```

- **StateIdle**：等待用户输入，收到消息后创建 `Conversation` 记录（`EnsureConversation`）
- **StateAnalyzing**：Agent 并行分析，流式输出，每步追加 `StoredMsg`
- **StateAwaiting**：等待用户确认大纲（输入 1/2/3/4），`FlushConversation` 写库
- **StateWriting**：撰写终稿，流式输出
- **StateComplete**：保存 `Script`，`FlushConversation(state=1)` 标记完成，关联 `script_id`

**重入恢复**：若 session 卡在 Analyzing/Writing 超过 3 分钟，下次收到消息自动降级到前一个稳定状态。

**新建会话**：前端调用 `POST /api/chat/reset`，后端先 `FlushConversation(state=0)` 保存当前进度，再清空内存 session。

## 开发环境

### 启动后端

```bash
cd backend && go run .
# 热重载（需先安装 air: go install github.com/air-verse/air@latest）
cd backend && air
```

### 启动前端（热更新，自动代理 /api → localhost:3004）

```bash
cd frontend && npm run dev
```

访问 http://localhost:5173

## 构建生产版本

```bash
./build.sh
```

产物：
- `frontend/dist/` — 静态文件，由 nginx 托管
- `content-creator-imm` — Go 二进制，运行时工作目录为 `backend/`

## 服务管理

```bash
./manage.sh start          # 启动后端服务（后台运行，日志写入 server.log）
./manage.sh stop           # 停止后端服务
./manage.sh restart        # 重启后端服务
./manage.sh status         # 查看运行状态
./manage.sh logs           # 实时日志（tail -f server.log）
```

## 用户管理

```bash
# 添加用户（需服务已启动）
./manage.sh add-user <username> <email> <password>

# 列出所有用户（需本机 mysql 命令行工具）
./manage.sh list-users
```

## 配置

配置文件：`backend/config.json`（参考 `backend/config.example.json`）

| 字段 | 说明 | 默认值 |
|------|------|--------|
| `port` | 后端监听端口 | `3004` |
| `jwt_secret` | JWT 签名密钥 | **请修改！** |
| `db_host` | MySQL 主机 | `127.0.0.1` |
| `db_port` | MySQL 端口 | `3306` |
| `db_user` | MySQL 用户 | `root` |
| `db_password` | MySQL 密码 | `` |
| `db_name` | 数据库名 | `content_creator` |
| `anthropic_api_key` | Anthropic API Key | 必填 |
| `llm_base_url` | LLM API 地址 | `https://api.anthropic.com` |
| `cors_origins` | 允许跨域来源（逗号分隔） | `http://localhost:5173` |
| `storage_type` | 脚本存储方式（`local`/`oss`） | `local` |
| `storage_path` | 本地脚本存储路径 | `data/scripts` |
| `base_path` | URL 前缀（如 `/creator`） | `` |

所有字段支持对应的环境变量覆盖（如 `PORT`、`JWT_SECRET`、`ANTHROPIC_API_KEY`、`CORS_ORIGINS` 等）。

## API 列表

```
POST /api/auth/register          注册（username, email, password）
POST /api/auth/login             登录 → { token, user }

GET  /api/user/profile           获取风格档案（需认证）
PUT  /api/user/style             更新风格档案（需认证）

GET  /api/chat/session           获取当前会话状态（需认证）
POST /api/chat/reset             重置会话（需认证）
POST /api/chat/message           发送消息，SSE 流式响应（需认证）

GET  /api/scripts                稿件列表（需认证）
GET  /api/scripts/:id            稿件详情 + 内容（需认证）

GET  /api/conversations          会话列表（需认证，返回最近 50 条）
GET  /api/conversations/:id      会话详情，含 messages JSON（需认证）

GET  /api/feishu/bots            飞书机器人列表（需认证）
DELETE /api/feishu/bots/:id      解绑飞书机器人（需认证）
```

### SSE 消息格式（POST /api/chat/message）

响应 `Content-Type: text/event-stream`，每条消息格式：`data: <JSON>\n\n`

| type | 字段 | 说明 |
|------|------|------|
| `token` | `content` | 流式文本 token（逐字追加） |
| `step` | `step`, `name` | 流程步骤提示 |
| `info` | `content` | 状态信息（如"已提取 N 字"） |
| `outline` | `data` | 大纲数据（等待用户确认） |
| `action` | `options` | 操作按钮选项列表 |
| `similarity` | `data` | 相似度检测结果 |
| `complete` | `scriptId` | 完成，返回已保存稿件 ID |
| `error` | `message` | 错误信息 |

## 生产部署（nginx 实际配置，路径前缀 `/creator`）

```nginx
# API 反代（必须在静态资源 location 之前）
location /creator/api/ {
    proxy_pass http://127.0.0.1:3004/api/;
    proxy_http_version 1.1;
    proxy_set_header Connection "";
    proxy_buffering off;
    proxy_cache off;
    proxy_set_header X-Accel-Buffering no;   # 保证 SSE 实时推送
    proxy_read_timeout 300s;
}

# 前端 SPA（alias + fallback）
location /creator/ {
    alias /data/code/content_creator_imm/frontend/dist/;
    index index.html;
    try_files $uri $uri/ /creator/index.html;
}
```

> **注意**：`proxy_buffering off` 和 `X-Accel-Buffering no` 缺一不可，否则 SSE 流会被 nginx 缓存后批量推送。

启动流程：
```bash
./build.sh          # 构建前后端
./manage.sh start   # 启动后端服务
# nginx reload 使前端 dist 生效
```

## 开发规范

**Go 代码改动后必须重新编译：**
```bash
cd backend && go build -o ../content-creator-imm .
./manage.sh restart
```

**每次修改前端后必须验证：**
```bash
# 1. 类型检查
cd frontend && npx tsc --noEmit

# 2. 构建
cd frontend && npm run build

# 3. 验证构建产物可访问（通过 nginx）
curl -s -o /dev/null -w "HTTP %{http_code}\n" http://localhost/creator/

# 4. 验证 API 联通
curl -s http://localhost/creator/api/auth/login -X POST \
  -H "Content-Type: application/json" \
  -d '{"email":"test2@test.com","password":"Test1234"}' | python3 -c "import sys,json; d=json.load(sys.stdin); print('OK' if 'token' in d else 'FAIL')"
```

## 关键文件索引

| 文件 | 职责 |
|------|------|
| `backend/main.go` | 路由注册、服务启动 |
| `backend/internal/service/pipeline.go` | Session 状态机、StoredMsg、会话持久化 |
| `backend/internal/handler/chat_handler.go` | SSE 消息处理、消息追踪、会话 handler |
| `backend/internal/service/llm_service.go` | Claude API 调用（流式/非流式），HTTP client 超时 300s |
| `backend/internal/service/prompts.go` | 所有 AI prompt 构建 |
| `backend/internal/model/conversation.go` | Conversation 数据模型 |
| `backend/internal/repository/conversation_repo.go` | 会话 CRUD |
| `frontend/src/contexts/AuthContext.tsx` | React Context 认证状态，`useAuth()` hook |
| `frontend/src/api/chat.ts` | SSE fetch（baseURL 为 `/creator/api`） |
| `frontend/src/api/conversations.ts` | 会话列表/详情 API |
| `frontend/src/pages/Dashboard.tsx` | 主创作页：useReducer 状态机 + SSE 事件处理 |
| `frontend/src/components/create/MessageList.tsx` | 聊天消息渲染（所有 ChatMsg 类型） |
| `frontend/src/lib/sse.ts` | SSEEvent union type + parseSSELine 解析函数 |
| `frontend/src/components/Sidebar.tsx` | 侧边栏（历史会话列表） |

### 飞书集成

| 文件 | 职责 |
|------|------|
| `backend/internal/feishu/` | 飞书模块（WebSocket 连接池、Card SSE、消息路由） |
| `backend/internal/feishu/types.go` | 飞书事件类型定义（WSEvent, Card 等） |
| `backend/internal/feishu/ws_pool.go` | WebSocket 连接池管理 |
| `backend/internal/feishu/card_sse.go` | FeishuSSEWriter（SSE → Card 更新） |
| `backend/internal/feishu/router.go` | 消息路由（飞书事件 → workflow） |
| `backend/internal/service/feishu_api.go` | 飞书 API 封装（Token, Card 创建/更新） |
| `backend/internal/service/feishu_session.go` | 飞书会话状态管理 |
| `backend/internal/model/feishu_bot.go` | FeishuBot 数据模型 |
| `backend/internal/model/feishu_user.go` | FeishuUser 数据模型 |
| `backend/internal/model/feishu_conversation.go` | FeishuConversation 数据模型 |
| `backend/internal/handler/feishu_handler.go` | 飞书绑定 API handler |
| `frontend/src/pages/FeishuBind.tsx` | 飞书绑定页面 |
| `frontend/src/api/feishu.ts` | 前端飞书 API 封装 |
