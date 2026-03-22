import { useState } from "react";
import { Copy, Download, RotateCcw, Check } from "lucide-react";

interface ScriptEditorProps {
  content: string;
  onChange: (content: string) => void;
  onRegenerate?: () => void;
}

export function ScriptEditor({ content, onChange, onRegenerate }: ScriptEditorProps) {
  const [copied, setCopied] = useState(false);

  const handleCopy = () => {
    navigator.clipboard.writeText(content);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  const handleDownload = () => {
    const blob = new Blob([content], { type: "text/plain" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `爆款口播稿_${new Date().toLocaleDateString()}.txt`;
    a.click();
    URL.revokeObjectURL(url);
  };

  return (
    <div className="h-full w-full flex flex-col bg-white">
      {/* 工具栏 */}
      <div className="flex-shrink-0 p-4 border-b border-gray-200 flex items-center justify-between">
        <h3 className="text-lg font-semibold">完整口播稿</h3>
        <div className="flex gap-2">
          <button
            onClick={handleCopy}
            className="flex items-center gap-1.5 px-3 py-1.5 text-sm text-gray-700 hover:bg-gray-100 rounded-lg transition-colors"
          >
            {copied ? (
              <>
                <Check className="w-4 h-4 text-green-600" />
                <span className="text-green-600">已复制</span>
              </>
            ) : (
              <>
                <Copy className="w-4 h-4" />
                <span>复制</span>
              </>
            )}
          </button>
          <button
            onClick={handleDownload}
            className="flex items-center gap-1.5 px-3 py-1.5 text-sm text-gray-700 hover:bg-gray-100 rounded-lg transition-colors"
          >
            <Download className="w-4 h-4" />
            <span>导出</span>
          </button>
          {onRegenerate && (
            <button
              onClick={onRegenerate}
              className="flex items-center gap-1.5 px-3 py-1.5 text-sm text-blue-600 hover:bg-blue-50 rounded-lg transition-colors"
            >
              <RotateCcw className="w-4 h-4" />
              <span>重新生成</span>
            </button>
          )}
        </div>
      </div>

      {/* 内容区 */}
      <div className="flex-1 overflow-y-auto">
        <textarea
          value={content}
          onChange={(e) => onChange(e.target.value)}
          className="w-full h-full p-6 text-base leading-relaxed resize-none focus:outline-none"
          placeholder="口播稿内容..."
        />
      </div>
    </div>
  );
}