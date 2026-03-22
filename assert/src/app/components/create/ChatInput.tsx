import { useState } from "react";
import { Send } from "lucide-react";

interface ChatInputProps {
  onSend: (message: string) => void;
  placeholder?: string;
  disabled?: boolean;
}

export function ChatInput({ onSend, placeholder = "输入你的需求...", disabled = false }: ChatInputProps) {
  const [input, setInput] = useState("");

  const handleSend = () => {
    if (!input.trim() || disabled) return;
    onSend(input.trim());
    setInput("");
  };

  const handleKeyPress = (e: React.KeyboardEvent) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
  };

  return (
    <div className="p-4 border-t border-gray-200 bg-white">
      <div className="relative">
        <textarea
          value={input}
          onChange={(e) => setInput(e.target.value)}
          onKeyPress={handleKeyPress}
          placeholder={placeholder}
          disabled={disabled}
          className="w-full px-4 py-3 pr-12 border border-gray-200 rounded-xl resize-none focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent disabled:bg-gray-50 disabled:cursor-not-allowed"
          rows={2}
        />
        <button
          onClick={handleSend}
          disabled={!input.trim() || disabled}
          className="absolute bottom-2 right-2 w-8 h-8 bg-gradient-to-br from-blue-500 to-purple-600 rounded-lg flex items-center justify-center hover:scale-105 transition-all disabled:opacity-40 disabled:cursor-not-allowed disabled:hover:scale-100"
        >
          <Send className="w-4 h-4 text-white" />
        </button>
      </div>
    </div>
  );
}
