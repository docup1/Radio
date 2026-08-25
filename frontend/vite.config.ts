import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { fileURLToPath, URL } from 'node:url'

export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  server: {
    host: '127.0.0.1',
    port: 5173,
    open: 'http://127.0.0.1:5173',
    proxy: {
      '/api': {
        target: 'http://127.0.0.1:18000',
        changeOrigin: true,
        ws: true,
        timeout: 0,
        proxyTimeout: 0,
      },
      '/dev-docs': { target: 'http://127.0.0.1:18000', changeOrigin: true },
      '/swagger': { target: 'http://127.0.0.1:18000', changeOrigin: true },
    },
  },
})
