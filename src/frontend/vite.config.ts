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
  build: {
    // 按依赖分组产出 vendor chunk：首屏主包更小，第三方库可长期缓存复用。
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (!id.includes('node_modules')) return
          if (id.includes('naive-ui')) return 'vendor-ui'
          if (id.includes('date-fns')) return 'vendor-date'
          if (id.includes('lucide-vue-next')) return 'vendor-icons'
          if (id.includes('vue-router') || id.includes('pinia') || id.includes('/vue/') || id.includes('@vue/')) return 'vendor-vue'
          return 'vendor'
        }
      }
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
