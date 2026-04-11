import { useState, useEffect } from 'react'
import { FeishuQRCode } from '../components/FeishuQRCode'
import { getFeishuBots, unbindFeishuBot, type FeishuBot } from '../api/feishu'

export function FeishuBind() {
  const [bots, setBots] = useState<FeishuBot[]>([])
  const [status] = useState<'waiting' | 'success' | 'error'>('waiting')
  const [qrUrl] = useState('')
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    loadBots()
  }, [])

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
        <p className="mb-4">扫码创建飞书机器人，可在飞书中使用口播稿创作服务</p>
        {/* 注意：实际的扫码流程需要通过飞书开放平台的 App Manifest 导入 API 生成二维码 URL。
            当前版本先显示提示信息，后续根据飞书官方文档实现完整的扫码创建流程。 */}
        {qrUrl ? (
          <FeishuQRCode qrUrl={qrUrl} status={status} onRefresh={loadBots} />
        ) : (
          <div className="w-64 h-64 mx-auto border rounded-lg flex items-center justify-center bg-gray-50 dark:bg-gray-800 dark:border-gray-700">
            <p className="text-gray-500 text-center p-4">
              飞书扫码绑定功能需要配置飞书开放平台应用。<br/>
              请联系管理员获取绑定链接。
            </p>
          </div>
        )}
      </div>
    </div>
  )
}