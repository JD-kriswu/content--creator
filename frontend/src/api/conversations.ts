import request from './request'

export interface Conversation {
  id: number
  user_id: number
  title: string
  script_id?: number
  state: number // 0=in_progress 1=completed
  created_at: string
  updated_at: string
}

export interface StoredMsg {
  role: string
  type: string
  content?: string
  data?: any
  options?: string[]
  step?: number
  name?: string
}

export function listConversations() {
  return request.get<{ conversations: Conversation[] }>('/conversations')
}

export function getConversation(id: number) {
  return request.get<{ conversation: Conversation; messages: string }>(`/conversations/${id}`)
}
