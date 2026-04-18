import { api } from '../lib/request'

export function getSession() {
  return api.get<{ state: string }>('/chat/session')
}

export function resetSession() {
  return api.post<{ message: string; conv_id: number }>('/chat/reset')
}

// Returns raw Response — caller handles ReadableStream
export async function sendMessage(message: string, convId?: number): Promise<Response> {
  const token = localStorage.getItem('token') ?? ''
  const body: { message: string; conv_id?: number } = { message }
  if (convId !== undefined) {
    body.conv_id = convId
  }
  return fetch('/creator/api/chat/message', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify(body),
  })
}
