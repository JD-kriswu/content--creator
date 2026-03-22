# React 前端迁移实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 `frontend/` 从 Vue 3 + Element Plus 原地替换为 React 18 + Tailwind CSS + shadcn/ui，实现轻写Claw 新设计（Landing/Auth/Dashboard/Result/History + 暗色模式 + 移动端）。

**Architecture:** 所有文件在 `frontend/` 原地替换，产物仍输出至 `frontend/dist/`，nginx 和 build.sh 零改动。React Router 使用 `basename="/creator"` 对应生产路径前缀，Vite dev 代理 `/creator/api/*` → `http://localhost:3004/api/*`。

**Tech Stack:** React 18 · Vite 6 · React Router 7 · Tailwind CSS 4 · shadcn/ui (Radix UI) · next-themes · lucide-react · sonner · Vitest + @testing-library/react

**参考设计稿:** `assert/src/` — 可直接复制 `components/ui/*`、`styles/*`、各页面组件（需适配真实 API）

**规范文档:** `docs/superpowers/specs/2026-03-22-react-migration-design.md`

---

## 文件清单

| 文件 | 操作 | 说明 |
|------|------|------|
| `frontend/` (所有 Vue 文件) | 删除 | 清空旧代码 |
| `frontend/package.json` | 创建 | React 依赖 |
| `frontend/vite.config.ts` | 创建 | Vite + Tailwind + 代理 |
| `frontend/tsconfig.json` | 创建 | TypeScript 配置 |
| `frontend/index.html` | 创建 | HTML 入口 |
| `frontend/postcss.config.mjs` | 创建 | PostCSS（Tailwind 需要） |
| `frontend/vitest.config.ts` | 创建 | 测试配置 |
| `frontend/src/test/setup.ts` | 创建 | Jest-dom matchers |
| `frontend/src/styles/theme.css` | 复制 | 从 assert/src/styles/theme.css |
| `frontend/src/styles/index.css` | 复制 | 从 assert/src/styles/index.css |
| `frontend/src/lib/utils.ts` | 创建 | cn() 工具函数 |
| `frontend/src/lib/sse.ts` | 创建 | SSE 事件解析（纯函数，可测试） |
| `frontend/src/lib/request.ts` | 创建 | fetch 封装（baseURL + 401 处理） |
| `frontend/src/components/ui/*` | 复制 | 从 assert/src/app/components/ui/ |
| `frontend/src/contexts/AuthContext.tsx` | 创建 | Auth 状态 + login/logout |
| `frontend/src/api/auth.ts` | 创建 | login / register |
| `frontend/src/api/chat.ts` | 创建 | sendMessage(SSE) / reset / session |
| `frontend/src/api/conversations.ts` | 创建 | list / detail |
| `frontend/src/api/scripts.ts` | 创建 | list / detail |
| `frontend/src/router.tsx` | 创建 | 路由 + ProtectedRoute |
| `frontend/src/main.tsx` | 创建 | React 挂载入口 |
| `frontend/src/App.tsx` | 创建 | RouterProvider + ThemeProvider + Toaster |
| `frontend/src/components/Layout.tsx` | 创建 | Header + 移动底部导航 + 暗色切换 |
| `frontend/src/components/Sidebar.tsx` | 创建 | 会话/稿件 Tab + 历史列表 |
| `frontend/src/components/create/MessageList.tsx` | 创建 | SSE 消息渲染 |
| `frontend/src/components/create/ChatInput.tsx` | 创建 | 底部输入框 |
| `frontend/src/components/create/LoadingState.tsx` | 创建 | 生成中动画 |
| `frontend/src/components/create/OutlineEditor.tsx` | 创建 | 大纲可编辑区 |
| `frontend/src/components/create/ScriptEditor.tsx` | 创建 | 稿件展示 + 复制 + 导出 |
| `frontend/src/pages/Home.tsx` | 创建 | Landing 营销页 |
| `frontend/src/pages/Auth.tsx` | 创建 | 登录/注册（邮箱+密码） |
| `frontend/src/pages/Dashboard.tsx` | 创建 | 主创作区（SSE 聊天） |
| `frontend/src/pages/Result.tsx` | 创建 | 稿件结果详情 |
| `frontend/src/pages/History.tsx` | 创建 | 历史记录列表 |

---

## Task 1: 清空旧前端，搭建项目骨架

**Files:**
- Delete: `frontend/src/` (all Vue files)
- Create: `frontend/package.json`
- Create: `frontend/vite.config.ts`
- Create: `frontend/tsconfig.json`
- Create: `frontend/index.html`
- Create: `frontend/postcss.config.mjs`
- Create: `frontend/vitest.config.ts`
- Create: `frontend/src/test/setup.ts`

- [ ] **Step 1: 删除旧 Vue 文件**

```bash
cd /data/code/content_creator_imm/frontend
# 删除旧源码和依赖
rm -rf src/ node_modules/ dist/ package.json package-lock.json \
       vite.config.ts tsconfig*.json tsconfig.json index.html \
       postcss.config.* .eslintrc* env.d.ts
```

- [ ] **Step 2: 创建 package.json**

```bash
cat > /data/code/content_creator_imm/frontend/package.json << 'EOF'
{
  "name": "content-creator-imm-frontend",
  "private": true,
  "version": "0.0.1",
  "type": "module",
  "scripts": {
    "dev": "vite",
    "build": "tsc -b && vite build",
    "preview": "vite preview",
    "test": "vitest run",
    "test:watch": "vitest"
  },
  "dependencies": {
    "react": "^18.3.1",
    "react-dom": "^18.3.1",
    "react-router": "^7.1.0",
    "@radix-ui/react-dialog": "^1.1.6",
    "@radix-ui/react-progress": "^1.1.2",
    "@radix-ui/react-separator": "^1.1.2",
    "@radix-ui/react-slot": "^1.1.2",
    "@radix-ui/react-tabs": "^1.1.3",
    "class-variance-authority": "^0.7.1",
    "clsx": "^2.1.1",
    "lucide-react": "^0.487.0",
    "next-themes": "^0.4.6",
    "sonner": "^2.0.3",
    "tailwind-merge": "^3.2.0"
  },
  "devDependencies": {
    "@tailwindcss/vite": "^4.1.12",
    "@testing-library/jest-dom": "^6.6.3",
    "@testing-library/react": "^16.0.0",
    "@testing-library/user-event": "^14.5.2",
    "@types/react": "^18.3.5",
    "@types/react-dom": "^18.3.0",
    "@vitejs/plugin-react": "^4.3.4",
    "jsdom": "^25.0.1",
    "tailwindcss": "^4.1.12",
    "typescript": "^5.6.2",
    "vite": "^6.3.5",
    "vitest": "^2.0.0"
  }
}
EOF
```

- [ ] **Step 3: 创建 vite.config.ts**

```bash
cat > /data/code/content_creator_imm/frontend/vite.config.ts << 'EOF'
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

export default defineConfig({
  plugins: [react(), tailwindcss()],
  base: '/creator/',
  server: {
    port: 5173,
    proxy: {
      '/creator/api': {
        target: 'http://localhost:3004',
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/creator\/api/, '/api'),
      },
    },
  },
  build: {
    outDir: 'dist',
  },
})
EOF
```

- [ ] **Step 4: 创建 tsconfig.json**

```bash
cat > /data/code/content_creator_imm/frontend/tsconfig.json << 'EOF'
{
  "files": [],
  "references": [
    { "path": "./tsconfig.app.json" },
    { "path": "./tsconfig.node.json" }
  ]
}
EOF

cat > /data/code/content_creator_imm/frontend/tsconfig.app.json << 'EOF'
{
  "compilerOptions": {
    "tsBuildInfoFile": "./node_modules/.tmp/tsconfig.app.tsbuildinfo",
    "target": "ES2020",
    "useDefineForClassFields": true,
    "lib": ["ES2020", "DOM", "DOM.Iterable"],
    "module": "ESNext",
    "skipLibCheck": true,
    "moduleResolution": "bundler",
    "allowImportingTsExtensions": true,
    "isolatedModules": true,
    "moduleDetection": "force",
    "noEmit": true,
    "jsx": "react-jsx",
    "strict": true,
    "noUnusedLocals": true,
    "noUnusedParameters": true,
    "noFallthroughCasesInSwitch": true,
    "noUncheckedSideEffectImports": true
  },
  "include": ["src"]
}
EOF

# NOTE: include both vite.config.ts AND vitest.config.ts so tsc -b doesn't error
cat > /data/code/content_creator_imm/frontend/tsconfig.node.json << 'EOF'
{
  "compilerOptions": {
    "tsBuildInfoFile": "./node_modules/.tmp/tsconfig.node.tsbuildinfo",
    "target": "ES2022",
    "lib": ["ES2023"],
    "module": "ESNext",
    "skipLibCheck": true,
    "moduleResolution": "bundler",
    "allowImportingTsExtensions": true,
    "isolatedModules": true,
    "moduleDetection": "force",
    "noEmit": true,
    "strict": true,
    "noUnusedLocals": true,
    "noUnusedParameters": true,
    "noFallthroughCasesInSwitch": true,
    "noUncheckedSideEffectImports": true
  },
  "include": ["vite.config.ts", "vitest.config.ts"]
}
EOF
```

- [ ] **Step 5: 创建 index.html**

```bash
cat > /data/code/content_creator_imm/frontend/index.html << 'EOF'
<!doctype html>
<html lang="zh-CN">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>轻写Claw - AI文案助手</title>
  </head>
  <body>
    <div id="root"></div>
    <script type="module" src="/src/main.tsx"></script>
  </body>
</html>
EOF
```

- [ ] **Step 6: 创建 postcss.config.mjs**

```bash
cat > /data/code/content_creator_imm/frontend/postcss.config.mjs << 'EOF'
export default {
  plugins: {},
}
EOF
```

- [ ] **Step 7: 创建 vitest.config.ts**

```bash
cat > /data/code/content_creator_imm/frontend/vitest.config.ts << 'EOF'
import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  test: {
    environment: 'jsdom',
    setupFiles: ['./src/test/setup.ts'],
    globals: true,
  },
})
EOF
```

- [ ] **Step 8: 创建测试 setup 文件**

```bash
mkdir -p /data/code/content_creator_imm/frontend/src/test
cat > /data/code/content_creator_imm/frontend/src/test/setup.ts << 'EOF'
import '@testing-library/jest-dom'
EOF
```

- [ ] **Step 9: 安装依赖**

```bash
cd /data/code/content_creator_imm/frontend && npm install
```

预期：`node_modules/` 目录创建完成，无报错。

- [ ] **Step 10: 创建最小 main.tsx + App.tsx 验证可启动**

```bash
mkdir -p /data/code/content_creator_imm/frontend/src
cat > /data/code/content_creator_imm/frontend/src/main.tsx << 'EOF'
import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <div>轻写Claw 加载中...</div>
  </StrictMode>
)
EOF
```

- [ ] **Step 11: 验证开发服务器可启动**

