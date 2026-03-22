# 前端重构设计规范：Vue 3 → React

**日期：** 2026-03-22
**状态：** 已批准
**范围：** `frontend/` 目录全量替换

---

## 1. 背景与目标

现有前端使用 Vue 3 + Element Plus，视觉风格为紫色主题，品牌名称"口播稿助手"。`assert/` 目录下存有 Figma 设计导出的 React + Tailwind 代码，包含全新品牌（轻写Claw）、蓝紫渐变设计系统、全新 Landing 页和重构后的 Dashboard 布局。

**目标：** 将 `frontend/` 原地替换为 React 实现，贴合 Figma 设计，后端 API 不变。

---

## 2. 决策记录

| 问题 | 决策 | 理由 |
|------|------|------|
| 框架 | 迁移到 React 18 | 直接沿用 Figma React 代码，减少翻译成本 |
| 原地替换 vs 新目录 | 原地替换 `frontend/` | build.sh 和 nginx 配置零改动 |
| 认证方式 | 保留邮箱+密码，换新视觉 | 后端不支持手机号/短信，不扩展后端 |
| Landing 页 | 新增 `/` 营销首页 | 提升产品形象，未登录用户有落地页 |
| Dashboard 布局 | 初始态保留侧边栏，创作态侧边栏收起 | 忠实设计稿，给创作区最大空间 |
| UI 库 | Tailwind CSS 4 + shadcn/ui | 直接复用 assert/src/app/components/ui/ |

---

## 3. 技术栈

```
React 18 + Vite + React Router 7
Tailwind CSS 4 (@tailwindcss/vite 插件)
shadcn/ui 组件（Radix UI 基础）
lucide-react（图标）
sonner（Toast 通知）
clsx + tailwind-merge + class-variance-authority
```

**不引入（YAGNI）：** react-dnd、react-hook-form、recharts、embla-carousel、motion、canvas-confetti、@mui/material

---

## 4. 目录结构

```
frontend/
├── index.html
├── vite.config.ts
├── package.json
├── postcss.config.mjs
└── src/
    ├── main.tsx
    ├── App.tsx                      # RouterProvider + Toaster
    ├── router.tsx                   # 路由定义（含 ProtectedRoute）
    ├── contexts/
    │   └── AuthContext.tsx          # token/user/login/logout/401处理
    ├── api/
    │   ├── auth.ts                  # login / register
    │   ├── chat.ts                  # sendMessage(SSE) / reset / session
    │   ├── conversations.ts         # list / detail
    │   └── scripts.ts               # list / detail（GET /api/scripts, /api/scripts/:id）
    ├── components/
    │   ├── ui/                      # 从 assert/ 复制的 shadcn 组件
    │   │   ├── button.tsx
    │   │   ├── card.tsx
    │   │   ├── input.tsx
    │   │   ├── label.tsx
    │   │   ├── tabs.tsx
    │   │   ├── separator.tsx
    │   │   └── ...（按需复制）
    │   ├── Layout.tsx               # 顶部 Header + <Outlet />
    │   ├── Sidebar.tsx              # 新建对话 + 集成入口 + 历史列表 + 稿件列表
    │   └── create/
    │       ├── MessageList.tsx      # SSE 消息渲染
    │       ├── ChatInput.tsx        # 底部输入框（Enter 发送）
    │       ├── LoadingState.tsx     # 生成中动画
    │       ├── OutlineEditor.tsx    # 大纲预览区（右侧）
    │       └── ScriptEditor.tsx     # 稿件展示 + 复制 + 导出（只读，不提交后端）
    └── pages/
        ├── Home.tsx                 # Landing 营销首页
        ├── Auth.tsx                 # 登录 / 注册
        └── Dashboard.tsx           # 主创作区（受保护）
```

---

## 5. 页面设计

### 5.1 Landing（`/`）

从 `assert/src/app/pages/Home.tsx` 直接沿用，内容：
- Hero：标题 + 三痛点解决清单 + CTA 按钮（→ `/auth`）+ 右侧数据卡片
- 第二屏：你能获得什么（4张卡片）
- 第三屏：为什么选择轻写Claw（3张痛点卡片）
- 第四屏：独特AI创作能力（3张功能卡片）
- 第五屏：数据统计（4张数据卡片）
- 第六屏：CTA 行动号召

### 5.2 Auth（`/auth`）

从 `assert/src/app/pages/Auth.tsx` 改造：
- 保留整体视觉（蓝紫渐变、Card 布局、Tabs）
- **表单字段改为邮箱 + 密码**（删除手机号/验证码/微信登录）
- 调用 `POST /api/auth/login` 和 `POST /api/auth/register`
- 登录成功后 token 存入 AuthContext + localStorage，跳转 `/dashboard`

### 5.3 Dashboard（`/dashboard`，受保护）

全屏布局，不使用 Layout 组件。

**初始态（`stage === 'idle'`）：**
```
┌─────────────────────────────────────┐
│  Sidebar(w-64)  │  居中输入区        │
│  + 新建对话      │  Hi，今天想创作... │
│  配置入口(开发中) │  [粘贴参考文案...]  │
│  历史记录列表    │  字数统计          │
└─────────────────────────────────────┘
```

**创作态（`stage !== 'idle'`）：**
```
┌──────────────────────────────────────────────┐
│  ChatPanel(w-2/5)     │  PreviewPanel(w-3/5)  │
│  消息列表(可滚动)      │  LoadingState         │
│                       │  或 OutlineEditor      │
│  [底部输入框]          │  或 ScriptEditor       │
└──────────────────────────────────────────────┘
```

**Stage 状态机（对应后端 pipeline 状态）：**

