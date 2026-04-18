import { WorkerPanel } from './WorkerPanel'

export interface WorkerStream {
  name: string
  displayName: string
  content: string
  status: 'running' | 'done'
}

interface ParallelStageViewProps {
  stageName: string
  workers: WorkerStream[]
  synthContent?: string
  synthStatus?: 'running' | 'done'
}

export function ParallelStageView({ stageName, workers, synthContent, synthStatus }: ParallelStageViewProps) {
  const doneCount = workers.filter(w => w.status === 'done').length

  return (
    <div className="border rounded-xl p-4 bg-gray-50 dark:bg-gray-900 space-y-3">
      <div className="flex items-center justify-between">
        <span className="font-semibold text-sm">{stageName}</span>
        <span className="text-xs text-gray-500">{doneCount}/{workers.length} 完成</span>
      </div>
      <div className="grid grid-cols-1 sm:grid-cols-3 gap-2">
        {workers.map(w => (
          <WorkerPanel key={w.name} {...w} />
        ))}
      </div>
      {synthContent !== undefined && (
        <div className="border-t pt-2">
          <span className="text-xs font-medium text-gray-500">
            {synthStatus === 'running' ? '汇总分析中...' : '汇总完成'}
          </span>
          <div className="mt-1 text-sm text-gray-600 whitespace-pre-wrap max-h-40 overflow-y-auto">
            {synthContent || (synthStatus === 'running' ? '...' : '')}
            {synthStatus === 'running' && synthContent && (
              <span className="inline-block w-0.5 h-4 ml-0.5 bg-blue-500 animate-pulse align-middle" />
            )}
          </div>
        </div>
      )}
    </div>
  )
}
