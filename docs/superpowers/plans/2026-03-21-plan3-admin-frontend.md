# Admin Backend Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 构建口播稿改写系统的 Web 管理后台，支持超管视角（用户管理、全量稿件、系统配置、数据看板）和用户自助视角（我的稿件、风格档案、素材收藏），前端构建产物通过 Go embed 打包进二进制。

**Architecture:** React 18 + TypeScript + Ant Design Pro 5，前后端分离开发，构建后产物放入 `server/admin/dist/`，通过 Go `embed.FS` 嵌入二进制。开发时 Vite 代理到本地 Go 服务（`:8080`），生产时 Go 直接 serve 静态文件。

**Tech Stack:** React 18, TypeScript, Ant Design Pro 5, Vite, @ant-design/pro-components, axios

**前置条件:** Plan 1 已完成，API 服务运行在 `:8080`。

---

## 文件结构

```
admin-frontend/                   # 独立前端项目目录
├── package.json
├── vite.config.ts
├── tsconfig.json
├── index.html
└── src/
    ├── main.tsx
    ├── app.tsx                   # 路由 + 布局
    ├── api/
    │   ├── client.ts             # axios 实例 + 拦截器（token 注入、401处理）
    │   ├── auth.ts               # 登录接口
    │   ├── user.ts               # 用户/风格接口
    │   ├── script.ts             # 稿件接口
    │   └── material.ts           # 素材/热点接口
    ├── store/
    │   └── auth.ts               # 全局登录状态（zustand）
    ├── pages/
    │   ├── login/
    │   │   └── index.tsx         # 登录页
    │   ├── admin/
    │   │   ├── users/
    │   │   │   └── index.tsx     # 超管: 用户管理
    │   │   ├── scripts/
    │   │   │   └── index.tsx     # 超管: 全量稿件
    │   │   └── dashboard/
    │   │       └── index.tsx     # 超管: 数据看板
    │   └── user/
    │       ├── scripts/
    │       │   └── index.tsx     # 用户: 我的稿件
    │       ├── style/
    │       │   └── index.tsx     # 用户: 风格档案
    │       └── materials/
    │           └── index.tsx     # 用户: 素材收藏
    └── components/
        └── script-detail/
            └── index.tsx         # 稿件详情弹窗（含质量报告）
server/
└── admin.go                      # Go embed + 静态文件服务
```

---

## Task 1: 前端项目初始化

**Files:**
- Create: `admin-frontend/package.json`
- Create: `admin-frontend/vite.config.ts`
- Create: `admin-frontend/tsconfig.json`
- Create: `admin-frontend/index.html`
- Create: `admin-frontend/src/main.tsx`

- [ ] **Step 1: 初始化项目**

```bash
mkdir -p /data/code/content_creator_imm/admin-frontend
cd /data/code/content_creator_imm/admin-frontend
npm create vite@latest . -- --template react-ts
```

- [ ] **Step 2: 安装依赖**

```bash
cd /data/code/content_creator_imm/admin-frontend
npm install
npm install antd @ant-design/pro-components @ant-design/icons axios zustand react-router-dom
```

- [ ] **Step 3: 配置 vite.config.ts（开发代理）**

```typescript
// vite.config.ts
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      '/api': 'http://localhost:8080',
    },
  },
  build: {
    outDir: '../server/admin/dist',
    emptyOutDir: true,
  },
})
```

- [ ] **Step 4: 验证开发服务器启动**

```bash
cd /data/code/content_creator_imm/admin-frontend
npm run dev
# 访问 http://localhost:5173 看到默认 React 页面即可
```

- [ ] **Step 5: Commit**

```bash
cd /data/code/content_creator_imm
git add admin-frontend/
git commit -m "feat: initialize admin frontend with Vite + React + Ant Design"
```

---

## Task 2: API 客户端 + 认证状态

**Files:**
- Create: `admin-frontend/src/api/client.ts`
- Create: `admin-frontend/src/api/auth.ts`
- Create: `admin-frontend/src/store/auth.ts`

- [ ] **Step 1: 创建 axios 客户端**

