import { createContext, useContext, useState, ReactNode } from 'react'
import type { User } from '../api/auth'

interface AuthContextValue {
  user: User | null
  token: string | null
  login: (email: string, password: string) => Promise<void>
  register: (username: string, email: string, password: string) => Promise<void>
  logout: () => void
}

const AuthContext = createContext<AuthContextValue | null>(null)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [token, setToken] = useState<string | null>(() => localStorage.getItem('token'))
  const [user, setUser] = useState<User | null>(() => {
    const u = localStorage.getItem('user')
    return u ? (JSON.parse(u) as User) : null
  })

  const _store = (t: string, u: User) => {
    localStorage.setItem('token', t)
    localStorage.setItem('user', JSON.stringify(u))
    setToken(t)
    setUser(u)
  }

  const login = async (email: string, password: string) => {
    const res = await fetch('/creator/api/auth/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email, password }),
    })
    if (!res.ok) {
      const body = await res.json().catch(() => ({}))
      throw new Error((body as { error?: string }).error ?? '登录失败')
    }
    const data = (await res.json()) as { token: string; user: User }
    _store(data.token, data.user)
  }

  const register = async (username: string, email: string, password: string) => {
    const res = await fetch('/creator/api/auth/register', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ username, email, password }),
    })
    if (!res.ok) {
      const body = await res.json().catch(() => ({}))
      throw new Error((body as { error?: string }).error ?? '注册失败')
    }
    const data = (await res.json()) as { token: string; user: User }
    _store(data.token, data.user)
  }

  const logout = () => {
    localStorage.removeItem('token')
    localStorage.removeItem('user')
    setToken(null)
    setUser(null)
  }

  return (
    <AuthContext.Provider value={{ user, token, login, register, logout }}>
      {children}
    </AuthContext.Provider>
  )
}

export function useAuth() {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth must be used within AuthProvider')
  return ctx
}
