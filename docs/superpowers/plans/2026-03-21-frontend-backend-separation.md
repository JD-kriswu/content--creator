# Frontend-Backend Separation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Refactor content_creator_imm from Go-embedded frontend to monorepo with `backend/` (Go API) and `frontend/` (Vue 3 + Vite + Element Plus).

**Architecture:** Go code moves to `backend/`, embed logic removed, CORS updated. Vue 3 SPA in `frontend/` calls `/api/*` via Vite proxy in dev, nginx reverse proxy in prod.

**Tech Stack:** Go 1.22 + Gin, Vue 3 + Vite 5 + TypeScript, Element Plus 2, Pinia 2, Vue Router 4, Axios 1

**Spec:** `docs/superpowers/specs/2026-03-21-frontend-backend-separation-design.md`

---

## File Map

### New files (frontend/)
- `frontend/package.json` — dependencies
- `frontend/vite.config.ts` — dev server + proxy
- `frontend/tsconfig.json` — TypeScript config
- `frontend/index.html` — SPA entry
- `frontend/src/main.ts` — app bootstrap
- `frontend/src/App.vue` — root with router-view
- `frontend/src/router/index.ts` — routes + auth guard
- `frontend/src/stores/user.ts` — user/token state (Pinia)
- `frontend/src/stores/chat.ts` — chat messages + SSE (Pinia)
- `frontend/src/api/request.ts` — axios instance
- `frontend/src/api/auth.ts` — login/register
- `frontend/src/api/chat.ts` — session/reset; SSE sendMessage
- `frontend/src/api/scripts.ts` — list/get scripts
- `frontend/src/api/user.ts` — profile/style
- `frontend/src/views/Login.vue` — login + register tabs
- `frontend/src/views/Home.vue` — layout shell (header + sidebar + slot)
- `frontend/src/components/ChatPanel.vue` — messages + input + SSE rendering
- `frontend/src/components/ScriptList.vue` — sidebar script list

### Modified files (backend/)
- `backend/main.go` — remove embed, add CORS origins config
- All existing Go files — move from root to `backend/` (no code changes)

### New root files
- `build.sh` — build frontend + backend binary
- `manage.sh` — start/stop/status/logs/add-user/list-users
- `CLAUDE.md` — project docs + all commands

---

## Task 1: Move Go code to backend/

**Files:**
- Create: `backend/` (directory)
- Move: all Go source files and dirs into `backend/`
- Modify: `backend/main.go` — remove embed logic + update CORS

- [ ] **Step 1: Create backend/ and move files**

```bash
cd /data/code/content_creator_imm
mkdir -p backend
# Move Go source
mv main.go go.mod go.sum backend/
mv config internal middleware backend/
# Move runtime data and config
mv config.json config.example.json backend/ 2>/dev/null || true
mv db_init.sql backend/
# data/ stays at root for now, will be referenced from backend/
mv data backend/ 2>/dev/null || true
```

- [ ] **Step 2: Verify go.mod module name is correct**

```bash
head -3 /data/code/content_creator_imm/backend/go.mod
```
Expected: `module content-creator-imm`

- [ ] **Step 3: Rewrite backend/main.go — remove embed logic, update CORS**

Replace the full content of `backend/main.go`:

```go
package main

import (
	"log"
	"net/http"
	"strings"

	"content-creator-imm/config"
	"content-creator-imm/internal/db"
	"content-creator-imm/internal/handler"
	"content-creator-imm/middleware"

	"github.com/gin-gonic/gin"
)

func main() {
	config.Load()

	if err := db.Init(); err != nil {
		log.Fatalf("database init failed: %v", err)
	}
	log.Println("database connected")

	if config.C.AnthropicKey == "" {
		log.Println("⚠️  ANTHROPIC_API_KEY 未配置！请在 config.json 中设置 anthropic_api_key")
	}

	r := gin.Default()

	// CORS — allow configured origins (dev: localhost:5173, prod: same-origin via nginx)
	allowedOrigins := strings.Split(config.C.CORSOrigins, ",")
	r.Use(func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		for _, allowed := range allowedOrigins {
			if strings.TrimSpace(allowed) == origin || strings.TrimSpace(allowed) == "*" {
				c.Header("Access-Control-Allow-Origin", origin)
				break
			}
		}
		c.Header("Access-Control-Allow-Headers", "Authorization,Content-Type")
		c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	})

	base := strings.TrimRight(config.C.BasePath, "/")

	// Auth routes
	auth := r.Group(base + "/api/auth")
	{
		auth.POST("/register", handler.Register)
		auth.POST("/login", handler.Login)
	}

	// Protected routes
	api := r.Group(base+"/api", middleware.Auth())
	{
		api.GET("/user/profile", handler.GetProfile)
		api.PUT("/user/style", handler.UpdateStyle)

		api.GET("/chat/session", handler.GetSession)
		api.POST("/chat/reset", handler.ResetSession)
		api.POST("/chat/message", handler.SendMessage)

		api.GET("/scripts", handler.GetScripts)
		api.GET("/scripts/:id", handler.GetScript)
	}

	addr := ":" + config.C.Port
	log.Printf("server starting on %s (base: %q)", addr, base)
	if err := r.Run(addr); err != nil {
		log.Fatal(err)
	}
}
```

- [ ] **Step 4: Add CORSOrigins field to config**

Edit `backend/config/config.go` — add `CORSOrigins` to `Config` struct and `Load()`:

In the struct, add:
```go
CORSOrigins  string `json:"cors_origins"`  // comma-separated, e.g. "http://localhost:5173"
```

In `C = Config{...}` defaults block, add:
```go
CORSOrigins: getEnv("CORS_ORIGINS", "http://localhost:5173"),
```

In the env override section, add:
```go
if v := os.Getenv("CORS_ORIGINS"); v != "" { C.CORSOrigins = v }
```

- [ ] **Step 5: Verify backend compiles**

```bash
cd /data/code/content_creator_imm/backend && go build ./...
```
Expected: no errors, produces no output (or outputs binary name)

- [ ] **Step 6: Commit**

```bash
cd /data/code/content_creator_imm
git add backend/
git add -u  # stage moved/deleted files
git commit -m "refactor: move Go code to backend/, remove embed logic"
```

