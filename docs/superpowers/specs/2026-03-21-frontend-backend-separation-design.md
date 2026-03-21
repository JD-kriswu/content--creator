# 前后端分离架构设计

**日期**: 2026-03-21
**项目**: content_creator_imm
**状态**: 待实施

---

## 1. 背景

当前项目是 Go + Gin 服务，通过 `//go:embed public` 将前端 HTML/JS/CSS 打包进二进制。前后端耦合导致：
- 前端无法独立开发（无热更新）
- 前端改动需重编译 Go
- 无法独立部署/扩展

目标：将项目改造为 monorepo 前后端分离架构。

---

## 2. 目录结构

```
content_creator_imm/
├── backend/              # Go API 服务
│   ├── main.go
│   ├── go.mod
│   ├── go.sum
│   ├── config/
│   │   └── config.go
│   ├── internal/
│   │   ├── db/
│   │   ├── handler/
│   │   ├── model/
│   │   ├── repository/
│   │   └── service/
│   ├── middleware/
│   ├── data/             # 本地脚本存储（运行时生成）
│   ├── config.json       # 本地配置（gitignored）
│   └── config.example.json
├── frontend/             # Vue 3 + Vite + Element Plus
│   ├── src/
│   │   ├── main.ts
│   │   ├── App.vue
│   │   ├── router/
│   │   │   └── index.ts
│   │   ├── stores/
│   │   │   ├── user.ts
│   │   │   └── chat.ts
│   │   ├── api/
│   │   │   ├── auth.ts
│   │   │   ├── chat.ts
│   │   │   ├── scripts.ts
│   │   │   └── user.ts
│   │   ├── views/
│   │   │   ├── Login.vue
│   │   │   └── Home.vue
│   │   └── components/
│   │       ├── ChatPanel.vue
│   │       └── ScriptList.vue
│   ├── index.html
│   ├── vite.config.ts
│   ├── tsconfig.json
│   └── package.json
├── build.sh              # 构建脚本（前端 + 后端）
├── manage.sh             # 服务管理（启停、用户管理）
├── CLAUDE.md             # 项目文档 + 常用命令
└── docs/
```

---

## 3. 后端变更

### 3.1 删除 embed 逻辑

从 `main.go` 中移除：
- `//go:embed public` 声明
- `publicFS` 变量
- `serveHTML` 函数
- `r.StaticFS(...)` 静态文件服务
- `r.GET(...index.html)` / `r.GET(...login)` 页面路由

### 3.2 精简后的 main.go 结构

```
config.Load()
db.Init()
gin.Default()
CORS 中间件
API 路由注册
r.Run()
```

### 3.3 CORS 配置

开发期允许 `http://localhost:5173`，生产期通过配置项控制（或允许同源）。

### 3.4 API 路由（不变）

```
POST /api/auth/register
POST /api/auth/login
GET  /api/user/profile      (auth)
PUT  /api/user/style        (auth)
GET  /api/chat/session      (auth)
POST /api/chat/reset        (auth)
POST /api/chat/message      (auth, SSE streaming)
GET  /api/scripts           (auth)
GET  /api/scripts/:id       (auth)
```

### 3.5 go.mod

module 名保持 `content-creator-imm`，目录迁移到 `backend/`。

---

## 4. 前端架构

### 4.1 技术栈

| 工具 | 版本 |
|------|------|
| Vue | 3.x |
| Vite | 5.x |
| TypeScript | 5.x |
| Element Plus | 2.x |
| Pinia | 2.x |
| Vue Router | 4.x |
| Axios | 1.x |

### 4.2 路由

| 路径 | 组件 | 说明 |
|------|------|------|
| `/login` | Login.vue | 登录/注册，未登录自动跳转 |
| `/` | Home.vue | 主界面，需认证 |

路由守卫：检查 Pinia user store 中的 token，未登录跳转 `/login`。

### 4.3 状态管理（Pinia）

**useUserStore**
- state: `token`, `user` (id, username, email, style)
- actions: `login(email, password)`, `register(...)`, `logout()`, `fetchProfile()`
- 持久化：localStorage

**useChatStore**
- state: `messages`, `sending`, `session`
- actions: `sendMessage(text)` —— 使用原生 `fetch + ReadableStream` 处理 SSE 流式响应（POST 请求，不能用 EventSource）, `resetSession()`, `loadSession()`

### 4.4 API 层

统一 axios 实例（`src/api/request.ts`）：
- baseURL: `/api`
- 请求拦截器：自动注入 `Authorization: Bearer <token>`
- 响应拦截器：401 时清除 store 并跳转登录

SSE 流式消息（`/api/chat/message`）使用原生 `fetch` + `ReadableStream`，不用 axios。

### 4.5 Vite 开发代理

```ts
// vite.config.ts
server: {
  proxy: {
    '/api': {
      target: 'http://localhost:3004',
      changeOrigin: true
    }
  }
}
```

---

## 5. 构建与部署

### 5.1 build.sh

```bash
#!/bin/bash
set -e

echo "=== 构建前端 ==="
cd frontend
npm install
npm run build
cd ..

echo "=== 构建后端 ==="
cd backend
go build -o ../content-creator-imm .
cd ..

echo "=== 构建完成 ==="
```

构建产物：
- `frontend/dist/` —— 静态文件，由 nginx 托管
- `content-creator-imm` —— Go 二进制

### 5.2 manage.sh

```
用法: ./manage.sh <command>

Commands:
  start           启动后端服务
  stop            停止后端服务
  restart         重启后端服务
  status          查看服务状态
  add-user        <username> <email> <password>  添加用户
  list-users      列出所有用户
  logs            查看服务日志（tail -f）
```

实现：使用 PID 文件（`server.pid`）管理进程。

### 5.3 生产 nginx 配置示意

```nginx
server {
    listen 80;
    root /path/to/frontend/dist;
    index index.html;

    # 前端 SPA fallback
    location / {
        try_files $uri $uri/ /index.html;
    }

    # API 反代
    location /api {
        proxy_pass http://127.0.0.1:3004;
    }
}
```

---

## 6. CLAUDE.md 内容要点

- 项目架构概述
- 目录结构
- 开发环境启动命令
- 构建命令
- 服务管理命令
- 用户管理命令
- API 列表
- 配置说明

---

## 7. 迁移步骤

1. 创建 `backend/` 目录，迁移所有 Go 文件
2. 精简 `backend/main.go`（删除 embed 逻辑）
3. 创建 `frontend/` Vue 3 + Vite 项目
4. 实现 Pinia stores、API 层、路由守卫
5. 重写 Login.vue（对应原 login.html）
6. 重写 Home.vue + ChatPanel.vue + ScriptList.vue（对应原 index.html + app.js）
7. 编写 `build.sh` 和 `manage.sh`
8. 编写 `CLAUDE.md`
9. 删除旧的 `public/` 目录
10. 更新 `.gitignore`

---

## 8. 不在本次范围内

- 用户权限/角色系统
- 管理后台
- 容器化（Docker）
- CI/CD 流水线
