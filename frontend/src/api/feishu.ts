import { api } from '../lib/request'

const BASE = '/creator/api'

export interface FeishuBot {
  id: number
  user_id: number
  app_id: string
  bot_name: string
  ws_connected: boolean
  created_at: string
}

export function getFeishuBots() {
  return api.get<{ bots: FeishuBot[] }>('/feishu/bots')
}

export function unbindFeishuBot(botId: number) {
  return api.delete<{ message: string }>(`/feishu/bots/${botId}`)
}

export interface BindStatusResponse {
  status: 'pending' | 'scanning' | 'creating' | 'success' | 'error'
  app_id?: string
  bot_name?: string
  qrcode?: string
  error?: string
}

export function getBindStatus(token: string): Promise<BindStatusResponse> {
  return api.get(`/feishu/bind-status/${token}`)
}

export function cancelBind(token: string) {
  return api.delete(`/feishu/bind/${token}`)
}

// SSE event types for bind flow
export interface BindSSEEvent {
  type: 'init' | 'qrcode' | 'status' | 'info' | 'success' | 'error'
  data: {
    bind_token?: string
    line?: string
    status?: string
    message?: string
    app_id?: string
    bot_name?: string
  }
}

// Start bind flow with SSE stream
export function startBindFlow(): Promise<Response> {
  const url = `${BASE}/feishu/bind-stream`

  return fetch(url, {
    headers: {
      'Authorization': `Bearer ${localStorage.getItem('token')}`,
    },
  })
}