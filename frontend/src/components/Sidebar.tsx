import { useState, useEffect } from 'react'
import { Plus, MessageSquare, Send, Trash2, FileCode } from 'lucide-react'
import { listConversations, deleteConversation, type Conversation } from '../api/conversations'
import { Link } from 'react-router'

interface SidebarProps {
  onNewChat: () => void
  onSelectConversation: (conv: Conversation) => void
  activeConvId?: number
  refreshTrigger?: number
}

export function Sidebar({ onNewChat, onSelectConversation, activeConvId, refreshTrigger }: SidebarProps) {
  const [conversations, setConversations] = useState<Conversation[]>([])

  const loadConversations = async () => {
    try {
      const data = await listConversations()
      setConversations(data.conversations ?? [])
    } catch { /* ignore */ }
  }

  useEffect(() => { loadConversations() }, [refreshTrigger])

  const handleDelete = async (e: React.MouseEvent, convId: number) => {
    e.stopPropagation()
    if (!confirm('确定要删除这个会话吗？')) return
    try {
      await deleteConversation(convId)
      setConversations(prev => prev.filter(c => c.id !== convId))
    } catch { /* ignore */ }
  }

  return (
    <div data-testid="sidebar" className="w-72 h-full bg-gray-100 dark:bg-gray-950 flex flex-col flex-shrink-0 p-4 pt-3">
      {/* White card container */}
      <div className="flex-1 bg-white dark:bg-gray-900 rounded-xl shadow-sm border border-gray-200 dark:border-gray-800 flex flex-col overflow-hidden">
        {/* New chat button */}
        <div className="p-3">
          <button
            onClick={onNewChat}
            className="w-full flex items-center justify-center gap-2 px-4 py-2.5 bg-gray-50 dark:bg-gray-800 border border-gray-200 dark:border-gray-700 text-gray-700 dark:text-gray-300 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
          >
            <Plus className="w-4 h-4" />
            <span className="font-medium text-sm">新建对话</span>
          </button>
        </div>

        {/* Integration placeholders */}
        <div className="px-3 pb-3 border-b border-gray-100 dark:border-gray-800">
          <Link
            to="/feishu"
            className="w-full flex items-center justify-between px-3 py-2 text-gray-600 dark:text-gray-400 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-800 text-sm transition-colors"
          >
            <div className="flex items-center gap-2.5">
              <Send className="w-4 h-4" />
              <span>配置到飞书</span>
            </div>
          </Link>
          {[
            { label: '配置到钉钉' },
            { label: '配置到企业微信' },
          ].map((item) => (
            <button
              key={item.label}
              disabled
              className="w-full flex items-center justify-between px-3 py-2 text-gray-400 dark:text-gray-600 rounded-lg text-sm cursor-not-allowed"
            >
              <div className="flex items-center gap-2.5">
                <Send className="w-4 h-4" />
                <span>{item.label}</span>
              </div>
              <span className="text-xs bg-gray-100 dark:bg-gray-800 px-1.5 py-0.5 rounded text-gray-500 dark:text-gray-400">开发中</span>
            </button>
          ))}
        </div>

        {/* Conversations list */}
        <div className="flex-1 overflow-y-auto px-2 py-2">
          <div className="px-3 py-2 text-xs font-medium text-gray-400 dark:text-gray-500">会话</div>
          {conversations.length === 0 ? (
            <div className="text-center py-8 text-sm text-gray-400">暂无会话记录</div>
          ) : (
            <div className="space-y-0.5">
              {conversations.map((conv) => (
                <div
                  key={conv.id}
                  data-testid="conversation-item"
                  onClick={() => onSelectConversation(conv)}
                  className={`w-full flex items-center gap-2 px-3 py-2 text-left text-sm rounded-lg transition-colors cursor-pointer group ${
                    activeConvId === conv.id
                      ? 'bg-blue-50 dark:bg-blue-950 text-blue-700 dark:text-blue-300'
                      : 'text-gray-700 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-800'
                  }`}
                >
                  <MessageSquare className="w-4 h-4 flex-shrink-0 text-gray-400" />
                  <div className="flex-1 truncate">{conv.title || '未命名会话'}</div>
                  <button
                    onClick={(e) => handleDelete(e, conv.id)}
                    className="p-1 text-gray-400 hover:text-red-500 opacity-0 group-hover:opacity-100 transition-all"
                    title="删除会话"
                  >
                    <Trash2 className="w-4 h-4" />
                  </button>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Prompt Editor link */}
        <div className="p-3 border-t border-gray-100 dark:border-gray-800">
          <Link
            to="/prompts"
            className="w-full flex items-center gap-2.5 px-3 py-2 text-gray-600 dark:text-gray-400 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-800 text-sm transition-colors"
          >
            <FileCode className="w-4 h-4" />
            <span>Prompt 编辑器</span>
          </Link>
        </div>
      </div>
    </div>
  )
}