import { Loader2 } from "lucide-react";

interface LoadingStateProps {
  message: string;
}

export function LoadingState({ message }: LoadingStateProps) {
  return (
    <div className="h-full flex flex-col">
      <div className="flex items-center gap-3 p-4 border-b border-gray-100 dark:border-gray-800">
        <Loader2 className="w-5 h-5 text-blue-500 animate-spin" />
        <span className="text-sm text-gray-600 dark:text-gray-400">{message}</span>
      </div>
      <div className="flex-1 flex items-center justify-center text-gray-400 dark:text-gray-500 text-sm">
        内容将在这里展示...
      </div>
    </div>
  );
}
