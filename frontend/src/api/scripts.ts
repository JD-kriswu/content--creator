import request from './request'

export interface Script {
  id: number
  title: string
  source_url: string
  platform: string
  similarity_score: number
  viral_score: number
  created_at: string
}

export function getScripts() {
  return request.get<{ scripts: Script[]; total: number }>('/scripts')
}

export function getScript(id: number) {
  return request.get<{ script: Script; content: string }>(`/scripts/${id}`)
}
