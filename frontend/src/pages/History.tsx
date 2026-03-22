import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router'
import { Card } from '../components/ui/card'
import { Button } from '../components/ui/button'
import { Clock, Eye, FileText } from 'lucide-react'
import { listConversations, type Conversation } from '../api/conversations'
import { toast } from 'sonner'

export function History() {
  const [conversations, setConversations] = useState<Conversation[]>([])
  const [loading, setLoading] = useState(true)
  const navigate = useNavigate()

  useEffect(() => {
    listConversations()
      .then((data) => setConversations(data.conversations ?? []))
      .catch(() => toast.error('加载历史记录失败'))
      .finally(() => setLoading(false))
  }, [])

  if (loading) return <div className="p-8 text-center text-gray-500">加载中...</div>

  if (conversations.length === 0) {
    return (
      <div className="max-w-4xl mx-auto text-center py-16">
        <div className="inline-flex items-center justify-center w-20 h-20 bg-gray-100 dark:bg-gray-800 rounded-full mb-4">
          <FileText className="w-10 h-10 text-gray-400" />
        </div>
        <h2 className="text-2xl mb-2 text-gray-600 dark:text-gray-400">暂无历史记录</h2>
        <p className="text-gray-500 mb-6">开始创作以查看历史记录</p>
        <Button
          onClick={() => navigate('/dashboard')}
          className="bg-gradient-to-r from-blue-600 to-purple-600 text-white"
        >
          开始创作
        </Button>
      </div>
    )
  }

  return (
    <div className="max-w-4xl mx-auto">
      <div className="mb-6">
        <h1 className="text-2xl sm:text-3xl font-semibold mb-2 text-gray-900 dark:text-gray-100">历史记录</h1>
        <p className="text-gray-600 dark:text-gray-400">共 {conversations.length} 条会话记录</p>
      </div>

      <div className="space-y-4">
        {conversations.map((conv) => (
          <Card
            key={conv.id}
            className="p-4 sm:p-6 hover:shadow-lg transition-shadow border border-gray-200 dark:border-gray-800 bg-white dark:bg-gray-900"
          >
            <div className="flex flex-col sm:flex-row sm:items-center gap-4">
              <div className="flex-1 min-w-0">
                <h3 className="text-base font-medium text-gray-900 dark:text-gray-100 truncate mb-1">
                  {conv.title || '未命名会话'}
                </h3>
                <div className="flex items-center gap-3 text-xs text-gray-500 dark:text-gray-400">
                  <span className="flex items-center gap-1">
                    <Clock className="w-3 h-3" />
                    {new Date(conv.created_at).toLocaleString('zh-CN')}
                  </span>
                  <span className={`px-2 py-0.5 rounded text-xs ${
                    conv.state === 1
                      ? 'bg-green-100 dark:bg-green-900 text-green-700 dark:text-green-300'
                      : 'bg-yellow-100 dark:bg-yellow-900 text-yellow-700 dark:text-yellow-300'
                  }`}>
                    {conv.state === 1 ? '已完成' : '进行中'}
                  </span>
                </div>
              </div>
              <div className="flex gap-2">
                {conv.script_id ? (
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => navigate(`/result/${conv.script_id}`)}
                  >
                    <Eye className="w-4 h-4 mr-1" />查看稿件
                  </Button>
                ) : (
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => navigate('/dashboard')}
                  >
                    继续创作
                  </Button>
                )}
              </div>
            </div>
          </Card>
        ))}
      </div>
    </div>
  )
}
