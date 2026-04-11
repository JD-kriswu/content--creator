# 飞书集成设计文档

> 日期：2026-04-11
> 作者：Claude Code

---

## 概述

**目标**：用户通过飞书扫码创建机器人，通过飞书聊天使用口播稿创作服务。

**核心技术选型**：
- **App Manifest**：扫码即创建机器人（参考 EasyClaw）
- **WebSocket 模式**：接收飞书消息事件，无需公网 Webhook
- **飞书 Card**：实现流式输出效果，实时推送生成内容

---

## 整体架构

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         Go Backend (:3004)                                   │
│                                                                              │
│  ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────────────┐ │
│  │  Web Handler    │    │  Feishu Module  │    │  Workflow Engine        │ │
│  │  (现有)         │    │  (新增)         │    │  (复用)                 │ │
│  │                 │    │                 │    │                         │ │
│  │  • SSE 推送     │    │  • WS Client    │    │  • loader.go            │ │
│  │  • Gin 路由     │    │  • Card Pusher  │    │  • engine.go            │ │
│  │                 │    │  • 消息路由     │    │  • context.go           │ │
│  └────────────┬────┘    └────────────┬────┘    │  • sse.go               │ │
│               │                      │         └────────────┬────────────┘ │
│               │                      │                      │              │
│  ┌────────────▼──────────────────────▼──────────────────────▼────────────┐ │
│  │                     Service 层 (复用 + 扩展)                           │ │
│  │                                                                        │ │
│  │  • llm_service.go      (复用：LLM调用)                                │ │
│  │  • pipeline.go         (复用：Session状态机)                          │ │
│  │  • prompts.go          (复用：Prompt构建)                             │ │
│  │  • extractor.go        (复用：URL提取)                                │ │
│  │  • feishu_service.go   (新增：飞书API封装)                            │ │
│  │  • feishu_session.go   (新增：飞书会话管理)                           │ │
│  └───────────────────────────────────────────────────────────────────────┘ │
│                                                                              │
│  ┌───────────────────────────────────────────────────────────────────────┐ │
│  │                     Repository 层 (扩展)                               │ │
│  │                                                                        │ │
│  │  现有表: users, user_styles, conversations, messages, scripts         │ │
│  │  新增表: feishu_bots, feishu_users, feishu_conversations              │ │
│  └───────────────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 新增模块说明

| 模块 | 文件 | 职责 |
|------|------|------|
| `WS Client` | `internal/feishu/ws_client.go` | WebSocket 连接管理，接收飞书消息事件 |
| `Card Pusher` | `internal/feishu/card_pusher.go` | 飞书 Card 消息推送，实现流式输出效果 |
| `消息路由` | `internal/feishu/router.go` | 解析飞书消息，路由到 workflow engine |
| `飞书 Service` | `internal/service/feishu_service.go` | 飞书 API 封装（创建 Card、更新 Card、发送消息） |
| `飞书 Session` | `internal/service/feishu_session.go` | 飞书用户会话管理 |

---

## 数据模型设计

### feishu_bots 表（飞书机器人实例）

```go
type FeishuBot struct {
    ID            uint      `gorm:"primaryKey"`
    UserID        uint      `gorm:"index;not null"`           // 关联的 Web 用户 ID
    AppID         string    `gorm:"size:64;unique;not null"`  // 飞书 App ID
    AppSecret     string    `gorm:"size:128;not null"`        // 飞书 App Secret（加密存储）
    TenantKey     string    `gorm:"size:64"`                  // 租户标识
    BotName       string    `gorm:"size:128"`                 // 机器人名称
    WSConnected   bool      `gorm:"default:false"`            // WebSocket 连接状态
    LastHeartbeat time.Time                                   // 最后心跳时间
    CreatedAt     time.Time
    UpdatedAt     time.Time
}
```

### feishu_users 表（飞书用户身份映射）

