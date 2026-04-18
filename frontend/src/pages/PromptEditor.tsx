import { useEffect, useState } from 'react'
import { getPrompts, updatePrompt, PromptFile } from '../api/prompts'

export function PromptEditor() {
  const [prompts, setPrompts] = useState<PromptFile[]>([])
  const [loading, setLoading] = useState(true)
  const [selectedPath, setSelectedPath] = useState<string | null>(null)
  const [editingContent, setEditingContent] = useState('')
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [success, setSuccess] = useState<string | null>(null)

  useEffect(() => {
    loadPrompts()
  }, [])

  async function loadPrompts() {
    try {
      setLoading(true)
      const data = await getPrompts()
      setPrompts(data.prompts)
      if (data.prompts.length > 0 && !selectedPath) {
        setSelectedPath(data.prompts[0].path)
        setEditingContent(data.prompts[0].content)
      }
    } catch (e) {
      setError('加载 prompt 失败')
    } finally {
      setLoading(false)
    }
  }

  function selectPrompt(p: PromptFile) {
    setSelectedPath(p.path)
    setEditingContent(p.content)
    setError(null)
    setSuccess(null)
  }

  async function handleSave() {
    if (!selectedPath) return
    setSaving(true)
    setError(null)
    setSuccess(null)
    try {
      await updatePrompt({ path: selectedPath, content: editingContent })
      setSuccess('保存成功')
      // Update local state
      setPrompts(prompts.map(p =>
        p.path === selectedPath ? { ...p, content: editingContent } : p
      ))
    } catch (e: any) {
      setError(e.message || '保存失败')
    } finally {
      setSaving(false)
    }
  }

  const selected = prompts.find(p => p.path === selectedPath)

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-gray-500">加载中...</div>
      </div>
    )
  }

  return (
    <div className="p-6 max-w-6xl mx-auto">
      <h1 className="text-2xl font-bold mb-6">Prompt 编辑器</h1>

      <div className="flex gap-6">
        {/* Left: Prompt list */}
        <div className="w-64 border rounded-lg p-4">
          <h2 className="text-sm font-semibold text-gray-500 mb-3">Prompt 文件</h2>
          <div className="space-y-1">
            {prompts.map(p => (
              <button
                key={p.path}
                onClick={() => selectPrompt(p)}
                className={`w-full text-left px-3 py-2 rounded text-sm ${
                  p.path === selectedPath
                    ? 'bg-blue-100 text-blue-700'
                    : 'hover:bg-gray-100'
                }`}
              >
                <div className="font-medium">{p.display_name || p.name}</div>
                <div className="text-xs text-gray-400">{p.path}</div>
              </button>
            ))}
          </div>
        </div>

        {/* Right: Editor */}
        <div className="flex-1 border rounded-lg p-4">
          {selected && (
            <>
              <div className="flex items-center justify-between mb-4">
                <div>
                  <h2 className="font-semibold">{selected.display_name || selected.name}</h2>
                  <div className="text-sm text-gray-500">{selected.path}</div>
                </div>
                <button
                  onClick={handleSave}
                  disabled={saving}
                  className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700 disabled:opacity-50"
                >
                  {saving ? '保存中...' : '保存'}
                </button>
              </div>

              {error && (
                <div className="mb-4 p-3 bg-red-50 text-red-600 rounded">{error}</div>
              )}
              {success && (
                <div className="mb-4 p-3 bg-green-50 text-green-600 rounded">{success}</div>
              )}

              <textarea
                value={editingContent}
                onChange={(e) => setEditingContent(e.target.value)}
                className="w-full h-[500px] p-4 border rounded font-mono text-sm resize-none focus:outline-none focus:ring-2 focus:ring-blue-500"
                spellCheck={false}
              />
            </>
          )}
        </div>
      </div>
    </div>
  )
}