import { Plus, Send, MessageSquare, MoreHorizontal } from 'lucide-react';

interface HistoryItem {
  id: string;
  title: string;
  timestamp: Date;
}

interface SidebarProps {
  onNewChat: () => void;
  historyItems: HistoryItem[];
  currentChatId?: string;
  onSelectHistory: (id: string) => void;
}

export function Sidebar({ onNewChat, historyItems, currentChatId, onSelectHistory }: SidebarProps) {
  return (
    <div className="w-64 h-screen bg-gray-50 border-r border-gray-200 flex flex-col">
      {/* 顶部：新建对话按钮 */}
      <div className="p-3">
        <button
          onClick={onNewChat}
          className="w-full flex items-center justify-center gap-2 px-4 py-2.5 bg-white border border-gray-200 text-gray-700 rounded-lg hover:bg-gray-50 transition-colors"
        >
          <Plus className="w-4 h-4" />
          <span className="font-medium text-sm">新建对话</span>
        </button>
      </div>

      {/* 功能导航区 */}
      <div className="px-3 pb-3 border-b border-gray-200">
        <div className="space-y-0.5">
          <button className="w-full flex items-center justify-between px-3 py-2 text-gray-700 hover:bg-gray-100 rounded-lg transition-colors group text-sm">
            <div className="flex items-center gap-2.5">
              <Send className="w-4 h-4" />
              <span>配置到飞书</span>
            </div>
            <span className="text-xs text-gray-400 bg-gray-200 px-1.5 py-0.5 rounded">开发中</span>
          </button>
          <button className="w-full flex items-center justify-between px-3 py-2 text-gray-700 hover:bg-gray-100 rounded-lg transition-colors group text-sm">
            <div className="flex items-center gap-2.5">
              <Send className="w-4 h-4" />
              <span>配置到钉钉</span>
            </div>
            <span className="text-xs text-gray-400 bg-gray-200 px-1.5 py-0.5 rounded">开发中</span>
          </button>
          <button className="w-full flex items-center justify-between px-3 py-2 text-gray-700 hover:bg-gray-100 rounded-lg transition-colors group text-sm">
            <div className="flex items-center gap-2.5">
              <Send className="w-4 h-4" />
              <span>配置到企业微信</span>
            </div>
            <span className="text-xs text-gray-400 bg-gray-200 px-1.5 py-0.5 rounded">开发中</span>
          </button>
        </div>
      </div>

      {/* 历史对话列表 */}
      <div className="flex-1 overflow-y-auto">
        {historyItems.length > 0 && (
          <div className="px-3 py-2">
            <div className="text-xs text-gray-500 px-3 py-1.5 font-medium">
              所有记录
            </div>
          </div>
        )}
        
        <div className="px-2">
          {historyItems.length === 0 ? (
            <div className="px-4 py-8 text-center text-sm text-gray-400">
              暂无历史记录
            </div>
          ) : (
            <div className="space-y-0.5">
              {historyItems.map((item) => (
                <button
                  key={item.id}
                  onClick={() => onSelectHistory(item.id)}
                  className={`w-full flex items-center gap-2 px-3 py-2 text-left text-sm rounded-lg transition-colors group ${
                    currentChatId === item.id
                      ? 'bg-white text-gray-900 shadow-sm'
                      : 'text-gray-700 hover:bg-gray-100'
                  }`}
                >
                  <MessageSquare className="w-4 h-4 flex-shrink-0 text-gray-400" />
                  <div className="flex-1 truncate">{item.title}</div>
                  <MoreHorizontal className="w-4 h-4 text-gray-400 opacity-0 group-hover:opacity-100 transition-opacity" />
                </button>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}