```bash
cd /data/code/content_creator_imm/frontend && npm run dev &
sleep 3
curl -s -o /dev/null -w "HTTP %{http_code}\n" http://localhost:5173/creator/
# 预期: HTTP 200
kill %1  # 停止后台 vite
```

- [ ] **Step 12: Commit**

```bash
cd /data/code/content_creator_imm
git add frontend/
git commit -m "chore: 替换 Vue 前端，初始化 React + Vite + Tailwind 项目骨架"
```

---

## Task 2: 样式系统 + UI 组件库

**Files:**
- Create: `frontend/src/styles/theme.css` (from assert/)
- Create: `frontend/src/styles/index.css` (from assert/)
- Create: `frontend/src/lib/utils.ts`
- Create: `frontend/src/components/ui/` (从 assert/ 复制所需组件)

- [ ] **Step 1: 复制样式文件（创建干净的 index.css）**

`assert/src/styles/index.css` 包含 `@import './fonts.css'` 和 `@import './tailwind.css'`，这两个文件不存在于目标目录，直接复制会导致样式加载失败。改为只复制 `theme.css`，并手工创建适配 Tailwind CSS 4 的 `index.css`：

```bash
mkdir -p /data/code/content_creator_imm/frontend/src/styles
cp /data/code/content_creator_imm/assert/src/styles/theme.css \
   /data/code/content_creator_imm/frontend/src/styles/

# 创建干净的 index.css（不 @import fonts.css / tailwind.css）
# Tailwind CSS 4 通过 @tailwindcss/vite 插件注入，不需要 @import
cat > /data/code/content_creator_imm/frontend/src/styles/index.css << 'EOF'
@import "./theme.css";

*, *::before, *::after {
  box-sizing: border-box;
}

body {
  margin: 0;
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', system-ui, sans-serif;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
}
EOF
```

- [ ] **Step 2: 创建 lib/utils.ts**

```bash
mkdir -p /data/code/content_creator_imm/frontend/src/lib
cat > /data/code/content_creator_imm/frontend/src/lib/utils.ts << 'EOF'
import { clsx, type ClassValue } from 'clsx'
import { twMerge } from 'tailwind-merge'

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}
EOF
```

- [ ] **Step 3: 复制 shadcn UI 组件**

```bash
mkdir -p /data/code/content_creator_imm/frontend/src/components/ui

# 复制所需组件（按需选择）
for comp in button card input label tabs separator progress dialog badge; do
  src="/data/code/content_creator_imm/assert/src/app/components/ui/${comp}.tsx"
  if [ -f "$src" ]; then
    cp "$src" /data/code/content_creator_imm/frontend/src/components/ui/
  fi
done

# 复制 utils（shadcn 内部使用）
cp /data/code/content_creator_imm/assert/src/app/components/ui/utils.ts \
   /data/code/content_creator_imm/frontend/src/components/ui/ 2>/dev/null || true
```

- [ ] **Step 4: 修复 ui 组件中的 import 路径，删除冗余 utils.ts**

assert/ 里的组件 import 自 `./utils`。替换路径后删除 `utils.ts` 副本（`lib/utils.ts` 是正式版本）：

```bash
grep -r "from.*utils" /data/code/content_creator_imm/frontend/src/components/ui/

# 将 "./utils" 替换为 "../../lib/utils"
sed -i 's|from "./utils"|from "../../lib/utils"|g' \
    /data/code/content_creator_imm/frontend/src/components/ui/*.tsx

# 将 "@/lib/utils" 替换为 "../../lib/utils"（备用）
sed -i 's|from "@/lib/utils"|from "../../lib/utils"|g' \
    /data/code/content_creator_imm/frontend/src/components/ui/*.tsx

# 删除 ui/utils.ts 副本（避免重复，lib/utils.ts 是唯一正式版本）
rm -f /data/code/content_creator_imm/frontend/src/components/ui/utils.ts

# 验证没有剩余的 ./utils 引用
grep -r 'from.*["\x27]\.\/utils["\x27]' /data/code/content_creator_imm/frontend/src/components/ui/ && echo "WARN: 仍有 ./utils 引用" || echo "OK: 无残留引用"
```

- [ ] **Step 5: 更新 main.tsx 引入全局样式**

```bash
cat > /data/code/content_creator_imm/frontend/src/main.tsx << 'EOF'
import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import './styles/theme.css'
import './styles/index.css'

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <div className="p-4 text-2xl font-semibold bg-gradient-to-r from-blue-600 to-purple-600 bg-clip-text text-transparent">
      轻写Claw 样式测试
    </div>
  </StrictMode>
)
EOF
```

- [ ] **Step 6: 验证样式生效**

```bash
cd /data/code/content_creator_imm/frontend && npm run dev &
sleep 3
curl -s http://localhost:5173/creator/ | grep -q "轻写Claw" && echo "OK" || echo "FAIL"
kill %1
```

- [ ] **Step 7: Commit**

```bash
cd /data/code/content_creator_imm
git add frontend/src/styles/ frontend/src/lib/utils.ts frontend/src/components/ui/
git commit -m "feat: 添加 Tailwind 样式系统和 shadcn UI 组件库"
```

---

## Task 3: 核心基础设施 — SSE 解析器 + API 层 + AuthContext

**Files:**
- Create: `frontend/src/lib/sse.ts`
- Create: `frontend/src/lib/request.ts`
- Create: `frontend/src/api/auth.ts`
- Create: `frontend/src/api/chat.ts`
- Create: `frontend/src/api/conversations.ts`
- Create: `frontend/src/api/scripts.ts`
- Create: `frontend/src/contexts/AuthContext.tsx`
- Test: `frontend/src/lib/__tests__/sse.test.ts`
- Test: `frontend/src/contexts/__tests__/AuthContext.test.tsx`

- [ ] **Step 1: 写 SSE 解析器测试（先写失败测试）**

```bash
mkdir -p /data/code/content_creator_imm/frontend/src/lib/__tests__
cat > /data/code/content_creator_imm/frontend/src/lib/__tests__/sse.test.ts << 'EOF'
import { describe, it, expect } from 'vitest'
import { parseSSELine } from '../sse'

describe('parseSSELine', () => {
  it('parses token event', () => {
    const result = parseSSELine('data: {"type":"token","content":"hello"}')
    expect(result).toEqual({ type: 'token', content: 'hello' })
  })

  it('parses step event', () => {
    const result = parseSSELine('data: {"type":"step","step":1,"name":"分析"}')
    expect(result).toEqual({ type: 'step', step: 1, name: '分析' })
  })

  it('parses info event', () => {
    const result = parseSSELine('data: {"type":"info","content":"已提取500字"}')
    expect(result).toEqual({ type: 'info', content: '已提取500字' })
  })

  it('parses outline event', () => {
    const result = parseSSELine('data: {"type":"outline","data":{"title":"标题","sections":[]}}')
    expect(result).toEqual({ type: 'outline', data: { title: '标题', sections: [] } })
  })

  it('parses action event', () => {
    const result = parseSSELine('data: {"type":"action","options":["方案1","方案2","方案3","方案4"]}')
    expect(result).toEqual({ type: 'action', options: ['方案1', '方案2', '方案3', '方案4'] })
  })

  it('parses similarity event', () => {
    const result = parseSSELine('data: {"type":"similarity","data":{"score":15}}')
    expect(result).toEqual({ type: 'similarity', data: { score: 15 } })
  })

  it('parses complete event', () => {
    const result = parseSSELine('data: {"type":"complete","scriptId":42}')
    expect(result).toEqual({ type: 'complete', scriptId: 42 })
  })

  it('returns null for empty line', () => {
    expect(parseSSELine('')).toBeNull()
  })

  it('returns null for comment line', () => {
    expect(parseSSELine(': keep-alive')).toBeNull()
  })

  it('returns null for malformed JSON', () => {
    expect(parseSSELine('data: {invalid}')).toBeNull()
  })
})
EOF
```

- [ ] **Step 2: 运行测试，确认失败**

```bash
cd /data/code/content_creator_imm/frontend && npm test 2>&1 | tail -20
# 预期: FAIL - Cannot find module '../sse'
```

- [ ] **Step 3: 实现 SSE 解析器**

```bash
cat > /data/code/content_creator_imm/frontend/src/lib/sse.ts << 'EOF'
export type SSEEvent =
  | { type: 'token'; content: string }
  | { type: 'step'; step: number; name: string }
  | { type: 'info'; content: string }
  | { type: 'outline'; data: unknown }
  | { type: 'action'; options: string[] }
  | { type: 'similarity'; data: unknown }
  | { type: 'complete'; scriptId: number }
  | { type: 'error'; message: string }

export function parseSSELine(line: string): SSEEvent | null {
  if (!line.startsWith('data: ')) return null
  try {
    return JSON.parse(line.slice(6)) as SSEEvent
  } catch {
    return null
  }
}
EOF
```

- [ ] **Step 4: 运行测试，确认通过**

```bash
cd /data/code/content_creator_imm/frontend && npm test 2>&1 | tail -10
# 预期: 10 passed
```

- [ ] **Step 5: 创建 fetch 封装（request.ts）**

```bash
cat > /data/code/content_creator_imm/frontend/src/lib/request.ts << 'EOF'
const BASE = '/creator/api'

export class ApiError extends Error {
  constructor(public status: number, message: string) {
    super(message)
  }
}

async function apiFetch<T>(path: string, init?: RequestInit): Promise<T> {
  const token = localStorage.getItem('token')
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...(token ? { Authorization: `Bearer ${token}` } : {}),
    ...(init?.headers as Record<string, string>),
  }
  const res = await fetch(`${BASE}${path}`, { ...init, headers })
  if (res.status === 401) {
    localStorage.removeItem('token')
    localStorage.removeItem('user')
    window.location.href = '/creator/auth'
    throw new ApiError(401, 'Unauthorized')
  }
  if (!res.ok) {
    const body = await res.json().catch(() => ({}))
    throw new ApiError(res.status, (body as { error?: string }).error ?? res.statusText)
  }
  return res.json() as Promise<T>
}

export const api = {
  get: <T>(path: string) => apiFetch<T>(path),
  post: <T>(path: string, body?: unknown) =>
    apiFetch<T>(path, { method: 'POST', body: body !== undefined ? JSON.stringify(body) : undefined }),
}
EOF
```

- [ ] **Step 6: 创建 API 模块**

