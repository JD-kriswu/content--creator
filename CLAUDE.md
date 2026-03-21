# 口播稿助手 (content-creator-imm)

AI 驱动的爆款口播稿改写工具。用户提交短视频链接或文案，系统通过5角色并行分析生成原创改写稿，相似度检测 < 30%。

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

## 生产部署（nginx 示例）

```nginx
server {
    listen 80;
    root /path/to/content_creator_imm/frontend/dist;
    index index.html;

    # 前端 SPA fallback
    location / {
        try_files $uri $uri/ /index.html;
    }

    # API 反代到后端
    location /api {
        proxy_pass http://127.0.0.1:3004;
        proxy_http_version 1.1;
        proxy_set_header Connection "";
        proxy_set_header X-Accel-Buffering no;  # 禁用缓冲，保证 SSE 实时推送
    }
}
```

启动流程：
```bash
./build.sh          # 构建前后端
./manage.sh start   # 启动后端服务
# nginx reload 使前端 dist 生效
```
