import { useState } from 'react'
import { Copy, Download, RotateCcw, Check, ExternalLink } from 'lucide-react'
import { useNavigate } from 'react-router'

interface ScriptEditorProps {
  content: string
  scriptId?: number | null
  onRegenerate?: () => void
}

export function ScriptEditor({ content, scriptId, onRegenerate }: ScriptEditorProps) {
  const [copied, setCopied] = useState(false)
  const navigate = useNavigate()

  const handleCopy = () => {
    navigator.clipboard.writeText(content)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  const handleDownload = () => {
    const blob = new Blob([content], { type: 'text/plain' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `爆款口播稿_${new Date().toLocaleDateString('zh-CN')}.txt`
    a.click()
    URL.revokeObjectURL(url)
  }

  return (
    <div data-testid="script-editor" className="h-full w-full flex flex-col bg-white dark:bg-gray-900">
      <div className="flex-shrink-0 px-6 py-4 border-b border-gray-200 dark:border-gray-800 flex items-center justify-between">
        <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100">完整口播稿</h3>
        <div className="flex gap-1">
          <button
            onClick={handleCopy}
            className="flex items-center gap-1.5 px-3 py-1.5 text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-800 rounded-lg transition-colors"
          >
            {copied ? <><Check className="w-4 h-4 text-green-600" /><span className="text-green-600">已复制</span></> : <><Copy className="w-4 h-4" /><span>复制</span></>}
          </button>
          <button
            onClick={handleDownload}
            className="flex items-center gap-1.5 px-3 py-1.5 text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-800 rounded-lg transition-colors"
          >
            <Download className="w-4 h-4" />
            <span>导出</span>
          </button>
          {onRegenerate && (
            <button
              onClick={onRegenerate}
              className="flex items-center gap-1.5 px-3 py-1.5 text-sm text-blue-600 hover:bg-blue-50 dark:hover:bg-blue-950 rounded-lg transition-colors"
            >
              <RotateCcw className="w-4 h-4" />
              <span>重新生成</span>
            </button>
          )}
        </div>
      </div>
      <div className="flex-1 overflow-y-auto p-6">
        <p className="text-sm leading-relaxed whitespace-pre-wrap text-gray-800 dark:text-gray-200">{content}</p>
      </div>
      {scriptId && (
        <div className="flex-shrink-0 p-4 border-t border-gray-200 dark:border-gray-800">
          <button
            onClick={() => navigate(`/result/${scriptId}`)}
            className="w-full flex items-center justify-center gap-2 py-2.5 text-sm bg-gradient-to-r from-blue-600 to-purple-600 text-white rounded-lg hover:opacity-90 transition-opacity"
          >
            <ExternalLink className="w-4 h-4" />
            查看完整结果（含相似度检测）
          </button>
        </div>
      )}
    </div>
  )
}
