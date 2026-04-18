import { useState } from 'react'
import { Sparkles, ChevronDown, ChevronUp, X, CheckCircle } from 'lucide-react'
import { toast } from 'sonner'
import { initStyleSSE } from '../api/style'

interface Props {
  onInitialized: () => void
}

function parseMsg(line: string): { type: string; content?: string } | null {
  if (!line.startsWith('data: ')) return null
  try {
    return JSON.parse(line.slice(6)) as { type: string; content?: string }
  } catch {
    return null
  }
}

export function StyleInitBanner({ onInitialized }: Props) {
  const [expanded, setExpanded] = useState(false)
  const [dismissed, setDismissed] = useState(false)
  const [scripts, setScripts] = useState(['', '', ''])
  const [running, setRunning] = useState(false)
  const [done, setDone] = useState(false)
  const [status, setStatus] = useState('')

  if (dismissed) return null

  const validCount = scripts.filter((s) => s.trim().length >= 50).length

  const handleSubmit = async () => {
    const validScripts = scripts.filter((s) => s.trim().length >= 50)
    if (validScripts.length === 0) {
      toast.error('请至少输入一篇口播稿（50字以上）')
      return
    }
    setRunning(true)
    setStatus('正在初始化风格档案...')

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
          const event = parseMsg(line)
          if (!event) continue
          if (event.type === 'style_init' && event.content) {
            setStatus(event.content)
          } else if (event.type === 'complete') {
            setDone(true)
            setStatus('✅ 风格档案初始化完成！')
            setTimeout(() => {
              onInitialized()
              setDismissed(true)
            }, 1500)
          } else if (event.type === 'error' && event.content) {
            toast.error(event.content)
            setRunning(false)
            return
          }
        }
      }
    } catch (e) {
      toast.error(e instanceof Error ? e.message : '连接失败')
    }
    setRunning(false)
  }

  return (
    <div data-testid="style-init-banner" className="mx-4 mt-4 rounded-2xl border border-indigo-200 dark:border-indigo-800 bg-indigo-50 dark:bg-indigo-950/40 overflow-hidden">
      {/* Header row */}
      <div className="flex items-center gap-3 px-4 py-3">
        <Sparkles className="w-4 h-4 text-indigo-500 flex-shrink-0" />
        <div className="flex-1 min-w-0">
          <p className="text-sm font-medium text-indigo-800 dark:text-indigo-200">
            个人风格档案未初始化
          </p>
          <p className="text-xs text-indigo-600/70 dark:text-indigo-400/70">
            提供 3 篇历史口播稿，AI 将生成专属风格档案；或直接使用通用爆款风格。
          </p>
        </div>
        <div className="flex items-center gap-1 flex-shrink-0">
          <button
            onClick={() => setExpanded((v) => !v)}
            className="flex items-center gap-1 text-xs font-medium text-indigo-600 dark:text-indigo-400 hover:text-indigo-800 dark:hover:text-indigo-200 transition-colors px-2 py-1 rounded-lg hover:bg-indigo-100 dark:hover:bg-indigo-900/40"
          >
            {expanded ? '收起' : '初始化'}
            {expanded ? <ChevronUp className="w-3.5 h-3.5" /> : <ChevronDown className="w-3.5 h-3.5" />}
          </button>
          <button
            onClick={() => setDismissed(true)}
            className="p-1 rounded-lg text-indigo-400 hover:text-indigo-600 dark:hover:text-indigo-200 hover:bg-indigo-100 dark:hover:bg-indigo-900/40 transition-colors"
            title="跳过，使用通用风格"
          >
            <X className="w-3.5 h-3.5" />
          </button>
        </div>
      </div>

      {/* Expanded form */}
      {expanded && (
        <div className="px-4 pb-4 space-y-3 border-t border-indigo-200 dark:border-indigo-800 pt-3">
          {!running && !done && (
            <>
              {scripts.map((script, i) => (
                <div key={i}>
                  <label className="text-xs font-medium text-indigo-700 dark:text-indigo-300 mb-1 block">
                    口播稿 {i + 1}
                    {script.trim().length > 0 && (
                      <span className={`ml-2 ${script.trim().length >= 50 ? 'text-green-600 dark:text-green-400' : 'text-amber-600 dark:text-amber-400'}`}>
                        {script.trim().length} 字
                      </span>
                    )}
                  </label>
                  <textarea
                    value={script}
                    onChange={(e) => setScripts((s) => s.map((x, idx) => idx === i ? e.target.value : x))}
                    placeholder={`粘贴第 ${i + 1} 篇历史口播稿（至少50字）...`}
                    rows={4}
                    className="w-full rounded-xl border border-indigo-200 dark:border-indigo-700 bg-white dark:bg-gray-900 text-sm text-gray-900 dark:text-gray-100 placeholder-gray-400 px-3 py-2 resize-none focus:outline-none focus:ring-2 focus:ring-indigo-400"
                  />
                </div>
              ))}
              <div className="flex gap-2">
                <button
                  onClick={handleSubmit}
                  disabled={validCount === 0}
                  className="flex-1 py-2 rounded-xl bg-indigo-600 hover:bg-indigo-700 disabled:bg-gray-200 dark:disabled:bg-gray-800 disabled:text-gray-400 text-white text-sm font-semibold transition-colors"
                >
                  开始分析（{validCount} 篇有效）
                </button>
                <button
                  onClick={() => setDismissed(true)}
                  className="px-4 py-2 rounded-xl border border-indigo-200 dark:border-indigo-700 text-sm text-indigo-600 dark:text-indigo-400 hover:bg-indigo-100 dark:hover:bg-indigo-900/40 transition-colors"
                >
                  跳过
                </button>
              </div>
            </>
          )}

          {(running || done) && (
            <div className="flex items-center gap-2 py-2">
              {done ? (
                <CheckCircle className="w-4 h-4 text-green-500 flex-shrink-0" />
              ) : (
                <div className="w-4 h-4 rounded-full border-2 border-indigo-500 border-t-transparent animate-spin flex-shrink-0" />
              )}
              <span className="text-sm text-indigo-700 dark:text-indigo-300">{status}</span>
            </div>
          )}
        </div>
      )}
    </div>
  )
}
