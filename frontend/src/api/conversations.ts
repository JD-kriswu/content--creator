import { api } from '../lib/request'

export interface Conversation {
  id: number
  user_id: number
  title: string
  script_id?: number
  state: number // 0=进行中 1=完成
  created_at: string
}

export interface StoredMsg {
  role: string
  type: string
  content?: string
  data?: unknown
  options?: string[]
  step?: number
  name?: string
}

export function listConversations() {
  return api.get<{ conversations: Conversation[] }>('/conversations')
}

export function getConversation(id: number) {
  return api.get<{ conversation: Conversation; messages: string }>(`/conversations/${id}`)
}
