import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { resolve } from 'path'
import tailwindcss from 'tailwindcss'
import autoprefixer from 'autoprefixer'

// 端口通过环境变量注入，默认与一键启动脚本的固定端口一致
// （scripts/start-dev.sh：前端 28301、后端 28302）。
const frontendPort = Number(process.env.PROMPTOS_FRONTEND_PORT || 28301)
const backendPort = Number(process.env.PROMPTOS_BACKEND_PORT || 28302)

export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      '@': resolve(__dirname, 'src')
    }
  },
  css: {
    postcss: {
      plugins: [tailwindcss, autoprefixer]
    }
  },
  server: {
    port: frontendPort,
    strictPort: true,
    proxy: {
      '/api': {
        target: `http://localhost:${backendPort}`,
        changeOrigin: true
      },
      '/uploads': {
        target: `http://localhost:${backendPort}`,
        changeOrigin: true
      }
    }
  }
})
