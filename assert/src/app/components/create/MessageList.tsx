import { Bot, User } from "lucide-react";

export interface Message {
  id: string;
  type: "user" | "ai" | "system" | "action";
  content: string;
  timestamp: Date;
  action?: {
    label: string;
    onClick: () => void;
  };
}

interface MessageListProps {
  messages: Message[];
}

export function MessageList({ messages }: MessageListProps) {
  return (
    <div className="flex-1 overflow-y-auto p-4 space-y-4">
      {messages.map((message) => (
        <div
          key={message.id}
          className={`flex gap-3 ${
            message.type === "user" ? "justify-end" : "justify-start"
          }`}
        >
          {message.type !== "user" && (
            <div className="flex-shrink-0 w-8 h-8 rounded-full bg-gradient-to-br from-blue-500 to-purple-600 flex items-center justify-center">
              <Bot className="w-5 h-5 text-white" />
            </div>
          )}
          
          <div
            className={`max-w-[80%] ${
              message.type === "user" ? "" : "space-y-2"
            }`}
          >
            {/* 消息内容 */}
            <div
              className={`rounded-2xl px-4 py-3 ${
                message.type === "user"
                  ? "bg-gradient-to-br from-blue-500 to-purple-600 text-white"
                  : message.type === "system"
                  ? "bg-blue-50 text-blue-900 border border-blue-200"
                  : "bg-gray-100 text-gray-900"
              }`}
            >
              <p className="text-sm leading-relaxed whitespace-pre-wrap">
                {message.content}
              </p>
            </div>

            {/* 操作按钮 */}
            {message.action && (
              <button
                onClick={message.action.onClick}
                className="px-4 py-2 bg-gradient-to-br from-blue-500 to-purple-600 text-white text-sm rounded-lg hover:shadow-md transition-all"
              >
                {message.action.label}
              </button>
            )}
          </div>

          {message.type === "user" && (
            <div className="flex-shrink-0 w-8 h-8 rounded-full bg-gray-200 flex items-center justify-center">
              <User className="w-5 h-5 text-gray-600" />
            </div>
          )}
        </div>
      ))}
    </div>
  );
}