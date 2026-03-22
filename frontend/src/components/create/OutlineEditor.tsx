interface OutlineEditorProps {
  content: string
  onChange: (content: string) => void
}

export function OutlineEditor({ content, onChange }: OutlineEditorProps) {
  return (
    <div className="h-full w-full flex flex-col bg-white dark:bg-gray-900">
      <div className="flex-shrink-0 p-6 border-b border-gray-200 dark:border-gray-800">
        <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100">大纲预览</h3>
        <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">可在此编辑大纲内容，确认后点击左侧按钮继续</p>
      </div>
      <div className="flex-1 overflow-y-auto">
        <textarea
          value={content}
          onChange={(e) => onChange(e.target.value)}
          className="w-full h-full p-6 text-sm leading-relaxed resize-none focus:outline-none bg-transparent text-gray-800 dark:text-gray-200"
          placeholder="大纲内容..."
        />
      </div>
    </div>
  )
}
