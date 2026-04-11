import { api } from '../lib/request'

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

export interface BindQRCodeResponse {
  qrcode_url: string
  bind_token: string
}

export interface BindStatusResponse {
  status: 'pending' | 'success' | 'error'
  app_id?: string
  bot_name?: string
}

export function getBindQRCode(): Promise<BindQRCodeResponse> {
  return api.get('/feishu/bind-qrcode')
}

export function getBindStatus(token: string): Promise<BindStatusResponse> {
  return api.get(`/feishu/bind-status/${token}`)
}