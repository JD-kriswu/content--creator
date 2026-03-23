import { useState, useEffect } from 'react'
import { useParams, useNavigate } from 'react-router'
import { ArrowLeft, Copy, Check, ThumbsUp, ThumbsDown, Feather } from 'lucide-react'
import { Card } from '../components/ui/card'
import { Button } from '../components/ui/button'
import { getScript } from '../api/scripts'
import { toast } from 'sonner'

export function Result() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const [content, setContent] = useState('')
  const [similarity, setSimilarity] = useState<number | null>(null)
  const [title, setTitle] = useState('')
  const [copied, setCopied] = useState(false)
  const [feedback, setFeedback] = useState<'like' | 'dislike' | null>(
    () => (localStorage.getItem(`feedback_${id}`) as 'like' | 'dislike' | null) ?? null
  )
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    if (!id) return
    getScript(Number(id))
      .then((data) => {
        setContent(data.content)
        setSimilarity(data.script.similarity_score)
        setTitle(data.script.title)
      })
      .catch(() => {
        toast.error('加载稿件失败')
        navigate('/dashboard')
      })
      .finally(() => setLoading(false))
  }, [id, navigate])

  const handleCopy = () => {
    navigator.clipboard.writeText(content)
    setCopied(true)
    toast.success('已复制到剪贴板')
    setTimeout(() => setCopied(false), 2000)
  }

  const handleFeedback = (type: 'like' | 'dislike') => {
    const next = feedback === type ? null : type
    setFeedback(next)
    if (next) localStorage.setItem(`feedback_${id}`, next)
    else localStorage.removeItem(`feedback_${id}`)
    toast.success(next === 'like' ? '感谢反馈！' : next === 'dislike' ? '我们会持续改进' : '已取消反馈')
  }

  if (loading) return (
    <div className="h-full overflow-y-auto flex items-center justify-center text-gray-500">加载中...</div>
  )

  const passed = similarity !== null && similarity < 30
  const statusColor = passed ? 'text-green-600' : 'text-red-600'
  const statusBg = passed ? 'bg-green-50 dark:bg-green-950' : 'bg-red-50 dark:bg-red-950'
  const statusBorder = passed ? 'border-green-200 dark:border-green-800' : 'border-red-200 dark:border-red-800'

  return (
    <div className="h-full overflow-y-auto">
    <div className="max-w-4xl mx-auto px-4 sm:px-6 py-8">
      <Button variant="ghost" onClick={() => navigate(-1)} className="mb-6 -ml-2">
        <ArrowLeft className="w-4 h-4 mr-2" />返回
      </Button>

      <h1 className="text-2xl font-semibold mb-6 text-gray-900 dark:text-gray-100">{title}</h1>

      <Card className={`p-6 sm:p-8 shadow-lg border-2 ${statusBorder} ${statusBg}`}>
        {/* Similarity */}
        {similarity !== null && (
          <div className="mb-6">
            <div className="flex items-center justify-between mb-2">
              <span className="text-sm text-gray-600 dark:text-gray-400">相似度</span>
              <span className={`text-3xl font-semibold ${statusColor}`}>{similarity}%</span>
            </div>
            <div className="w-full h-3 bg-gray-200 dark:bg-gray-700 rounded-full overflow-hidden">
              <div
                className={`h-full rounded-full ${passed ? 'bg-green-500' : 'bg-red-500'} transition-all`}
                style={{ width: `${Math.min(similarity, 100)}%` }}
              />
            </div>
            <div className={`mt-2 text-sm ${statusColor}`}>
              {passed ? '✅ 原创度达标，可直接使用' : '❌ 相似度较高，建议修改'}
            </div>
          </div>
        )}

        {/* Script content */}
        <div className="mb-6">
          <h3 className="text-sm font-medium text-gray-700 dark:text-gray-300 mb-3">稿件内容</h3>
          <div className="bg-white dark:bg-gray-900 rounded-lg p-4 border border-gray-200 dark:border-gray-700 max-h-[300px] overflow-y-auto">
            <p className="text-gray-800 dark:text-gray-200 whitespace-pre-wrap text-sm leading-relaxed">{content}</p>
          </div>
        </div>

        {/* Copy button */}
        <Button onClick={handleCopy} className="w-full mb-6 h-12 bg-blue-600 hover:bg-blue-700 text-white">
          {copied ? <><Check className="w-5 h-5 mr-2" />已复制</> : <><Copy className="w-5 h-5 mr-2" />📋 一键复制</>}
        </Button>

        {/* Feedback */}
        <div className="border-t border-gray-300 dark:border-gray-700 pt-6 mb-6">
          <h3 className="text-sm font-medium text-gray-700 dark:text-gray-300 mb-3">反馈</h3>
          <div className="flex gap-3">
            <Button
              variant={feedback === 'like' ? 'default' : 'outline'}
              onClick={() => handleFeedback('like')}
              className={`flex-1 h-11 ${feedback === 'like' ? 'bg-green-600 hover:bg-green-700 text-white' : ''}`}
            >
              <ThumbsUp className="w-4 h-4 mr-2" />👍 喜欢
            </Button>
            <Button
              variant={feedback === 'dislike' ? 'default' : 'outline'}
              onClick={() => handleFeedback('dislike')}
              className={`flex-1 h-11 ${feedback === 'dislike' ? 'bg-red-600 hover:bg-red-700 text-white' : ''}`}
            >
              <ThumbsDown className="w-4 h-4 mr-2" />👎 不喜欢
            </Button>
          </div>
        </div>

        {/* Continue creating */}
        <Button
          onClick={() => navigate('/dashboard')}
          className="w-full h-12 bg-gradient-to-r from-blue-600 to-purple-600 hover:from-blue-700 hover:to-purple-700 text-white"
        >
          <Feather className="w-5 h-5 mr-2" />继续创作
        </Button>
      </Card>
    </div>
    </div>
  )
}
