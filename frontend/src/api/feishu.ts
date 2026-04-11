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