```typescript
// src/api/client.ts
import axios from 'axios'

const client = axios.create({
  baseURL: '/api/v1',
  timeout: 10000,
})

client.interceptors.request.use((config) => {
  const token = localStorage.getItem('token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

client.interceptors.response.use(
  (res) => res,
  (err) => {
    if (err.response?.status === 401) {
      localStorage.removeItem('token')
      window.location.href = '/login'
    }
    return Promise.reject(err)
  }
)

export default client
```

- [ ] **Step 2: 创建 auth API**

```typescript
// src/api/auth.ts
import client from './client'

export const login = (email: string, password: string) =>
  client.post<{ token: string }>('/auth/login', { email, password })
```

- [ ] **Step 3: 创建 zustand 认证 store**

```typescript
// src/store/auth.ts
import { create } from 'zustand'

interface AuthState {
  token: string | null
  role: string | null
  login: (token: string) => void
  logout: () => void
}

// 从 JWT payload 解析 role
function parseRole(token: string): string {
  try {
    const payload = JSON.parse(atob(token.split('.')[1]))
    return payload.role ?? 'user'
  } catch {
    return 'user'
  }
}

export const useAuth = create<AuthState>((set) => ({
  token: localStorage.getItem('token'),
  role: (() => {
    const t = localStorage.getItem('token')
    return t ? parseRole(t) : null
  })(),
  login: (token) => {
    localStorage.setItem('token', token)
    set({ token, role: parseRole(token) })
  },
  logout: () => {
    localStorage.removeItem('token')
    set({ token: null, role: null })
  },
}))
```

- [ ] **Step 4: Commit**

```bash
cd /data/code/content_creator_imm
git add admin-frontend/
git commit -m "feat: add API client with JWT interceptor and auth store"
```

---

## Task 3: 登录页 + 路由布局

**Files:**
- Create: `admin-frontend/src/pages/login/index.tsx`
- Create: `admin-frontend/src/app.tsx`
- Modify: `admin-frontend/src/main.tsx`

- [ ] **Step 1: 创建登录页**

```typescript
// src/pages/login/index.tsx
import { LoginForm, ProFormText } from '@ant-design/pro-components'
import { message } from 'antd'
import { useNavigate } from 'react-router-dom'
import { login } from '../../api/auth'
import { useAuth } from '../../store/auth'

export default function LoginPage() {
  const navigate = useNavigate()
  const authLogin = useAuth((s) => s.login)
  const role = useAuth((s) => s.role)

  const handleSubmit = async (values: { email: string; password: string }) => {
    try {
      const res = await login(values.email, values.password)
      authLogin(res.data.token)
      // 按角色跳转
      const r = role ?? 'user'
      navigate(r === 'admin' ? '/admin/dashboard' : '/user/scripts')
    } catch {
      message.error('邮箱或密码错误')
    }
  }

  return (
    <div style={{ height: '100vh', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
      <LoginForm title="口播稿改写系统" onFinish={handleSubmit}>
        <ProFormText name="email" label="邮箱" rules={[{ required: true }]} />
        <ProFormText.Password name="password" label="密码" rules={[{ required: true }]} />
      </LoginForm>
    </div>
  )
}
```

- [ ] **Step 2: 创建主路由 app.tsx**