```bash
mkdir -p /data/code/content_creator_imm/frontend/src/api

cat > /data/code/content_creator_imm/frontend/src/api/auth.ts << 'EOF'
import { api } from '../lib/request'

export interface User {
  id: number
  username: string
  email: string
}

export function login(email: string, password: string) {
  return api.post<{ token: string; user: User }>('/auth/login', { email, password })
}

export function register(username: string, email: string, password: string) {
  return api.post<{ token: string; user: User }>('/auth/register', { username, email, password })
}
EOF

cat > /data/code/content_creator_imm/frontend/src/api/chat.ts << 'EOF'
import { api } from '../lib/request'

export function getSession() {
  return api.get<{ state: string }>('/chat/session')
}

export function resetSession() {
  return api.post<{ message: string; conv_id: number }>('/chat/reset')
}

// 返回原始 Response，调用方自行处理 ReadableStream
export async function sendMessage(message: string): Promise<Response> {
  const token = localStorage.getItem('token') ?? ''
  return fetch('/creator/api/chat/message', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify({ message }),
  })
}
EOF

cat > /data/code/content_creator_imm/frontend/src/api/conversations.ts << 'EOF'
import { api } from '../lib/request'

export interface Conversation {
  id: number
  user_id: number
  title: string
  script_id?: number
  state: number // 0=进行中 1=完成
  created_at: string
}

export interface StoredMsg {
  role: string
  type: string
  content?: string
  data?: unknown
  options?: string[]
  step?: number
  name?: string
}

export function listConversations() {
  return api.get<{ conversations: Conversation[] }>('/conversations')
}

export function getConversation(id: number) {
  return api.get<{ conversation: Conversation; messages: string }>(`/conversations/${id}`)
}
EOF

cat > /data/code/content_creator_imm/frontend/src/api/scripts.ts << 'EOF'
import { api } from '../lib/request'

export interface Script {
  id: number
  title: string
  source_url: string
  similarity_score: number
  viral_score: number
  created_at: string
}

export function getScripts() {
  return api.get<{ scripts: Script[]; total: number }>('/scripts')
}

export function getScript(id: number) {
  return api.get<{ script: Script; content: string }>(`/scripts/${id}`)
}
EOF
```

- [ ] **Step 7: 写 AuthContext 测试（先写）**

```bash
mkdir -p /data/code/content_creator_imm/frontend/src/contexts/__tests__
cat > /data/code/content_creator_imm/frontend/src/contexts/__tests__/AuthContext.test.tsx << 'EOF'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { AuthProvider, useAuth } from '../AuthContext'

function StateDisplay() {
  const { user, token } = useAuth()
  return (
    <div>
      <span data-testid="token">{token ?? 'none'}</span>
      <span data-testid="email">{user?.email ?? 'none'}</span>
    </div>
  )
}

describe('AuthContext', () => {
  beforeEach(() => {
    localStorage.clear()
    vi.restoreAllMocks()
  })

  it('初始化时从 localStorage 读取 token 和 user', () => {
    localStorage.setItem('token', 'stored-token')
    localStorage.setItem('user', JSON.stringify({ id: 1, username: 'u', email: 'u@test.com' }))
    render(<AuthProvider><StateDisplay /></AuthProvider>)
    expect(screen.getByTestId('token').textContent).toBe('stored-token')
    expect(screen.getByTestId('email').textContent).toBe('u@test.com')
  })

  it('login 成功后存入 localStorage 并更新状态', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({
        token: 'new-token',
        user: { id: 2, username: 'v', email: 'v@test.com' },
      }),
    }))

    function LoginBtn() {
      const { login } = useAuth()
      return <button onClick={() => login('v@test.com', 'pass123')}>login</button>
    }

    render(<AuthProvider><StateDisplay /><LoginBtn /></AuthProvider>)
    await userEvent.click(screen.getByRole('button'))
    await waitFor(() => expect(screen.getByTestId('token').textContent).toBe('new-token'))
    expect(localStorage.getItem('token')).toBe('new-token')
  })

  it('logout 清除 localStorage 并重置状态', async () => {
    localStorage.setItem('token', 'old-token')
    localStorage.setItem('user', JSON.stringify({ id: 1, username: 'u', email: 'u@test.com' }))

    function LogoutBtn() {
      const { logout } = useAuth()
      return <button onClick={logout}>logout</button>
    }

    render(<AuthProvider><StateDisplay /><LogoutBtn /></AuthProvider>)
    await userEvent.click(screen.getByRole('button'))
    await waitFor(() => expect(screen.getByTestId('token').textContent).toBe('none'))
    expect(localStorage.getItem('token')).toBeNull()
  })

  it('login 失败时抛出错误', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: false,
      json: () => Promise.resolve({ error: '密码错误' }),
    }))

    let caughtError = ''

    function LoginBtn() {
      const { login } = useAuth()
      return (
        <button
          onClick={() =>
            login('a@b.com', 'wrong').catch((e: Error) => {
              caughtError = e.message
            })
          }
        >
          login
        </button>
      )
    }

    render(<AuthProvider><StateDisplay /><LoginBtn /></AuthProvider>)
    await userEvent.click(screen.getByRole('button'))
    await waitFor(() => expect(caughtError).toBe('密码错误'))
  })
})
EOF
```

- [ ] **Step 8: 运行测试，确认失败**

```bash
cd /data/code/content_creator_imm/frontend && npm test 2>&1 | tail -20
# 预期: FAIL - Cannot find module '../AuthContext'
```

- [ ] **Step 9: 实现 AuthContext**

```bash
cat > /data/code/content_creator_imm/frontend/src/contexts/AuthContext.tsx << 'EOF'
import { createContext, useContext, useState, ReactNode } from 'react'
import type { User } from '../api/auth'

interface AuthContextValue {
  user: User | null
  token: string | null
  login: (email: string, password: string) => Promise<void>
  register: (username: string, email: string, password: string) => Promise<void>
  logout: () => void
}

const AuthContext = createContext<AuthContextValue | null>(null)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [token, setToken] = useState<string | null>(() => localStorage.getItem('token'))
  const [user, setUser] = useState<User | null>(() => {
    const u = localStorage.getItem('user')
    return u ? (JSON.parse(u) as User) : null
  })

  const _store = (t: string, u: User) => {
    localStorage.setItem('token', t)
    localStorage.setItem('user', JSON.stringify(u))
    setToken(t)
    setUser(u)
  }

  const login = async (email: string, password: string) => {
    const res = await fetch('/creator/api/auth/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email, password }),
    })
    if (!res.ok) {
      const body = await res.json().catch(() => ({}))
      throw new Error((body as { error?: string }).error ?? '登录失败')
    }
    const data = (await res.json()) as { token: string; user: User }
    _store(data.token, data.user)
  }

  const register = async (username: string, email: string, password: string) => {
    const res = await fetch('/creator/api/auth/register', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ username, email, password }),
    })
    if (!res.ok) {
      const body = await res.json().catch(() => ({}))
      throw new Error((body as { error?: string }).error ?? '注册失败')
    }
    const data = (await res.json()) as { token: string; user: User }
    _store(data.token, data.user)
  }

  const logout = () => {
    localStorage.removeItem('token')
    localStorage.removeItem('user')
    setToken(null)
    setUser(null)
  }

  return (
    <AuthContext.Provider value={{ user, token, login, register, logout }}>
      {children}
    </AuthContext.Provider>
  )
}

export function useAuth() {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth must be used within AuthProvider')
  return ctx
}
EOF
```

- [ ] **Step 10: 运行全部测试**

```bash
cd /data/code/content_creator_imm/frontend && npm test 2>&1 | tail -15
# 预期: 14 passed (10 SSE + 4 AuthContext)
```

- [ ] **Step 11: Commit**

```bash
cd /data/code/content_creator_imm
git add frontend/src/lib/ frontend/src/api/ frontend/src/contexts/
git commit -m "feat: 添加 SSE 解析器、API 层和 AuthContext（含测试）"
```

---

## Task 4: 路由 + App 根组件

**Files:**
- Create: `frontend/src/router.tsx`
- Create: `frontend/src/App.tsx`
- Modify: `frontend/src/main.tsx`

- [ ] **Step 1: 创建 router.tsx（含 ProtectedRoute）**

```bash
cat > /data/code/content_creator_imm/frontend/src/router.tsx << 'EOF'
import { createBrowserRouter, Navigate, Outlet } from 'react-router'
import { useAuth } from './contexts/AuthContext'
import { Layout } from './components/Layout'
import { Home } from './pages/Home'
import { Auth } from './pages/Auth'
import { Dashboard } from './pages/Dashboard'
import { Result } from './pages/Result'
import { History } from './pages/History'

function ProtectedRoute() {
  const { token } = useAuth()
  return token ? <Outlet /> : <Navigate to="/auth" replace />
}

export const router = createBrowserRouter(
  [
    {
      path: '/',
      Component: Layout,
      children: [
        { index: true, Component: Home },
        { path: 'auth', Component: Auth },
        {
          Component: ProtectedRoute,
          children: [
            { path: 'dashboard', Component: Dashboard },
            { path: 'result/:id', Component: Result },
            { path: 'history', Component: History },
          ],
        },
      ],
    },
  ],
  { basename: '/creator' }
)
EOF
```

- [ ] **Step 2: 创建 App.tsx**

```bash
cat > /data/code/content_creator_imm/frontend/src/App.tsx << 'EOF'
import { RouterProvider } from 'react-router'
import { ThemeProvider } from 'next-themes'
import { Toaster } from 'sonner'
import { AuthProvider } from './contexts/AuthContext'
import { router } from './router'

export default function App() {
  return (
    <ThemeProvider attribute="class" defaultTheme="system" enableSystem>
      <AuthProvider>
        <RouterProvider router={router} />
        <Toaster position="top-center" richColors />
      </AuthProvider>
    </ThemeProvider>
  )
}
EOF
```

- [ ] **Step 3: 更新 main.tsx**

```bash
cat > /data/code/content_creator_imm/frontend/src/main.tsx << 'EOF'
import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import App from './App'
import './styles/theme.css'
import './styles/index.css'

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <App />
  </StrictMode>
)
EOF
```

- [ ] **Step 4: 创建页面桩文件（占位，避免 import 报错）**

```bash
mkdir -p /data/code/content_creator_imm/frontend/src/pages
mkdir -p /data/code/content_creator_imm/frontend/src/components

for page in Home Auth Dashboard Result History; do
cat > /data/code/content_creator_imm/frontend/src/pages/${page}.tsx << STUB
export function ${page}() {
  return <div className="p-8 text-xl">${page} 页面（施工中）</div>
}
STUB
done

# Layout 桩
cat > /data/code/content_creator_imm/frontend/src/components/Layout.tsx << 'EOF'
import { Outlet } from 'react-router'
export function Layout() {
  return (
    <div className="min-h-screen bg-gradient-to-br from-blue-50 via-white to-purple-50 dark:from-gray-900 dark:via-gray-800 dark:to-gray-900">
      <Outlet />
    </div>
  )
}
EOF
```

- [ ] **Step 5: 运行测试确保无回归**

```bash
cd /data/code/content_creator_imm/frontend && npm test 2>&1 | tail -5
# 预期: 14 passed
```

- [ ] **Step 6: Commit**

```bash
cd /data/code/content_creator_imm
git add frontend/src/router.tsx frontend/src/App.tsx frontend/src/main.tsx \
        frontend/src/pages/ frontend/src/components/Layout.tsx
git commit -m "feat: 添加路由、ProtectedRoute 和 App 根组件"
```

---

## Task 5: Layout 组件（Header + 移动导航 + 暗色切换）

