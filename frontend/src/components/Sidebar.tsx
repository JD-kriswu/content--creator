import { useState, useEffect } from 'react'
import { Plus, MessageSquare, FileText, Send, MoreHorizontal } from 'lucide-react'
import { listConversations, type Conversation } from '../api/conversations'
import { getScripts, getScript, type Script } from '../api/scripts'

interface SidebarProps {
  onNewChat: () => void
  onSelectConversation: (conv: Conversation) => void
  onSelectScript: (content: string, title: string) => void
  activeConvId?: number
  refreshTrigger?: number  // external trigger to refresh conversation list
}

export function Sidebar({ onNewChat, onSelectConversation, onSelectScript, activeConvId, refreshTrigger }: SidebarProps) {
  const [activeTab, setActiveTab] = useState<'conversations' | 'scripts'>('conversations')
  const [conversations, setConversations] = useState<Conversation[]>([])
  const [scripts, setScripts] = useState<Script[]>([])

  const loadConversations = async () => {
    try {
      const data = await listConversations()
      setConversations(data.conversations ?? [])
    } catch { /* ignore */ }
  }

  const loadScripts = async () => {
    try {
      const data = await getScripts()
      setScripts(data.scripts ?? [])
    } catch { /* ignore */ }
  }

  useEffect(() => { loadConversations() }, [refreshTrigger])
  useEffect(() => { if (activeTab === 'scripts') loadScripts() }, [activeTab])

  const handleScriptClick = async (id: number, title: string) => {
    try {
      const data = await getScript(id)
      onSelectScript(data.content, title)
    } catch { /* ignore */ }
  }

  return (
    <div className="w-64 h-screen bg-gray-50 dark:bg-gray-900 border-r border-gray-200 dark:border-gray-800 flex flex-col flex-shrink-0">
      {/* New chat button */}
      <div className="p-3">
        <button
          onClick={onNewChat}
          className="w-full flex items-center justify-center gap-2 px-4 py-2.5 bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 text-gray-700 dark:text-gray-300 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors"
        >
          <Plus className="w-4 h-4" />
          <span className="font-medium text-sm">新建对话</span>
        </button>
      </div>

      {/* Integration placeholders */}
      <div className="px-3 pb-3 border-b border-gray-200 dark:border-gray-800">
        {[{ label: '配置到飞书' }, { label: '配置到钉钉' }, { label: '配置到企业微信' }].map((item) => (
          <button
            key={item.label}
            disabled
            className="w-full flex items-center justify-between px-3 py-2 text-gray-400 dark:text-gray-600 rounded-lg text-sm cursor-not-allowed"
          >
            <div className="flex items-center gap-2.5">
              <Send className="w-4 h-4" />
              <span>{item.label}</span>
            </div>
            <span className="text-xs bg-gray-200 dark:bg-gray-700 px-1.5 py-0.5 rounded">开发中</span>
          </button>
        ))}
      </div>

      {/* Tabs */}
      <div className="flex border-b border-gray-200 dark:border-gray-800">
        {(['conversations', 'scripts'] as const).map((tab) => (
          <button
            key={tab}
            onClick={() => setActiveTab(tab)}
            className={`flex-1 flex items-center justify-center gap-1.5 py-2 text-xs font-medium transition-colors ${
              activeTab === tab
                ? 'text-blue-600 dark:text-blue-400 border-b-2 border-blue-600'
                : 'text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-300'
            }`}
          >
            {tab === 'conversations' ? <MessageSquare className="w-3.5 h-3.5" /> : <FileText className="w-3.5 h-3.5" />}
            {tab === 'conversations' ? '会话' : '稿件'}
          </button>
        ))}
      </div>

      {/* List */}
      <div className="flex-1 overflow-y-auto px-2 py-2">
        {activeTab === 'conversations' ? (
          conversations.length === 0 ? (
            <div className="text-center py-8 text-sm text-gray-400">暂无会话记录</div>
          ) : (
            <div className="space-y-0.5">
              {conversations.map((conv) => (
                <button
                  key={conv.id}
                  onClick={() => onSelectConversation(conv)}
                  className={`w-full flex items-center gap-2 px-3 py-2 text-left text-sm rounded-lg transition-colors group ${
                    activeConvId === conv.id
                      ? 'bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 shadow-sm'
                      : 'text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-800'
                  }`}
                >
                  <MessageSquare className="w-4 h-4 flex-shrink-0 text-gray-400" />
                  <div className="flex-1 truncate">{conv.title || '未命名会话'}</div>
                  <MoreHorizontal className="w-4 h-4 text-gray-400 opacity-0 group-hover:opacity-100 transition-opacity" />
                </button>
              ))}
            </div>
          )
        ) : (
          scripts.length === 0 ? (
            <div className="text-center py-8 text-sm text-gray-400">暂无稿件</div>
          ) : (
            <div className="space-y-0.5">
              {scripts.map((script) => (
                <button
                  key={script.id}
                  onClick={() => handleScriptClick(script.id, script.title)}
                  className="w-full flex items-center gap-2 px-3 py-2 text-left text-sm rounded-lg transition-colors text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-800"
                >
                  <FileText className="w-4 h-4 flex-shrink-0 text-gray-400" />
                  <div className="flex-1 truncate">{script.title || '未命名稿件'}</div>
                </button>
              ))}
            </div>
          )
        )}
      </div>
    </div>
  )
}
