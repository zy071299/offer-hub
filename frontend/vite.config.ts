import { fileURLToPath, URL } from 'node:url'
import tailwindcss from '@tailwindcss/vite'
import react from '@vitejs/plugin-react'
import { defineConfig } from 'vite'

export default defineConfig({
  plugins: [react(), tailwindcss()],
  server: {
    port: 3001,
    proxy: {
      '/api': {
        target: 'http://localhost:8180',
        changeOrigin: true,
      },
      '/auth': {
        target: 'http://localhost:8180',
        changeOrigin: true,
      },
      '/home': {
        target: 'http://localhost:8180',
        changeOrigin: true,
      },
    },
  },
  resolve: {
    alias: { '@': fileURLToPath(new URL('./src', import.meta.url)) },
  },
})