**Files:**
- Modify: `frontend/src/components/Layout.tsx`

从 `assert/src/app/components/Layout.tsx` 改造：保留视觉结构，接入真实 Auth 状态，加入暗色切换按钮。

- [ ] **Step 1: 实现完整 Layout**

```bash
cat > /data/code/content_creator_imm/frontend/src/components/Layout.tsx << 'EOF'
import { Outlet, Link, useLocation } from 'react-router'
import { Feather, Home, History, LogOut, Sun, Moon } from 'lucide-react'
import { useTheme } from 'next-themes'
import { Button } from './ui/button'
import { useAuth } from '../contexts/AuthContext'

export function Layout() {
  const location = useLocation()
  const { user, logout } = useAuth()
  const { theme, setTheme } = useTheme()

  const isActive = (path: string) => {
    if (path === '/') return location.pathname === '/'
    return location.pathname.startsWith(path)
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-blue-50 via-white to-purple-50 dark:from-gray-950 dark:via-gray-900 dark:to-gray-950">
      {/* Header */}
      <header className="sticky top-0 z-50 bg-white/80 dark:bg-gray-900/80 backdrop-blur-lg border-b border-gray-200 dark:border-gray-800">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex items-center justify-between h-16">
            <Link to="/" className="flex items-center gap-3">
              <div className="w-10 h-10 bg-gradient-to-br from-blue-500 to-purple-600 rounded-xl flex items-center justify-center shadow-lg shadow-blue-200">
                <Feather className="w-5 h-5 text-white" strokeWidth={2.5} />
              </div>
              <div className="flex flex-col">
                <span className="text-xl font-semibold bg-gradient-to-r from-blue-600 to-purple-600 bg-clip-text text-transparent">
                  轻写Claw
                </span>
                <span className="text-xs text-gray-500 dark:text-gray-400 hidden sm:block">你的AI文案助手</span>
              </div>
            </Link>

            <nav className="flex items-center gap-2">
              {user && (
                <>
                  <Link
                    to="/dashboard"
                    className={`hidden sm:flex items-center gap-2 px-3 py-2 rounded-lg transition-colors text-sm ${
                      isActive('/dashboard') ? 'bg-blue-50 text-blue-600 dark:bg-blue-950 dark:text-blue-400' : 'text-gray-600 dark:text-gray-400 hover:bg-gray-50 dark:hover:bg-gray-800'
                    }`}
                  >
                    <Home className="w-4 h-4" />
                    <span>创作</span>
                  </Link>
                  <Link
                    to="/history"
                    className={`hidden sm:flex items-center gap-2 px-3 py-2 rounded-lg transition-colors text-sm ${
                      isActive('/history') ? 'bg-blue-50 text-blue-600 dark:bg-blue-950 dark:text-blue-400' : 'text-gray-600 dark:text-gray-400 hover:bg-gray-50 dark:hover:bg-gray-800'
                    }`}
                  >
                    <History className="w-4 h-4" />
                    <span>历史</span>
                  </Link>
                </>
              )}

              {/* 暗色切换 */}
              <Button
                variant="ghost"
                size="icon"
                onClick={() => setTheme(theme === 'dark' ? 'light' : 'dark')}
                className="w-9 h-9 text-gray-600 dark:text-gray-400"
              >
                <Sun className="h-4 w-4 rotate-0 scale-100 transition-all dark:-rotate-90 dark:scale-0" />
                <Moon className="absolute h-4 w-4 rotate-90 scale-0 transition-all dark:rotate-0 dark:scale-100" />
              </Button>

              {user ? (
                <Button variant="ghost" size="sm" onClick={logout} className="text-gray-600 dark:text-gray-400">
                  <LogOut className="w-4 h-4 mr-1" />
                  <span className="hidden sm:inline">退出</span>
                </Button>
              ) : (
                <>
                  <Link to="/auth">
                    <Button variant="ghost" className="text-gray-700 dark:text-gray-300">登录</Button>
                  </Link>
                  <Link to="/auth">
                    <Button className="bg-gradient-to-r from-blue-600 to-purple-600 hover:from-blue-700 hover:to-purple-700 text-white border-0">
                      免费试用
                    </Button>
                  </Link>
                </>
              )}
            </nav>
          </div>
        </div>
      </header>

      <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <Outlet />
      </main>

      {/* 移动端底部导航（仅登录后显示） */}
      {user && (
        <>
          <nav className="md:hidden fixed bottom-0 left-0 right-0 bg-white dark:bg-gray-900 border-t border-gray-200 dark:border-gray-800 z-50">
            <div className="grid grid-cols-3 gap-1 px-2 py-2">
              <Link
                to="/dashboard"
                className={`flex flex-col items-center gap-1 px-3 py-2 rounded-lg transition-colors ${
                  isActive('/dashboard') ? 'bg-blue-50 text-blue-600 dark:bg-blue-950 dark:text-blue-400' : 'text-gray-600 dark:text-gray-400'
                }`}
              >
                <Home className="w-5 h-5" />
                <span className="text-xs">创作</span>
              </Link>
              <Link
                to="/history"
                className={`flex flex-col items-center gap-1 px-3 py-2 rounded-lg transition-colors ${
                  isActive('/history') ? 'bg-blue-50 text-blue-600 dark:bg-blue-950 dark:text-blue-400' : 'text-gray-600 dark:text-gray-400'
                }`}
              >
                <History className="w-5 h-5" />
                <span className="text-xs">历史</span>
              </Link>
              <button
                onClick={logout}
                className="flex flex-col items-center gap-1 px-3 py-2 rounded-lg transition-colors text-gray-600 dark:text-gray-400"
              >
                <LogOut className="w-5 h-5" />
                <span className="text-xs">退出</span>
              </button>
            </div>
          </nav>
          <div className="md:hidden h-20" />
        </>
      )}
    </div>
  )
}
EOF
```

- [ ] **Step 2: 运行测试**

```bash
cd /data/code/content_creator_imm/frontend && npm test 2>&1 | tail -5
# 预期: 14 passed
```

- [ ] **Step 3: Commit**

```bash
cd /data/code/content_creator_imm
git add frontend/src/components/Layout.tsx
git commit -m "feat: 实现 Layout 组件（Header + 移动导航 + 暗色切换）"
```

---

## Task 6: Landing 页 + Auth 页

**Files:**
- Modify: `frontend/src/pages/Home.tsx`
- Modify: `frontend/src/pages/Auth.tsx`

- [ ] **Step 1: 实现 Landing 页（从设计稿移植）**

从 `assert/src/app/pages/Home.tsx` 复制，检查并修复 import 路径。

```bash
cp /data/code/content_creator_imm/assert/src/app/pages/Home.tsx \
   /data/code/content_creator_imm/frontend/src/pages/Home.tsx
```

- [ ] **Step 2: 检查并修复 Home.tsx 的 import**

```bash
head -20 /data/code/content_creator_imm/frontend/src/pages/Home.tsx
# 查看实际 import 路径，按需修复
# 常见路径：../components/ui/ → ../components/ui/（通常已正确）
# 若有 @/components/ 前缀，替换为相对路径：
grep "from.*@/" /data/code/content_creator_imm/frontend/src/pages/Home.tsx && \
  sed -i 's|from "@/components/|from "../components/|g' \
      /data/code/content_creator_imm/frontend/src/pages/Home.tsx || echo "无 @/ 路径，跳过"
```

- [ ] **Step 3: 实现 Auth 页（从设计稿改造，换为邮箱+密码）**

```bash
cat > /data/code/content_creator_imm/frontend/src/pages/Auth.tsx << 'EOF'
import { useState } from 'react'
import { useNavigate, Navigate } from 'react-router'
import { Card } from '../components/ui/card'
import { Button } from '../components/ui/button'
import { Input } from '../components/ui/input'
import { Label } from '../components/ui/label'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '../components/ui/tabs'
import { Feather, Mail, Lock, User } from 'lucide-react'
import { toast } from 'sonner'
import { useAuth } from '../contexts/AuthContext'

export function Auth() {
  const { login, register, token } = useAuth()
  const navigate = useNavigate()
  const [loading, setLoading] = useState(false)

  // 已登录直接跳转
  if (token) return <Navigate to="/dashboard" replace />

  const handleLogin = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault()
    const form = e.currentTarget
    const email = (form.elements.namedItem('email') as HTMLInputElement).value
    const password = (form.elements.namedItem('password') as HTMLInputElement).value
    setLoading(true)
    try {
      await login(email, password)
      toast.success('登录成功！欢迎回来')
      navigate('/dashboard')
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '登录失败')
    } finally {
      setLoading(false)
    }
  }

  const handleRegister = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault()
    const form = e.currentTarget
    const username = (form.elements.namedItem('username') as HTMLInputElement).value
    const email = (form.elements.namedItem('email') as HTMLInputElement).value
    const password = (form.elements.namedItem('password') as HTMLInputElement).value
    setLoading(true)
    try {
      await register(username, email, password)
      toast.success('注册成功！')
      navigate('/dashboard')
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '注册失败')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="max-w-md mx-auto">
      <div className="text-center mb-8">
        <div className="inline-flex items-center justify-center w-16 h-16 bg-gradient-to-br from-blue-500 to-purple-600 rounded-2xl mb-4 shadow-xl shadow-blue-200">
          <Feather className="w-8 h-8 text-white" strokeWidth={2.5} />
        </div>
        <h1 className="text-3xl mb-2">欢迎来到轻写Claw</h1>
        <p className="text-gray-600 dark:text-gray-400">登录后可保存您的创作记录</p>
      </div>

      <Card className="p-6 sm:p-8 shadow-lg border-0 bg-white/80 dark:bg-gray-900/80 backdrop-blur">
        <Tabs defaultValue="login" className="w-full">
          <TabsList className="grid w-full grid-cols-2 mb-6">
            <TabsTrigger value="login">登录</TabsTrigger>
            <TabsTrigger value="register">注册</TabsTrigger>
          </TabsList>

          <TabsContent value="login">
            <form onSubmit={handleLogin} className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="login-email">邮箱</Label>
                <div className="relative">
                  <Mail className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
                  <Input id="login-email" name="email" type="email" placeholder="请输入邮箱" className="pl-10" required />
                </div>
              </div>
              <div className="space-y-2">
                <Label htmlFor="login-password">密码</Label>
                <div className="relative">
                  <Lock className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
                  <Input id="login-password" name="password" type="password" placeholder="请输入密码" className="pl-10" required />
                </div>
              </div>
              <Button
                type="submit"
                disabled={loading}
                className="w-full h-11 bg-gradient-to-r from-blue-600 to-purple-600 hover:from-blue-700 hover:to-purple-700 text-white"
              >
                {loading ? '登录中...' : '登录'}
              </Button>
            </form>
          </TabsContent>

          <TabsContent value="register">
            <form onSubmit={handleRegister} className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="reg-username">用户名</Label>
                <div className="relative">
                  <User className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
                  <Input id="reg-username" name="username" type="text" placeholder="请输入用户名" className="pl-10" required />
                </div>
              </div>
              <div className="space-y-2">
                <Label htmlFor="reg-email">邮箱</Label>
                <div className="relative">
                  <Mail className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
                  <Input id="reg-email" name="email" type="email" placeholder="请输入邮箱" className="pl-10" required />
                </div>
              </div>
              <div className="space-y-2">
                <Label htmlFor="reg-password">密码</Label>
                <div className="relative">
                  <Lock className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
                  <Input id="reg-password" name="password" type="password" placeholder="请输入密码（至少6位）" className="pl-10" required minLength={6} />
                </div>
              </div>
              <Button
                type="submit"
                disabled={loading}
                className="w-full h-11 bg-gradient-to-r from-blue-600 to-purple-600 hover:from-blue-700 hover:to-purple-700 text-white"
              >
                {loading ? '注册中...' : '注册'}
              </Button>
            </form>
          </TabsContent>
        </Tabs>
      </Card>
    </div>
  )
}
EOF
```

