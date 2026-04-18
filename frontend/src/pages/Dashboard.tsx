import { useRef, useState, useEffect, useCallback, useReducer } from 'react'
import { ArrowUp, ArrowLeft, Plus, Sparkles } from 'lucide-react'
import { toast } from 'sonner'
import { Sidebar } from '../components/Sidebar'
import { MessageList, type ChatMsg } from '../components/create/MessageList'
import { ChatInput } from '../components/create/ChatInput'
import { OutlineEditor } from '../components/create/OutlineEditor'
import { ScriptEditor } from '../components/create/ScriptEditor'
import { sendMessage, resetSession } from '../api/chat'
import { getConversation, type Conversation } from '../api/conversations'
import { getScript } from '../api/scripts'
import { parseSSELine } from '../lib/sse'
import { getStyleDoc } from '../api/style'
import { StyleInitBanner } from '../components/StyleInitBanner'

function formatOutline(data: unknown): string {
  if (!data || typeof data !== 'object') return JSON.stringify(data, null, 2)
  const d = data as Record<string, unknown>
  const lines: string[] = []

  if (Array.isArray(d.elements) && d.elements.length > 0) {
    lines.push('【爆款元素】')
    ;(d.elements as string[]).forEach((e, i) => lines.push(`  ${i + 1}. ${e}`))
    lines.push('')
  }

  if (Array.isArray(d.outline) && d.outline.length > 0) {
    lines.push('【内容大纲】')
    ;(d.outline as Array<{ part?: string; duration?: string; content?: string; emotion?: string }>).forEach((o) => {
      lines.push(`  ${o.part ?? ''}（${o.duration ?? ''}）`)
      if (o.content) lines.push(`    内容：${o.content}`)
      if (o.emotion) lines.push(`    情绪：${o.emotion}`)
    })
    lines.push('')
  }

  if (d.strategy) {
    lines.push('【创作策略】')
    lines.push(`  ${d.strategy}`)
    lines.push('')
  }

  if (d.estimated_similarity) {
    lines.push(`【预估相似度】${d.estimated_similarity}`)
  }

  return lines.join('\n').trim()
}

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
  | { type: 'UPDATE_LAST_MSG'; content: string }
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
    case 'UPDATE_LAST_MSG': {
      const msgs = [...state.messages]
      const last = msgs[msgs.length - 1]
      if (last) {
        msgs[msgs.length - 1] = { ...last, content: action.content }
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
  const [styleInitialized, setStyleInitialized] = useState<boolean | null>(null)
  const streamingTextRef = useRef('')
  const streamingMsgIdRef = useRef<string | null>(null)
  const flushTimerRef = useRef<number | null>(null)

  // Check style initialization on mount
  useEffect(() => {
    getStyleDoc()
      .then((doc) => setStyleInitialized(doc.is_initialized))
      .catch(() => setStyleInitialized(false))
  }, [])

  // Flush streaming content to state (called periodically)
  const flushStreamingContent = useCallback(() => {
    if (streamingMsgIdRef.current && streamingTextRef.current) {
      const QUALITY_MARKER = '---QUALITY_CHECK_START---'
      const OUTLINE_START = '---OUTLINE_START---'
      const OUTLINE_END = '---OUTLINE_END---'
      let content = streamingTextRef.current
      // Strip inline outline block
      const osIdx = content.indexOf(OUTLINE_START)
      if (osIdx !== -1) {
        const oeIdx = content.indexOf(OUTLINE_END)
        content = oeIdx !== -1
          ? (content.slice(0, osIdx) + content.slice(oeIdx + OUTLINE_END.length)).trimStart()
          : content.slice(0, osIdx)
      }
      // Strip quality check section
      const markerIdx = content.indexOf(QUALITY_MARKER)
      if (markerIdx !== -1) content = content.slice(0, markerIdx)
      dispatch({ type: 'UPDATE_LAST_MSG', content })
    }
  }, [])

  // Schedule a flush
  const scheduleFlush = useCallback(() => {
    if (flushTimerRef.current) return
    flushTimerRef.current = window.setTimeout(() => {
      flushStreamingContent()
      flushTimerRef.current = null
    }, 100) // Flush every 100ms
  }, [flushStreamingContent])

  const runSSE = useCallback(async (message: string, convId?: number) => {
    streamingTextRef.current = ''
    streamingMsgIdRef.current = null
    if (flushTimerRef.current) {
      clearTimeout(flushTimerRef.current)
      flushTimerRef.current = null
    }

    try {
      const res = await sendMessage(message, convId)
      if (!res.ok) {
        const err = await res.json().catch(() => ({ error: '请求失败' }))
        dispatch({ type: 'ADD_MSG', msg: { id: `${Date.now()}`, type: 'error', content: (err as { error?: string }).error ?? '请求失败' } })
        dispatch({ type: 'SET_STAGE', stage: 'idle' })
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
              if (!streamingMsgIdRef.current) {
                // Start a new streaming message if not already
                streamingMsgIdRef.current = `${Date.now()}-t`
                dispatch({ type: 'ADD_MSG', msg: { id: streamingMsgIdRef.current, type: 'ai', content: '', streaming: true } })
              }
              streamingTextRef.current += event.content
              scheduleFlush()
              break
            case 'step':
              dispatch({ type: 'ADD_MSG', msg: { id: `${Date.now()}-s`, type: 'step', content: `Step ${event.step}：${event.name}` } })
              break
            case 'info':
              dispatch({ type: 'ADD_MSG', msg: { id: `${Date.now()}-i`, type: 'info', content: event.content } })
              break
            case 'outline':
              dispatch({ type: 'SET_OUTLINE', text: formatOutline(event.data) })
              dispatch({ type: 'ADD_MSG', msg: { id: `${Date.now()}-o`, type: 'outline', data: event.data } })
              break
            case 'action':
              dispatch({ type: 'ADD_MSG', msg: { id: `${Date.now()}-a`, type: 'action', options: event.options } })
              break
            case 'similarity':
              dispatch({ type: 'ADD_MSG', msg: { id: `${Date.now()}-sim`, type: 'similarity', data: event.data } })
              break
            case 'final_draft':
              streamingTextRef.current = event.content
              dispatch({ type: 'SET_SCRIPT', text: event.content, scriptId: null })
              break
            case 'complete': {
              if (flushTimerRef.current) {
                clearTimeout(flushTimerRef.current)
                flushTimerRef.current = null
              }
              flushStreamingContent()
              const QUALITY_MARKER = '---QUALITY_CHECK_START---'
              const idx = streamingTextRef.current.indexOf(QUALITY_MARKER)
              const cleanText = idx !== -1
                ? streamingTextRef.current.slice(0, idx).trimEnd()
                : streamingTextRef.current
              dispatch({ type: 'SET_SCRIPT', text: cleanText, scriptId: event.scriptId })
              setRefreshTrigger((n) => n + 1)
              break
            }
            case 'error':
              dispatch({ type: 'ADD_MSG', msg: { id: `${Date.now()}-e`, type: 'error', content: event.message } })
              break
            // Worker events - treat as normal streaming text for serial stages
            case 'stage_start':
              dispatch({ type: 'ADD_MSG', msg: { id: `${Date.now()}-ss`, type: 'info', content: `▶ ${event.stage_name}` } })
              break
            case 'stage_done':
              break
            case 'worker_start':
              dispatch({ type: 'ADD_MSG', msg: { id: `${Date.now()}-ws`, type: 'step', content: event.worker_display } })
              // Start streaming message
              streamingTextRef.current = ''
              streamingMsgIdRef.current = `${Date.now()}-w`
              dispatch({ type: 'ADD_MSG', msg: { id: streamingMsgIdRef.current, type: 'ai', content: '', streaming: true } })
              break
            case 'worker_token':
              streamingTextRef.current += event.content
              scheduleFlush()
              break
            case 'worker_done':
              if (flushTimerRef.current) {
                clearTimeout(flushTimerRef.current)
                flushTimerRef.current = null
              }
              flushStreamingContent()
              dispatch({ type: 'STREAM_DONE' })
              streamingMsgIdRef.current = null
              break
            case 'synth_start':
              dispatch({ type: 'ADD_MSG', msg: { id: `${Date.now()}-syn`, type: 'info', content: '综合分析...' } })
              break
            case 'synth_token':
              streamingTextRef.current += event.content
              scheduleFlush()
              break
            case 'synth_done':
              if (flushTimerRef.current) {
                clearTimeout(flushTimerRef.current)
                flushTimerRef.current = null
              }
              flushStreamingContent()
              dispatch({ type: 'STREAM_DONE' })
              break
          }
        }
      }
    } catch (e) {
      const msg = e instanceof Error ? e.message : '连接失败'
      dispatch({ type: 'ADD_MSG', msg: { id: `${Date.now()}-e`, type: 'error', content: msg } })
    } finally {
      if (flushTimerRef.current) {
        clearTimeout(flushTimerRef.current)
        flushTimerRef.current = null
      }
      flushStreamingContent()
      dispatch({ type: 'STREAM_DONE' })
      setRefreshTrigger((n) => n + 1)
    }
  }, [flushStreamingContent, scheduleFlush])

  const handleSend = useCallback(async (text: string) => {
    if (state.sending) return
    dispatch({ type: 'SEND', text })
    await runSSE(text, activeConvId)
  }, [state.sending, runSSE, activeConvId])

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
      const stored = JSON.parse(data.messages || '[]') as Array<{
        role: string; type: string; content?: string; data?: unknown;
        options?: string[]; step?: number; name?: string
      }>
      const QUALITY_MARKER = '---QUALITY_CHECK_START---'
      const msgs: ChatMsg[] = stored
        .filter(m => m.type !== 'complete')
        .map((m, i) => {
          let content: string | undefined
          if (m.type === 'step') {
            content = `Step ${m.step}：${m.name}`
          } else if (m.content) {
            const idx = m.content.indexOf(QUALITY_MARKER)
            content = idx !== -1 ? m.content.slice(0, idx).trimEnd() : m.content
          }
          return {
            id: `restore-${i}`,
            type: (m.role === 'user' ? 'user'
              : m.type === 'action' ? 'action'
              : m.type === 'error' ? 'error'
              : m.type === 'step' ? 'step'
              : m.type === 'info' ? 'info'
              : m.type === 'similarity' ? 'similarity'
              : m.type === 'outline' ? 'outline'
              : 'ai') as ChatMsg['type'],
            content,
            options: m.options,
            data: m.data,
          }
        })

      const fullConv = data.conversation
      const stage: Stage = fullConv.state === 1 ? 'complete' : 'idle'
      dispatch({ type: 'RESTORE', messages: msgs, stage })
      setActiveConvId(conv.id)

      const outlineMsg = stored.find(m => m.type === 'outline')
      if (outlineMsg?.data && fullConv.state !== 1) {
        dispatch({ type: 'SET_OUTLINE', text: formatOutline(outlineMsg.data) })
      }

      if (fullConv.state === 1 && fullConv.script_id) {
        try {
          const scriptData = await getScript(fullConv.script_id)
          dispatch({ type: 'SET_SCRIPT', text: scriptData.content, scriptId: fullConv.script_id })
        } catch { /* script load failed */ }
      }
    } catch { toast.error('加载会话失败') }
  }

  const handleAction = useCallback((option: string) => {
    dispatch({ type: 'SET_STAGE', stage: 'writing' })
    handleSend(option)
  }, [handleSend])

  // Idle state: sidebar + centered textarea
  if (state.stage === 'idle') {
    return (
      <div className="h-full flex overflow-hidden">
        <Sidebar
          onNewChat={handleNewChat}
          onSelectConversation={handleSelectConversation}
          activeConvId={activeConvId}
          refreshTrigger={refreshTrigger}
        />
        <div className="flex-1 overflow-y-auto bg-gradient-to-br from-blue-50 via-white to-purple-50 dark:from-gray-950 dark:via-gray-900 dark:to-gray-950">
          {styleInitialized === false && (
            <StyleInitBanner onInitialized={() => setStyleInitialized(true)} />
          )}
          <div className="max-w-3xl mx-auto px-4 pt-16">
            <div className="text-center mb-8">
              <h1 className="text-4xl sm:text-5xl font-medium mb-3 text-gray-900 dark:text-gray-100">
                Hi，今天想创作什么爆款文案？
              </h1>
              <p className="text-lg text-gray-500 dark:text-gray-400 mb-4">粘贴你的参考口播稿，AI 会学习风格并为你创作</p>
              <div className="inline-flex items-center gap-2 px-4 py-2 bg-white/70 dark:bg-gray-800/70 rounded-full border border-blue-200 dark:border-blue-800 shadow-sm">
                <Sparkles className="w-4 h-4 text-blue-600 dark:text-blue-400" />
                <span className="text-sm font-medium text-blue-700 dark:text-blue-300">越用越懂你</span>
              </div>
            </div>
            <div className="relative mb-4 shadow-lg rounded-2xl">
              <textarea
                value={initialInput}
                onChange={(e) => setInitialInput(e.target.value)}
                placeholder="粘贴你喜欢的爆款口播稿..."
                className="w-full h-[200px] p-6 pr-6 pb-16 text-base border-0 bg-white dark:bg-gray-900 text-gray-900 dark:text-gray-100 rounded-2xl resize-none focus:outline-none transition-all"
              />
              <button
                onClick={handleInitialCreate}
                disabled={!initialInput.trim() || initialInput.length < 10}
                className="absolute bottom-4 right-4 w-10 h-10 bg-gradient-to-br from-blue-500 to-purple-600 rounded-full flex items-center justify-center shadow-lg hover:scale-105 hover:shadow-xl transition-all disabled:opacity-40 disabled:cursor-not-allowed disabled:hover:scale-100"
              >
                <ArrowUp className="w-5 h-5 text-white" strokeWidth={2.5} />
              </button>
            </div>
            <p className="text-sm text-gray-400 px-2">{initialInput.length} 字</p>
            <div className="text-center pb-16 mt-8 text-sm text-gray-400">
              提示：提供参考文案可以帮助 AI 更好地理解你想要的风格
            </div>
          </div>
        </div>
      </div>
    )
  }

  // Chat state: left chat panel + right preview
  return (
    <div className="h-full flex overflow-hidden bg-gray-100 dark:bg-gray-950 gap-4 p-4">
      {/* Left: chat panel */}
      <div className="w-full md:w-1/2 flex flex-col bg-white dark:bg-gray-900 rounded-xl shadow-sm border border-gray-200 dark:border-gray-800 overflow-hidden">
        {/* Top toolbar */}
        <div className="flex-shrink-0 flex items-center justify-between px-4 py-3 border-b border-gray-200 dark:border-gray-800">
          <button
            data-testid="back-to-idle-btn"
            onClick={() => dispatch({ type: 'RESET' })}
            className="flex items-center gap-1.5 text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-200 transition-colors"
          >
            <ArrowLeft className="w-4 h-4" />
          </button>
          <button
            data-testid="new-chat-btn"
            onClick={handleNewChat}
            className="flex items-center gap-1 text-sm text-gray-600 dark:text-gray-400 hover:text-blue-600 dark:hover:text-blue-400 transition-colors"
          >
            <Plus className="w-4 h-4" />
            <span>New</span>
          </button>
        </div>
        <MessageList
          messages={state.messages}
          onAction={handleAction}
          disabled={state.sending}
        />
        <ChatInput
          onSend={handleSend}
          placeholder="随时告诉我你的想法..."
          disabled={state.sending}
        />
      </div>

      {/* Right: preview panel */}
      <div className="hidden md:flex md:flex-1 h-full bg-white dark:bg-gray-900 rounded-xl shadow-sm border border-gray-200 dark:border-gray-800 overflow-hidden">
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