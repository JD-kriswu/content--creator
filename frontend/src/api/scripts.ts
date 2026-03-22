import { api } from '../lib/request'

export interface Script {
  id: number
  title: string
  source_url: string
  similarity_score: number
  viral_score: number
  created_at: string
}

export function getScripts() {
  return api.get<{ scripts: Script[]; total: number }>('/scripts')
}

export function getScript(id: number) {
  return api.get<{ script: Script; content: string }>(`/scripts/${id}`)
}