- [ ] **Step 4: 运行测试**

```bash
cd /data/code/content_creator_imm/frontend && npm test 2>&1 | tail -5
# 预期: 14 passed
```

- [ ] **Step 5: Commit**

```bash
cd /data/code/content_creator_imm
git add frontend/src/pages/Home.tsx frontend/src/pages/Auth.tsx
git commit -m "feat: 实现 Landing 首页和登录/注册页"
```

---

## Task 7: Sidebar 组件

**Files:**
- Create: `frontend/src/components/Sidebar.tsx`

- [ ] **Step 1: 实现 Sidebar（会话 Tab + 稿件 Tab + 集成入口）**

```bash
cat > /data/code/content_creator_imm/frontend/src/components/Sidebar.tsx << 'EOF'
import { useState, useEffect } from 'react'
import { Plus, MessageSquare, FileText, Send, MoreHorizontal } from 'lucide-react'
import { listConversations, type Conversation } from '../api/conversations'
import { getScripts, getScript, type Script } from '../api/scripts'

interface SidebarProps {
  onNewChat: () => void
  onSelectConversation: (conv: Conversation) => void
  onSelectScript: (content: string, title: string) => void
  activeConvId?: number
  refreshTrigger?: number  // 外部触发刷新
}

export function Sidebar({ onNewChat, onSelectConversation, onSelectScript, activeConvId, refreshTrigger }: SidebarProps) {
  const [activeTab, setActiveTab] = useState<'conversations' | 'scripts'>('conversations')
  const [conversations, setConversations] = useState<Conversation[]>([])
  const [scripts, setScripts] = useState<Script[]>([])

  const loadConversations = async () => {
    try {
      const data = await listConversations()
      setConversations(data.conversations ?? [])
    } catch { /* ignore */ }
  }

  const loadScripts = async () => {
    try {
      const data = await getScripts()
      setScripts(data.scripts ?? [])
    } catch { /* ignore */ }
  }

  useEffect(() => { loadConversations() }, [refreshTrigger])
  useEffect(() => { if (activeTab === 'scripts') loadScripts() }, [activeTab])

  const handleScriptClick = async (id: number, title: string) => {
    try {
      const data = await getScript(id)
      onSelectScript(data.content, title)
    } catch { /* ignore */ }
  }

  return (
    <div className="w-64 h-screen bg-gray-50 dark:bg-gray-900 border-r border-gray-200 dark:border-gray-800 flex flex-col flex-shrink-0">
      {/* 新建对话 */}
      <div className="p-3">
        <button
          onClick={onNewChat}
          className="w-full flex items-center justify-center gap-2 px-4 py-2.5 bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 text-gray-700 dark:text-gray-300 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors"
        >
          <Plus className="w-4 h-4" />
          <span className="font-medium text-sm">新建对话</span>
        </button>
      </div>

      {/* 集成入口 */}
      <div className="px-3 pb-3 border-b border-gray-200 dark:border-gray-800">
        {[{ label: '配置到飞书' }, { label: '配置到钉钉' }, { label: '配置到企业微信' }].map((item) => (
          <button
            key={item.label}
            disabled
            className="w-full flex items-center justify-between px-3 py-2 text-gray-400 dark:text-gray-600 rounded-lg text-sm cursor-not-allowed"
          >
            <div className="flex items-center gap-2.5">
              <Send className="w-4 h-4" />
              <span>{item.label}</span>
            </div>
            <span className="text-xs bg-gray-200 dark:bg-gray-700 px-1.5 py-0.5 rounded">开发中</span>
          </button>
        ))}
      </div>

      {/* Tabs */}
      <div className="flex border-b border-gray-200 dark:border-gray-800">
        {(['conversations', 'scripts'] as const).map((tab) => (
          <button
            key={tab}
            onClick={() => setActiveTab(tab)}
            className={`flex-1 flex items-center justify-center gap-1.5 py-2 text-xs font-medium transition-colors ${
              activeTab === tab
                ? 'text-blue-600 dark:text-blue-400 border-b-2 border-blue-600'
                : 'text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-300'
            }`}
          >
            {tab === 'conversations' ? <MessageSquare className="w-3.5 h-3.5" /> : <FileText className="w-3.5 h-3.5" />}
            {tab === 'conversations' ? '会话' : '稿件'}
          </button>
        ))}
      </div>

      {/* 列表 */}
      <div className="flex-1 overflow-y-auto px-2 py-2">
        {activeTab === 'conversations' ? (
          conversations.length === 0 ? (
            <div className="text-center py-8 text-sm text-gray-400">暂无会话记录</div>
          ) : (
            <div className="space-y-0.5">
              {conversations.map((conv) => (
                <button
                  key={conv.id}
                  onClick={() => onSelectConversation(conv)}
                  className={`w-full flex items-center gap-2 px-3 py-2 text-left text-sm rounded-lg transition-colors group ${
                    activeConvId === conv.id
                      ? 'bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 shadow-sm'
                      : 'text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-800'
                  }`}
                >
                  <MessageSquare className="w-4 h-4 flex-shrink-0 text-gray-400" />
                  <div className="flex-1 truncate">{conv.title || '未命名会话'}</div>
                  <MoreHorizontal className="w-4 h-4 text-gray-400 opacity-0 group-hover:opacity-100 transition-opacity" />
                </button>
              ))}
            </div>
          )
        ) : (
          scripts.length === 0 ? (
            <div className="text-center py-8 text-sm text-gray-400">暂无稿件</div>
          ) : (
            <div className="space-y-0.5">
              {scripts.map((script) => (
                <button
                  key={script.id}
                  onClick={() => handleScriptClick(script.id, script.title)}
                  className="w-full flex items-center gap-2 px-3 py-2 text-left text-sm rounded-lg transition-colors text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-800"
                >
                  <FileText className="w-4 h-4 flex-shrink-0 text-gray-400" />
                  <div className="flex-1 truncate">{script.title || '未命名稿件'}</div>
                </button>
              ))}
            </div>
          )
        )}
      </div>
    </div>
  )
}
EOF
```

- [ ] **Step 2: 运行测试**

```bash
cd /data/code/content_creator_imm/frontend && npm test 2>&1 | tail -5
# 预期: 14 passed
```

- [ ] **Step 3: Commit**

```bash
cd /data/code/content_creator_imm
git add frontend/src/components/Sidebar.tsx
git commit -m "feat: 实现 Sidebar（会话/稿件 Tab + 集成入口占位）"
```

---

## Task 8: Dashboard 聊天子组件

**Files:**
- Create: `frontend/src/components/create/MessageList.tsx`
- Create: `frontend/src/components/create/ChatInput.tsx`
- Create: `frontend/src/components/create/LoadingState.tsx`
- Create: `frontend/src/components/create/OutlineEditor.tsx`
- Create: `frontend/src/components/create/ScriptEditor.tsx`

- [ ] **Step 1: 创建 MessageList.tsx**

```bash
mkdir -p /data/code/content_creator_imm/frontend/src/components/create

cat > /data/code/content_creator_imm/frontend/src/components/create/MessageList.tsx << 'EOF'
import { Bot, User } from 'lucide-react'

export interface ChatMsg {
  id: string
  type: 'user' | 'ai' | 'step' | 'info' | 'action' | 'similarity' | 'error'
  content?: string
  options?: string[]        // for action type
  data?: unknown            // for outline/similarity
  streaming?: boolean
}

interface MessageListProps {
  messages: ChatMsg[]
  onAction?: (option: string) => void
  disabled?: boolean
}

export function MessageList({ messages, onAction, disabled }: MessageListProps) {
  return (
    <div className="flex-1 overflow-y-auto p-4 space-y-4">
      {messages.map((msg) => (
        <div
          key={msg.id}
          className={`flex gap-3 ${msg.type === 'user' ? 'justify-end' : 'justify-start'}`}
        >
          {msg.type !== 'user' && (
            <div className="flex-shrink-0 w-8 h-8 rounded-full bg-gradient-to-br from-blue-500 to-purple-600 flex items-center justify-center">
              <Bot className="w-4 h-4 text-white" />
            </div>
          )}

          <div className={`max-w-[80%] space-y-2 ${msg.type === 'user' ? '' : ''}`}>
            {msg.type === 'user' && (
              <div className="rounded-2xl px-4 py-3 bg-gradient-to-br from-blue-500 to-purple-600 text-white">
                <p className="text-sm leading-relaxed whitespace-pre-wrap">{msg.content}</p>
              </div>
            )}

            {(msg.type === 'ai') && (
              <div className="rounded-2xl px-4 py-3 bg-gray-100 dark:bg-gray-800 text-gray-900 dark:text-gray-100">
                <p className="text-sm leading-relaxed whitespace-pre-wrap">
                  {msg.content}
                  {msg.streaming && <span className="inline-block w-1 h-4 ml-0.5 bg-blue-500 animate-pulse" />}
                </p>
              </div>
            )}

            {msg.type === 'step' && (
              <div className="flex items-center gap-2 text-xs text-gray-500 dark:text-gray-400">
                <span className="w-1.5 h-1.5 rounded-full bg-blue-500" />
                {msg.content}
              </div>
            )}

            {msg.type === 'info' && (
              <div className="rounded-xl px-3 py-2 bg-blue-50 dark:bg-blue-950 text-blue-800 dark:text-blue-200 text-sm border border-blue-200 dark:border-blue-800">
                {msg.content}
              </div>
            )}

            {msg.type === 'error' && (
              <div className="rounded-xl px-3 py-2 bg-red-50 dark:bg-red-950 text-red-700 dark:text-red-300 text-sm border border-red-200 dark:border-red-800">
                ❌ {msg.content}
              </div>
            )}

            {msg.type === 'action' && msg.options && (
              <div className="flex flex-wrap gap-2">
                {msg.options.map((opt, i) => (
                  <button
                    key={i}
                    disabled={disabled}
                    onClick={() => onAction?.(String(i + 1))}
                    className="px-4 py-2 text-sm bg-gradient-to-br from-blue-500 to-purple-600 text-white rounded-lg hover:opacity-90 disabled:opacity-50 disabled:cursor-not-allowed transition-opacity"
                  >
                    {opt}
                  </button>
                ))}
              </div>
            )}

            {msg.type === 'similarity' && msg.data && (
              <div className="rounded-xl px-3 py-2 bg-green-50 dark:bg-green-950 text-green-800 dark:text-green-200 text-sm border border-green-200 dark:border-green-800">
                相似度检测完成 ✅
              </div>
            )}
          </div>

          {msg.type === 'user' && (
            <div className="flex-shrink-0 w-8 h-8 rounded-full bg-gray-200 dark:bg-gray-700 flex items-center justify-center">
              <User className="w-4 h-4 text-gray-600 dark:text-gray-400" />
            </div>
          )}
        </div>
      ))}
    </div>
  )
}
EOF
```

