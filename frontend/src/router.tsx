// frontend/src/router.tsx
import { createBrowserRouter, Navigate, Outlet } from 'react-router'
import { useAuth } from './contexts/AuthContext'
import { Layout } from './components/Layout'
import { Home } from './pages/Home'
import { Auth } from './pages/Auth'
import { Dashboard } from './pages/Dashboard'
import { Result } from './pages/Result'
import { History } from './pages/History'

function ProtectedRoute() {
  const { token } = useAuth()
  return token ? <Outlet /> : <Navigate to="/auth" replace />
}

export const router = createBrowserRouter(
  [
    {
      path: '/',
      Component: Layout,
      children: [
        { index: true, Component: Home },
        { path: 'auth', Component: Auth },
        {
          Component: ProtectedRoute,
          children: [
            { path: 'dashboard', Component: Dashboard },
            { path: 'result/:id', Component: Result },
            { path: 'history', Component: History },
          ],
        },
      ],
    },
  ],
  { basename: '/creator' }
)
