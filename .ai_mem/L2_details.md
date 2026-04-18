# L2 — 代码实现细节

> ← [L1_modules.md](L1_modules.md)

---

## 会话状态机详细流程

### SendMessage 恢复机制（Phase 1）

```go
// SendMessage 开始时检查并恢复 session 状态
if activeWorkflowID == 0 {
    pausedWf := repository.GetActiveWorkflow(userID) // 查找 paused 状态的 workflow
    if pausedWf.Status == "paused" {
        activeWorkflowID = pausedWf.ID
        sess.ActiveWorkflowID = pausedWf.ID
        // 从 Workflow.ConvID 或 InputJSON 恢复 convID
        if pausedWf.ConvID != nil && *pausedWf.ConvID > 0 {
            convID = *pausedWf.ConvID
        } else {
            // Fallback: 从 InputJSON 解析
            var input WorkflowInput
            json.Unmarshal(pausedWf.InputJSON, &input)
            convID = input.ConvID
        }
        sess.ConvID = convID
        sess.SetState(StateAwaiting)
    }
}
```

### handleIdle（StateIdle → StateAnalyzing → StateAwaiting）

```go
1. sess.SetState(StateAnalyzing)
2. EnsureConversation(sess, title)          // 创建DB记录（若已存在则幂等）
3. UpdateConversationTitle(sess.ConvID, title)  // 更新为实际输入内容
4. addMsg(sess, user_msg)                   // 写内存 + 立即INSERT Message表
5. sendStep(1) + addMsg(step)
6. if IsURL(input): ExtractURL → sess.OriginalText
7. sendStep(2) → GetStyleByUserID → styleProfile
8. sendStep(3) → StreamClaude(analysisPrompt) → fullAnalysis
   每个token: sendToken + 写入fullAnalysis strings.Builder
9. addMsg(assistant, fullAnalysis)          // 全量分析文本落库
10. sendStep(5) → ParseOutlineFromAnalysis → outlineData
11. sendOutline(outlineData) + addMsg(outline)
12. sendAction(options) + addMsg(action)
13. sess.SetState(StateAwaiting)
14. FlushConversation(sess, 0, nil)         // 更新conversations.state=0
```

### handleAwaiting（StateAwaiting）

```
用户输入 → addMsg(user_msg)
"1" / "确认" → writeFinalDraft()
"4" / "重新" → SetState(StateIdle) + sendInfo
"2" / "调整" → sess.UserNote = input + sendInfo
"3" / "更换" → SetState(StateIdle) + sendInfo
其他         → sess.UserNote = input + sendInfo
每个分支末尾：FlushConversation(sess, 0, nil)
```

### writeFinalDraft（StateWriting → StateComplete）

```go
1. sess.SetState(StateWriting)
2. sendStep(6) → StreamClaude(finalDraftPrompt) → sess.FinalDraft
3. addMsg(assistant, finalDraft)
4. sendStep(8) → CallClaude(simPrompt, 256tokens) → simResult
5. parse JSON → simScore; sendSimilarity + addMsg(similarity)
6. sendStep(9) → SaveScript(userID, sess, simScore, 7.5)
   → os.WriteFile(path, stripQualityCheck(draft))
   → repository.CreateScript
   → FlushConversation(sess, 1, &script.ID)  // 标记完成，关联script
7. sendComplete(script.ID) + addMsg(complete)
8. sess.SetState(StateComplete)
```

---

## Prompt 关键约定

### 分析Prompt输出格式
- 5个角色 → 辩论决策 → **`---OUTLINE_START---{JSON}---OUTLINE_END---`**
- OutlineData JSON结构：
```json
{
  "elements": ["要素1"...],
  "materials": ["素材1"...],
  "outline": [{"part":"开场","duration":"3s","content":"...","emotion":"..."}...],
  "estimated_similarity": "约15%",
  "strategy": "改写核心策略"
}
```

### 终稿Prompt输出格式
- 正文直接输出
- 末尾：`---QUALITY_CHECK_START---` ... `---QUALITY_CHECK_END---`
- `stripQualityCheck()` 在保存前截断该段

### 相似度Prompt输出格式
```json
{"vocab":20,"sentence":15,"structure":18,"viewpoint":10,"total":16.25}
```
`total = vocab*0.30 + sentence*0.25 + structure*0.25 + viewpoint*0.20`

---

## 消息持久化机制

### 两层存储
1. **内存层**：`ChatSession.StoredMsgs []StoredMsg`（用于会话内快速访问）
2. **DB层**：`messages` 表，每条消息通过 `addMsg()` helper 立即 INSERT

### addMsg helper
```go
func addMsg(sess *service.ChatSession, msg service.StoredMsg) {
    sess.AddMsg(msg)                          // 写内存
    service.PersistMsg(sess.ConvID, msg)      // 立即INSERT到messages表
}
```

