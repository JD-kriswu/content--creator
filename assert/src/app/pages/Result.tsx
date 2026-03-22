import { useParams, useNavigate } from "react-router";
import { useState, useEffect } from "react";
import { Card } from "../components/ui/card";
import { Button } from "../components/ui/button";
import { Progress } from "../components/ui/progress";
import { 
  CheckCircle2, 
  XCircle, 
  Copy, 
  ThumbsUp, 
  ThumbsDown, 
  Feather,
  ArrowLeft,
  Check
} from "lucide-react";
import { toast } from "sonner";

interface ResultData {
  id: string;
  text: string;
  similarity: number;
  timestamp: string;
  feedback?: "like" | "dislike";
}

export function Result() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [result, setResult] = useState<ResultData | null>(null);
  const [copied, setCopied] = useState(false);
  const [feedback, setFeedback] = useState<"like" | "dislike" | null>(null);

  useEffect(() => {
    if (id) {
      const savedResult = localStorage.getItem(`result_${id}`);
      if (savedResult) {
        const data = JSON.parse(savedResult);
        setResult(data);
        setFeedback(data.feedback || null);
      } else {
        navigate("/");
      }
    }
  }, [id, navigate]);

  if (!result) {
    return null;
  }

  const isPassed = result.similarity < 30;
  const statusColor = isPassed ? "text-green-600" : "text-red-600";
  const statusBg = isPassed ? "bg-green-50" : "bg-red-50";
  const statusBorder = isPassed ? "border-green-200" : "border-red-200";

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(result.text);
      setCopied(true);
      toast.success("已复制到剪贴板");
      setTimeout(() => setCopied(false), 2000);
    } catch (err) {
      toast.error("复制失败，请重试");
    }
  };

  const handleFeedback = (type: "like" | "dislike") => {
    const newFeedback = feedback === type ? null : type;
    setFeedback(newFeedback);
    
    // 更新 localStorage
    const updatedResult = { ...result, feedback: newFeedback };
    localStorage.setItem(`result_${id}`, JSON.stringify(updatedResult));
    
    toast.success(newFeedback === "like" ? "感谢您的反馈！" : newFeedback === "dislike" ? "我们会持续改进" : "已取消反馈");
  };

  const handleContinue = () => {
    navigate("/");
  };

  return (
    <div className="max-w-4xl mx-auto">
      <Button
        variant="ghost"
        onClick={() => navigate("/")}
        className="mb-6 -ml-2"
      >
        <ArrowLeft className="w-4 h-4 mr-2" />
        返回首页
      </Button>

      <Card className={`p-6 sm:p-8 shadow-lg border-2 ${statusBorder} ${statusBg}`}>
        {/* 检测结果标题 */}
        <div className="text-center mb-6">
          <div className={`inline-flex items-center justify-center w-16 h-16 rounded-full mb-4 ${isPassed ? "bg-green-100" : "bg-red-100"}`}>
            {isPassed ? (
              <CheckCircle2 className={`w-8 h-8 ${statusColor}`} />
            ) : (
              <XCircle className={`w-8 h-8 ${statusColor}`} />
            )}
          </div>
          <h2 className="text-2xl mb-2">相似度检测结果</h2>
        </div>

        {/* 相似度显示 */}
        <div className="bg-white rounded-xl p-6 mb-6 border border-gray-200">
          <div className="flex items-center justify-between mb-3">
            <span className="text-sm text-gray-600">相似度</span>
            <span className={`text-3xl ${statusColor}`}>
              {result.similarity}%
            </span>
          </div>
          <Progress value={result.similarity} className="h-3 mb-2" />
          <div className="flex items-center justify-between text-xs text-gray-500">
            <span>0%</span>
            <span className={isPassed ? "text-green-600" : "text-red-600"}>
              {isPassed ? "✅ 原创度达标，可直接复制使用" : "❌ 相似度较高，建议修改"}
            </span>
            <span>100%</span>
          </div>
        </div>

        {/* 通过/不通过状态 */}
        <div className={`p-4 rounded-lg mb-6 ${statusBg} border ${statusBorder}`}>
          <div className="flex items-center gap-2">
            {isPassed ? (
              <>
                <CheckCircle2 className={`w-5 h-5 ${statusColor}`} />
                <span className={statusColor}>
                  原创度达标，可直接复制使用
                </span>
              </>
            ) : (
              <>
                <XCircle className={`w-5 h-5 ${statusColor}`} />
                <span className={statusColor}>
                  相似度：{result.similarity}% (&gt;30%不通过)
                </span>
              </>
            )}
          </div>
        </div>

        {/* 终稿展示区 */}
        <div className="mb-6">
          <h3 className="text-sm text-gray-700 mb-3">终稿展示区</h3>
          <div className="bg-gray-50 rounded-lg p-4 border border-gray-200 max-h-[300px] overflow-y-auto">
            <p className="text-gray-800 whitespace-pre-wrap text-sm leading-relaxed">
              {result.text}
            </p>
          </div>
        </div>

        {/* 一键复制按钮 */}
        <Button
          onClick={handleCopy}
          className="w-full mb-6 h-12 bg-blue-600 hover:bg-blue-700 text-white"
        >
          {copied ? (
            <>
              <Check className="w-5 h-5 mr-2" />
              已复制
            </>
          ) : (
            <>
              <Copy className="w-5 h-5 mr-2" />
              📋 一键复制
            </>
          )}
        </Button>

        {/* 反馈区 */}
        <div className="border-t border-gray-300 pt-6 mb-6">
          <h3 className="text-sm text-gray-700 mb-3">反馈区</h3>
          <div className="flex gap-3">
            <Button
              variant={feedback === "like" ? "default" : "outline"}
              onClick={() => handleFeedback("like")}
              className={`flex-1 h-11 ${
                feedback === "like"
                  ? "bg-green-600 hover:bg-green-700 text-white"
                  : "border-gray-300 hover:border-green-600 hover:text-green-600"
              }`}
            >
              <ThumbsUp className="w-4 h-4 mr-2" />
              👍 喜欢
            </Button>
            <Button
              variant={feedback === "dislike" ? "default" : "outline"}
              onClick={() => handleFeedback("dislike")}
              className={`flex-1 h-11 ${
                feedback === "dislike"
                  ? "bg-red-600 hover:bg-red-700 text-white"
                  : "border-gray-300 hover:border-red-600 hover:text-red-600"
              }`}
            >
              <ThumbsDown className="w-4 h-4 mr-2" />
              👎 不喜欢
            </Button>
          </div>
        </div>

        {/* 继续创作按钮 */}
        <Button
          onClick={handleContinue}
          className="w-full h-12 bg-gradient-to-r from-blue-600 to-purple-600 hover:from-blue-700 hover:to-purple-700 text-white"
        >
          <Feather className="w-5 h-5 mr-2" />
          继续创作
        </Button>
      </Card>

      {/* 检测时间 */}
      <div className="mt-4 text-center text-sm text-gray-500">
        <p>检测时间：{new Date(result.timestamp).toLocaleString("zh-CN")}</p>
      </div>
    </div>
  );
}