```typescript
// src/app.tsx
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { ProLayout } from '@ant-design/pro-components'
import LoginPage from './pages/login'
import UserScriptsPage from './pages/user/scripts'
import UserStylePage from './pages/user/style'
import UserMaterialsPage from './pages/user/materials'
import AdminUsersPage from './pages/admin/users'
import AdminScriptsPage from './pages/admin/scripts'
import AdminDashboardPage from './pages/admin/dashboard'
import { useAuth } from './store/auth'

function PrivateLayout({ children, role }: { children: React.ReactNode; role: 'admin' | 'user' }) {
  const authRole = useAuth((s) => s.role)
  const logout = useAuth((s) => s.logout)

  if (!authRole) return <Navigate to="/login" />
  if (role === 'admin' && authRole !== 'admin') return <Navigate to="/user/scripts" />

  const adminMenus = [
    { path: '/admin/dashboard', name: '数据看板' },
    { path: '/admin/users', name: '用户管理' },
    { path: '/admin/scripts', name: '全量稿件' },
  ]
  const userMenus = [
    { path: '/user/scripts', name: '我的稿件' },
    { path: '/user/style', name: '风格档案' },
    { path: '/user/materials', name: '素材收藏' },
  ]

  return (
    <ProLayout
      title="口播稿改写系统"
      menuDataRender={() => authRole === 'admin' ? [...adminMenus, ...userMenus] : userMenus}
      avatarProps={{ title: authRole, onClick: logout }}
    >
      {children}
    </ProLayout>
  )
}

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/login" element={<LoginPage />} />
        <Route path="/user/scripts" element={<PrivateLayout role="user"><UserScriptsPage /></PrivateLayout>} />
        <Route path="/user/style" element={<PrivateLayout role="user"><UserStylePage /></PrivateLayout>} />
        <Route path="/user/materials" element={<PrivateLayout role="user"><UserMaterialsPage /></PrivateLayout>} />
        <Route path="/admin/dashboard" element={<PrivateLayout role="admin"><AdminDashboardPage /></PrivateLayout>} />
        <Route path="/admin/users" element={<PrivateLayout role="admin"><AdminUsersPage /></PrivateLayout>} />
        <Route path="/admin/scripts" element={<PrivateLayout role="admin"><AdminScriptsPage /></PrivateLayout>} />
        <Route path="*" element={<Navigate to="/login" />} />
      </Routes>
    </BrowserRouter>
  )
}
```

- [ ] **Step 3: 更新 main.tsx**

```typescript
// src/main.tsx
import React from 'react'
import ReactDOM from 'react-dom/client'
import App from './app'
import 'antd/dist/reset.css'

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode><App /></React.StrictMode>
)
```

- [ ] **Step 4: 验证页面可访问**

```bash
cd /data/code/content_creator_imm/admin-frontend && npm run dev
# 访问 http://localhost:5173/login 看到登录表单
```

- [ ] **Step 5: Commit**

```bash
cd /data/code/content_creator_imm
git add admin-frontend/
git commit -m "feat: add login page and role-based routing layout"
```

---

## Task 4: 用户侧页面（我的稿件 + 风格档案 + 素材收藏）

**Files:**
- Create: `admin-frontend/src/api/user.ts`
- Create: `admin-frontend/src/api/script.ts`
- Create: `admin-frontend/src/api/material.ts`
- Create: `admin-frontend/src/pages/user/scripts/index.tsx`
- Create: `admin-frontend/src/pages/user/style/index.tsx`
- Create: `admin-frontend/src/pages/user/materials/index.tsx`
- Create: `admin-frontend/src/components/script-detail/index.tsx`

- [ ] **Step 1: 创建 API 模块**

```typescript
// src/api/user.ts
import client from './client'
export const getProfile = () => client.get('/user/profile')
export const updateStyle = (data: object) => client.put('/user/style', data)
```

```typescript
// src/api/script.ts
import client from './client'
export const listScripts = (page = 1, limit = 20) =>
  client.get('/scripts', { params: { page, limit } })
export const getScript = (id: string) => client.get(`/scripts/${id}`)
export const createScript = (data: object) => client.post('/scripts', data)
export const updateTags = (id: string, tags: string) =>
  client.put(`/scripts/${id}/tags`, { tags })
```

```typescript
// src/api/material.ts
import client from './client'
export const listMaterials = (topic?: string, limit = 10) =>
  client.get('/materials', { params: { topic, limit } })
export const listHotspots = (platform?: string) =>
  client.get('/hotspot', { params: { platform } })
```

- [ ] **Step 2: 稿件详情弹窗组件**

