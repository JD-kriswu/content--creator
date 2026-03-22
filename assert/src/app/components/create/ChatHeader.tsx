import { ArrowLeft } from 'lucide-react';

interface ChatHeaderProps {
  onBack: () => void;
  onNew: () => void;
}

export function ChatHeader({ onBack, onNew }: ChatHeaderProps) {
  return (
    <div className="flex-shrink-0 h-14 border-b border-gray-200 flex items-center justify-between px-4 bg-white">
      {/* 左侧：返回按钮 */}
      <button
        onClick={onBack}
        className="p-2 text-gray-400 hover:text-gray-600 hover:bg-gray-50 rounded-lg transition-colors"
        title="返回首页"
      >
        <ArrowLeft className="w-5 h-5" />
      </button>

      {/* 右侧：+New 按钮 */}
      <button
        onClick={onNew}
        className="px-3 py-1.5 text-sm text-gray-500 hover:text-gray-700 hover:bg-gray-50 rounded-lg transition-colors"
        title="开始新对话"
      >
        +New
      </button>
    </div>
  );
}
