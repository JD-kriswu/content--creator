# 口播稿助手 (content-creator-imm) 产品需求文档 (PRD)

**文档版本：** v1.0
**创建日期：** 2026-03-22
**创建人：** 呆瓜（基于项目记忆整理）

---

## 一、产品概述

### 1.1 产品定位
**口播稿助手**是一款AI驱动的爆款口播稿改写工具。用户提交短视频链接或文案，系统通过5角色并行分析生成原创改写稿，确保相似度 < 30%。

### 1.2 目标用户
- 短视频创作者（抖音、快手、视频号等）
- 内容运营人员
- MCN机构内容团队

### 1.3 核心价值
- **原创性保障**：相似度 < 30%，避免平台查重
- **风格定制**：根据用户风格档案个性化改写
- **爆款逻辑**：5角色分析提取爆款要素
- **高效产出**：流式输出，实时预览

---

## 二、核心功能

### 2.1 用户管理
- 用户注册/登录
- 风格档案管理（语言风格、情绪基调、口头禅等）

### 2.2 口播稿改写流程

```
用户输入链接/文案
    ↓
Step1: 提取文本（URL → HTML解析）
    ↓
Step2: 读取用户风格档案
    ↓
Step3: 5角色并行分析（流式输出）
       ① 爆款解构师  ② 风格建模师  ③ 素材补齐师
       ④ 创作代理   ⑤ 优化代理   → 辩论决策
    ↓
Step5: 大纲生成，等待用户确认
    ↓（用户输入"1"确认）
Step6: 撰写终稿（流式输出）
    ↓
Step8: 相似度检测（非流式，最多256 tokens）
    ↓
Step9: 保存稿件（本地 .md 文件 + DB记录）
    ↓
返回 scriptId，完成
```

### 2.3 会话管理
- 会话历史保存
- 会话恢复（支持中断后继续）
- 历史稿件列表
- 历史会话列表

### 2.4 数据持久化
- 用户账户数据
- 用户风格档案
- 会话记录（实时落库）
- 稿件存储（本地 .md 文件 + DB记录）

---

## 三、技术架构

### 3.1 架构设计
**前后端分离架构**

```
前端（Vue 3 SPA）          后端（Go + Gin）
     ↓                         ↓
Pinia 状态管理            REST API + SSE 流式响应
     ↓                         ↓
Element Plus UI           MySQL + GORM
     ↓                         ↓
nginx 反代                AI API (Anthropic-compatible)
```

### 3.2 技术栈

| 层 | 技术 |
|----|------|
| 后端语言 | Go 1.22 |
| Web框架 | Gin |
| ORM | GORM + MySQL |
| AI接口 | Anthropic-compatible API（`/v1/messages`），模型 `glm-5` |
| 实时通信 | SSE（Server-Sent Events），`text/event-stream` |
| 前端框架 | Vue 3 + Vite |
| UI库 | Element Plus |
| 状态管理 | Pinia |
| 路由 | Vue Router（WebHistory，base: `/creator/`）|
| HTTP客户端 | Axios（baseURL: `/creator/api`）+ 原生 fetch（SSE）|
| 部署 | nginx 反代，`/creator/api/` → `:3004`，`/creator/` → dist/ |

### 3.3 数据库设计

| 表 | 说明 |
|----|------|
| `users` | 用户账号 |
| `user_styles` | 用户风格档案（每用户一条） |
| `conversations` | 会话记录（标题、状态） |
| `messages` | 消息实体（每条消息独立记录，实时落库） |
| `scripts` | 生成的稿件（含相似度分、本地文件路径） |

### 3.4 会话状态机

```
StateIdle → StateAnalyzing → StateAwaiting → StateWriting → StateComplete
```

- **StateIdle**：等待用户输入
- **StateAnalyzing**：5角色分析中
- **StateAwaiting**：等待用户确认大纲
- **StateWriting**：撰写终稿中
- **StateComplete**：完成

**重入恢复机制**：
- 若 session 卡在 Analyzing/Writing 超过 3 分钟，下次收到消息自动降级到前一个稳定状态

---

## 四、接口设计

### 4.1 认证接口

```
POST /api/auth/register          注册（username, email, password）
POST /api/auth/login             登录 → { token, user }
```

