interface OutlineEditorProps {
  content: string;
  onChange: (content: string) => void;
}

export function OutlineEditor({ content, onChange }: OutlineEditorProps) {
  return (
    <div className="h-full w-full flex flex-col bg-white">
      <div className="flex-shrink-0 p-6 border-b border-gray-200">
        <h3 className="text-lg font-semibold">大纲</h3>
      </div>
      
      <div className="flex-1 overflow-y-auto">
        <textarea
          value={content}
          onChange={(e) => onChange(e.target.value)}
          className="w-full h-full p-6 text-base leading-relaxed resize-none focus:outline-none"
          placeholder="大纲内容..."
        />
      </div>
    </div>
  );
}