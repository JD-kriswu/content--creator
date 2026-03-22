import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

export default defineConfig({
  plugins: [react(), tailwindcss()],
  base: '/creator/',
  server: {
    port: 5173,
    proxy: {
      '/creator/api': {
        target: 'http://localhost:3004',
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/creator\/api/, '/api'),
      },
    },
  },
  build: {
    outDir: 'dist',
  },
})
