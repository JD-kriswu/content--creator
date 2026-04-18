import { api } from '../lib/request'

export interface PromptFile {
  path: string
  name: string
  display_name: string
  content: string
}

export interface PromptsResponse {
  prompts: PromptFile[]
}

export interface UpdatePromptRequest {
  path: string
  content: string
}

export function getPrompts() {
  return api.get<PromptsResponse>('/prompts')
}

export function updatePrompt(data: UpdatePromptRequest) {
  return api.put<{ message: string; path: string; name: string; display_name: string }>('/prompts', data)
}