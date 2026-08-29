import { fileURLToPath, URL } from 'node:url'
import vue from '@vitejs/plugin-vue'
import tailwindcss from 'tailwindcss'
import autoprefixer from 'autoprefixer'
import { defineConfig } from 'vitest/config'

// 与 vite.config.ts 保持一致：同样的别名与 PostCSS（Tailwind @apply 需要）。
// 测试环境固定走 mock 数据（VITE_ENABLE_PROMPT_API=false），不访问网络。
export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url))
    }
  },
  css: {
    postcss: {
      plugins: [tailwindcss, autoprefixer]
    }
  },
  test: {
    environment: 'jsdom',
    setupFiles: ['./vitest.setup.ts'],
    env: {
      VITE_ENABLE_PROMPT_API: 'false'
    }
  }
})
