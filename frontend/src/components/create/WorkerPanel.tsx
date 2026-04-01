import { useState } from 'react'

interface WorkerPanelProps {
  name: string
  displayName: string
  content: string
  status: 'running' | 'done'
}

export function WorkerPanel({ displayName, content, status }: WorkerPanelProps) {
  const [expanded, setExpanded] = useState(false)

  return (
    <div className="border rounded-lg p-3 bg-white dark:bg-gray-800">
      <div className="flex items-center justify-between cursor-pointer" onClick={() => setExpanded(!expanded)}>
        <span className="font-medium text-sm">{displayName}</span>
        <div className="flex items-center gap-2">
          {status === 'running' ? (
            <span className="text-xs text-blue-500 animate-pulse">输出中...</span>
          ) : (
            <span className="text-xs text-green-500">完成</span>
          )}
          <span className="text-xs text-gray-400">{expanded ? '收起' : '展开'}</span>
        </div>
      </div>
      {expanded && content && (
        <div className="mt-2 text-sm text-gray-600 dark:text-gray-300 whitespace-pre-wrap max-h-60 overflow-y-auto border-t pt-2">
          {content}
        </div>
      )}
    </div>
  )
}