### FlushConversation（简化版）
仅更新 `conversations.state` 和 `conversations.script_id`，不再写messages JSON。
```go
repository.UpdateConversationMeta(id, map[string]interface{}{"state": state})
```

### 会话恢复（GetConversationDetail）
```go
msgs := repository.ListMessagesByConvID(convID)  // 按id ASC排序
for m := range msgs {
    sm := service.StoredMsg{...}
    // DataJSON → sm.Data (json.RawMessage)
    // OptionsJSON → sm.Options ([]string)
    storedMsgs = append(storedMsgs, sm)
}
return json.Marshal(storedMsgs)  // 返回给前端
```

---

## 前端消息渲染

### ChatMessage 结构
```typescript
interface ChatMessage {
  id: number
  role: 'user' | 'assistant'
  html: string
  rawText?: string
  streaming?: boolean
  retryable?: boolean     // error消息专用，显示重试按钮
  outlineData?: OutlineData
  actionOptions?: string[]
  simData?: SimilarityData
}
```

### restoreMessages（从DB恢复）
```typescript
JSON.parse(storedRaw) → Array
type映射：
  text   → html = renderMarkdown(content)
  step   → html = <div class="step-badge">⚙️ Step N：name</div>
  info   → html = <div class="info-badge">ℹ️ content</div>
  outline → { outlineData: data }
  action  → { actionOptions: options }
  similarity → { simData: data }
  complete → html = <span class="ok-text">✅ 对话已完成</span>
  error   → html = <span class="err-text">❌ content</span>
```

### 重试机制
```typescript
retry() {
  // 移除末尾的 user + retryable error 消息
  while (msgs.length && (msgs.last.retryable || msgs.last.role === 'user')) pop()
  send(lastSentText.value)  // 重新发送
}
```

### 会话切换逻辑（Home.vue）
```typescript
loadConversation(conv) {
  if (chatStore.sending && conv.id === chatStore.currentConvId)
    → 跳过（流式中，保留内存消息）
  else
    → getConversation(id) → restoreMessages(data.messages)
}
```

---

## LLM 接口实现细节

### StreamClaude
- 协议：Anthropic SSE格式，监听 `content_block_delta` 事件
- Scanner buffer: 64KB（防止长行截断）
- 每行格式：`data: {...}` 或 `data: [DONE]`

### CallClaude
- 非流式，直接读取 `content[0].text`
- 相似度检测用 maxTokens=256（节省响应时间）

---

## nginx 配置要点

```nginx
location /creator/api/ {
    proxy_pass http://127.0.0.1:3004/api/;
    proxy_buffering off;          # SSE必须关闭buffering
    proxy_cache off;
    proxy_set_header X-Accel-Buffering no;
    proxy_read_timeout 300s;      # 匹配后端HTTP client超时
}
location /creator/ {
    alias /data/code/content_creator_imm/frontend/dist/;
    try_files $uri $uri/ /creator/index.html;  # SPA fallback
}
```

---

## 扩展指引

### 新增后端接口
1. 在 `handler/` 下添加 handler 函数
2. 在 `main.go` 的 `api` 路由组注册
3. 如需新DB表：在 `model/` 添加 struct，`repository/` 添加CRUD，`db/db.go` AutoMigrate添加

### 新增前端页面/功能
1. `api/` 添加对应接口封装
2. 如需全局状态：在 `stores/` 添加 Pinia store
3. 组件放 `components/`，页面放 `views/`

### 新增 Workflow Stage
1. 在 `workflows/viral_script/workflow.yaml` 添加新阶段定义
2. 在 `workflows/viral_script/prompts/` 创建 worker prompt YAML
3. 如需条件跳过：设置 `skip_if` 表达式
4. 如需新 SSE event type：更新 `sse.go` 和前端处理
5. 如需静默执行（不流式输出）：在 prompt YAML 添加 `silent_output: true`

### skip_if 条件语法
```yaml
skip_if: "{{stage.X.worker.Y.output.field}} == false"
```
- 支持 `==` 和 `!=` 操作符
- 变量值从 `buildVarsMap` 解析
- JSON 字段自动提取（如 `need_material` 从 `{"need_material": true}`）

### silent_output 控制流式输出
```yaml
# prompt YAML 中添加
silent_output: true  # 不发送 worker_token，只显示进度
```
- 默认：流式输出（发送 worker_token SSE）
- `silent_output: true`：静默执行，只发送 worker_start 和 worker_done
- 适用场景：中间分析阶段，用户不需要看到详细过程
- draft_writer 保持流式输出，用户可实时看到稿子生成

### 构建与部署
```bash
# Go代码改动后
cd backend && go build -o ../content-creator-imm .
./manage.sh restart

# 前端改动后
cd frontend && npm run build
# 验证
curl -s -o /dev/null -w "HTTP %{http_code}\n" http://localhost/creator/
```