```typescript
// src/components/script-detail/index.tsx
import { Drawer, Descriptions, Tag, Typography } from 'antd'

interface Script {
  id: string; title: string; content: string; platform: string;
  similarity_score: number; viral_score: number; tags: string; created_at: string;
}

export default function ScriptDetail({ script, onClose }: { script: Script | null; onClose: () => void }) {
  if (!script) return null
  const tags: string[] = (() => { try { return JSON.parse(script.tags || '[]') } catch { return [] } })()

  return (
    <Drawer title={script.title || '稿件详情'} open={!!script} onClose={onClose} width={600}>
      <Descriptions column={2} bordered size="small" style={{ marginBottom: 16 }}>
        <Descriptions.Item label="平台">{script.platform}</Descriptions.Item>
        <Descriptions.Item label="创建时间">{new Date(script.created_at).toLocaleString()}</Descriptions.Item>
        <Descriptions.Item label="相似度">{(script.similarity_score * 100).toFixed(1)}%</Descriptions.Item>
        <Descriptions.Item label="爆款评分">{script.viral_score.toFixed(1)}/10</Descriptions.Item>
        <Descriptions.Item label="标签" span={2}>
          {tags.map((t) => <Tag key={t}>{t}</Tag>)}
        </Descriptions.Item>
      </Descriptions>
      <Typography.Title level={5}>正文</Typography.Title>
      <Typography.Paragraph style={{ whiteSpace: 'pre-wrap' }}>{script.content}</Typography.Paragraph>
    </Drawer>
  )
}
```

- [ ] **Step 3: 我的稿件页面**

```typescript
// src/pages/user/scripts/index.tsx
import { useState } from 'react'
import { ProTable } from '@ant-design/pro-components'
import { Button } from 'antd'
import { listScripts } from '../../../api/script'
import ScriptDetail from '../../../components/script-detail'

export default function UserScriptsPage() {
  const [detail, setDetail] = useState<any>(null)

  return (
    <>
      <ProTable
        headerTitle="我的稿件"
        rowKey="id"
        request={async ({ current, pageSize }) => {
          const res = await listScripts(current, pageSize)
          return { data: res.data.data, total: res.data.total, success: true }
        }}
        columns={[
          { title: '标题', dataIndex: 'title', ellipsis: true },
          { title: '平台', dataIndex: 'platform', width: 100 },
          { title: '相似度', dataIndex: 'similarity_score', width: 100, render: (v: number) => `${(v*100).toFixed(1)}%` },
          { title: '爆款评分', dataIndex: 'viral_score', width: 100, render: (v: number) => `${v.toFixed(1)}/10` },
          { title: '创建时间', dataIndex: 'created_at', width: 180, render: (v: string) => new Date(v).toLocaleString() },
          { title: '操作', render: (_, row) => <Button type="link" onClick={() => setDetail(row)}>查看</Button> },
        ]}
      />
      <ScriptDetail script={detail} onClose={() => setDetail(null)} />
    </>
  )
}
```

- [ ] **Step 4: 风格档案页面**

```typescript
// src/pages/user/style/index.tsx
import { useEffect, useState } from 'react'
import { ProForm, ProFormText, ProFormSelect } from '@ant-design/pro-components'
import { Card, message } from 'antd'
import { getProfile, updateStyle } from '../../../api/user'

export default function UserStylePage() {
  const [style, setStyle] = useState<any>({})

  useEffect(() => {
    getProfile().then((res) => setStyle(res.data.style || {}))
  }, [])

  return (
    <Card title="我的风格档案">
      <ProForm
        initialValues={style}
        onFinish={async (values) => {
          await updateStyle(values)
          message.success('风格档案已更新')
        }}
      >
        <ProFormSelect name="language_style" label="语言风格"
          options={['口语化', '书面化', '专业', '接地气'].map((v) => ({ label: v, value: v }))} />
        <ProFormSelect name="emotion_tone" label="情绪基调"
          options={['理性', '感性', '幽默', '严肃'].map((v) => ({ label: v, value: v }))} />
        <ProFormText name="opening_style" label="典型开场方式" />
        <ProFormText name="closing_style" label="典型结尾方式" />
        <ProFormText name="catchphrases" label="口头禅（逗号分隔）" />
      </ProForm>
    </Card>
  )
}
```

- [ ] **Step 5: 素材收藏页面**

```typescript
// src/pages/user/materials/index.tsx
import { ProTable } from '@ant-design/pro-components'
import { Tag } from 'antd'
import { listMaterials } from '../../../api/material'

const TYPE_LABELS: Record<string, string> = {
  data: '数据', case: '案例', quote: '金句', contrast: '反差'
}

export default function UserMaterialsPage() {
  return (
    <ProTable
      headerTitle="素材收藏"
      rowKey="id"
      request={async (params) => {
        const res = await listMaterials(params.topic)
        return { data: res.data.data, success: true }
      }}
      search={{ labelWidth: 'auto' }}
      columns={[
        { title: '主题', dataIndex: 'topic', width: 120 },
        { title: '类型', dataIndex: 'type', width: 80, render: (v: string) => <Tag>{TYPE_LABELS[v] ?? v}</Tag> },
        { title: '内容', dataIndex: 'content', ellipsis: true },
        { title: '来源', dataIndex: 'source', width: 150 },
      ]}
    />
  )
}
```