| Stage | 触发条件 | 右侧面板 |
|-------|----------|----------|
| `idle` | 初始/重置 | 居中输入框 |
| `analyzing` | 发送第一条消息 | LoadingState |
| `awaiting` | 收到 `outline` SSE 事件 | OutlineEditor |
| `writing` | 用户点击确认按钮（发送 1/2/3/4） | LoadingState |
| `complete` | 收到 `complete` SSE 事件 | ScriptEditor |

**Outline 确认交互细节：**

- `outline` SSE 事件：在右侧显示 `OutlineEditor`（大纲预览），同时左侧 `MessageList` 中显示 `action` 事件的选项按钮（1/2/3/4）
- 用户点击左侧 action 按钮 → 调用 `sendMessage('1')`（或对应数字）→ 进入 `writing` 状态
- `OutlineEditor` 为只读展示，不支持编辑大纲内容，确认动作通过左侧 action 按钮完成

**ScriptEditor 数据来源：**

- 稿件内容来源于 SSE `token` 事件的累积流式文本（在内存中逐字追加）
- 收到 `complete` 事件后，将累积文本传入 `ScriptEditor` 展示
- `ScriptEditor` 只支持客户端复制和导出（.txt），不提交后端（无 PUT /api/scripts/:id 接口）
- 稿件已在后端持久化，`scriptId` 可用于将来跳转查看

---

## 6. 核心逻辑

### 6.1 认证（AuthContext）

```typescript
interface AuthContext {
  user: { id: number; username: string; email: string } | null;
  token: string | null;
  login(email: string, password: string): Promise<void>;
  logout(): void;
}
```

- token 存入 `localStorage`，初始化时读取
- 所有 API 请求带 `Authorization: Bearer <token>` header
- ProtectedRoute：token 为空则重定向到 `/auth`
- **401 自动登出：** API 请求封装层收到 HTTP 401 响应时，调用 `logout()` 并跳转到 `/auth`（与 Vue 版行为一致）

### 6.2 SSE 处理（Dashboard）

使用原生 `fetch + ReadableStream`，所有 API 请求的 baseURL 为 `/creator/api`：

```typescript
const response = await fetch('/creator/api/chat/message', {
  method: 'POST',
  headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' },
  body: JSON.stringify({ message }),
});
// 逐行解析 data: <JSON>\n\n 格式
```

**SSE 事件处理：**

| type | 行为 |
|------|------|
| `token` | 追加 `content` 到当前流式消息（内存累积，用于最终 ScriptEditor） |
| `step` | 在消息列表中插入步骤提示 |
| `info` | 在消息列表中插入信息消息 |
| `outline` | 设置 `stage = 'awaiting'`，`outlineData = data`，右侧显示 OutlineEditor |
| `action` | 在消息列表中显示操作按钮（用户点击后调用 sendMessage） |
| `similarity` | 在消息列表中显示相似度结果 |
| `complete` | 设置 `stage = 'complete'`，`scriptId = data.scriptId`，将累积流式文本传入 ScriptEditor |
| `error` | 在消息列表中显示错误，重置 stage 为 `idle` |

**SSE 完成后刷新：** 每次 SSE 流结束（stream 关闭）后，重新调用 `GET /api/conversations` 刷新侧边栏会话列表（新对话首次出现）。

### 6.3 历史会话加载（Sidebar）

- 初始化时调用 `GET /api/conversations`（最近 50 条）
- 点击历史会话：调用 `GET /api/conversations/:id`，用返回的 `messages` JSON 恢复消息列表，**stage 设为 `complete`（若 `conversation.state === 1`）或 `idle`（若 `state === 0`）**，不做消息类型推断
- 历史会话恢复后为只读查看模式（不能继续创作），用户需点击"新建对话"开始新会话

**稿件列表（Sidebar）：**
- Sidebar 中包含"会话"和"稿件"两个 Tab（与 Vue 版一致）
- 稿件 Tab：调用 `GET /api/scripts` 获取列表，点击稿件调用 `GET /api/scripts/:id` 获取内容，在右侧 PreviewPanel 展示（使用 ScriptEditor 组件，只读）

### 6.4 新建对话

调用 `POST /api/chat/reset`，清空本地消息状态，重置 stage 为 `idle`，刷新会话列表。

---

## 7. 样式系统

直接沿用 `assert/src/styles/theme.css`（CSS 变量定义）和 `assert/src/styles/index.css`。

**主色调：**
- 品牌渐变：`from-blue-600 to-purple-600`（`#2563eb` → `#7c3aed`）
- 背景：`bg-gradient-to-br from-blue-50 via-white to-purple-50`
- 侧边栏：`bg-gray-50`

---

## 8. 构建与部署

`build.sh` 中构建命令不变（`cd frontend && npm run build`），产物仍输出到 `frontend/dist/`，nginx 配置无需修改。

**React Router basename：** 使用 `createBrowserRouter(routes, { basename: '/creator' })`，与 nginx `/creator/` 前缀对应。

**Vite 开发代理：**
```typescript
// vite.config.ts
proxy: {
  '/creator/api': {
    target: 'http://localhost:3004',
    rewrite: (path) => path.replace(/^\/creator\/api/, '/api'),
  }
}
```

开发时访问 `http://localhost:5173/creator/`，代理将 `/creator/api/*` 转发到 `http://localhost:3004/api/*`。

---

## 9. 不在本次范围内

- 后端任何改动
- 手机号/短信/微信登录
- 稿件详情独立页面（`/result/:id`）
- 历史记录独立页面（`/history`），历史会话通过 Dashboard 侧边栏访问
- 暗色模式
- 移动端适配（Dashboard 仅桌面端）
- 大纲编辑（OutlineEditor 只读，确认通过 action 按钮完成）
