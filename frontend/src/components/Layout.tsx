// frontend/src/components/Layout.tsx (stub)
import { Outlet } from 'react-router'
export function Layout() {
  return (
    <div className="min-h-screen bg-gradient-to-br from-blue-50 via-white to-purple-50 dark:from-gray-950 dark:via-gray-900 dark:to-gray-950">
      <Outlet />
    </div>
  )
}
