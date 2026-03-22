import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { AuthProvider, useAuth } from '../AuthContext'

function StateDisplay() {
  const { user, token } = useAuth()
  return (
    <div>
      <span data-testid="token">{token ?? 'none'}</span>
      <span data-testid="email">{user?.email ?? 'none'}</span>
    </div>
  )
}

describe('AuthContext', () => {
  beforeEach(() => {
    localStorage.clear()
    vi.restoreAllMocks()
  })

  it('初始化时从 localStorage 读取 token 和 user', () => {
    localStorage.setItem('token', 'stored-token')
    localStorage.setItem('user', JSON.stringify({ id: 1, username: 'u', email: 'u@test.com' }))
    render(<AuthProvider><StateDisplay /></AuthProvider>)
    expect(screen.getByTestId('token').textContent).toBe('stored-token')
    expect(screen.getByTestId('email').textContent).toBe('u@test.com')
  })

  it('login 成功后存入 localStorage 并更新状态', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({
        token: 'new-token',
        user: { id: 2, username: 'v', email: 'v@test.com' },
      }),
    }))

    function LoginBtn() {
      const { login } = useAuth()
      return <button onClick={() => login('v@test.com', 'pass123')}>login</button>
    }

    render(<AuthProvider><StateDisplay /><LoginBtn /></AuthProvider>)
    await userEvent.click(screen.getByRole('button'))
    await waitFor(() => expect(screen.getByTestId('token').textContent).toBe('new-token'))
    expect(localStorage.getItem('token')).toBe('new-token')
  })

  it('logout 清除 localStorage 并重置状态', async () => {
    localStorage.setItem('token', 'old-token')
    localStorage.setItem('user', JSON.stringify({ id: 1, username: 'u', email: 'u@test.com' }))

    function LogoutBtn() {
      const { logout } = useAuth()
      return <button onClick={logout}>logout</button>
    }

    render(<AuthProvider><StateDisplay /><LogoutBtn /></AuthProvider>)
    await userEvent.click(screen.getByRole('button'))
    await waitFor(() => expect(screen.getByTestId('token').textContent).toBe('none'))
    expect(localStorage.getItem('token')).toBeNull()
  })

  it('login 失败时抛出错误', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: false,
      json: () => Promise.resolve({ error: '密码错误' }),
    }))

    let caughtError = ''

    function LoginBtn() {
      const { login } = useAuth()
      return (
        <button
          onClick={() =>
            login('a@b.com', 'wrong').catch((e: Error) => {
              caughtError = e.message
            })
          }
        >
          login
        </button>
      )
    }

    render(<AuthProvider><StateDisplay /><LoginBtn /></AuthProvider>)
    await userEvent.click(screen.getByRole('button'))
    await waitFor(() => expect(caughtError).toBe('密码错误'))
  })
})
