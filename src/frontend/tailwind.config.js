// 文件作用：集中定义 PromptOS 前端 Tailwind 设计 token，供页面和 Naive UI 主题复用。
/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{vue,js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        brand: {
          ink: '#0F172A',
          muted: '#475569',
          subtle: '#64748B',
          line: 'rgba(37, 99, 235, 0.14)',
        },
        surface: {
          page: '#F6F9FF',
          card: '#FFFFFF',
          soft: '#EEF5FF',
          wash: '#EAF2FF',
          inverse: '#0F172A',
        },
        accent: {
          success: '#22a06b',
          warning: '#b7791f',
          danger: '#c0392b',
          info: '#2563eb',
        },
        // Primary dark background
        dark: {
          900: '#0B0F19',
          800: '#111827',
          700: '#1F2937',
        },
        // Glassmorphism accent
        glass: {
          primary: '#2563EB',
          secondary: '#1D4ED8',
          accent: '#EEF5FF',
        },
        // Neutral
        neutral: {
          100: '#FFFFFF',
          400: '#A1A1AA',
          700: '#3F3F46',
          800: '#27272A',
          900: '#18181B',
        }
      },
      borderRadius: {
        'sm': '8px',
        'md': '12px',
        'lg': '16px',
        'xl': '20px',
        'card': '20px',
        'button': '9999px',
        'input': '16px',
      },
      boxShadow: {
        'panel': '0 16px 40px rgba(15, 23, 42, 0.06)',
        'panel-hover': '0 20px 48px rgba(15, 23, 42, 0.1)',
        'focus': '0 0 0 3px rgba(17, 17, 17, 0.12)',
        'glass': '0 16px 40px rgba(15, 23, 42, 0.06)',
        'glass-hover': '0 20px 48px rgba(15, 23, 42, 0.1)',
        'glow': '0 0 20px rgba(0, 0, 0, 0.08)',
      },
      transitionTimingFunction: {
        'standard': 'cubic-bezier(0.2, 0, 0, 1)',
      },
      backdropBlur: {
        'glass': '20px',
      },
      animation: {
        'hover-up': 'hoverUp 0.2s ease-out',
        'glow-pulse': 'glowPulse 2s ease-in-out infinite',
      },
      keyframes: {
        hoverUp: {
          '0%': { transform: 'translateY(0)' },
          '100%': { transform: 'translateY(-4px)' },
        },
        glowPulse: {
          '0%, 100%': { boxShadow: '0 0 20px rgba(0, 0, 0, 0.08)' },
          '50%': { boxShadow: '0 0 30px rgba(0, 0, 0, 0.12)' },
        },
      },
    },
  },
  plugins: [],
}
