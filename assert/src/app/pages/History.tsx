import { useState, useEffect } from "react";
import { useNavigate } from "react-router";
import { Card } from "../components/ui/card";
import { Button } from "../components/ui/button";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "../components/ui/dialog";
import { 
  CheckCircle2, 
  XCircle, 
  Clock, 
  Trash2,
  Eye,
  FileText
} from "lucide-react";
import { toast } from "sonner";

interface HistoryItem {
  id: string;
  text: string;
  similarity: number;
  timestamp: string;
  feedback?: "like" | "dislike";
}

export function History() {
  const [historyItems, setHistoryItems] = useState<HistoryItem[]>([]);
  const [selectedItem, setSelectedItem] = useState<HistoryItem | null>(null);
  const navigate = useNavigate();

  useEffect(() => {
    loadHistory();
  }, []);

  const loadHistory = () => {
    const items: HistoryItem[] = [];
    for (let i = 0; i < localStorage.length; i++) {
      const key = localStorage.key(i);
      if (key?.startsWith("result_")) {
        const data = localStorage.getItem(key);
        if (data) {
          items.push(JSON.parse(data));
        }
      }
    }
    // 按时间倒序排列
    items.sort((a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime());
    setHistoryItems(items);
  };

  const handleDelete = (id: string, e: React.MouseEvent) => {
    e.stopPropagation();
    localStorage.removeItem(`result_${id}`);
    loadHistory();
    toast.success("已删除记录");
  };

  const handleView = (item: HistoryItem) => {
    setSelectedItem(item);
  };

  const handleNavigateToResult = (id: string) => {
    navigate(`/result/${id}`);
    setSelectedItem(null);
  };

  if (historyItems.length === 0) {
    return (
      <div className="max-w-4xl mx-auto">
        <div className="text-center py-16">
          <div className="inline-flex items-center justify-center w-20 h-20 bg-gray-100 rounded-full mb-4">
            <FileText className="w-10 h-10 text-gray-400" />
          </div>
          <h2 className="text-2xl mb-2 text-gray-600">暂无历史记录</h2>
          <p className="text-gray-500 mb-6">开始检测内容以查看历史记录</p>
          <Button
            onClick={() => navigate("/")}
            className="bg-gradient-to-r from-blue-600 to-purple-600 hover:from-blue-700 hover:to-purple-700 text-white"
          >
            开始检测
          </Button>
        </div>
      </div>
    );
  }

  return (
    <div className="max-w-4xl mx-auto">
      <div className="mb-6">
        <h1 className="text-2xl sm:text-3xl mb-2">历史记录</h1>
        <p className="text-gray-600">共 {historyItems.length} 条检测记录</p>
      </div>

      <div className="space-y-4">
        {historyItems.map((item) => {
          const isPassed = item.similarity < 30;
          const statusColor = isPassed ? "text-green-600" : "text-red-600";
          const statusBg = isPassed ? "bg-green-50" : "bg-red-50";

          return (
            <Card
              key={item.id}
              className="p-4 sm:p-6 hover:shadow-lg transition-shadow cursor-pointer border border-gray-200"
              onClick={() => handleView(item)}
            >
              <div className="flex flex-col sm:flex-row sm:items-start gap-4">
                {/* 状态图标 */}
                <div className={`flex-shrink-0 w-12 h-12 rounded-lg ${statusBg} flex items-center justify-center`}>
                  {isPassed ? (
                    <CheckCircle2 className={`w-6 h-6 ${statusColor}`} />
                  ) : (
                    <XCircle className={`w-6 h-6 ${statusColor}`} />
                  )}
                </div>

                {/* 内容区 */}
                <div className="flex-1 min-w-0">
                  <div className="flex items-start justify-between gap-4 mb-2">
                    <div className="flex-1">
                      <div className="flex items-center gap-2 mb-1">
                        <span className={`text-lg ${statusColor}`}>
                          {item.similarity}%
                        </span>
                        <span className={`text-xs px-2 py-1 rounded ${statusBg} ${statusColor}`}>
                          {isPassed ? "通过" : "不通过"}
                        </span>
                        {item.feedback === "like" && (
                          <span className="text-xs text-green-600">👍</span>
                        )}
                        {item.feedback === "dislike" && (
                          <span className="text-xs text-red-600">👎</span>
                        )}
                      </div>
                      <p className="text-sm text-gray-600 line-clamp-2">
                        {item.text}
                      </p>
                    </div>
                  </div>

                  <div className="flex items-center gap-4 text-xs text-gray-500">
                    <span className="flex items-center gap-1">
                      <Clock className="w-3 h-3" />
                      {new Date(item.timestamp).toLocaleString("zh-CN")}
                    </span>
                  </div>
                </div>

                {/* 操作按钮 */}
                <div className="flex sm:flex-col gap-2">
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={(e) => {
                      e.stopPropagation();
                      handleNavigateToResult(item.id);
                    }}
                    className="flex-1 sm:flex-none"
                  >
                    <Eye className="w-4 h-4 mr-1" />
                    查看
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={(e) => handleDelete(item.id, e)}
                    className="flex-1 sm:flex-none text-red-600 hover:text-red-700 hover:border-red-600"
                  >
                    <Trash2 className="w-4 h-4 mr-1" />
                    删除
                  </Button>
                </div>
              </div>
            </Card>
          );
        })}
      </div>

      {/* 详情弹窗 */}
      <Dialog open={!!selectedItem} onOpenChange={() => setSelectedItem(null)}>
        <DialogContent className="max-w-2xl max-h-[80vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>检测详情</DialogTitle>
          </DialogHeader>
          
          {selectedItem && (
            <div className="space-y-4">
              <div className="flex items-center gap-4">
                <div className={`flex-shrink-0 w-16 h-16 rounded-lg ${
                  selectedItem.similarity < 30 ? "bg-green-50" : "bg-red-50"
                } flex items-center justify-center`}>
                  {selectedItem.similarity < 30 ? (
                    <CheckCircle2 className="w-8 h-8 text-green-600" />
                  ) : (
                    <XCircle className="w-8 h-8 text-red-600" />
                  )}
                </div>
                <div>
                  <div className={`text-3xl mb-1 ${
                    selectedItem.similarity < 30 ? "text-green-600" : "text-red-600"
                  }`}>
                    {selectedItem.similarity}%
                  </div>
                  <div className="text-sm text-gray-600">
                    {selectedItem.similarity < 30 ? "原创度达标" : "相似度较高"}
                  </div>
                </div>
              </div>

              <div>
                <h4 className="text-sm text-gray-700 mb-2">检测内容</h4>
                <div className="bg-gray-50 rounded-lg p-4 border border-gray-200 max-h-[300px] overflow-y-auto">
                  <p className="text-gray-800 whitespace-pre-wrap text-sm leading-relaxed">
                    {selectedItem.text}
                  </p>
                </div>
              </div>

              <div className="text-xs text-gray-500 flex items-center gap-2">
                <Clock className="w-4 h-4" />
                {new Date(selectedItem.timestamp).toLocaleString("zh-CN")}
              </div>

              <Button
                onClick={() => handleNavigateToResult(selectedItem.id)}
                className="w-full bg-gradient-to-r from-blue-600 to-purple-600 hover:from-blue-700 hover:to-purple-700 text-white"
              >
                查看完整结果
              </Button>
            </div>
          )}
        </DialogContent>
      </Dialog>
    </div>
  );
}
