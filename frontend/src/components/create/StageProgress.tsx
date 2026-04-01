interface StageProgressProps {
  currentStep: number
  totalSteps: number
  stageName: string
}

export function StageProgress({ currentStep, totalSteps, stageName }: StageProgressProps) {
  const percent = Math.round((currentStep / totalSteps) * 100)

  return (
    <div className="flex items-center gap-3 py-2">
      <div className="flex-1 h-2 bg-gray-200 dark:bg-gray-700 rounded-full overflow-hidden">
        <div
          className="h-full bg-blue-500 rounded-full transition-all duration-500"
          style={{ width: `${percent}%` }}
        />
      </div>
      <span className="text-xs text-gray-500 whitespace-nowrap">
        {currentStep}/{totalSteps} {stageName}
      </span>
    </div>
  )
}