- [ ] **Step 6: Commit**

```bash
cd /data/code/content_creator_imm
git add admin-frontend/
git commit -m "feat: add user-side pages: scripts, style profile, materials"
```

---

## Task 5: 超管侧页面（用户管理 + 全量稿件 + 数据看板）

**Files:**
- Create: `admin-frontend/src/pages/admin/users/index.tsx`
- Create: `admin-frontend/src/pages/admin/scripts/index.tsx`
- Create: `admin-frontend/src/pages/admin/dashboard/index.tsx`

- [ ] **Step 1: 超管用户管理页面**

```typescript
// src/pages/admin/users/index.tsx
import { ProTable } from '@ant-design/pro-components'
import { Button, Popconfirm, Tag, message } from 'antd'
import client from '../../../api/client'

export default function AdminUsersPage() {
  return (
    <ProTable
      headerTitle="用户管理"
      rowKey="id"
      request={async ({ current, pageSize }) => {
        const res = await client.get('/admin/users', { params: { page: current, limit: pageSize } })
        return { data: res.data.data, total: res.data.total, success: true }
      }}
      columns={[
        { title: '用户名', dataIndex: 'username' },
        { title: '邮箱', dataIndex: 'email' },
        { title: '角色', dataIndex: 'role', render: (v: string) => <Tag color={v === 'admin' ? 'red' : 'blue'}>{v}</Tag> },
        { title: '状态', dataIndex: 'active', render: (v: boolean) => <Tag color={v ? 'green' : 'default'}>{v ? '正常' : '禁用'}</Tag> },
        { title: '注册时间', dataIndex: 'created_at', render: (v: string) => new Date(v).toLocaleString() },
        {
          title: '操作',
          render: (_, row) => (
            <Popconfirm
              title={row.active ? '确认禁用该用户？' : '确认启用该用户？'}
              onConfirm={async () => {
                await client.put(`/admin/users/${row.id}/active`, { active: !row.active })
                message.success('操作成功')
              }}
            >
              <Button type="link" danger={row.active}>{row.active ? '禁用' : '启用'}</Button>
            </Popconfirm>
          ),
        },
      ]}
    />
  )
}
```

- [ ] **Step 2: 超管全量稿件页面**

```typescript
// src/pages/admin/scripts/index.tsx
import { useState } from 'react'
import { ProTable } from '@ant-design/pro-components'
import { Button } from 'antd'
import client from '../../../api/client'
import ScriptDetail from '../../../components/script-detail'

export default function AdminScriptsPage() {
  const [detail, setDetail] = useState<any>(null)

  return (
    <>
      <ProTable
        headerTitle="全量稿件"
        rowKey="id"
        request={async ({ current, pageSize }) => {
          const res = await client.get('/admin/scripts', { params: { page: current, limit: pageSize } })
          return { data: res.data.data, total: res.data.total, success: true }
        }}
        columns={[
          { title: '标题', dataIndex: 'title', ellipsis: true },
          { title: '用户ID', dataIndex: 'user_id', width: 120, ellipsis: true },
          { title: '平台', dataIndex: 'platform', width: 100 },
          { title: '相似度', dataIndex: 'similarity_score', width: 100, render: (v: number) => `${(v*100).toFixed(1)}%` },
          { title: '爆款评分', dataIndex: 'viral_score', width: 100, render: (v: number) => `${v.toFixed(1)}/10` },
          { title: '创建时间', dataIndex: 'created_at', width: 180, render: (v: string) => new Date(v).toLocaleString() },
          { title: '操作', render: (_, row) => <Button type="link" onClick={() => setDetail(row)}>查看</Button> },
        ]}
      />
      <ScriptDetail script={detail} onClose={() => setDetail(null)} />
    </>
  )
}
```

