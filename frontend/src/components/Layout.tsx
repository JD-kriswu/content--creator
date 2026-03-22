// frontend/src/components/Layout.tsx
import { Outlet, Link, useLocation } from 'react-router'
import { Feather, Home, History, LogOut, Sun, Moon } from 'lucide-react'
import { useTheme } from 'next-themes'
import { Button } from './ui/button'
import { useAuth } from '../contexts/AuthContext'

export function Layout() {
  const location = useLocation()
  const { user, logout } = useAuth()
  const { theme, setTheme } = useTheme()

  const isActive = (path: string) => {
    if (path === '/') return location.pathname === '/'
    return location.pathname.startsWith(path)
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-blue-50 via-white to-purple-50 dark:from-gray-950 dark:via-gray-900 dark:to-gray-950">
      {/* Header */}
      <header className="sticky top-0 z-50 bg-white/80 dark:bg-gray-900/80 backdrop-blur-lg border-b border-gray-200 dark:border-gray-800">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex items-center justify-between h-16">
            <Link to="/" className="flex items-center gap-3">
              <div className="w-10 h-10 bg-gradient-to-br from-blue-500 to-purple-600 rounded-xl flex items-center justify-center shadow-lg shadow-blue-200">
                <Feather className="w-5 h-5 text-white" strokeWidth={2.5} />
              </div>
              <div className="flex flex-col">
                <span className="text-xl font-semibold bg-gradient-to-r from-blue-600 to-purple-600 bg-clip-text text-transparent">
                  轻写Claw
                </span>
                <span className="text-xs text-gray-500 dark:text-gray-400 hidden sm:block">你的AI文案助手</span>
              </div>
            </Link>

            <nav className="flex items-center gap-2">
              {user && (
                <>
                  <Link
                    to="/dashboard"
                    className={`hidden sm:flex items-center gap-2 px-3 py-2 rounded-lg transition-colors text-sm ${
                      isActive('/dashboard') ? 'bg-blue-50 text-blue-600 dark:bg-blue-950 dark:text-blue-400' : 'text-gray-600 dark:text-gray-400 hover:bg-gray-50 dark:hover:bg-gray-800'
                    }`}
                  >
                    <Home className="w-4 h-4" />
                    <span>创作</span>
                  </Link>
                  <Link
                    to="/history"
                    className={`hidden sm:flex items-center gap-2 px-3 py-2 rounded-lg transition-colors text-sm ${
                      isActive('/history') ? 'bg-blue-50 text-blue-600 dark:bg-blue-950 dark:text-blue-400' : 'text-gray-600 dark:text-gray-400 hover:bg-gray-50 dark:hover:bg-gray-800'
                    }`}
                  >
                    <History className="w-4 h-4" />
                    <span>历史</span>
                  </Link>
                </>
              )}

              {/* Dark mode toggle */}
              <Button
                variant="ghost"
                size="icon"
                onClick={() => setTheme(theme === 'dark' ? 'light' : 'dark')}
                className="w-9 h-9 text-gray-600 dark:text-gray-400"
              >
                <Sun className="h-4 w-4 rotate-0 scale-100 transition-all dark:-rotate-90 dark:scale-0" />
                <Moon className="absolute h-4 w-4 rotate-90 scale-0 transition-all dark:rotate-0 dark:scale-100" />
              </Button>

              {user ? (
                <Button variant="ghost" size="sm" onClick={logout} className="text-gray-600 dark:text-gray-400">
                  <LogOut className="w-4 h-4 mr-1" />
                  <span className="hidden sm:inline">退出</span>
                </Button>
              ) : (
                <>
                  <Link to="/auth">
                    <Button variant="ghost" className="text-gray-700 dark:text-gray-300">登录</Button>
                  </Link>
                  <Link to="/auth">
                    <Button className="bg-gradient-to-r from-blue-600 to-purple-600 hover:from-blue-700 hover:to-purple-700 text-white border-0">
                      免费试用
                    </Button>
                  </Link>
                </>
              )}
            </nav>
          </div>
        </div>
      </header>

      <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <Outlet />
      </main>

      {/* Mobile bottom nav (only when logged in) */}
      {user && (
        <>
          <nav className="md:hidden fixed bottom-0 left-0 right-0 bg-white dark:bg-gray-900 border-t border-gray-200 dark:border-gray-800 z-50">
            <div className="grid grid-cols-3 gap-1 px-2 py-2">
              <Link
                to="/dashboard"
                className={`flex flex-col items-center gap-1 px-3 py-2 rounded-lg transition-colors ${
                  isActive('/dashboard') ? 'bg-blue-50 text-blue-600 dark:bg-blue-950 dark:text-blue-400' : 'text-gray-600 dark:text-gray-400'
                }`}
              >
                <Home className="w-5 h-5" />
                <span className="text-xs">创作</span>
              </Link>
              <Link
                to="/history"
                className={`flex flex-col items-center gap-1 px-3 py-2 rounded-lg transition-colors ${
                  isActive('/history') ? 'bg-blue-50 text-blue-600 dark:bg-blue-950 dark:text-blue-400' : 'text-gray-600 dark:text-gray-400'
                }`}
              >
                <History className="w-5 h-5" />
                <span className="text-xs">历史</span>
              </Link>
              <button
                onClick={logout}
                className="flex flex-col items-center gap-1 px-3 py-2 rounded-lg transition-colors text-gray-600 dark:text-gray-400"
              >
                <LogOut className="w-5 h-5" />
                <span className="text-xs">退出</span>
              </button>
            </div>
          </nav>
          <div className="md:hidden h-20" />
        </>
      )}
    </div>
  )
}