---

## Task 2: Create frontend/ Vue 3 + Vite scaffold

**Files:**
- Create: `frontend/package.json`
- Create: `frontend/vite.config.ts`
- Create: `frontend/tsconfig.json`
- Create: `frontend/index.html`
- Create: `frontend/src/main.ts`
- Create: `frontend/src/App.vue`

- [ ] **Step 1: Create frontend/package.json**

```json
{
  "name": "content-creator-imm-frontend",
  "version": "1.0.0",
  "private": true,
  "scripts": {
    "dev": "vite",
    "build": "vue-tsc && vite build",
    "preview": "vite preview"
  },
  "dependencies": {
    "axios": "^1.7.0",
    "element-plus": "^2.8.0",
    "pinia": "^2.2.0",
    "vue": "^3.5.0",
    "vue-router": "^4.4.0"
  },
  "devDependencies": {
    "@vitejs/plugin-vue": "^5.1.0",
    "typescript": "^5.5.0",
    "vite": "^5.4.0",
    "vue-tsc": "^2.1.0"
  }
}
```

- [ ] **Step 2: Create frontend/vite.config.ts**

```typescript
import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { resolve } from 'path'

export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: { '@': resolve(__dirname, 'src') }
  },
  server: {
    port: 5173,
    proxy: {
      '/api': {
        target: 'http://localhost:3004',
        changeOrigin: true
      }
    }
  }
})
```

- [ ] **Step 3: Create frontend/tsconfig.json**

```json
{
  "compilerOptions": {
    "target": "ES2020",
    "useDefineForClassFields": true,
    "module": "ESNext",
    "lib": ["ES2020", "DOM", "DOM.Iterable"],
    "skipLibCheck": true,
    "moduleResolution": "bundler",
    "allowImportingTsExtensions": true,
    "resolveJsonModule": true,
    "isolatedModules": true,
    "noEmit": true,
    "jsx": "preserve",
    "strict": true,
    "noUnusedLocals": true,
    "noUnusedParameters": true,
    "noFallthroughCasesInSwitch": true,
    "baseUrl": ".",
    "paths": { "@/*": ["src/*"] }
  },
  "include": ["src/**/*.ts", "src/**/*.d.ts", "src/**/*.tsx", "src/**/*.vue"],
  "references": [{ "path": "./tsconfig.node.json" }]
}
```

- [ ] **Step 4: Create frontend/tsconfig.node.json**

```json
{
  "compilerOptions": {
    "composite": true,
    "skipLibCheck": true,
    "module": "ESNext",
    "moduleResolution": "bundler",
    "allowSyntheticDefaultImports": true
  },
  "include": ["vite.config.ts"]
}
```

- [ ] **Step 5: Create frontend/index.html**

```html
<!DOCTYPE html>
<html lang="zh-CN">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>口播稿助手</title>
  </head>
  <body>
    <div id="app"></div>
    <script type="module" src="/src/main.ts"></script>
  </body>
</html>
```

- [ ] **Step 6: Create frontend/src/main.ts**

```typescript
import { createApp } from 'vue'
import { createPinia } from 'pinia'
import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css'
import zhCn from 'element-plus/es/locale/lang/zh-cn'
import App from './App.vue'
import router from './router'

const app = createApp(App)
app.use(createPinia())
app.use(router)
app.use(ElementPlus, { locale: zhCn })
app.mount('#app')
```

- [ ] **Step 7: Create frontend/src/App.vue**

```vue
<template>
  <router-view />
</template>
```

- [ ] **Step 8: Install dependencies**

```bash
cd /data/code/content_creator_imm/frontend && npm install
```
Expected: node_modules created, no errors

- [ ] **Step 9: Commit scaffold**

```bash
cd /data/code/content_creator_imm
git add frontend/
git commit -m "feat: add Vue 3 + Vite frontend scaffold"
```

---

## Task 3: API layer

**Files:**
- Create: `frontend/src/api/request.ts`
- Create: `frontend/src/api/auth.ts`
- Create: `frontend/src/api/chat.ts`
- Create: `frontend/src/api/scripts.ts`
- Create: `frontend/src/api/user.ts`

- [ ] **Step 1: Create frontend/src/api/request.ts**

```typescript
import axios from 'axios'
import router from '@/router'

const request = axios.create({
  baseURL: '/api',
  timeout: 30000
})

request.interceptors.request.use(config => {
  const token = localStorage.getItem('token')
  if (token) config.headers.Authorization = `Bearer ${token}`
  return config
})

request.interceptors.response.use(
  res => res,
  err => {
    if (err.response?.status === 401) {
      localStorage.removeItem('token')
      localStorage.removeItem('user')
      router.push('/login')
    }
    return Promise.reject(err)
  }
)

export default request
```

- [ ] **Step 2: Create frontend/src/api/auth.ts**

```typescript
import request from './request'

export interface LoginResp {
  token: string
  user: { id: number; username: string; email: string; role: string }
}

export function login(email: string, password: string) {
  return request.post<LoginResp>('/auth/login', { email, password })
}

export function register(username: string, email: string, password: string) {
  return request.post('/auth/register', { username, email, password })
}
```

- [ ] **Step 3: Create frontend/src/api/chat.ts**

```typescript
import request from './request'

export function getSession() {
  return request.get<{ session_id: string; state: string }>('/chat/session')
}

export function resetSession() {
  return request.post('/chat/reset')
}

// Returns a fetch Response with ReadableStream for SSE
export async function sendMessage(message: string): Promise<Response> {
  const token = localStorage.getItem('token') ?? ''
  return fetch('/api/chat/message', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
    body: JSON.stringify({ message })
  })
}
```

- [ ] **Step 4: Create frontend/src/api/scripts.ts**

```typescript
import request from './request'

export interface Script {
  id: number
  title: string
  source_url: string
  platform: string
  similarity_score: number
  viral_score: number
  created_at: string
}

export function getScripts() {
  return request.get<{ scripts: Script[]; total: number }>('/scripts')
}

export function getScript(id: number) {
  return request.get<{ script: Script; content: string }>(`/scripts/${id}`)
}
```