### 4.2 用户接口

```
GET  /api/user/profile           获取风格档案（需认证）
PUT  /api/user/style             更新风格档案（需认证）
```

### 4.3 聊天接口

```
GET  /api/chat/session           获取当前会话状态（需认证）
POST /api/chat/reset             重置会话（需认证）
POST /api/chat/message           发送消息，SSE 流式响应（需认证）
```

### 4.4 稿件接口

```
GET  /api/scripts                稿件列表（需认证）
GET  /api/scripts/:id            稿件详情 + 内容（需认证）
```

### 4.5 会话接口

```
GET  /api/conversations          会话列表（需认证，返回最近 50 条）
GET  /api/conversations/:id      会话详情，含 messages JSON（需认证）
```

### 4.6 SSE 消息协议

所有消息格式：`data: <JSON>\n\n`

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

---

## 五、部署架构

### 5.1 生产部署

**nginx 配置（路径前缀 `/creator`）**：

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

### 5.2 服务管理

```bash
./build.sh          # 构建前后端
./manage.sh start   # 启动后端服务
./manage.sh stop    # 停止后端服务
./manage.sh restart # 重启后端服务
./manage.sh status  # 查看运行状态
./manage.sh logs    # 实时日志
```

### 5.3 配置

配置文件：`backend/config.json`

| 字段 | 说明 | 默认值 |
|------|------|--------|
| `port` | 后端监听端口 | `3004` |
| `jwt_secret` | JWT 签名密钥 | **请修改！** |
| `anthropic_api_key` | AI API Key | 必填 |
| `llm_base_url` | LLM API 地址 | `https://api.anthropic.com` |
| `storage_type` | 脚本存储方式（`local`/`oss`） | `local` |
| `storage_path` | 本地脚本存储路径 | `data/scripts` |
| `base_path` | URL 前缀（如 `/creator`） | `` |

---

## 六、开发规范

### 6.1 Go 代码改动

```bash
cd backend && go build -o ../content-creator-imm .
./manage.sh restart
```

### 6.2 前端改动

```bash
cd frontend && npm run build
# 验证构建产物
curl -s -o /dev/null -w "HTTP %{http_code}\n" http://localhost/creator/
```

### 6.3 项目记忆更新

**每次功能开发后，必须同步更新 `.ai_mem/` 文件**：

| 文件 | 内容 |
|------|------|
| `.ai_mem/L0_overview.md` | 项目整体框架、技术栈、目录结构、核心流程 |
| `.ai_mem/L1_modules.md` | 各模块功能、所有API接口、SSE协议、前端store/组件说明 |
| `.ai_mem/L2_details.md` | 代码实现细节、状态机流程、Prompt约定、扩展指引 |

---

## 七、未来规划（待确认）

### 7.1 功能增强
- [ ] 支持更多平台（小红书、B站等）
- [ ] 批量改写功能
- [ ] 稿件版本管理
- [ ] 团队协作功能

### 7.2 技术优化
- [ ] 缓存优化（风格档案、常用素材）
- [ ] 异步任务队列（长时间改写任务）
- [ ] 相似度算法优化
- [ ] 多模型支持

---

## 八、附录

### 8.1 关键文件索引

| 文件 | 职责 |
|------|------|
| `backend/main.go` | 路由注册、服务启动 |
| `backend/internal/service/pipeline.go` | Session 状态机、StoredMsg、会话持久化 |
| `backend/internal/handler/chat_handler.go` | SSE 消息处理、消息追踪、会话 handler |
| `backend/internal/service/llm_service.go` | Claude API 调用（流式/非流式） |
| `backend/internal/service/prompts.go` | 所有 AI prompt 构建 |
| `frontend/src/stores/chat.ts` | 聊天状态、SSE 事件处理 |
| `frontend/src/api/chat.ts` | SSE fetch |
| `frontend/src/views/Home.vue` | 主页面布局 |
| `frontend/src/components/ChatPanel.vue` | 聊天渲染、输入框 |

### 8.2 测试账号

- **测试账号：** `test2@test.com / Test1234`

---

**文档结束**

*本 PRD 基于 `.ai_mem/` 项目记忆整理，如有更新请同步修改。*