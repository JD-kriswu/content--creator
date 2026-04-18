# L0 — 项目整体框架

> 索引层级：L0（鸟瞰）→ [L1_modules.md](L1_modules.md)（模块）→ [L2_details.md](L2_details.md)（代码细节）

## 项目定位

**口播稿助手 (content-creator-imm)**：AI 驱动的爆款口播稿改写工具。
用户提交短视频链接或文案，系统通过 5 角色并行分析生成原创改写稿，相似度 < 30%。

---

## 仓库结构

```
content_creator_imm/
├── backend/              Go 1.22 + Gin，REST API + SSE 流式响应
│   ├── main.go           路由注册、服务入口
│   ├── config/           配置加载（config.json + 环境变量）
│   ├── middleware/        JWT 鉴权中间件
│   └── internal/
│       ├── db/           GORM MySQL 初始化 + AutoMigrate
│       ├── model/        数据模型（5张表）
│       ├── repository/   数据库 CRUD
│       ├── service/      业务逻辑（LLM调用、会话状态机、Prompt构建）
│       └── handler/      HTTP 处理器
├── frontend/             React 18 + Vite + Tailwind v4 + Radix UI SPA
│   └── src/
│       ├── api/          原生 fetch 封装（SSE 用 raw fetch，JSON 用 lib/request）
│       ├── contexts/     React Context（AuthContext）
│       ├── pages/        页面组件（Auth、Home、Dashboard、History、Result）
│       ├── components/   UI 组件（Sidebar、Layout、create/MessageList 等）
│       ├── components/ui/ shadcn/ui 风格 Radix UI 封装（button、card、dialog 等）
│       └── lib/          工具库（sse.ts、request.ts、utils.ts）
├── .ai_mem/              项目记忆索引（本目录）
├── CLAUDE.md             开发规范与架构说明
├── build.sh              一键构建前后端
└── manage.sh             服务启停 + 用户管理
```

---

## 技术栈

| 层 | 技术 |
|----|------|
| 后端语言 | Go 1.22 |
| Web框架 | Gin |
| ORM | GORM + MySQL |
| AI接口 | Anthropic-compatible API（`/v1/messages`），模型 `glm-5` |
| 实时通信 | SSE（Server-Sent Events），`text/event-stream` |
| 前端框架 | React 18 + Vite |
| UI库 | Radix UI + shadcn/ui 风格组件 |
| 样式 | Tailwind CSS v4 |
| 状态管理 | React Context（AuthContext）+ useReducer（Dashboard） |
| 路由 | React Router v7 |
| HTTP客户端 | `lib/request.ts`（JSON API）+ 原生 fetch（SSE）|
| 部署 | nginx 反代，`/creator/api/` → `:3004`，`/creator/` → dist/ |

---

## 核心流程（用户视角）

```
用户输入链接/文案
    ↓
Step1: 提取文本（URL → HTML解析）
    ↓
Step2: 研究分析（viral_decoder 爆款解构师，流式输出）
    ↓
Step3: 素材需求判断（material_check 输出 need_material: true/false）
    ↓（如 need_material=true）
Step4: 素材补齐（material_curator，否则跳过此步骤）
    ↓
Step5: 大纲创作（creative_agent 基于分析生成大纲）
    ↓
Step6: 优化审查（optimization_agent 审查大纲）
    ↓
Step7: 大纲确认（等待用户输入 1/2/3/4）
    ↓（用户输入"1"确认）
Step8: 撰写终稿（draft_writer 流式输出）
    ↓
Step9: 相似度检测（similarity_checker 输出评分）
    ↓
Step10: 保存稿件（本地 .md 文件 + DB记录）
    ↓
返回 scriptId，完成
```

**条件执行机制**：
- `material_curator` 阶段使用 `skip_if` 条件跳过
- 当 `material_check` 输出 `need_material=false` 时，自动跳过素材补齐

---

## 数据库表（5张）

| 表 | 说明 |
|----|------|
| `users` | 用户账号 |
| `user_styles` | 用户风格档案（每用户一条） |
| `conversations` | 会话记录（标题、状态） |
| `messages` | 消息实体（每条消息独立记录，实时落库） |
| `scripts` | 生成的稿件（含相似度分、本地文件路径） |

---

## 部署环境（当前）

- 后端进程：`:3004`，binary: `content-creator-imm`
- 前端静态：`frontend/dist/`，由 nginx 托管
- nginx 路径前缀：`/creator/`
- 数据库：MySQL，库名 `content_creator`
- 稿件存储：`backend/data/scripts/`
- 测试账号：`test2@test.com / Test1234`
