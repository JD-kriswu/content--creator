import { api } from '../lib/request'

export interface StyleDoc {
  style_doc: string
  style_vector: string
  style_version: number
  is_initialized: boolean
}

export function getStyleDoc() {
  return api.get<StyleDoc>('/user/style/doc')
}

// Returns raw Response — caller handles SSE ReadableStream
export async function initStyleSSE(scripts: string[]): Promise<Response> {
  const token = localStorage.getItem('token') ?? ''
  return fetch('/creator/api/user/style/init', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify({ scripts }),
  })
}