- [ ] **Step 5: Create frontend/src/api/user.ts**

```typescript
import request from './request'

export interface UserStyle {
  language_style: string
  emotion_tone: string
  opening_style: string
  closing_style: string
  catchphrases: string
}

export function getProfile() {
  return request.get<{ style: UserStyle | null }>('/user/profile')
}

export function updateStyle(style: UserStyle) {
  return request.put('/user/style', style)
}
```

- [ ] **Step 6: Commit**

```bash
cd /data/code/content_creator_imm
git add frontend/src/api/
git commit -m "feat: add frontend API layer (axios + auth/chat/scripts/user)"
```

---

## Task 4: Pinia stores

**Files:**
- Create: `frontend/src/stores/user.ts`
- Create: `frontend/src/stores/chat.ts`

- [ ] **Step 1: Create frontend/src/stores/user.ts**

```typescript
import { defineStore } from 'pinia'
import { ref } from 'vue'
import { login as apiLogin, register as apiRegister } from '@/api/auth'

interface User {
  id: number
  username: string
  email: string
  role: string
}

export const useUserStore = defineStore('user', () => {
  const token = ref<string | null>(localStorage.getItem('token'))
  const user = ref<User | null>(JSON.parse(localStorage.getItem('user') ?? 'null'))

  async function login(email: string, password: string) {
    const { data } = await apiLogin(email, password)
    token.value = data.token
    user.value = data.user
    localStorage.setItem('token', data.token)
    localStorage.setItem('user', JSON.stringify(data.user))
  }

  async function register(username: string, email: string, password: string) {
    await apiRegister(username, email, password)
  }

  function logout() {
    token.value = null
    user.value = null
    localStorage.removeItem('token')
    localStorage.removeItem('user')
  }

  const isLoggedIn = () => !!token.value

  return { token, user, login, register, logout, isLoggedIn }
})
```

- [ ] **Step 2: Create frontend/src/stores/chat.ts**

This store owns SSE message parsing. The backend sends:
`data: {"type":"token"|"step"|"info"|"outline"|"action"|"similarity"|"complete"|"error", ...}\n\n`

```typescript
import { defineStore } from 'pinia'
import { ref } from 'vue'
import { sendMessage as apiSendMessage, resetSession as apiResetSession } from '@/api/chat'

export type MsgRole = 'user' | 'assistant'

export interface ChatMessage {
  id: number
  role: MsgRole
  html: string      // rendered content
  rawText?: string  // accumulated token text (for streaming)
  streaming?: boolean
}

export interface OutlineData {
  outline?: Array<{ part: string; content: string; duration: string }>
  elements?: string[]
  estimated?: string
  strategy?: string
}

export interface SimilarityData {
  vocab: number; sentence: number; structure: number; viewpoint: number; total: number
}

export type SSEEvent =
  | { type: 'token'; content: string }
  | { type: 'step'; step: number; name: string }
  | { type: 'info'; content: string }
  | { type: 'outline'; data: OutlineData }
  | { type: 'action'; options: string[] }
  | { type: 'similarity'; data: SimilarityData }
  | { type: 'complete'; scriptId: number }
  | { type: 'error'; message: string }

export const useChatStore = defineStore('chat', () => {
  const messages = ref<ChatMessage[]>([])
  const sending = ref(false)
  let nextId = 1

  function addMessage(role: MsgRole, html: string, opts?: Partial<ChatMessage>): ChatMessage {
    const msg: ChatMessage = { id: nextId++, role, html, ...opts }
    messages.value.push(msg)
    return msg
  }

  function addStepBadge(step: number, name: string) {
    messages.value.push({ id: nextId++, role: 'assistant', html: `<div class="step-badge">⚙️ Step ${step}：${name}</div>` })
  }

  function addInfoBadge(content: string) {
    messages.value.push({ id: nextId++, role: 'assistant', html: `<div class="info-badge">ℹ️ ${content}</div>` })
  }

  async function send(text: string) {
    if (sending.value || !text.trim()) return
    sending.value = true

    addMessage('user', escapeHtml(text))
    let streamingMsg: ChatMessage | null = null

    try {
      const res = await apiSendMessage(text)
      if (!res.ok) {
        const err = await res.json()
        addMessage('assistant', `<span class="err-text">❌ ${escapeHtml(err.error ?? '请求失败')}</span>`)
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
          if (!line.startsWith('data: ')) continue
          try {
            const event: SSEEvent = JSON.parse(line.slice(6))
            streamingMsg = handleEvent(event, streamingMsg)
          } catch { /* ignore parse errors */ }
        }
      }
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : String(e)
      addMessage('assistant', `<span class="err-text">❌ 连接失败：${escapeHtml(msg)}</span>`)
    } finally {
      if (streamingMsg) streamingMsg.streaming = false
      sending.value = false
    }
  }

  function handleEvent(event: SSEEvent, streamingMsg: ChatMessage | null): ChatMessage | null {
    switch (event.type) {
      case 'token': {
        if (!streamingMsg) {
          streamingMsg = addMessage('assistant', '', { streaming: true, rawText: '' })
        }
        streamingMsg.rawText = (streamingMsg.rawText ?? '') + event.content
        streamingMsg.html = renderMarkdown(streamingMsg.rawText)
        return streamingMsg
      }
      case 'step':
        if (streamingMsg) { streamingMsg.streaming = false; streamingMsg = null }
        addStepBadge(event.step, event.name)
        return null
      case 'info':
        addInfoBadge(event.content)
        return streamingMsg
      case 'outline':
        messages.value.push({ id: nextId++, role: 'assistant', html: '__outline__', ...{ outlineData: event.data } } as unknown as ChatMessage)
        return streamingMsg
      case 'action':
        messages.value.push({ id: nextId++, role: 'assistant', html: '__action__', ...{ actionOptions: event.options } } as unknown as ChatMessage)
        return streamingMsg
      case 'similarity':
        messages.value.push({ id: nextId++, role: 'assistant', html: '__similarity__', ...{ simData: event.data } } as unknown as ChatMessage)
        return streamingMsg
      case 'complete':
        if (streamingMsg) { streamingMsg.streaming = false; streamingMsg = null }
        addMessage('assistant', `<span class="ok-text">✅ 稿件已保存！ID: ${event.scriptId}</span><br><span class="hint-text">输入新内容开始下一轮，或点击「新建对话」重置。</span>`)
        return null
      case 'error':
        if (streamingMsg) { streamingMsg.streaming = false; streamingMsg = null }
        addMessage('assistant', `<span class="err-text">❌ ${escapeHtml(event.message)}</span>`)
        return null
    }
  }

  async function reset() {
    await apiResetSession()
    messages.value = []
  }

  function escapeHtml(s: string) {
    return s.replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;').replace(/"/g,'&quot;')
  }

  function renderMarkdown(text: string): string {
    return text
      .replace(/^### (.+)$/gm, '<h3>$1</h3>')
      .replace(/^## (.+)$/gm, '<h3>$1</h3>')
      .replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>')
      .replace(/^---+$/gm, '<hr>')
      .replace(/`([^`]+)`/g, '<code>$1</code>')
      .replace(/\n\n/g, '</p><p>')
      .replace(/\n/g, '<br>')
  }

  return { messages, sending, send, reset }
})
```

- [ ] **Step 3: Commit**

```bash
cd /data/code/content_creator_imm
git add frontend/src/stores/
git commit -m "feat: add Pinia stores (user auth + chat SSE)"
```

---

## Task 5: Vue Router with auth guard

**Files:**
- Create: `frontend/src/router/index.ts`

- [ ] **Step 1: Create frontend/src/router/index.ts**

```typescript
import { createRouter, createWebHistory } from 'vue-router'
import Login from '@/views/Login.vue'
import Home from '@/views/Home.vue'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/login', component: Login },
    { path: '/', component: Home, meta: { requiresAuth: true } },
    { path: '/:pathMatch(.*)*', redirect: '/' }
  ]
})

