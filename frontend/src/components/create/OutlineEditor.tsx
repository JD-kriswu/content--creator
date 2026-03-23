interface OutlineEditorProps {
  content: string
  onChange: (content: string) => void
}

export function OutlineEditor({ content, onChange }: OutlineEditorProps) {
  return (
    <div className="h-full w-full flex flex-col bg-white dark:bg-gray-900">
      <div className="flex-shrink-0 px-6 py-4 border-b border-gray-200 dark:border-gray-800">
        <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100">大纲</h3>
      </div>
      <div className="flex-1 overflow-y-auto">
        <textarea
          value={content}
          onChange={(e) => onChange(e.target.value)}
          className="w-full h-full p-6 text-sm leading-relaxed resize-none focus:outline-none bg-transparent text-gray-700 dark:text-gray-300"
          placeholder="大纲内容..."
        />
      </div>
    </div>
  )
}