```go
type FeishuUser struct {
    ID          uint   `gorm:"primaryKey"`
    FeishuID    string `gorm:"size:64;unique;not null"`  // 飞书 open_id
    OpenID      string `gorm:"size:64;unique"`           // 飞书 open_id
    UnionID     string `gorm:"size:64"`                  // 飞书 union_id
    UserID      uint   `gorm:"index"`                    // 关联的 Web 用户 ID（可为空）
    BindStatus  string `gorm:"size:20;default:'independent'"` // independent/merged
    CreatedAt   time.Time
    UpdatedAt   time.Time
}
```

### feishu_conversations 表（飞书会话映射）

```go
type FeishuConversation struct {
    ID           uint   `gorm:"primaryKey"`
    BotID        uint   `gorm:"index;not null"`           // 关联的飞书 Bot
    ConvID       uint   `gorm:"index;not null"`           // 关联的 conversations 表 ID
    FeishuChatID string `gorm:"size:64;unique;not null"`  // 飞书聊天窗口 ID
    CreatedAt    time.Time
    UpdatedAt    time.Time
}
```

---

## WebSocket 连接管理

### WSConnectionPool（连接池）

```go
type WSConnectionPool struct {
    connections map[string]*WSConnection  // appID → connection
    mu          sync.RWMutex
}

type WSConnection struct {
    AppID            string
    AppSecret        string
    Conn             *websocket.Conn
    Status           string  // connected/disconnected/reconnecting
    MessageHandler   func(event FeishuEvent)
    HeartbeatTimer   *time.Timer
    ReconnectAttempts int
    MaxReconnectAttempts int
}
```

### 连接生命周期

```
服务启动 ──────────────────────────────────────────────────────────────────────
│
│  1. 从 feishu_bots 表加载所有 ws_connected=true 的 Bot
│  2. 对每个 Bot 调用 WSConnectionPool.Connect()
│  3. 启动心跳检测定时器 (每 30s)
│
用户扫码创建新 Bot ────────────────────────────────────────────────────────────
│
│  1. 用户扫码 → 飞书回调返回 app_id, app_secret
│  2. 创建 feishu_bots 记录
│  3. WSConnectionPool.Connect(app_id, app_secret)
│  4. 注册消息处理器
│
心跳检测 ──────────────────────────────────────────────────────────────────────
│
│  每 30s 检查所有连接:
│  • 发送心跳包
│  • 超时未响应 → 标记 disconnected → 触发重连
│  • 重连失败 3 次 → 标记 disconnected，等待下次消息触发重连
│
服务重启 ──────────────────────────────────────────────────────────────────────
│
│  1. 从 feishu_bots 表读取所有 Bot
│  2. 自动重建 WebSocket 连接
│  3. 恢复暂停的 workflow
```

---

## 飞书 Card 流式输出

### SSE 事件 →飞书 Card 更新映射

| SSE 事件 | 飞书 Card 更新动作 |
|----------|-------------------|
| `stage_start` | 更新 header.title + 显示阶段名称 |
| `worker_start` | 显示 worker 名称 + "正在执行..." |
| `worker_token` | 追加 token 内容到流式文本元素 |
| `worker_done` | 显示 "✅ 完成" |
| `outline` | 创建折叠元素展示大纲数据 |
| `action` | 创建交互按钮元素 |
| `similarity` | 创建表格元素展示相似度数据 |
| `complete` | 更新 header.template=green + 显示完成消息 |
| `error` | 更新 header.template=red + 显示错误消息 |

### CardBuilder（Card 内容构建器）

```go
type CardBuilder struct {
    currentStage      string
    streamingContent  strings.Builder
    outlineData       *OutlineData
    actionOptions     []string
    simData           *SimilarityData
}

func (b *CardBuilder) Build() FeishuCardJSON {
    // 构建飞书 Card JSON 结构
}
```

### CardPusher（Card API 调用）

