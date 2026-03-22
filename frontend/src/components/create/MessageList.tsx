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

          <div className="max-w-[80%] space-y-2">
            {msg.type === 'user' && (
              <div className="rounded-2xl px-4 py-3 bg-gradient-to-br from-blue-500 to-purple-600 text-white">
                <p className="text-sm leading-relaxed whitespace-pre-wrap">{msg.content}</p>
              </div>
            )}

            {msg.type === 'ai' && (
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