- [ ] **Step 3: 数据看板页面**

```typescript
// src/pages/admin/dashboard/index.tsx
import { useEffect, useState } from 'react'
import { Card, Col, Row, Statistic } from 'antd'
import client from '../../../api/client'

export default function AdminDashboardPage() {
  const [stats, setStats] = useState({ totalScripts: 0, totalUsers: 0, avgViralScore: 0 })

  useEffect(() => {
    Promise.all([
      client.get('/admin/scripts', { params: { page: 1, limit: 1 } }),
      client.get('/admin/users', { params: { page: 1, limit: 1 } }),
    ]).then(([scripts, users]) => {
      setStats({
        totalScripts: scripts.data.total,
        totalUsers: users.data.total,
        avgViralScore: 0, // 后续扩展聚合接口
      })
    })
  }, [])

  return (
    <Row gutter={16}>
      <Col span={8}>
        <Card><Statistic title="累计稿件数" value={stats.totalScripts} /></Card>
      </Col>
      <Col span={8}>
        <Card><Statistic title="注册用户数" value={stats.totalUsers} /></Card>
      </Col>
      <Col span={8}>
        <Card><Statistic title="平均爆款评分" value={stats.avgViralScore} suffix="/10" precision={1} /></Card>
      </Col>
    </Row>
  )
}
```

- [ ] **Step 4: Commit**

```bash
cd /data/code/content_creator_imm
git add admin-frontend/
git commit -m "feat: add admin-side pages: user management, all scripts, dashboard"
```

---

## Task 6: 构建 + Go embed 集成

**Files:**
- Create: `server/admin.go`
- Modify: `server/main.go`

- [ ] **Step 1: 构建前端产物**

```bash
cd /data/code/content_creator_imm/admin-frontend && npm run build
# 产物输出到 ../server/admin/dist/
```

- [ ] **Step 2: 创建 server/admin.go**

```go
// server/admin.go
package main

import (
    "embed"
    "io/fs"
    "net/http"
    "github.com/gin-gonic/gin"
)

//go:embed admin/dist
var adminFS embed.FS

func serveAdmin(r *gin.Engine) {
    sub, _ := fs.Sub(adminFS, "admin/dist")
    fileServer := http.FileServer(http.FS(sub))
    r.NoRoute(func(c *gin.Context) {
        // SPA fallback: 所有非 /api 路由返回 index.html
        if len(c.Request.URL.Path) > 4 && c.Request.URL.Path[:4] == "/api" {
            c.Status(http.StatusNotFound)
            return
        }
        fileServer.ServeHTTP(c.Writer, c.Request)
    })
}
```

- [ ] **Step 3: 在 main.go 中调用 serveAdmin**

```go
// main.go 路由注册完成后追加
serveAdmin(r)
```

- [ ] **Step 4: 编译验证（含 embed）**

```bash
cd /data/code/content_creator_imm/server && go build ./...
```

Expected: 无报错，二进制包含前端资源

- [ ] **Step 5: 验证访问**

```bash
cd /data/code/content_creator_imm/server
docker compose up -d db && sleep 3
go run . &
# 访问 http://localhost:8080/login 看到登录页面
curl http://localhost:8080/api/v1/ping  # 仍然返回 {"status":"ok"}
kill %1 && docker compose down
```

- [ ] **Step 6: Commit**

```bash
cd /data/code/content_creator_imm
git add server/ admin-frontend/
git commit -m "feat: build frontend and embed into Go binary - Plan 3 complete"
```

---

## 验收标准

- [ ] `npm run build` 成功，产物在 `server/admin/dist/`
- [ ] `go build ./...` 成功，前端资源嵌入二进制
- [ ] 访问 `http://localhost:8080/login` 显示登录页
- [ ] 普通用户登录后看到"我的稿件"等 3 个菜单
- [ ] 管理员登录后看到全部 6 个菜单
- [ ] 用户管理页可启用/禁用用户
- [ ] 全量稿件页可查看稿件详情
- [ ] `/api/v1/ping` 仍正常响应（API 路由不被前端覆盖）

---

## 下一步

- **Plan 4**: Skill 改造（本地缓存层 + 远端 API 接入）