- [ ] **Step 2: 创建 ChatInput.tsx**

```bash
cat > /data/code/content_creator_imm/frontend/src/components/create/ChatInput.tsx << 'EOF'
import { useState } from 'react'
import { Send } from 'lucide-react'

interface ChatInputProps {
  onSend: (message: string) => void
  placeholder?: string
  disabled?: boolean
}

export function ChatInput({ onSend, placeholder = '输入你的需求...', disabled }: ChatInputProps) {
  const [input, setInput] = useState('')

  const handleSend = () => {
    if (!input.trim() || disabled) return
    onSend(input.trim())
    setInput('')
  }

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      handleSend()
    }
  }

  return (
    <div className="p-4 border-t border-gray-200 dark:border-gray-800 bg-white dark:bg-gray-900">
      <div className="relative">
        <textarea
          value={input}
          onChange={(e) => setInput(e.target.value)}
          onKeyDown={handleKeyDown}
          placeholder={placeholder}
          disabled={disabled}
          rows={2}
          className="w-full px-4 py-3 pr-12 border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 rounded-xl resize-none focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:bg-gray-50 dark:disabled:bg-gray-900 disabled:cursor-not-allowed text-sm"
        />
        <button
          onClick={handleSend}
          disabled={!input.trim() || disabled}
          className="absolute bottom-2 right-2 w-8 h-8 bg-gradient-to-br from-blue-500 to-purple-600 rounded-lg flex items-center justify-center hover:opacity-90 transition-all disabled:opacity-40 disabled:cursor-not-allowed"
        >
          <Send className="w-4 h-4 text-white" />
        </button>
      </div>
    </div>
  )
}
EOF
```

- [ ] **Step 3: 创建 LoadingState.tsx**

```bash
cp /data/code/content_creator_imm/assert/src/app/components/create/LoadingState.tsx \
   /data/code/content_creator_imm/frontend/src/components/create/LoadingState.tsx
# 修复 import: lucide-react 路径已正确
```

- [ ] **Step 4: 创建 OutlineEditor.tsx（可编辑）**

```bash
cat > /data/code/content_creator_imm/frontend/src/components/create/OutlineEditor.tsx << 'EOF'
interface OutlineEditorProps {
  content: string
  onChange: (content: string) => void
}

export function OutlineEditor({ content, onChange }: OutlineEditorProps) {
  return (
    <div className="h-full w-full flex flex-col bg-white dark:bg-gray-900">
      <div className="flex-shrink-0 p-6 border-b border-gray-200 dark:border-gray-800">
        <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100">大纲预览</h3>
        <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">可在此编辑大纲内容，确认后点击左侧按钮继续</p>
      </div>
      <div className="flex-1 overflow-y-auto">
        <textarea
          value={content}
          onChange={(e) => onChange(e.target.value)}
          className="w-full h-full p-6 text-sm leading-relaxed resize-none focus:outline-none bg-transparent text-gray-800 dark:text-gray-200"
          placeholder="大纲内容..."
        />
      </div>
    </div>
  )
}
EOF
```

- [ ] **Step 5: 创建 ScriptEditor.tsx**

```bash
cat > /data/code/content_creator_imm/frontend/src/components/create/ScriptEditor.tsx << 'EOF'
import { useState } from 'react'
import { Copy, Download, RotateCcw, Check, ExternalLink } from 'lucide-react'
import { useNavigate } from 'react-router'

interface ScriptEditorProps {
  content: string
  scriptId?: number | null
  onRegenerate?: () => void
}

export function ScriptEditor({ content, scriptId, onRegenerate }: ScriptEditorProps) {
  const [copied, setCopied] = useState(false)
  const navigate = useNavigate()

  const handleCopy = () => {
    navigator.clipboard.writeText(content)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  const handleDownload = () => {
    const blob = new Blob([content], { type: 'text/plain' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `爆款口播稿_${new Date().toLocaleDateString('zh-CN')}.txt`
    a.click()
    URL.revokeObjectURL(url)
  }

  return (
    <div className="h-full w-full flex flex-col bg-white dark:bg-gray-900">
      <div className="flex-shrink-0 p-4 border-b border-gray-200 dark:border-gray-800 flex items-center justify-between">
        <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100">完整口播稿</h3>
        <div className="flex gap-1">
          <button
            onClick={handleCopy}
            className="flex items-center gap-1.5 px-3 py-1.5 text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-800 rounded-lg transition-colors"
          >
            {copied ? <><Check className="w-4 h-4 text-green-600" /><span className="text-green-600">已复制</span></> : <><Copy className="w-4 h-4" /><span>复制</span></>}
          </button>
          <button
            onClick={handleDownload}
            className="flex items-center gap-1.5 px-3 py-1.5 text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-800 rounded-lg transition-colors"
          >
            <Download className="w-4 h-4" />
            <span>导出</span>
          </button>
          {onRegenerate && (
            <button
              onClick={onRegenerate}
              className="flex items-center gap-1.5 px-3 py-1.5 text-sm text-blue-600 hover:bg-blue-50 dark:hover:bg-blue-950 rounded-lg transition-colors"
            >
              <RotateCcw className="w-4 h-4" />
              <span>重新生成</span>
            </button>
          )}
        </div>
      </div>
      <div className="flex-1 overflow-y-auto p-6">
        <p className="text-sm leading-relaxed whitespace-pre-wrap text-gray-800 dark:text-gray-200">{content}</p>
      </div>
      {scriptId && (
        <div className="flex-shrink-0 p-4 border-t border-gray-200 dark:border-gray-800">
          <button
            onClick={() => navigate(`/result/${scriptId}`)}
            className="w-full flex items-center justify-center gap-2 py-2.5 text-sm bg-gradient-to-r from-blue-600 to-purple-600 text-white rounded-lg hover:opacity-90 transition-opacity"
          >
            <ExternalLink className="w-4 h-4" />
            查看完整结果（含相似度检测）
          </button>
        </div>
      )}
    </div>
  )
}
EOF
```

- [ ] **Step 6: 运行测试**

```bash
cd /data/code/content_creator_imm/frontend && npm test 2>&1 | tail -5
# 预期: 14 passed
```

- [ ] **Step 7: Commit**

```bash
cd /data/code/content_creator_imm
git add frontend/src/components/create/
git commit -m "feat: 实现 Dashboard 子组件（MessageList/ChatInput/LoadingState/OutlineEditor/ScriptEditor）"
```

---

## Task 9: Dashboard 页面（主创作区）

**Files:**
- Modify: `frontend/src/pages/Dashboard.tsx`

- [ ] **Step 1: 实现完整 Dashboard**