```go
type CardPusher struct {
    feishuService    *FeishuService
    updateThrottle   time.Duration  // 200ms
    pendingUpdates   map[string]*CardBuilder
    updateTimer      *time.Timer
}

func (p *CardPusher) UpdateCard(cardID string, cardJSON FeishuCardJSON) {
    // throttled update，避免频繁调用飞书 API
}
```

---

## App Manifest 扫码创建流程

### App Manifest 配置（feishu_manifest.yaml）

```yaml
app:
  name: "口播稿助手"
  description: "AI驱动的爆款口播稿改写工具"
  
permissions:
  - im:message:receive_as_bot
  - im:message:send_as_bot
  - im:card
  - contact:user.base:readonly
  
events:
  - im.message.receive_v1
  - card.action.trigger
  
event_subscription:
  type: websocket
```

### 扫码创建流程

```
步骤1：用户访问 Web 端「绑定飞书」页面
│  GET /api/feishu/bind-qrcode
│  返回: { qrcode_url: "https://...", bind_token: "uuid-xxx" }

步骤2：用户扫码确认
│  飞书创建自建应用

步骤3：飞书回调
│  WebSocket 推送事件: { type: "app_manifest.created", app_id, app_secret, bind_token }

步骤4：服务端处理回调
│  1. 验证 bind_token
│  2. 创建 feishu_bots 记录
│  3. 建立 WebSocket 连接
│  4. 推送绑定成功消息

步骤5：用户开始飞书聊天
│  创建 feishu_users 记录 → 开始 workflow
```

---

## 消息处理与用户隔离

### MessageRouter（消息路由）

```
飞书推送事件 ──────────────────────────────────────────────────────────────────
│  { app_id, tenant_key, event: { sender: { open_id }, message: { chat_id, content } } }
│
└───────────┬─────────────────────────────────────────────────────────────────────
            │
┌───────────▼───────────────────────────────────────────────────────────────────┐
│                  MessageRouter                                                │
│                                                                                │
│  1. 根据 app_id 查找 feishu_bots → 获取 bot_id, user_id                       │
│  2. 根据 open_id 查找/创建 feishu_users → 获取/创建飞书用户                   │
│  3. 根据 chat_id 查找/创建 feishu_conversations → 获取 conv_id               │
│  4. 构建 FeishuSessionContext                                                  │
│  5. 调用 workflow engine.Start() 或 engine.Resume()                          │
│  6. 注册 SSE 回调 → 转换为飞书 Card 更新                                       │
└───────────────────────────────────────────────────────────────────────────────┘
```

### FeishuSessionManager（飞书会话管理）

```go
type FeishuSessionManager struct {
    sessions map[string]*FeishuSession  // chatID → session
    mu       sync.RWMutex
}

type FeishuSession struct {
    ChatID      string
    ConvID      uint
    WorkflowID  uint
    CardID      string
    State       string  // idle/analyzing/awaiting/writing
    Lock        sync.Mutex
}
```

---

## API 接口设计

| 接口 | 方法 | 说明 |
|------|------|------|
| `/api/feishu/bind-qrcode` | GET | 生成绑定二维码（需认证） |
| `/api/feishu/bind-status` | GET | 查询绑定状态（需认证） |
| `/api/feishu/unbind` | DELETE | 解绑飞书机器人（需认证） |
| `/api/feishu/bots` | GET | 获取用户绑定的飞书机器人列表（需认证） |

---

## 错误处理

| 错误类型 | 处理策略 |
|----------|----------|
| WebSocket 连接断开 | 自动重连（最多3次），超过后等待下次消息触发 |
| 飞书 API 调用失败 | 重试（最多2次），失败后发送错误 Card |
| LLM 调用超时 | 发送错误 Card，用户可重试 |
| Workflow 执行失败 | 发送错误 Card，清理 session |
| 用户发送无效输入 | 发送提示 Card |
| URL 提取失败 | 发送提示 Card，建议直接输入文案 |