router.beforeEach(to => {
  const token = localStorage.getItem('token')
  if (to.meta.requiresAuth && !token) return '/login'
  if (to.path === '/login' && token) return '/'
  return true
})

export default router
```

- [ ] **Step 2: Commit**

```bash
cd /data/code/content_creator_imm
git add frontend/src/router/
git commit -m "feat: add Vue Router with auth guard"
```

---

## Task 6: Login.vue

**Files:**
- Create: `frontend/src/views/Login.vue`

- [ ] **Step 1: Create frontend/src/views/Login.vue**

```vue
<template>
  <div class="login-page">
    <el-card class="login-card">
      <h1>🎙 口播稿助手</h1>
      <p class="subtitle">AI 驱动的爆款口播稿改写工具</p>

      <el-tabs v-model="activeTab">
        <el-tab-pane label="登录" name="login">
          <el-form :model="loginForm" @submit.prevent="doLogin">
            <el-form-item>
              <el-input v-model="loginForm.email" type="email" placeholder="邮箱" @keyup.enter="doLogin" />
            </el-form-item>
            <el-form-item>
              <el-input v-model="loginForm.password" type="password" placeholder="密码" @keyup.enter="doLogin" />
            </el-form-item>
            <el-button type="primary" :loading="loading" style="width:100%" @click="doLogin">登录</el-button>
          </el-form>
        </el-tab-pane>

        <el-tab-pane label="注册" name="register">
          <el-form :model="regForm" @submit.prevent="doRegister">
            <el-form-item>
              <el-input v-model="regForm.username" placeholder="用户名（2-64字符）" />
            </el-form-item>
            <el-form-item>
              <el-input v-model="regForm.email" type="email" placeholder="邮箱" />
            </el-form-item>
            <el-form-item>
              <el-input v-model="regForm.password" type="password" placeholder="密码（至少6位）" />
            </el-form-item>
            <el-button type="primary" :loading="loading" style="width:100%" @click="doRegister">注册</el-button>
          </el-form>
        </el-tab-pane>
      </el-tabs>

      <p v-if="errorMsg" class="error-msg">{{ errorMsg }}</p>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useUserStore } from '@/stores/user'
import { ElMessage } from 'element-plus'

const router = useRouter()
const userStore = useUserStore()

const activeTab = ref('login')
const loading = ref(false)
const errorMsg = ref('')

const loginForm = ref({ email: '', password: '' })
const regForm = ref({ username: '', email: '', password: '' })

async function doLogin() {
  errorMsg.value = ''
  loading.value = true
  try {
    await userStore.login(loginForm.value.email, loginForm.value.password)
    router.push('/')
  } catch (e: unknown) {
    errorMsg.value = extractError(e) || '登录失败'
  } finally {
    loading.value = false
  }
}

async function doRegister() {
  errorMsg.value = ''
  loading.value = true
  try {
    await userStore.register(regForm.value.username, regForm.value.email, regForm.value.password)
    ElMessage.success('注册成功，请登录')
    activeTab.value = 'login'
    loginForm.value.email = regForm.value.email
  } catch (e: unknown) {
    errorMsg.value = extractError(e) || '注册失败'
  } finally {
    loading.value = false
  }
}

function extractError(e: unknown): string {
  if (e && typeof e === 'object' && 'response' in e) {
    const resp = (e as { response?: { data?: { error?: string } } }).response
    return resp?.data?.error ?? ''
  }
  return e instanceof Error ? e.message : ''
}
</script>