```bash
cat > /data/code/content_creator_imm/frontend/src/pages/Dashboard.tsx << 'EOF'
import { useRef, useState, useEffect, useCallback, useReducer } from 'react'
import { ArrowUp, Sparkles } from 'lucide-react'
import { toast } from 'sonner'
import { Sidebar } from '../components/Sidebar'
import { MessageList, type ChatMsg } from '../components/create/MessageList'
import { ChatInput } from '../components/create/ChatInput'
import { LoadingState } from '../components/create/LoadingState'
import { OutlineEditor } from '../components/create/OutlineEditor'
import { ScriptEditor } from '../components/create/ScriptEditor'
import { sendMessage, resetSession } from '../api/chat'
import { getConversation, type Conversation } from '../api/conversations'
import { parseSSELine } from '../lib/sse'

type Stage = 'idle' | 'analyzing' | 'awaiting' | 'writing' | 'complete'

interface DashState {
  stage: Stage
  messages: ChatMsg[]
  outlineText: string
  scriptText: string
  scriptId: number | null
  sending: boolean
}

type Action =
  | { type: 'RESET' }
  | { type: 'SEND'; text: string }
  | { type: 'ADD_MSG'; msg: ChatMsg }
  | { type: 'APPEND_TOKEN'; content: string }
  | { type: 'SET_STAGE'; stage: Stage }
  | { type: 'SET_OUTLINE'; text: string }
  | { type: 'SET_SCRIPT'; text: string; scriptId: number | null }
  | { type: 'STREAM_DONE' }
  | { type: 'RESTORE'; messages: ChatMsg[]; stage: Stage }
  | { type: 'UPDATE_OUTLINE'; text: string }

function reducer(state: DashState, action: Action): DashState {
  switch (action.type) {
    case 'RESET':
      return { stage: 'idle', messages: [], outlineText: '', scriptText: '', scriptId: null, sending: false }
    case 'SEND':
      return {
        ...state,
        stage: 'analyzing',
        sending: true,
        messages: [...state.messages, { id: `${Date.now()}`, type: 'user', content: action.text }],
      }
    case 'ADD_MSG':
      return { ...state, messages: [...state.messages, action.msg] }
    case 'APPEND_TOKEN': {
      const msgs = [...state.messages]
      const last = msgs[msgs.length - 1]
      if (last?.streaming) {
        msgs[msgs.length - 1] = { ...last, content: (last.content ?? '') + action.content }
      } else {
        msgs.push({ id: `${Date.now()}-t`, type: 'ai', content: action.content, streaming: true })
      }
      return { ...state, messages: msgs }
    }
    case 'SET_STAGE':
      return { ...state, stage: action.stage }
    case 'SET_OUTLINE':
      return { ...state, stage: 'awaiting', outlineText: action.text }
    case 'SET_SCRIPT':
      return { ...state, stage: 'complete', scriptText: action.text, scriptId: action.scriptId, sending: false }
    case 'STREAM_DONE': {
      const msgs = state.messages.map((m) => (m.streaming ? { ...m, streaming: false } : m))
      return { ...state, messages: msgs, sending: false }
    }
    case 'RESTORE':
      return { ...state, messages: action.messages, stage: action.stage, sending: false }
    case 'UPDATE_OUTLINE':
      return { ...state, outlineText: action.text }
    default:
      return state
  }
}

export function Dashboard() {
  const [state, dispatch] = useReducer(reducer, {
    stage: 'idle', messages: [], outlineText: '', scriptText: '', scriptId: null, sending: false,
  })
  const [initialInput, setInitialInput] = useState('')
  const [activeConvId, setActiveConvId] = useState<number | undefined>()
  const [refreshTrigger, setRefreshTrigger] = useState(0)
  const messagesEndRef = useRef<HTMLDivElement>(null)
  const streamingTextRef = useRef('')  // accumulate token content for ScriptEditor

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [state.messages])

  const runSSE = useCallback(async (message: string) => {
    streamingTextRef.current = ''
    try {
      const res = await sendMessage(message)
      if (!res.ok) {
        const err = await res.json().catch(() => ({ error: '请求失败' }))
        dispatch({ type: 'ADD_MSG', msg: { id: `${Date.now()}`, type: 'error', content: (err as { error?: string }).error ?? '请求失败' } })
        dispatch({ type: 'STREAM_DONE' })
        return
      }

      const reader = res.body!.getReader()
      const decoder = new TextDecoder()
      let buffer = ''

      while (true) {
        const { done, value } = await reader.read()
        if (done) break
        buffer += decoder.decode(value, { stream: true })
        const lines = buffer.split('\n')
        buffer = lines.pop() ?? ''
        for (const line of lines) {
          const event = parseSSELine(line)
          if (!event) continue
          switch (event.type) {
            case 'token':
              streamingTextRef.current += event.content
              dispatch({ type: 'APPEND_TOKEN', content: event.content })
              break
            case 'step':
              dispatch({ type: 'ADD_MSG', msg: { id: `${Date.now()}-s`, type: 'step', content: `Step ${event.step}：${event.name}` } })
              break
            case 'info':
              dispatch({ type: 'ADD_MSG', msg: { id: `${Date.now()}-i`, type: 'info', content: event.content } })
              break
            case 'outline':
              dispatch({ type: 'SET_OUTLINE', text: JSON.stringify(event.data, null, 2) })
              break
            case 'action':
              dispatch({ type: 'ADD_MSG', msg: { id: `${Date.now()}-a`, type: 'action', options: event.options } })
              break
            case 'similarity':
              dispatch({ type: 'ADD_MSG', msg: { id: `${Date.now()}-sim`, type: 'similarity', data: event.data } })
              break
            case 'complete':
              dispatch({ type: 'SET_SCRIPT', text: streamingTextRef.current, scriptId: event.scriptId })
              setRefreshTrigger((n) => n + 1)
              break
            case 'error':
              dispatch({ type: 'ADD_MSG', msg: { id: `${Date.now()}-e`, type: 'error', content: event.message } })
              dispatch({ type: 'SET_STAGE', stage: 'idle' })
              break
          }
        }
      }
    } catch (e) {
      const msg = e instanceof Error ? e.message : '连接失败'
      dispatch({ type: 'ADD_MSG', msg: { id: `${Date.now()}-e`, type: 'error', content: msg } })
    } finally {
      dispatch({ type: 'STREAM_DONE' })
      setRefreshTrigger((n) => n + 1)
    }
  }, [])

  const handleSend = useCallback(async (text: string) => {
    if (state.sending) return
    dispatch({ type: 'SEND', text })
    await runSSE(text)
  }, [state.sending, runSSE])

  const handleInitialCreate = () => {
    if (!initialInput.trim() || initialInput.length < 10) return
    handleSend(initialInput)
    setInitialInput('')
  }

  const handleNewChat = async () => {
    try {
      await resetSession()
      dispatch({ type: 'RESET' })
      setActiveConvId(undefined)
      setRefreshTrigger((n) => n + 1)
      toast.success('新会话已开始')
    } catch { toast.error('重置失败') }
  }

  const handleSelectConversation = async (conv: Conversation) => {
    try {
      const data = await getConversation(conv.id)
      const stored = JSON.parse(data.messages || '[]') as Array<{ role: string; type: string; content?: string; data?: unknown; options?: string[] }>
      const msgs: ChatMsg[] = stored.map((m, i) => ({
        id: `restore-${i}`,
        type: m.role === 'user' ? 'user' : (m.type === 'action' ? 'action' : m.type === 'error' ? 'error' : m.type === 'step' ? 'step' : m.type === 'info' ? 'info' : 'ai') as ChatMsg['type'],
        content: m.content,
        options: m.options,
        data: m.data,
      }))
      const stage: Stage = conv.state === 1 ? 'complete' : 'idle'
      dispatch({ type: 'RESTORE', messages: msgs, stage })
      setActiveConvId(conv.id)
    } catch { toast.error('加载会话失败') }
  }

  const handleSelectScript = (content: string, _title: string) => {
    dispatch({ type: 'SET_SCRIPT', text: content, scriptId: null })
  }

  const handleAction = useCallback((option: string) => {
    handleSend(option)
    dispatch({ type: 'SET_STAGE', stage: 'writing' })
  }, [handleSend])

  // 初始态：侧边栏 + 居中输入
  if (state.stage === 'idle') {
    return (
      <div className="h-screen flex overflow-hidden">
        <Sidebar
          onNewChat={handleNewChat}
          onSelectConversation={handleSelectConversation}
          onSelectScript={handleSelectScript}
          activeConvId={activeConvId}
          refreshTrigger={refreshTrigger}
        />
        <div className="flex-1 overflow-y-auto bg-white dark:bg-gray-950">
          <div className="max-w-3xl mx-auto px-4 pt-16">
            <div className="text-center mb-8">
              <h1 className="text-4xl sm:text-5xl font-medium mb-3 text-gray-900 dark:text-gray-100">
                Hi，今天想创作什么爆款文案？
              </h1>
              <p className="text-lg text-gray-500 dark:text-gray-400 mb-4">粘贴你的参考口播稿，AI 会学习风格并为你创作</p>
              <div className="inline-flex items-center gap-2 px-4 py-2 bg-gradient-to-r from-blue-50 to-purple-50 dark:from-blue-950 dark:to-purple-950 rounded-full border border-blue-200 dark:border-blue-800">
                <Sparkles className="w-4 h-4 text-blue-600 dark:text-blue-400" />
                <span className="text-sm font-medium text-blue-900 dark:text-blue-300">越用越懂你</span>
              </div>
            </div>
            <div className="relative mb-4">
              <textarea
                value={initialInput}
                onChange={(e) => setInitialInput(e.target.value)}
                placeholder="粘贴你喜欢的爆款口播稿..."
                className="w-full h-[200px] p-6 pr-20 pb-16 text-base border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900 text-gray-900 dark:text-gray-100 rounded-2xl resize-none focus:outline-none focus:ring-2 focus:ring-blue-500 transition-all"
              />
              <button
                onClick={handleInitialCreate}
                disabled={!initialInput.trim() || initialInput.length < 10}
                className="absolute bottom-4 right-4 w-10 h-10 bg-gradient-to-br from-blue-500 to-purple-600 rounded-lg flex items-center justify-center shadow-lg hover:scale-105 hover:shadow-xl transition-all disabled:opacity-40 disabled:cursor-not-allowed disabled:hover:scale-100"
              >
                <ArrowUp className="w-5 h-5 text-white" strokeWidth={2.5} />
              </button>
            </div>
            <p className="text-sm text-gray-400 px-2">{initialInput.length} 字</p>
            <div className="text-center pb-16 mt-8 text-sm text-gray-400">
              💡 提示：提供参考文案可以帮助 AI 更好地理解你想要的风格
            </div>
          </div>
        </div>
      </div>
    )
  }

  // 创作态：左侧聊天 + 右侧预览（无侧边栏）
  return (
    <div className="h-screen flex overflow-hidden">
      {/* 左侧聊天区 2/5 */}
      <div className="w-full md:w-2/5 border-r border-gray-200 dark:border-gray-800 flex flex-col bg-white dark:bg-gray-950">
        <MessageList
          messages={state.messages}
          onAction={handleAction}
          disabled={state.sending}
        />
        <div ref={messagesEndRef} />
        <ChatInput
          onSend={handleSend}
          placeholder="随时告诉我你的想法..."
          disabled={state.sending}
        />
      </div>

      {/* 右侧预览区 3/5（桌面端） */}
      <div className="hidden md:flex md:w-3/5 h-full">
        {(state.stage === 'analyzing' || state.stage === 'writing') && (
          <LoadingState message={state.stage === 'analyzing' ? '正在分析并生成大纲...' : '正在创作爆款口播稿...'} />
        )}
        {state.stage === 'awaiting' && (
          <OutlineEditor
            content={state.outlineText}
            onChange={(text) => dispatch({ type: 'UPDATE_OUTLINE', text })}
          />
        )}
        {state.stage === 'complete' && (
          <ScriptEditor
            content={state.scriptText}
            scriptId={state.scriptId}
          />
        )}
      </div>
    </div>
  )
}
EOF
```

- [ ] **Step 2: 运行测试**

```bash
cd /data/code/content_creator_imm/frontend && npm test 2>&1 | tail -5
# 预期: 14 passed
```

- [ ] **Step 3: Commit**

```bash
cd /data/code/content_creator_imm
git add frontend/src/pages/Dashboard.tsx
git commit -m "feat: 实现 Dashboard 主创作区（SSE 聊天 + 大纲/稿件预览）"
```

---

## Task 10: Result 页 + History 页

**Files:**
- Modify: `frontend/src/pages/Result.tsx`
- Modify: `frontend/src/pages/History.tsx`

- [ ] **Step 1: 实现 Result 页**

