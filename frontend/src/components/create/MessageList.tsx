import { useEffect, useRef } from 'react'
import { Bot } from 'lucide-react'
import { ParallelStageView, type WorkerStream } from './ParallelStageView'

export interface ChatMsg {
  id: string
  type: 'user' | 'ai' | 'step' | 'info' | 'action' | 'similarity' | 'error' | 'outline' | 'parallel_stage'
  content?: string
  options?: string[]        // for action type
  data?: unknown            // for outline/similarity
  streaming?: boolean
  workers?: WorkerStream[]  // for parallel_stage type
  synthContent?: string     // for parallel_stage type
}

interface OutlineData {
  elements?: string[]
  materials?: string[]
  outline?: Array<{ part?: string; duration?: string; content?: string; emotion?: string }>
  estimated_similarity?: string
  strategy?: string
}

function OutlineCard({ data }: { data: unknown }) {
  const d = (data ?? {}) as OutlineData
  return (
    <div className="rounded-2xl border border-blue-100 dark:border-blue-900 bg-blue-50/60 dark:bg-blue-950/30 overflow-hidden text-sm w-full">
      <div className="px-4 py-2.5 bg-blue-100/80 dark:bg-blue-900/40 font-semibold text-blue-800 dark:text-blue-200 text-xs uppercase tracking-wide">
        内容大纲
      </div>
      {d.elements && d.elements.length > 0 && (
        <div className="px-4 py-3 border-b border-blue-100 dark:border-blue-900/60">
          <p className="text-xs font-medium text-blue-600 dark:text-blue-400 mb-1.5">爆款元素</p>
          <ul className="space-y-1">
            {d.elements.map((e, i) => (
              <li key={i} className="flex items-start gap-1.5 text-gray-700 dark:text-gray-300">
                <span className="mt-1.5 w-1.5 h-1.5 rounded-full bg-blue-400 flex-shrink-0" />
                {e}
              </li>
            ))}
          </ul>
        </div>
      )}
      {d.outline && d.outline.length > 0 && (
        <div className="px-4 py-3 border-b border-blue-100 dark:border-blue-900/60 space-y-2.5">
          <p className="text-xs font-medium text-blue-600 dark:text-blue-400">结构分镜</p>
          {d.outline.map((o, i) => (
            <div key={i} className="rounded-lg bg-white/70 dark:bg-gray-800/50 px-3 py-2">
              <div className="flex items-center gap-2 mb-1">
                <span className="font-semibold text-gray-800 dark:text-gray-200">{o.part}</span>
                {o.duration && (
                  <span className="text-xs px-1.5 py-0.5 rounded-full bg-blue-100 dark:bg-blue-900/50 text-blue-600 dark:text-blue-400">{o.duration}</span>
                )}
                {o.emotion && (
                  <span className="text-xs text-purple-500 dark:text-purple-400 ml-auto">{o.emotion}</span>
                )}
              </div>
              {o.content && <p className="text-gray-600 dark:text-gray-400 leading-relaxed">{o.content}</p>}
            </div>
          ))}
        </div>
      )}
      {d.strategy && (
        <div className="px-4 py-3 border-b border-blue-100 dark:border-blue-900/60">
          <p className="text-xs font-medium text-blue-600 dark:text-blue-400 mb-1">创作策略</p>
          <p className="text-gray-700 dark:text-gray-300 leading-relaxed">{d.strategy}</p>
        </div>
      )}
      {d.estimated_similarity && (
        <div className="px-4 py-2.5 flex items-center gap-2">
          <span className="text-xs text-gray-500 dark:text-gray-400">预估相似度</span>
          <span className="text-xs font-medium text-orange-500">{d.estimated_similarity}</span>
        </div>
      )}
    </div>
  )
}

interface MessageListProps {
  messages: ChatMsg[]
  onAction?: (option: string) => void
  disabled?: boolean
}

export function MessageList({ messages, onAction, disabled }: MessageListProps) {
  const listRef = useRef<HTMLDivElement>(null)
  const bottomRef = useRef<HTMLDivElement>(null)

  // Auto-scroll when messages change or when streaming content updates
  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages])

  return (
    <div ref={listRef} className="flex-1 overflow-y-auto p-4 space-y-3">
      {messages.map((msg) => (
        <div
          key={msg.id}
          className={`flex gap-3 ${msg.type === 'user' ? 'justify-end' : 'justify-start'}`}
        >
          {msg.type !== 'user' && (
            <div className="flex-shrink-0 w-8 h-8 rounded-xl bg-gradient-to-br from-blue-500 to-purple-600 flex items-center justify-center shadow-sm">
              <Bot className="w-4 h-4 text-white" />
            </div>
          )}

          <div className={`space-y-2 ${msg.type === 'outline' ? 'flex-1 min-w-0' : 'max-w-[85%]'}`}>
            {msg.type === 'user' && (
              <div className="rounded-2xl px-4 py-3 bg-gradient-to-br from-blue-500 to-purple-600 text-white shadow-sm">
                <p className="text-sm leading-relaxed whitespace-pre-wrap">{msg.content}</p>
              </div>
            )}

            {msg.type === 'step' && (
              <div className="flex items-center gap-2 py-1">
                <span className="w-2 h-2 rounded-full bg-blue-400 flex-shrink-0" />
                <span className="text-sm text-gray-600 dark:text-gray-400">{msg.content}</span>
              </div>
            )}

            {msg.type === 'ai' && (
              <div className="rounded-2xl px-4 py-3 bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 shadow-sm border border-gray-100 dark:border-gray-700">
                <p className="text-sm leading-relaxed whitespace-pre-wrap">
                  {msg.content}
                  {msg.streaming && <span className="inline-block w-0.5 h-4 ml-0.5 bg-blue-500 animate-pulse align-middle" />}
                </p>
              </div>
            )}

            {msg.type === 'info' && (
              <div className="rounded-xl px-3 py-2 bg-blue-50 dark:bg-blue-950/50 text-blue-700 dark:text-blue-300 text-sm">
                {msg.content}
              </div>
            )}

            {msg.type === 'error' && (
              <div className="rounded-xl px-3 py-2 bg-red-50 dark:bg-red-950/50 text-red-600 dark:text-red-400 text-sm">
                ❌ {msg.content}
              </div>
            )}

            {msg.type === 'action' && msg.options && (
              <div className="flex flex-col gap-2">
                {msg.options.map((opt, i) => (
                  <button
                    key={i}
                    disabled={disabled}
                    onClick={() => onAction?.(String(i + 1))}
                    className="px-4 py-2.5 text-sm bg-gradient-to-r from-blue-500 to-purple-600 text-white rounded-xl hover:opacity-90 disabled:opacity-40 disabled:cursor-not-allowed transition-opacity text-left shadow-sm"
                  >
                    {opt}
                  </button>
                ))}
              </div>
            )}

            {msg.type === 'outline' && (
              <OutlineCard data={msg.data} />
            )}

            {msg.type === 'similarity' && !!msg.data && (
              <div className="rounded-xl px-3 py-2 bg-green-50 dark:bg-green-950/50 text-green-700 dark:text-green-400 text-sm">
                相似度检测完成 ✅
              </div>
            )}

            {msg.type === 'parallel_stage' && msg.workers && (
              <ParallelStageView
                stageName={msg.content ?? ''}
                workers={msg.workers}
                synthContent={msg.synthContent}
                synthStatus="done"
              />
            )}
          </div>
        </div>
      ))}
      <div ref={bottomRef} />
    </div>
  )
}
