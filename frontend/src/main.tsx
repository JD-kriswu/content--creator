// src/main.tsx
import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import './styles/theme.css'
import './styles/index.css'

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <div className="p-4 text-2xl font-semibold bg-gradient-to-r from-blue-600 to-purple-600 bg-clip-text text-transparent">
      轻写Claw 样式测试
    </div>
  </StrictMode>
)
