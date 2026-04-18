import { useState, useEffect, useCallback } from 'react'
import { getFeishuBots, unbindFeishuBot, startBindFlow, cancelBind, type FeishuBot, type BindSSEEvent } from '../api/feishu'

export function FeishuBind() {
  const [bots, setBots] = useState<FeishuBot[]>([])
  const [bindToken, setBindToken] = useState('')
  const [bindStatus, setBindStatus] = useState<'idle' | 'scanning' | 'creating' | 'success' | 'error'>('idle')
  const [qrCodeLines, setQrCodeLines] = useState<string[]>([])
  const [infoMessage, setInfoMessage] = useState('')
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

  const handleStartBind = useCallback(() => {
    setBindStatus('scanning')
    setQrCodeLines([])
    setInfoMessage('正在启动绑定流程...')
    setBindToken('') // Will be set when init event arrives

    // Start SSE connection
    startBindFlow().then(response => {
      if (!response.ok) {
        setBindStatus('error')
        setInfoMessage('启动绑定流程失败')
        return
      }

      const reader = response.body?.getReader()
      if (!reader) {
        setBindStatus('error')
        setInfoMessage('无法读取响应流')
        return
      }

      const decoder = new TextDecoder()
      let buffer = ''

      const processLine = (line: string) => {
        if (!line.startsWith('data: ')) return

        const jsonStr = line.slice(6).trim()
        if (!jsonStr) return

        try {
          const event: BindSSEEvent = JSON.parse(jsonStr)

          switch (event.type) {
            case 'init':
              if (event.data.bind_token) {
                setBindToken(event.data.bind_token)
              }
              break

            case 'qrcode':
              if (event.data.line) {
                // Convert ANSI codes to visible characters
                const cleanLine = cleanANSI(event.data.line)
                setQrCodeLines(prev => [...prev, cleanLine])
              }
              break

            case 'status':
              if (event.data.status === 'scanning') {
                setBindStatus('scanning')
                setInfoMessage('请使用飞书 APP 扫描二维码')
              } else if (event.data.status === 'creating') {
                setBindStatus('creating')
                setInfoMessage('正在创建机器人...')
              }
              break

            case 'info':
              if (event.data.message) {
                setInfoMessage(event.data.message)
              }
              break

            case 'success':
              setBindStatus('success')
              setInfoMessage(`绑定成功！机器人: ${event.data.bot_name || '口播稿助手'}`)
              loadBots()
              // Stop reading
              reader.cancel()
              break

            case 'error':
              setBindStatus('error')
              setInfoMessage(event.data.message || '绑定失败')
              // Stop reading
              reader.cancel()
              break
          }
        } catch (e) {
          console.error('Failed to parse SSE event:', jsonStr, e)
        }
      }

      const readLoop = async () => {
        try {
          while (true) {
            const { done, value } = await reader.read()
            if (done) break

            buffer += decoder.decode(value, { stream: true })

            // Process complete lines
            const lines = buffer.split('\n')
            buffer = lines.pop() || '' // Keep incomplete line in buffer

            for (const line of lines) {
              processLine(line)
            }
          }
        } catch (e) {
          if (bindStatus !== 'success' && bindStatus !== 'error') {
            setBindStatus('error')
            setInfoMessage('连接中断')
          }
        }
      }

      readLoop()
    }).catch(err => {
      setBindStatus('error')
      setInfoMessage(`启动绑定流程失败: ${err.message}`)
    })
  }, [bindStatus])

  const handleReset = async () => {
    // Cancel ongoing bind if there's a token
    if (bindToken) {
      try {
        await cancelBind(bindToken)
      } catch {
        // ignore
      }
    }
    setBindToken('')
    setBindStatus('idle')
    setQrCodeLines([])
    setInfoMessage('')
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

        {(bindStatus === 'scanning' || bindStatus === 'creating') && (
          <div className="py-4">
            {/* ASCII QR Code display */}
            {qrCodeLines.length > 0 && (
              <div className="mb-4 p-4 bg-gray-900 rounded-lg inline-block">
                <pre className="text-white font-mono text-xs leading-none whitespace-pre overflow-hidden">
                  {qrCodeLines.join('\n')}
                </pre>
              </div>
            )}
            <p className="text-gray-600 mb-2">{infoMessage}</p>
            {bindStatus === 'scanning' && qrCodeLines.length === 0 && (
              <p className="text-gray-500">正在生成二维码...</p>
            )}
            {bindStatus === 'creating' && (
              <p className="text-blue-500 animate-pulse">正在创建机器人应用...</p>
            )}
            <button
              onClick={handleReset}
              className="mt-4 px-4 py-2 text-sm border rounded-lg hover:bg-gray-50 dark:hover:bg-gray-800 transition-colors"
            >
              取消
            </button>
          </div>
        )}

        {bindStatus === 'success' && (
          <div className="py-8">
            <p className="text-green-600 text-lg mb-4">{infoMessage}</p>
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
            <p className="text-red-600 text-lg mb-4">{infoMessage}</p>
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

// Clean ANSI color codes and convert to visible characters
function cleanANSI(line: string): string {
  // Remove ANSI color codes like [47m[30m and [0m
  // But keep the visual characters (▄, █, ▀)
  let cleaned = line

  // Remove ANSI escape sequences
  cleaned = cleaned.replace(/\x1b\[[0-9;]*m/g, '')

  // Handle the special pattern [47m[30m text [0m
  cleaned = cleaned.replace(/\[47m\[30m/g, '')
  cleaned = cleaned.replace(/\[0m/g, '')

  // The QR code uses inverse video (white background, black text)
  // In terminal: [47m (white bg) + [30m (black fg) = white block with black border
  // The ▄ character represents half-block (top half filled)
  // The █ character represents full block
  // We display them as-is in white text on dark background

  return cleaned
}