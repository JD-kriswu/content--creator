import { useState, useEffect } from 'react'
import { FeishuQRCode } from '../components/FeishuQRCode'
import { getFeishuBots, unbindFeishuBot, getBindQRCode, getBindStatus, type FeishuBot } from '../api/feishu'

export function FeishuBind() {
  const [bots, setBots] = useState<FeishuBot[]>([])
  const [qrUrl, setQrUrl] = useState('')
  const [bindToken, setBindToken] = useState('')
  const [bindStatus, setBindStatus] = useState<'idle' | 'waiting' | 'success' | 'error'>('idle')
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    loadBots()
  }, [])

  // Poll status when bindToken is set and status is waiting
  useEffect(() => {
    if (!bindToken || bindStatus !== 'waiting') return

    const pollInterval = setInterval(async () => {
      try {
        const status = await getBindStatus(bindToken)
        if (status.status === 'success') {
          setBindStatus('success')
          clearInterval(pollInterval)
          loadBots()
        } else if (status.status === 'error') {
          setBindStatus('error')
          clearInterval(pollInterval)
        }
        // pending: continue polling
      } catch {
        // token expired or error
        setBindStatus('error')
        clearInterval(pollInterval)
      }
    }, 2000)

    return () => clearInterval(pollInterval)
  }, [bindToken, bindStatus])

  const loadBots = async () => {
    try {
      setLoading(true)
      const data = await getFeishuBots()
      setBots(data.bots)
    } catch {
      // ignore error
    } finally {
      setLoading(false)
    }
  }

  const handleStartBind = async () => {
    try {
      const data = await getBindQRCode()
      setQrUrl(data.qrcode_url)
      setBindToken(data.bind_token)
      setBindStatus('waiting')
    } catch {
      setBindStatus('error')
    }
  }

  const handleReset = () => {
    setQrUrl('')
    setBindToken('')
    setBindStatus('idle')
  }

  const handleUnbind = async (botId: number) => {
    if (!confirm('确定要解绑这个机器人吗？')) return
    try {
      await unbindFeishuBot(botId)
      loadBots()
    } catch {
      // ignore error
    }
  }

  return (
    <div className="p-8 max-w-2xl mx-auto">
      <h1 className="text-2xl font-bold mb-6">飞书机器人绑定</h1>

      {/* 已绑定的机器人列表 */}
      {loading ? (
        <div className="mb-8 text-center py-8 text-gray-500">加载中...</div>
      ) : bots.length > 0 && (
        <div className="mb-8">
          <h2 className="text-lg font-semibold mb-4">已绑定的机器人</h2>
          <div className="space-y-2">
            {bots.map(bot => (
              <div key={bot.id} className="flex items-center justify-between p-4 border rounded-lg dark:border-gray-700">
                <div>
                  <p className="font-medium">{bot.bot_name || '口播稿助手'}</p>
                  <p className="text-sm text-gray-500">
                    {bot.ws_connected ? '🟢 已连接' : '🔴 未连接'}
                  </p>
                </div>
                <button
                  onClick={() => handleUnbind(bot.id)}
                  className="px-3 py-1.5 text-sm text-red-600 border border-red-200 rounded-lg hover:bg-red-50 dark:border-red-800 dark:text-red-400 dark:hover:bg-red-950 transition-colors"
                >
                  解绑
                </button>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* 扫码绑定 */}
      <div className="text-center">
        {bindStatus === 'idle' && (
          <>
            <p className="mb-4">扫码创建飞书机器人，可在飞书中使用口播稿创作服务</p>
            <button
              onClick={handleStartBind}
              className="px-6 py-3 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors"
            >
              创建新机器人
            </button>
          </>
        )}

        {bindStatus === 'waiting' && qrUrl && (
          <FeishuQRCode qrUrl={qrUrl} status="waiting" onRefresh={handleReset} />
        )}

        {bindStatus === 'success' && (
          <div className="py-8">
            <p className="text-green-600 text-lg mb-4">绑定成功！</p>
            <button
              onClick={handleReset}
              className="px-6 py-2 border rounded-lg hover:bg-gray-50 dark:hover:bg-gray-800 transition-colors"
            >
              继续绑定其他机器人
            </button>
          </div>
        )}

        {bindStatus === 'error' && (
          <div className="py-8">
            <p className="text-red-600 text-lg mb-4">绑定失败，请重试</p>
            <button
              onClick={handleReset}
              className="px-6 py-2 border rounded-lg hover:bg-gray-50 dark:hover:bg-gray-800 transition-colors"
            >
              重试
            </button>
          </div>
        )}
      </div>
    </div>
  )
}