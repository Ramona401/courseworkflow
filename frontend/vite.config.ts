import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import path from 'path'

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  build: {
    rollupOptions: {
      output: {
        /**
         * v142 分包优化:
         *   1. vendor-react: React 生态核心库(极少变化,长期缓存)
         *   2. vendor-docx:  Word 导出库(仅点击导出时按需加载)
         *   3. vendor-axios: HTTP 客户端(独立缓存)
         */
        manualChunks(id) {
          // React 生态核心 → vendor-react
          if (
            id.includes('node_modules/react/') ||
            id.includes('node_modules/react-dom/') ||
            id.includes('node_modules/react-router') ||
            id.includes('node_modules/scheduler/')
          ) {
            return 'vendor-react'
          }
          // docx + file-saver → vendor-docx(动态 import 按需加载)
          if (id.includes('node_modules/docx/') || id.includes('node_modules/file-saver/')) {
            return 'vendor-docx'
          }
          // axios → vendor-axios
          if (id.includes('node_modules/axios/')) {
            return 'vendor-axios'
          }
        },
      },
    },
  },
  server: {
    proxy: {
      '/api': {
        target: 'http://127.0.0.1:8080',
        changeOrigin: true,
      },
    },
  },
})
