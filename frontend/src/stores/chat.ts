import { defineStore } from 'pinia'
import { ref } from 'vue'
import { sendMessage as apiSendMessage, resetSession as apiResetSession } from '@/api/chat'

export type MsgRole = 'user' | 'assistant'

export interface ChatMessage {
  id: number
  role: MsgRole
  html: string
  rawText?: string
  streaming?: boolean
  [key: string]: unknown  // allow extra fields for outline/action/similarity
}

export interface OutlineData {
  outline?: Array<{ part: string; content: string; duration: string }>
  elements?: string[]
  estimated?: string
  strategy?: string
}

export interface SimilarityData {
  vocab: number
  sentence: number
  structure: number
  viewpoint: number
  total: number
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
  const justCompleted = ref(0)
  const messagesUpdated = ref(0)
  const currentConvId = ref(0)
  const lastSentText = ref('')   // keeps last user input for retry
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
    lastSentText.value = text
    addMessage('user', escapeHtml(text))
    let streamingMsg: ChatMessage | null = null

    try {
      const res = await apiSendMessage(text)
      if (!res.ok) {
        const err = await res.json()
        addMessage('assistant', `<span class="err-text">❌ ${escapeHtml((err as { error?: string }).error ?? '请求失败')}</span>`)
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
            const event = JSON.parse(line.slice(6)) as SSEEvent
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
      messagesUpdated.value++
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
        messages.value.push({ id: nextId++, role: 'assistant', html: '__outline__', outlineData: event.data })
        return streamingMsg
      case 'action':
        messages.value.push({ id: nextId++, role: 'assistant', html: '__action__', actionOptions: event.options })
        return streamingMsg
      case 'similarity':
        messages.value.push({ id: nextId++, role: 'assistant', html: '__similarity__', simData: event.data })
        return streamingMsg
      case 'complete':
        if (streamingMsg) { streamingMsg.streaming = false; streamingMsg = null }
        addMessage('assistant', `<span class="ok-text">✅ 稿件已保存！ID: ${event.scriptId}</span><br><span class="hint-text">输入新内容开始下一轮，或点击「新建对话」重置。</span>`)
        justCompleted.value++
        return null
      case 'error':
        if (streamingMsg) { streamingMsg.streaming = false; streamingMsg = null }
        addMessage('assistant', `<span class="err-text">❌ ${escapeHtml(event.message)}</span>`, { retryable: true })
        return null
    }
  }

  function retry() {
    if (lastSentText.value && !sending.value) {
      // Remove the last user message + error message pair before retrying
      const msgs = messages.value
      while (msgs.length && (msgs[msgs.length - 1].retryable || msgs[msgs.length - 1].role === 'user')) {
        msgs.pop()
      }
      send(lastSentText.value)
    }
  }

  async function reset() {
    const { data } = await apiResetSession()
    currentConvId.value = data.conv_id ?? 0
    messages.value = []
  }

  // Restore persisted messages from a conversation record
  function restoreMessages(storedRaw: string) {
    messages.value = []
    let stored: Array<{ role: string; type: string; content?: string; data?: unknown; options?: string[]; step?: number; name?: string }> = []
    try {
      const parsed = JSON.parse(storedRaw)
      if (!Array.isArray(parsed)) return
      stored = parsed
    } catch { return }
    for (const m of stored) {
      const role = (m.role === 'user' ? 'user' : 'assistant') as MsgRole
      switch (m.type) {
        case 'text':
          messages.value.push({ id: nextId++, role, html: renderMarkdown(m.content ?? ''), rawText: m.content ?? '' })
          break
        case 'step':
          messages.value.push({ id: nextId++, role: 'assistant', html: `<div class="step-badge">⚙️ Step ${m.step ?? ''}：${m.name ?? ''}</div>` })
          break
        case 'info':
          messages.value.push({ id: nextId++, role: 'assistant', html: `<div class="info-badge">ℹ️ ${m.content ?? ''}</div>` })
          break
        case 'outline':
          messages.value.push({ id: nextId++, role: 'assistant', html: '__outline__', outlineData: m.data })
          break
        case 'action':
          messages.value.push({ id: nextId++, role: 'assistant', html: '__action__', actionOptions: m.options })
          break
        case 'similarity':
          messages.value.push({ id: nextId++, role: 'assistant', html: '__similarity__', simData: m.data })
          break
        case 'complete':
          messages.value.push({ id: nextId++, role: 'assistant', html: '<span class="ok-text">✅ 对话已完成</span>' })
          break
        case 'error':
          messages.value.push({ id: nextId++, role: 'assistant', html: `<span class="err-text">❌ ${escapeHtml(m.content ?? '')}</span>` })
          break
      }
    }
  }

  function escapeHtml(s: string): string {
    return s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;')
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

  return { messages, sending, justCompleted, messagesUpdated, currentConvId, lastSentText, send, retry, reset, restoreMessages }
})
