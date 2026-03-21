import request from './request'

export interface LoginResp {
  token: string
  user: { id: number; username: string; email: string; role: string }
}

export function login(email: string, password: string) {
  return request.post<LoginResp>('/auth/login', { email, password })
}

export function register(username: string, email: string, password: string) {
  return request.post('/auth/register', { username, email, password })
}