---

## 用户合并流程

Web 端「绑定飞书」功能：

```
1. 验证飞书身份（通过 open_id）
2. 更新 feishu_users.user_id = web_user.id
3. 更新 feishu_bots.user_id = web_user.id
4. 迁移历史数据：
   • conversations: 保持不变，通过 user_id 关联
   • scripts: 保持不变，通过 user_id 关联
   • messages: 保持不变，通过 conv_id 关联
5. 更新 feishu_users.bind_status = "merged"
```

---

## 配置新增

```json
// backend/config.json 新增字段
{
  "feishu_enabled": true,
  "feishu_manifest_path": "feishu_manifest.yaml",
  "feishu_ws_reconnect_max": 3,
  "feishu_ws_heartbeat_interval": 30,
  "feishu_card_update_throttle": 200
}
```

---

## 测试方案

| 测试类型 | 测试内容 | 测试方法 |
|----------|----------|----------|
| 单元测试 | WebSocket 连接管理、CardBuilder 构建 | Go test + mock |
| 集成测试 | 扫码创建流程、消息处理流程 | 测试环境 + 飞书测试机器人 |
| E2E 测试 | 完整创作流程 | 真实飞书机器人 + 手动测试 |
| 压力测试 | 多用户并发、连接池稳定性 | 模拟多用户消息推送 |

---

## 部署计划

```
Phase 1: 后端模块部署
│  1. 添加飞书配置
│  2. 部署新代码
│  3. 数据库迁移
│  4. 验证 WebSocket

Phase 2: Web 端绑定页面
│  1. 添加飞书绑定页面
│  2. 构建前端
│  3. nginx reload

Phase 3: 测试验证
│  1. 扫码绑定测试
│  2. 飞书聊天测试
│  3. 流式输出验证
│  4. 错误处理验证

Phase 4: 生产上线
│  1. 配置生产参数
│  2. 部署生产服务器
│  3. 监控连接状态
│  4. 用户文档更新
```

---

## 文件清单

### 新增文件

```
backend/
├── feishu_manifest.yaml                 # App Manifest 配置
├── internal/
│   ├── feishu/
│   │   ├── ws_client.go                 # WebSocket 连接管理
│   │   ├── ws_pool.go                   # 连接池
│   │   ├── card_pusher.go               # Card 推送
│   │   ├── card_builder.go              # Card 构建
│   │   ├── router.go                    # 消息路由
│   │   └── types.go                     # 飞书事件类型定义
│   ├── service/
│   │   ├── feishu_service.go            # 飞书 API 封装
│   │   └ feishu_session.go              # 飞书会话管理
│   ├── model/
│   │   ├── feishu_bot.go                # FeishuBot 模型
│   │   ├── feishu_user.go               # FeishuUser 模型
│   │   └── feishu_conversation.go       # FeishuConversation 模型
│   ├── repository/
│   │   ├── feishu_bot_repo.go           # FeishuBot CRUD
│   │   ├── feishu_user_repo.go          # FeishuUser CRUD
│   │   └── feishu_conv_repo.go          # FeishuConversation CRUD
│   └── handler/
│       └── feishu_handler.go            #飞书绑定 API handler

frontend/
└── src/
    ├── pages/
    │   └── FeishuBind.tsx                # 飞书绑定页面
    ├── api/
    │   └── feishu.ts                     # 飞书 API 封装
    └── components/
        └── FeishuQRCode.tsx              # 二维码显示组件
```

### 修改文件

```
backend/
├── main.go                               # 添加飞书路由
├── config/config.go                      # 添加飞书配置字段
├── internal/db/db.go                     # AutoMigrate 新增表
├── internal/workflow/engine.go           # 添加飞书 SSE 回调支持

frontend/
├── src/router.tsx                        # 添加飞书绑定页面路由
└── src/components/Sidebar.tsx            # 添加飞书绑定入口
```