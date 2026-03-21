import request from './request'

export function getSession() {
  return request.get<{ session_id: string; state: string }>('/chat/session')
}

export function resetSession() {
  return request.post<{ message: string; conv_id: number }>('/chat/reset')
}

// Returns a fetch Response with ReadableStream for SSE (POST, not EventSource)
export async function sendMessage(message: string): Promise<Response> {
  const token = localStorage.getItem('token') ?? ''
  return fetch('/creator/api/chat/message', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
    body: JSON.stringify({ message })
  })
}
