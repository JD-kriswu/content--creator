import { api } from '../lib/request'

export interface User {
  id: number
  username: string
  email: string
}

export function login(email: string, password: string) {
  return api.post<{ token: string; user: User }>('/auth/login', { email, password })
}

export function register(username: string, email: string, password: string) {
  return api.post<{ token: string; user: User }>('/auth/register', { username, email, password })
}
