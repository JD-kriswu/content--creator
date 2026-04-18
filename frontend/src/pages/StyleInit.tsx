import { useState, useRef } from 'react'
import { useNavigate } from 'react-router'
import { Plus, Trash2, Sparkles, CheckCircle } from 'lucide-react'
import { toast } from 'sonner'
import { initStyleSSE } from '../api/style'

function parseSSEMsg(line: string): { type: string; content?: string; data?: unknown } | null {
  if (!line.startsWith('data: ')) return null
  try {
    return JSON.parse(line.slice(6)) as { type: string; content?: string; data?: unknown }
  } catch {
    return null
  }
}

export function StyleInit() {
  const navigate = useNavigate()
  const [scripts, setScripts] = useState<string[]>([''])
  const [running, setRunning] = useState(false)
  const [done, setDone] = useState(false)
  const [statusLog, setStatusLog] = useState<string[]>([])
  const [streamText, setStreamText] = useState('')
  const statusEndRef = useRef<HTMLDivElement>(null)

  const addScript = () => setScripts((s) => [...s, ''])
  const removeScript = (i: number) => setScripts((s) => s.filter((_, idx) => idx !== i))
  const updateScript = (i: number, v: string) =>
    setScripts((s) => s.map((x, idx) => (idx === i ? v : x)))

  const validScripts = scripts.filter((s) => s.trim().length >= 50)

  const handleSubmit = async () => {
    if (validScripts.length === 0) {
      toast.error('请至少输入一篇口播稿（50字以上）')
      return
    }
    setRunning(true)
    setStatusLog([])
    setStreamText('')

    try {
      const res = await initStyleSSE(validScripts)
      if (!res.ok) {
        const err = await res.json().catch(() => ({ error: '请求失败' }))
        toast.error((err as { error?: string }).error ?? '初始化失败')
        setRunning(false)
        return
      }

      const reader = res.body!.getReader()
      const decoder = new TextDecoder()
      let buffer = ''

      while (true) {
        const { done: streamDone, value } = await reader.read()
        if (streamDone) break
        buffer += decoder.decode(value, { stream: true })
        const lines = buffer.split('\n')
        buffer = lines.pop() ?? ''
        for (const line of lines) {
          const event = parseSSEMsg(line)
          if (!event) continue
          if (event.type === 'style_init' && event.content) {
            setStatusLog((log) => [...log, event.content!])
            setTimeout(() => statusEndRef.current?.scrollIntoView({ behavior: 'smooth' }), 50)
          } else if (event.type === 'token' && event.content) {
            setStreamText((t) => t + event.content)
          } else if (event.type === 'complete') {
            setDone(true)
          } else if (event.type === 'error' && event.content) {
            toast.error(event.content)
            setRunning(false)
            return
          }
        }
      }
    } catch (e) {
      toast.error(e instanceof Error ? e.message : '连接失败')
      setRunning(false)
      return
    }

    setRunning(false)
    if (done || true) {
      toast.success('风格档案初始化完成！')
      setTimeout(() => navigate('/dashboard'), 1500)
    }
  }

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-950 flex flex-col items-center py-12 px-4">
      <div className="w-full max-w-2xl space-y-8">
        {/* Header */}
        <div className="text-center space-y-2">
          <div className="inline-flex items-center justify-center w-12 h-12 rounded-2xl bg-indigo-100 dark:bg-indigo-900/40 mb-2">
            <Sparkles className="w-6 h-6 text-indigo-600 dark:text-indigo-400" />
          </div>
          <h1 className="text-2xl font-bold text-gray-900 dark:text-gray-100">初始化个人风格档案</h1>
          <p className="text-gray-500 dark:text-gray-400 text-sm leading-relaxed max-w-md mx-auto">
            粘贴您过去写的口播稿，AI 将分析您的语言风格、情绪基调和表达习惯，生成专属风格档案，帮您写出更像自己的爆款内容。
          </p>
        </div>

        {/* Script inputs */}
        {!running && !done && (
          <div className="space-y-4">
            {scripts.map((script, i) => (
              <div key={i} className="relative">
                <div className="flex items-center justify-between mb-1.5">
                  <span className="text-xs font-medium text-gray-500 dark:text-gray-400">
                    口播稿 {i + 1}
                    {script.trim().length > 0 && (
                      <span className={`ml-2 ${script.trim().length >= 50 ? 'text-green-500' : 'text-amber-500'}`}>
                        {script.trim().length} 字
                      </span>
                    )}
                  </span>
                  {scripts.length > 1 && (
                    <button
                      onClick={() => removeScript(i)}
                      className="text-gray-400 hover:text-red-500 transition-colors"
                    >
                      <Trash2 className="w-3.5 h-3.5" />
                    </button>
                  )}
                </div>
                <textarea
                  value={script}
                  onChange={(e) => updateScript(i, e.target.value)}
                  placeholder="粘贴您的口播稿内容（至少50字）..."
                  rows={6}
                  className="w-full rounded-xl border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900 text-sm text-gray-900 dark:text-gray-100 placeholder-gray-400 px-4 py-3 resize-none focus:outline-none focus:ring-2 focus:ring-indigo-500 dark:focus:ring-indigo-400"
                />
              </div>
            ))}

            <button
              onClick={addScript}
              className="flex items-center gap-1.5 text-sm text-indigo-600 dark:text-indigo-400 hover:text-indigo-700 dark:hover:text-indigo-300 transition-colors"
            >
              <Plus className="w-4 h-4" />
              添加更多口播稿（最多10篇，效果更好）
            </button>

            <button
              onClick={handleSubmit}
              disabled={validScripts.length === 0}
              className="w-full py-3 rounded-xl bg-indigo-600 hover:bg-indigo-700 disabled:bg-gray-200 dark:disabled:bg-gray-800 disabled:text-gray-400 text-white font-semibold text-sm transition-colors"
            >
              开始分析风格（{validScripts.length} 篇）
            </button>
          </div>
        )}

        {/* Progress panel */}
        {(running || done) && (
          <div className="rounded-2xl border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900 overflow-hidden">
            <div className="px-4 py-3 border-b border-gray-100 dark:border-gray-800 flex items-center gap-2">
              {done ? (
                <CheckCircle className="w-4 h-4 text-green-500" />
              ) : (
                <div className="w-4 h-4 rounded-full border-2 border-indigo-500 border-t-transparent animate-spin" />
              )}
              <span className="text-sm font-medium text-gray-700 dark:text-gray-300">
                {done ? '风格档案生成完成' : 'AI 正在分析您的风格...'}
              </span>
            </div>

            {/* Status log */}
            {statusLog.length > 0 && (
              <div className="px-4 py-3 border-b border-gray-100 dark:border-gray-800 space-y-1.5">
                {statusLog.map((log, i) => (
                  <p key={i} className="text-xs text-gray-500 dark:text-gray-400">{log}</p>
                ))}
                <div ref={statusEndRef} />
              </div>
            )}

            {/* Streaming text preview */}
            {streamText && (
              <div className="px-4 py-3 max-h-64 overflow-y-auto">
                <p className="text-xs text-gray-400 dark:text-gray-500 mb-2">风格档案预览</p>
                <p className="text-sm text-gray-700 dark:text-gray-300 whitespace-pre-wrap leading-relaxed">
                  {streamText}
                  {running && <span className="inline-block w-0.5 h-4 bg-indigo-500 animate-pulse ml-0.5 align-middle" />}
                </p>
              </div>
            )}
          </div>
        )}

        {done && (
          <p className="text-center text-sm text-gray-400 dark:text-gray-500">
            正在跳转到创作页面...
          </p>
        )}
      </div>
    </div>
  )
}