```bash
cat > /data/code/content_creator_imm/frontend/src/pages/Result.tsx << 'EOF'
import { useState, useEffect } from 'react'
import { useParams, useNavigate } from 'react-router'
import { ArrowLeft, Copy, Check, ThumbsUp, ThumbsDown, Feather } from 'lucide-react'
import { Card } from '../components/ui/card'
import { Button } from '../components/ui/button'
import { getScript } from '../api/scripts'
import { toast } from 'sonner'

export function Result() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const [content, setContent] = useState('')
  const [similarity, setSimilarity] = useState<number | null>(null)
  const [title, setTitle] = useState('')
  const [copied, setCopied] = useState(false)
  const [feedback, setFeedback] = useState<'like' | 'dislike' | null>(
    () => (localStorage.getItem(`feedback_${id}`) as 'like' | 'dislike' | null) ?? null
  )
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    if (!id) return
    getScript(Number(id))
      .then((data) => {
        setContent(data.content)
        setSimilarity(data.script.similarity_score)
        setTitle(data.script.title)
      })
      .catch(() => {
        toast.error('加载稿件失败')
        navigate('/dashboard')
      })
      .finally(() => setLoading(false))
  }, [id, navigate])

  const handleCopy = () => {
    navigator.clipboard.writeText(content)
    setCopied(true)
    toast.success('已复制到剪贴板')
    setTimeout(() => setCopied(false), 2000)
  }

  const handleFeedback = (type: 'like' | 'dislike') => {
    const next = feedback === type ? null : type
    setFeedback(next)
    if (next) localStorage.setItem(`feedback_${id}`, next)
    else localStorage.removeItem(`feedback_${id}`)
    toast.success(next === 'like' ? '感谢反馈！' : next === 'dislike' ? '我们会持续改进' : '已取消反馈')
  }

  if (loading) return <div className="p-8 text-center text-gray-500">加载中...</div>

  const passed = similarity !== null && similarity < 30
  const statusColor = passed ? 'text-green-600' : 'text-red-600'
  const statusBg = passed ? 'bg-green-50 dark:bg-green-950' : 'bg-red-50 dark:bg-red-950'
  const statusBorder = passed ? 'border-green-200 dark:border-green-800' : 'border-red-200 dark:border-red-800'

  return (
    <div className="max-w-4xl mx-auto">
      <Button variant="ghost" onClick={() => navigate(-1)} className="mb-6 -ml-2">
        <ArrowLeft className="w-4 h-4 mr-2" />返回
      </Button>

      <h1 className="text-2xl font-semibold mb-6 text-gray-900 dark:text-gray-100">{title}</h1>

      <Card className={`p-6 sm:p-8 shadow-lg border-2 ${statusBorder} ${statusBg}`}>
        {/* 相似度 */}
        {similarity !== null && (
          <div className="mb-6">
            <div className="flex items-center justify-between mb-2">
              <span className="text-sm text-gray-600 dark:text-gray-400">相似度</span>
              <span className={`text-3xl font-semibold ${statusColor}`}>{similarity}%</span>
            </div>
            <div className="w-full h-3 bg-gray-200 dark:bg-gray-700 rounded-full overflow-hidden">
              <div
                className={`h-full rounded-full ${passed ? 'bg-green-500' : 'bg-red-500'} transition-all`}
                style={{ width: `${Math.min(similarity, 100)}%` }}
              />
            </div>
            <div className={`mt-2 text-sm ${statusColor}`}>
              {passed ? '✅ 原创度达标，可直接使用' : '❌ 相似度较高，建议修改'}
            </div>
          </div>
        )}

        {/* 稿件内容 */}
        <div className="mb-6">
          <h3 className="text-sm font-medium text-gray-700 dark:text-gray-300 mb-3">稿件内容</h3>
          <div className="bg-white dark:bg-gray-900 rounded-lg p-4 border border-gray-200 dark:border-gray-700 max-h-[300px] overflow-y-auto">
            <p className="text-gray-800 dark:text-gray-200 whitespace-pre-wrap text-sm leading-relaxed">{content}</p>
          </div>
        </div>

        {/* 复制按钮 */}
        <Button onClick={handleCopy} className="w-full mb-6 h-12 bg-blue-600 hover:bg-blue-700 text-white">
          {copied ? <><Check className="w-5 h-5 mr-2" />已复制</> : <><Copy className="w-5 h-5 mr-2" />📋 一键复制</>}
        </Button>

        {/* 反馈 */}
        <div className="border-t border-gray-300 dark:border-gray-700 pt-6 mb-6">
          <h3 className="text-sm font-medium text-gray-700 dark:text-gray-300 mb-3">反馈</h3>
          <div className="flex gap-3">
            <Button
              variant={feedback === 'like' ? 'default' : 'outline'}
              onClick={() => handleFeedback('like')}
              className={`flex-1 h-11 ${feedback === 'like' ? 'bg-green-600 hover:bg-green-700 text-white' : ''}`}
            >
              <ThumbsUp className="w-4 h-4 mr-2" />👍 喜欢
            </Button>
            <Button
              variant={feedback === 'dislike' ? 'default' : 'outline'}
              onClick={() => handleFeedback('dislike')}
              className={`flex-1 h-11 ${feedback === 'dislike' ? 'bg-red-600 hover:bg-red-700 text-white' : ''}`}
            >
              <ThumbsDown className="w-4 h-4 mr-2" />👎 不喜欢
            </Button>
          </div>
        </div>

        {/* 继续创作 */}
        <Button
          onClick={() => navigate('/dashboard')}
          className="w-full h-12 bg-gradient-to-r from-blue-600 to-purple-600 hover:from-blue-700 hover:to-purple-700 text-white"
        >
          <Feather className="w-5 h-5 mr-2" />继续创作
        </Button>
      </Card>
    </div>
  )
}
EOF
```

- [ ] **Step 2: 实现 History 页**

```bash
cat > /data/code/content_creator_imm/frontend/src/pages/History.tsx << 'EOF'
import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router'
import { Card } from '../components/ui/card'
import { Button } from '../components/ui/button'
import { Clock, Eye, FileText } from 'lucide-react'
import { listConversations, type Conversation } from '../api/conversations'
import { toast } from 'sonner'

export function History() {
  const [conversations, setConversations] = useState<Conversation[]>([])
  const [loading, setLoading] = useState(true)
  const navigate = useNavigate()

  useEffect(() => {
    listConversations()
      .then((data) => setConversations(data.conversations ?? []))
      .catch(() => toast.error('加载历史记录失败'))
      .finally(() => setLoading(false))
  }, [])

  if (loading) return <div className="p-8 text-center text-gray-500">加载中...</div>

  if (conversations.length === 0) {
    return (
      <div className="max-w-4xl mx-auto text-center py-16">
        <div className="inline-flex items-center justify-center w-20 h-20 bg-gray-100 dark:bg-gray-800 rounded-full mb-4">
          <FileText className="w-10 h-10 text-gray-400" />
        </div>
        <h2 className="text-2xl mb-2 text-gray-600 dark:text-gray-400">暂无历史记录</h2>
        <p className="text-gray-500 mb-6">开始创作以查看历史记录</p>
        <Button
          onClick={() => navigate('/dashboard')}
          className="bg-gradient-to-r from-blue-600 to-purple-600 text-white"
        >
          开始创作
        </Button>
      </div>
    )
  }

  return (
    <div className="max-w-4xl mx-auto">
      <div className="mb-6">
        <h1 className="text-2xl sm:text-3xl font-semibold mb-2 text-gray-900 dark:text-gray-100">历史记录</h1>
        <p className="text-gray-600 dark:text-gray-400">共 {conversations.length} 条会话记录</p>
      </div>

      <div className="space-y-4">
        {conversations.map((conv) => (
          <Card
            key={conv.id}
            className="p-4 sm:p-6 hover:shadow-lg transition-shadow border border-gray-200 dark:border-gray-800 bg-white dark:bg-gray-900"
          >
            <div className="flex flex-col sm:flex-row sm:items-center gap-4">
              <div className="flex-1 min-w-0">
                <h3 className="text-base font-medium text-gray-900 dark:text-gray-100 truncate mb-1">
                  {conv.title || '未命名会话'}
                </h3>
                <div className="flex items-center gap-3 text-xs text-gray-500 dark:text-gray-400">
                  <span className="flex items-center gap-1">
                    <Clock className="w-3 h-3" />
                    {new Date(conv.created_at).toLocaleString('zh-CN')}
                  </span>
                  <span className={`px-2 py-0.5 rounded text-xs ${
                    conv.state === 1
                      ? 'bg-green-100 dark:bg-green-900 text-green-700 dark:text-green-300'
                      : 'bg-yellow-100 dark:bg-yellow-900 text-yellow-700 dark:text-yellow-300'
                  }`}>
                    {conv.state === 1 ? '已完成' : '进行中'}
                  </span>
                </div>
              </div>
              <div className="flex gap-2">
                {conv.script_id ? (
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => navigate(`/result/${conv.script_id}`)}
                  >
                    <Eye className="w-4 h-4 mr-1" />查看稿件
                  </Button>
                ) : (
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => navigate('/dashboard')}
                  >
                    继续创作
                  </Button>
                )}
              </div>
            </div>
          </Card>
        ))}
      </div>
    </div>
  )
}
EOF
```

- [ ] **Step 3: 运行测试**

```bash
cd /data/code/content_creator_imm/frontend && npm test 2>&1 | tail -5
# 预期: 14 passed
```

- [ ] **Step 4: Commit**

```bash
cd /data/code/content_creator_imm
git add frontend/src/pages/Result.tsx frontend/src/pages/History.tsx
git commit -m "feat: 实现 Result 详情页和 History 历史列表页"
```

---

## Task 11: 构建验证 + 端对端测试

**Goal:** 确认生产构建可用，nginx 能正确服务前端。

- [ ] **Step 1: 运行完整测试套件**

```bash
cd /data/code/content_creator_imm/frontend && npm test 2>&1
# 预期: 14 passed, 0 failed
```

- [ ] **Step 2: TypeScript 类型检查**

```bash
# tsc -b 使用 tsconfig.app.json 中的 noEmit: true，不需要额外 --noEmit 标志
cd /data/code/content_creator_imm/frontend && npx tsc -b 2>&1
# 预期: 无报错输出
```

如有报错，逐个修复后再继续。

- [ ] **Step 3: 生产构建**

```bash
cd /data/code/content_creator_imm && ./build.sh 2>&1 | tail -30
# 预期: 构建成功，frontend/dist/ 包含 index.html 和 assets/
ls frontend/dist/
```

- [ ] **Step 4: 验证 nginx 可服务**

```bash
# 确认 nginx 配置正常并 reload
nginx -t && nginx -s reload

# 检查前端首页可访问
curl -s -o /dev/null -w "HTTP %{http_code}\n" http://localhost/creator/
# 预期: HTTP 200

# 检查 API 联通
curl -s http://localhost/creator/api/auth/login \
  -X POST \
  -H "Content-Type: application/json" \
  -d '{"email":"test2@test.com","password":"Test1234"}' \
  | python3 -c "import sys,json; d=json.load(sys.stdin); print('OK' if 'token' in d else 'FAIL: ' + str(d))"
# 预期: OK
```

- [ ] **Step 5: 启动后端服务（若未运行）**

```bash
cd /data/code/content_creator_imm
./manage.sh status  # 检查后端是否运行
# 若未运行: ./manage.sh start
```

- [ ] **Step 6: 最终 commit**

```bash
cd /data/code/content_creator_imm
git add -A
git commit -m "feat: 完成 React 前端迁移（Landing/Auth/Dashboard/Result/History + 暗色模式 + 移动端）"
```

---

## 附录：常见问题

**Q: Tailwind 样式不生效**
- 确认 `vite.config.ts` 中 `tailwindcss()` 插件已加载
- 确认 `styles/index.css` 已在 `main.tsx` 中 import

**Q: shadcn 组件报 import 错误**
- 检查 `components/ui/*.tsx` 中 utils import 路径（应为 `../../lib/utils`）
- 确认对应 Radix UI 包已在 package.json 中声明

**Q: SSE 在开发环境收不到数据**
- 检查 `vite.config.ts` proxy 配置：`/creator/api` → `http://localhost:3004`，rewrite 去掉前缀
- 确认后端服务运行：`./manage.sh status`

**Q: 生产环境路由 404**
- 确认 nginx `try_files $uri $uri/ /creator/index.html` 已配置
- 确认 React Router `basename="/creator"` 设置正确

**Q: TypeScript 严格模式报错**
- `noUnusedLocals/noUnusedParameters`: 删除未用变量或加 `_` 前缀
- 类型断言优先用 `as Type` 而非 `!`