<style scoped>
.login-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #0f1117;
}
.login-card {
  width: 380px;
  background: #1a1d27;
  border: 1px solid #2a2d3e;
  border-radius: 16px;
}
h1 { text-align: center; color: #e2e8f0; font-size: 22px; margin: 0 0 8px; }
.subtitle { text-align: center; color: #64748b; font-size: 14px; margin-bottom: 24px; }
.error-msg { color: #f87171; font-size: 13px; text-align: center; margin-top: 12px; }
</style>
```

- [ ] **Step 2: Commit**

```bash
cd /data/code/content_creator_imm
git add frontend/src/views/Login.vue
git commit -m "feat: add Login.vue with login/register tabs"
```

---

## Task 7: Home.vue layout + ScriptList.vue

**Files:**
- Create: `frontend/src/views/Home.vue`
- Create: `frontend/src/components/ScriptList.vue`

- [ ] **Step 1: Create frontend/src/views/Home.vue**

```vue
<template>
  <div class="app-shell">
    <header class="app-header">
      <div class="logo">🎙 口播稿助手</div>
      <div class="user-info">
        <span>{{ userStore.user?.username }}</span>
        <el-button size="small" text @click="logout">退出</el-button>
      </div>
    </header>

    <div class="main-content">
      <aside class="sidebar">
        <div class="sidebar-header">
          <span>历史稿件</span>
          <el-button size="small" @click="newChat">+ 新建</el-button>
        </div>
        <ScriptList @select="viewScript" />
      </aside>

      <main class="chat-area">
        <ChatPanel ref="chatPanelRef" />
      </main>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useUserStore } from '@/stores/user'
import { useChatStore } from '@/stores/chat'
import ChatPanel from '@/components/ChatPanel.vue'
import ScriptList from '@/components/ScriptList.vue'
import { getScript } from '@/api/scripts'

const router = useRouter()
const userStore = useUserStore()
const chatStore = useChatStore()
const chatPanelRef = ref<InstanceType<typeof ChatPanel> | null>(null)

function logout() {
  userStore.logout()
  router.push('/login')
}

async function newChat() {
  await chatStore.reset()
  chatPanelRef.value?.showWelcome()
}

async function viewScript(id: number) {
  try {
    const { data } = await getScript(id)
    chatPanelRef.value?.showScriptDetail(data.script.title, data.content)
  } catch {
    // ignore
  }
}
</script>

<style scoped>
.app-shell { display: flex; flex-direction: column; height: 100vh; background: #0f1117; color: #e2e8f0; }
.app-header { display: flex; align-items: center; justify-content: space-between; padding: 0 20px; height: 56px; background: #1a1d27; border-bottom: 1px solid #2a2d3e; flex-shrink: 0; }
.logo { font-size: 16px; font-weight: 600; color: #a78bfa; }
.user-info { display: flex; align-items: center; gap: 12px; font-size: 14px; color: #94a3b8; }
.main-content { display: flex; flex: 1; overflow: hidden; }
.sidebar { width: 240px; border-right: 1px solid #2a2d3e; display: flex; flex-direction: column; overflow: hidden; }
.sidebar-header { display: flex; align-items: center; justify-content: space-between; padding: 12px 16px; border-bottom: 1px solid #2a2d3e; font-size: 14px; font-weight: 600; }
.chat-area { flex: 1; display: flex; flex-direction: column; overflow: hidden; }
</style>
```

- [ ] **Step 2: Create frontend/src/components/ScriptList.vue**

```vue
<template>
  <div class="script-list">
    <div v-if="loading" class="hint">加载中...</div>
    <div v-else-if="!scripts.length" class="hint">暂无历史稿件</div>
    <div
      v-for="s in scripts"
      :key="s.id"
      class="script-item"
      @click="$emit('select', s.id)"
    >
      <div class="title">{{ s.title || '未命名' }}</div>
      <div class="meta">{{ formatDate(s.created_at) }} · 相似度 {{ ((s.similarity_score || 0) * 100).toFixed(0) }}%</div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { getScripts, type Script } from '@/api/scripts'

defineEmits<{ select: [id: number] }>()

const scripts = ref<Script[]>([])
const loading = ref(true)

onMounted(async () => {
  try {
    const { data } = await getScripts()
    scripts.value = data.scripts ?? []
  } finally {
    loading.value = false
  }
})

function formatDate(dateStr: string): string {
  if (!dateStr) return ''
  const d = new Date(dateStr)
  const diff = Date.now() - d.getTime()
  if (diff < 60000) return '刚刚'
  if (diff < 3600000) return Math.floor(diff / 60000) + '分钟前'
  if (diff < 86400000) return Math.floor(diff / 3600000) + '小时前'
  return `${d.getMonth() + 1}/${d.getDate()}`
}
</script>

<style scoped>
.script-list { flex: 1; overflow-y: auto; }
.hint { padding: 16px; color: #475569; font-size: 13px; text-align: center; }
.script-item { padding: 12px 16px; cursor: pointer; border-bottom: 1px solid #1e2133; }
.script-item:hover { background: #1e2133; }
.title { font-size: 13px; color: #e2e8f0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.meta { font-size: 11px; color: #475569; margin-top: 4px; }
</style>
```

- [ ] **Step 3: Commit**

```bash
cd /data/code/content_creator_imm
git add frontend/src/views/Home.vue frontend/src/components/ScriptList.vue
git commit -m "feat: add Home layout + ScriptList sidebar component"
```

---

## Task 8: ChatPanel.vue — SSE rendering

**Files:**
- Create: `frontend/src/components/ChatPanel.vue`

The chat store already handles SSE parsing. This component renders `chatStore.messages` and adds special rendering for `__outline__`, `__action__`, `__similarity__` message types.

- [ ] **Step 1: Create frontend/src/components/ChatPanel.vue**

```vue
<template>
  <div class="chat-panel">
    <!-- Messages -->
    <div ref="messagesEl" class="messages">
      <div v-if="!chatStore.messages.length" class="welcome">
        <h2>🎙 口播稿助手</h2>
        <p>粘贴短视频链接，或直接粘贴口播文案<br>AI 5角色分析，生成原创改写稿</p>
      </div>

      <template v-for="msg in chatStore.messages" :key="msg.id">
        <!-- Outline card -->
        <div v-if="(msg as any).outlineData" class="msg-row assistant">
          <div class="msg-avatar ai">📋</div>
          <div class="outline-card">
            <h4>📋 大纲方案（确认后开始撰写）</h4>
            <div v-if="(msg as any).outlineData.elements?.length" class="elements">
              <strong>保留要素：</strong>
              <el-tag v-for="e in (msg as any).outlineData.elements" :key="e" size="small" style="margin:2px">{{ e }}</el-tag>
            </div>
            <div v-for="p in (msg as any).outlineData.outline" :key="p.part" class="outline-row">
              <el-tag type="info" size="small">{{ p.part }}</el-tag>
              <span class="part-content">{{ p.content }} <span class="duration">{{ p.duration }}</span></span>
            </div>
          </div>
        </div>

        <!-- Action buttons -->
        <div v-else-if="(msg as any).actionOptions" class="msg-row assistant">
          <div class="msg-avatar ai">💬</div>
          <div class="action-btns">
            <el-button
              v-for="(opt, i) in (msg as any).actionOptions"
              :key="i"
              :type="i === 0 ? 'primary' : 'default'"
              size="small"
              @click="quickSend(opt, i + 1)"
            >{{ opt }}</el-button>
          </div>
        </div>

        <!-- Similarity card -->
        <div v-else-if="(msg as any).simData" class="msg-row assistant">
          <div class="msg-avatar ai">📊</div>
          <div class="sim-card">
            <div v-for="(val, key) in simDisplay((msg as any).simData)" :key="key" class="sim-item">
              <div class="label">{{ val.label }}</div>
              <div :class="['value', val.cls]">{{ val.text }}</div>
            </div>
          </div>
        </div>

        <!-- Regular message -->
        <div v-else class="msg-row" :class="msg.role">
          <div class="msg-avatar" :class="msg.role === 'user' ? 'user-av' : 'ai'">
            {{ msg.role === 'user' ? '👤' : '🤖' }}
          </div>
          <!-- eslint-disable-next-line vue/no-v-html -->
          <div class="msg-bubble" :class="{ streaming: msg.streaming }" v-html="msg.html" />
        </div>
      </template>

      <div v-if="scriptDetail" class="script-detail">
        <h3>{{ scriptDetail.title }}</h3>
        <pre>{{ scriptDetail.content }}</pre>
      </div>
    </div>

    <!-- Input -->
    <div class="input-area">
      <el-input
        v-model="inputText"
        type="textarea"
        :autosize="{ minRows: 1, maxRows: 6 }"
        placeholder="粘贴视频链接 或 直接粘贴口播文案... (Enter 发送，Shift+Enter 换行)"
        :disabled="chatStore.sending"
        @keydown.enter.exact.prevent="doSend"
      />
      <el-button
        type="primary"
        :loading="chatStore.sending"
        :disabled="!inputText.trim()"
        @click="doSend"
      >▶</el-button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, nextTick } from 'vue'
import { useChatStore, type SimilarityData } from '@/stores/chat'

const chatStore = useChatStore()
const inputText = ref('')
const messagesEl = ref<HTMLElement | null>(null)
const scriptDetail = ref<{ title: string; content: string } | null>(null)

watch(() => chatStore.messages.length, () => {
  nextTick(() => {
    if (messagesEl.value) messagesEl.value.scrollTop = messagesEl.value.scrollHeight
  })
})

function doSend() {
  const text = inputText.value.trim()
  if (!text || chatStore.sending) return
  inputText.value = ''
  scriptDetail.value = null
  chatStore.send(text)
}

function quickSend(opt: string, num: number) {
  inputText.value = String(num)
  doSend()
}

function showWelcome() {
  chatStore.messages.length = 0
  scriptDetail.value = null
}

function showScriptDetail(title: string, content: string) {
  scriptDetail.value = { title, content }
  chatStore.messages.length = 0
}

defineExpose({ showWelcome, showScriptDetail })

function simDisplay(data: SimilarityData) {
  const total = data.total ?? 0
  const cls = total < 25 ? 'ok' : total < 30 ? 'warn' : 'bad'
  return {
    total: { label: '综合相似度', text: `${total.toFixed(1)}%`, cls },
    vocab: { label: '词汇', text: `${(data.vocab ?? 0).toFixed(1)}%`, cls: '' },
    sentence: { label: '句式', text: `${(data.sentence ?? 0).toFixed(1)}%`, cls: '' },
    structure: { label: '结构', text: `${(data.structure ?? 0).toFixed(1)}%`, cls: '' },
    viewpoint: { label: '观点', text: `${(data.viewpoint ?? 0).toFixed(1)}%`, cls: '' },
    result: { label: '结论', text: total < 30 ? '✅ 通过' : '❌ 超标', cls }
  }
}
</script>

<style scoped>
.chat-panel { display: flex; flex-direction: column; height: 100%; background: #0f1117; }
.messages { flex: 1; overflow-y: auto; padding: 16px; display: flex; flex-direction: column; gap: 8px; }
.welcome { text-align: center; margin: auto; color: #64748b; }
.welcome h2 { font-size: 24px; color: #a78bfa; margin-bottom: 12px; }
.msg-row { display: flex; gap: 8px; align-items: flex-start; }
.msg-row.user { flex-direction: row-reverse; }
.msg-avatar { width: 32px; height: 32px; border-radius: 50%; display: flex; align-items: center; justify-content: center; font-size: 16px; flex-shrink: 0; background: #2a2d3e; }
.user-av { background: #7c3aed; }
.ai { background: #1e293b; }
.msg-bubble { background: #1a1d27; border: 1px solid #2a2d3e; border-radius: 12px; padding: 10px 14px; max-width: 720px; font-size: 14px; line-height: 1.6; color: #e2e8f0; }
.msg-row.user .msg-bubble { background: #4c1d95; border-color: #6d28d9; }
.streaming::after { content: '▊'; animation: blink .7s step-end infinite; }
@keyframes blink { 50% { opacity: 0; } }
.outline-card { background: #1a1d27; border: 1px solid #2a2d3e; border-radius: 12px; padding: 16px; max-width: 680px; }
.outline-card h4 { color: #a78bfa; margin: 0 0 12px; font-size: 14px; }
.elements { margin-bottom: 10px; font-size: 12px; color: #94a3b8; }
.outline-row { display: flex; gap: 8px; align-items: flex-start; margin: 6px 0; font-size: 13px; }
.part-content { color: #e2e8f0; }
.duration { color: #64748b; font-size: 12px; }
.action-btns { display: flex; flex-wrap: wrap; gap: 8px; }
.sim-card { display: flex; flex-wrap: wrap; gap: 12px; background: #1a1d27; border: 1px solid #2a2d3e; border-radius: 12px; padding: 14px; }
.sim-item { text-align: center; min-width: 60px; }
.label { font-size: 11px; color: #64748b; }
.value { font-size: 16px; font-weight: 700; color: #e2e8f0; }
.value.ok { color: #34d399; }
.value.warn { color: #fbbf24; }
.value.bad { color: #f87171; }
.script-detail { background: #1a1d27; border: 1px solid #2a2d3e; border-radius: 12px; padding: 20px; }
.script-detail h3 { color: #a78bfa; margin: 0 0 12px; }
.script-detail pre { white-space: pre-wrap; color: #e2e8f0; font-size: 14px; line-height: 1.6; }
.input-area { display: flex; gap: 8px; padding: 12px 16px; border-top: 1px solid #2a2d3e; background: #1a1d27; }
.input-area .el-textarea { flex: 1; }
:deep(.step-badge) { background: #1e2133; border-radius: 8px; padding: 6px 12px; font-size: 13px; color: #94a3b8; }
:deep(.info-badge) { font-size: 12px; color: #64748b; padding: 4px 8px; }
:deep(.err-text) { color: #f87171; }
:deep(.ok-text) { color: #34d399; }
:deep(.hint-text) { color: #64748b; font-size: 13px; }
</style>
```

- [ ] **Step 2: Commit**

```bash
cd /data/code/content_creator_imm
git add frontend/src/components/ChatPanel.vue
git commit -m "feat: add ChatPanel with SSE streaming rendering"
```

---

## Task 9: Verify frontend builds and runs

- [ ] **Step 1: Type-check**

```bash
cd /data/code/content_creator_imm/frontend && npm run build
```
Expected: build succeeds, `dist/` created. Fix any TypeScript errors before proceeding.

- [ ] **Step 2: Start backend and frontend, verify end-to-end**

Terminal 1:
```bash
cd /data/code/content_creator_imm/backend && go run .
```

Terminal 2:
```bash
cd /data/code/content_creator_imm/frontend && npm run dev
```

Open `http://localhost:5173` — should show login page.

- [ ] **Step 3: Commit any fixes**

```bash
cd /data/code/content_creator_imm
git add -p
git commit -m "fix: resolve frontend build/type errors"
```

---

## Task 10: build.sh + manage.sh

**Files:**
- Create: `build.sh`
- Create: `manage.sh`

- [ ] **Step 1: Create build.sh**

```bash
#!/bin/bash
set -e
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "=== [1/2] 构建前端 ==="
cd "$ROOT/frontend"
npm install
npm run build

echo "=== [2/2] 构建后端 ==="
cd "$ROOT/backend"
go build -o "$ROOT/content-creator-imm" .

echo "=== ✅ 构建完成 ==="
echo "  前端: frontend/dist/"
echo "  后端: content-creator-imm"
```

- [ ] **Step 2: Create manage.sh**

```bash
#!/bin/bash
set -e
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BINARY="$ROOT/content-creator-imm"
PID_FILE="$ROOT/server.pid"
LOG_FILE="$ROOT/server.log"
CONFIG="$ROOT/backend/config.json"

usage() {
  echo "用法: $0 <command>"
  echo ""
  echo "Commands:"
  echo "  start              启动后端服务"
  echo "  stop               停止后端服务"
  echo "  restart            重启后端服务"
  echo "  status             查看服务状态"
  echo "  logs               查看日志 (tail -f)"
  echo "  add-user <username> <email> <password>  添加用户"
  echo "  list-users         列出所有用户"
  exit 1
}

cmd_start() {
  if [ -f "$PID_FILE" ] && kill -0 "$(cat "$PID_FILE")" 2>/dev/null; then
    echo "服务已在运行 (PID $(cat "$PID_FILE"))"
    exit 0
  fi
  if [ ! -f "$BINARY" ]; then
    echo "❌ 未找到二进制文件，请先运行 ./build.sh"
    exit 1
  fi
  cd "$ROOT/backend"
  nohup "$BINARY" >> "$LOG_FILE" 2>&1 &
  echo $! > "$PID_FILE"
  sleep 1
  if kill -0 "$(cat "$PID_FILE")" 2>/dev/null; then
    echo "✅ 服务已启动 (PID $(cat "$PID_FILE"))"
  else
    echo "❌ 启动失败，查看日志: $LOG_FILE"
    exit 1
  fi
}

cmd_stop() {
  if [ ! -f "$PID_FILE" ]; then
    echo "服务未运行"
    return
  fi
  PID=$(cat "$PID_FILE")
  if kill -0 "$PID" 2>/dev/null; then
    kill "$PID"
    rm -f "$PID_FILE"
    echo "✅ 服务已停止"
  else
    rm -f "$PID_FILE"
    echo "服务未运行（清理了旧 PID 文件）"
  fi
}

cmd_status() {
  if [ -f "$PID_FILE" ] && kill -0 "$(cat "$PID_FILE")" 2>/dev/null; then
    echo "✅ 运行中 (PID $(cat "$PID_FILE"))"
  else
    echo "⏹  未运行"
  fi
}

cmd_add_user() {
  [ $# -lt 3 ] && { echo "用法: $0 add-user <username> <email> <password>"; exit 1; }
  # Read port from config.json
  PORT=$(python3 -c "import json,sys; d=json.load(open('$CONFIG')); print(d.get('port','3004'))" 2>/dev/null || echo "3004")
  curl -s -X POST "http://localhost:$PORT/api/auth/register" \
    -H "Content-Type: application/json" \
    -d "{\"username\":\"$1\",\"email\":\"$2\",\"password\":\"$3\"}" | python3 -m json.tool
}

cmd_list_users() {
  DB_HOST=$(python3 -c "import json; d=json.load(open('$CONFIG')); print(d.get('db_host','127.0.0.1'))" 2>/dev/null || echo "127.0.0.1")
  DB_PORT=$(python3 -c "import json; d=json.load(open('$CONFIG')); print(d.get('db_port','3306'))" 2>/dev/null || echo "3306")
  DB_USER=$(python3 -c "import json; d=json.load(open('$CONFIG')); print(d.get('db_user','root'))" 2>/dev/null || echo "root")
  DB_PASS=$(python3 -c "import json; d=json.load(open('$CONFIG')); print(d.get('db_password',''))" 2>/dev/null || echo "")
  DB_NAME=$(python3 -c "import json; d=json.load(open('$CONFIG')); print(d.get('db_name','content_creator'))" 2>/dev/null || echo "content_creator")
  mysql -h"$DB_HOST" -P"$DB_PORT" -u"$DB_USER" -p"$DB_PASS" "$DB_NAME" \
    -e "SELECT id, username, email, role, active, created_at FROM users ORDER BY id;"
}

case "${1:-}" in
  start)   cmd_start ;;
  stop)    cmd_stop ;;
  restart) cmd_stop; sleep 1; cmd_start ;;
  status)  cmd_status ;;
  logs)    tail -f "$LOG_FILE" ;;
  add-user) shift; cmd_add_user "$@" ;;
  list-users) cmd_list_users ;;
  *) usage ;;
esac
```

- [ ] **Step 3: Make scripts executable and commit**

```bash
chmod +x /data/code/content_creator_imm/build.sh
chmod +x /data/code/content_creator_imm/manage.sh
cd /data/code/content_creator_imm
git add build.sh manage.sh
git commit -m "feat: add build.sh and manage.sh scripts"
```

---

## Task 11: CLAUDE.md

**Files:**
- Create: `CLAUDE.md`

- [ ] **Step 1: Create CLAUDE.md**

```markdown
# 口播稿助手 (content-creator-imm)

AI 驱动的爆款口播稿改写工具。用户提交短视频链接或文案，系统通过5角色分析生成原创改写稿，相似度检测 < 30%。

## 架构

Monorepo，前后端分离：

```
content_creator_imm/
├── backend/        Go 1.22 + Gin API 服务（端口 3004）
├── frontend/       Vue 3 + Vite + Element Plus SPA（开发端口 5173）
├── build.sh        一键构建前后端
├── manage.sh       服务管理脚本
└── docs/           规范文档
```

**后端**：REST API + SSE 流式响应，JWT 认证，MySQL + GORM
**前端**：Pinia 状态管理，Vue Router，Axios，原生 fetch 处理 SSE

## 开发环境

### 启动后端（热重载需安装 air）

```bash
cd backend && go run .
# 或使用 air
cd backend && air
```

### 启动前端（热更新，代理 /api → localhost:3004）

```bash
cd frontend && npm run dev
```

访问 http://localhost:5173

## 构建生产版本

```bash
./build.sh
```

产物：
- `frontend/dist/` — 静态文件（nginx 托管）
- `content-creator-imm` — Go 二进制

## 服务管理

```bash
./manage.sh start          # 启动服务
./manage.sh stop           # 停止服务
./manage.sh restart        # 重启服务
./manage.sh status         # 查看状态
./manage.sh logs           # 实时日志
```

## 用户管理

```bash
# 添加用户（通过 API，需服务运行中）
./manage.sh add-user <username> <email> <password>

# 列出所有用户（需 mysql 命令行工具）
./manage.sh list-users
```

## 配置

配置文件：`backend/config.json`（参考 `backend/config.example.json`）

| 字段 | 说明 | 默认值 |
|------|------|--------|
| `port` | 后端端口 | `3004` |
| `jwt_secret` | JWT 签名密钥 | 请修改！ |
| `db_host/port/user/password/name` | MySQL 连接 | localhost:3306 |
| `anthropic_api_key` | Anthropic API Key | 必填 |
| `llm_base_url` | LLM API 地址 | https://api.anthropic.com |
| `cors_origins` | 允许的跨域来源（逗号分隔） | http://localhost:5173 |
| `storage_path` | 本地脚本存储路径 | data/scripts |

## API 列表

```
POST /api/auth/register       注册
POST /api/auth/login          登录 → 返回 JWT token

GET  /api/user/profile        获取风格档案
PUT  /api/user/style          更新风格档案

GET  /api/chat/session        获取当前会话状态
POST /api/chat/reset          重置会话
POST /api/chat/message        发送消息（SSE 流式响应）

GET  /api/scripts             稿件列表
GET  /api/scripts/:id         稿件详情 + 内容
```

### SSE 消息格式（/api/chat/message）

每条消息格式：`data: <JSON>\n\n`

| type | 字段 | 说明 |
|------|------|------|
| `token` | `content` | 流式文本 token |
| `step` | `step`, `name` | 流程步骤 |
| `info` | `content` | 状态信息 |
| `outline` | `data` | 大纲数据（待确认） |
| `action` | `options` | 操作按钮选项 |
| `similarity` | `data` | 相似度检测结果 |
| `complete` | `scriptId` | 完成，返回稿件 ID |
| `error` | `message` | 错误信息 |

## 生产部署（nginx）

```nginx
server {
    listen 80;
    root /path/to/frontend/dist;
    index index.html;

    location / {
        try_files $uri $uri/ /index.html;
    }

    location /api {
        proxy_pass http://127.0.0.1:3004;
        proxy_set_header X-Accel-Buffering no;  # 禁用 nginx 缓冲，确保 SSE 实时推送
    }
}
```
```

- [ ] **Step 2: Commit**

```bash
cd /data/code/content_creator_imm
git add CLAUDE.md
git commit -m "docs: add CLAUDE.md with architecture, commands, API reference"
```

---

## Task 12: Cleanup

**Files:**
- Delete: `public/` (old frontend)
- Update: `.gitignore`

- [ ] **Step 1: Remove old public/ directory**

```bash
rm -rf /data/code/content_creator_imm/public
```

- [ ] **Step 2: Create/update .gitignore**

```
# Backend
backend/config.json
backend/data/
backend/*.pid
server.pid
server.log
content-creator-imm

# Frontend
frontend/node_modules/
frontend/dist/

# Go
*.test
```

- [ ] **Step 3: Final commit**

```bash
cd /data/code/content_creator_imm
git add .gitignore
git rm -r --cached public/ 2>/dev/null || true
git add -A
git commit -m "chore: remove old public/, add .gitignore, finalize monorepo structure"
```

---

## Final Verification

- [ ] `./build.sh` 成功完成，无错误
- [ ] `./manage.sh start` 启动服务
- [ ] 访问 `http://localhost:5173`（开发）或构建后通过 nginx 访问
- [ ] 登录/注册正常
- [ ] 聊天 SSE 流式响应正常
- [ ] 历史稿件侧边栏加载正常
