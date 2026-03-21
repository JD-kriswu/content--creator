# 口播稿助手 (content-creator-imm)

AI 驱动的爆款口播稿改写工具。用户提交短视频链接或文案，系统通过5角色并行分析生成原创改写稿，相似度检测 < 30%。

## ⚠️ 开发必读：项目记忆索引

**任何功能开发前，必须先阅读 `.ai_mem/` 目录下的索引文件：**

| 文件 | 内容 |
|------|------|
| `.ai_mem/L0_overview.md` | 项目整体框架、技术栈、目录结构、核心流程 |
| `.ai_mem/L1_modules.md` | 各模块功能、所有API接口、SSE协议、前端store/组件说明 |
| `.ai_mem/L2_details.md` | 代码实现细节、状态机流程、Prompt约定、扩展指引 |

**每次功能开发完成后，必须同步更新对应的 `.ai_mem/` 文件**，保持索引与代码一致。

## 架构

Monorepo 前后端分离：

```
content_creator_imm/
├── backend/        Go 1.22 + Gin API 服务（默认端口 3004）
├── frontend/       Vue 3 + Vite + Element Plus SPA（开发端口 5173）
├── build.sh        一键构建前后端
├── manage.sh       服务管理脚本（启停、用户管理）
└── docs/           架构文档与实施计划
```

**后端**：REST API + SSE 流式响应，JWT 认证，MySQL + GORM，配置文件 `backend/config.json`

**前端**：Vue 3 SPA，Pinia 状态管理，Vue Router（含路由守卫），Axios，原生 `fetch + ReadableStream` 处理 SSE 流

## 核心数据模型

| 模型 | 说明 |
|------|------|
| `User` | 用户账户 |
| `UserStyle` | 用户风格档案（语言风格、情绪基调、口头禅等） |
| `Conversation` | 会话记录：`id, user_id, title, messages(longtext JSON), script_id, state(0=进行中/1=完成), created_at` |
| `Script` | 生成的稿件：`id, user_id, title, source_url, content_path, similarity_score, viral_score` |

**Conversation.messages** 存储 `[]StoredMsg` JSON，每条 StoredMsg 包含：`role(user/assistant), type, content, data, options, step, name`

## 会话生命周期

```
StateIdle → StateAnalyzing → StateAwaiting → StateWriting → StateComplete
```

- **StateIdle**：等待用户输入，收到消息后创建 `Conversation` 记录（`EnsureConversation`）
- **StateAnalyzing**：5角色分析，流式输出，每步追加 `StoredMsg`
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
# 1. 构建
cd frontend && npm run build

# 2. 验证构建产物可访问（通过 nginx）
curl -s -o /dev/null -w "HTTP %{http_code}\n" http://localhost/creator/

# 3. 验证 API 联通
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
| `frontend/src/stores/chat.ts` | 聊天状态、SSE 事件处理、`restoreMessages` |
| `frontend/src/api/chat.ts` | SSE fetch（注意 baseURL 为 `/creator/api`） |
| `frontend/src/api/conversations.ts` | 会话列表/详情 API |
| `frontend/src/views/Home.vue` | 主页面：侧边栏（会话/稿件 tab）+ 聊天区 |
| `frontend/src/components/ChatPanel.vue` | 聊天渲染、输入框、`restoreConversation` |
| `frontend/src/components/ConversationList.vue` | 历史会话列表 |
| `frontend/src/components/ScriptList.vue` | 历史稿件列表 |
