import { useState, useEffect } from 'react'
import { Plus, MessageSquare, Send, MoreHorizontal } from 'lucide-react'
import { listConversations, type Conversation } from '../api/conversations'

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

  return (
    <div className="w-72 h-full bg-gray-100 dark:bg-gray-950 flex flex-col flex-shrink-0 p-4 pt-3">
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
                <button
                  key={conv.id}
                  onClick={() => onSelectConversation(conv)}
                  className={`w-full flex items-center gap-2 px-3 py-2 text-left text-sm rounded-lg transition-colors group ${
                    activeConvId === conv.id
                      ? 'bg-blue-50 dark:bg-blue-950 text-blue-700 dark:text-blue-300'
                      : 'text-gray-700 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-800'
                  }`}
                >
                  <MessageSquare className="w-4 h-4 flex-shrink-0 text-gray-400" />
                  <div className="flex-1 truncate">{conv.title || '未命名会话'}</div>
                  <MoreHorizontal className="w-4 h-4 text-gray-400 opacity-0 group-hover:opacity-100 transition-opacity" />
                </button>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  )
}