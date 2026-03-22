import { Outlet, Link, useLocation } from "react-router";
import { Feather, Home, History, LogIn } from "lucide-react";
import { Button } from "./ui/button";

export function Layout() {
  const location = useLocation();
  
  // 模拟未登录状态
  const isLoggedIn = false;
  
  const isActive = (path: string) => {
    if (path === "/") {
      return location.pathname === "/";
    }
    return location.pathname.startsWith(path);
  };

  return (
    <div className="min-h-screen bg-gradient-to-br from-blue-50 via-white to-purple-50">
      {/* Header */}
      <header className="sticky top-0 z-50 bg-white/80 backdrop-blur-lg border-b border-gray-200">
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
                <span className="text-xs text-gray-500 hidden sm:block">你的AI文案助手</span>
              </div>
            </Link>

            <nav className="flex items-center gap-4">
              {isLoggedIn ? (
                <>
                  <Link
                    to="/"
                    className={`hidden sm:flex items-center gap-2 px-3 py-2 rounded-lg transition-colors ${
                      isActive("/") && !isActive("/history") && !isActive("/auth")
                        ? "bg-blue-50 text-blue-600"
                        : "text-gray-600 hover:text-gray-900 hover:bg-gray-50"
                    }`}
                  >
                    <Home className="w-4 h-4" />
                    <span>首页</span>
                  </Link>
                  <Link
                    to="/history"
                    className={`hidden sm:flex items-center gap-2 px-3 py-2 rounded-lg transition-colors ${
                      isActive("/history")
                        ? "bg-blue-50 text-blue-600"
                        : "text-gray-600 hover:text-gray-900 hover:bg-gray-50"
                    }`}
                  >
                    <History className="w-4 h-4" />
                    <span>历史记录</span>
                  </Link>
                </>
              ) : null}
              
              <Link to="/auth">
                <Button
                  variant="ghost"
                  className="text-gray-700 hover:text-gray-900"
                >
                  登录
                </Button>
              </Link>
              
              {!isLoggedIn && (
                <Link to="/">
                  <Button className="bg-gradient-to-r from-blue-600 to-purple-600 hover:from-blue-700 hover:to-purple-700 text-white border-0">
                    免费试用
                  </Button>
                </Link>
              )}
              
              {isLoggedIn && (
                <Link to="/auth">
                  <Button
                    variant="outline"
                  >
                    账户
                  </Button>
                </Link>
              )}
            </nav>
          </div>
        </div>
      </header>

      {/* Main Content */}
      <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <Outlet />
      </main>

      {/* Mobile Bottom Navigation - Only show when logged in */}
      {isLoggedIn && (
        <>
          <nav className="md:hidden fixed bottom-0 left-0 right-0 bg-white border-t border-gray-200 z-50">
            <div className="grid grid-cols-3 gap-1 px-2 py-2">
              <Link
                to="/"
                className={`flex flex-col items-center gap-1 px-3 py-2 rounded-lg transition-colors ${
                  isActive("/") && !isActive("/history") && !isActive("/auth")
                    ? "bg-blue-50 text-blue-600"
                    : "text-gray-600"
                }`}
              >
                <Home className="w-5 h-5" />
                <span className="text-xs">首页</span>
              </Link>
              <Link
                to="/history"
                className={`flex flex-col items-center gap-1 px-3 py-2 rounded-lg transition-colors ${
                  isActive("/history") ? "bg-blue-50 text-blue-600" : "text-gray-600"
                }`}
              >
                <History className="w-5 h-5" />
                <span className="text-xs">历史</span>
              </Link>
              <Link
                to="/auth"
                className={`flex flex-col items-center gap-1 px-3 py-2 rounded-lg transition-colors ${
                  isActive("/auth") ? "bg-blue-50 text-blue-600" : "text-gray-600"
                }`}
              >
                <LogIn className="w-5 h-5" />
                <span className="text-xs">账户</span>
              </Link>
            </div>
          </nav>

          {/* Spacer for mobile navigation */}
          <div className="md:hidden h-20" />
        </>
      )}
    </div>
  );
}