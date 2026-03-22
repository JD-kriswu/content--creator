const BASE = '/creator/api'

export class ApiError extends Error {
  constructor(public status: number, message: string) {
    super(message)
  }
}

async function apiFetch<T>(path: string, init?: RequestInit): Promise<T> {
  const token = localStorage.getItem('token')
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...(token ? { Authorization: `Bearer ${token}` } : {}),
    ...(init?.headers as Record<string, string>),
  }
  const res = await fetch(`${BASE}${path}`, { ...init, headers })
  if (res.status === 401) {
    localStorage.removeItem('token')
    localStorage.removeItem('user')
    window.location.href = '/creator/auth'
    throw new ApiError(401, 'Unauthorized')
  }
  if (!res.ok) {
    const body = await res.json().catch(() => ({}))
    throw new ApiError(res.status, (body as { error?: string }).error ?? res.statusText)
  }
  return res.json() as Promise<T>
}

export const api = {
  get: <T>(path: string) => apiFetch<T>(path),
  post: <T>(path: string, body?: unknown) =>
    apiFetch<T>(path, { method: 'POST', body: body !== undefined ? JSON.stringify(body) : undefined }),
}
