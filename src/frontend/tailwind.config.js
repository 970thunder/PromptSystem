/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{vue,js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        // Primary dark background
        dark: {
          900: '#0B0F19',
          800: '#111827',
          700: '#1F2937',
        },
        // Glassmorphism accent
        glass: {
          primary: '#111111',
          secondary: '#333333',
          accent: '#f5f3ee',
        },
        warm: {
          page: '#f5f3ee',
          surface: '#faf8f4',
          muted: '#f6f4ef',
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
        'card': '20px',
        'button': '14px',
        'input': '16px',
      },
      boxShadow: {
        'glass': '0 16px 40px rgba(15, 23, 42, 0.06)',
        'glass-hover': '0 20px 48px rgba(15, 23, 42, 0.1)',
        'glow': '0 0 20px rgba(0, 0, 0, 0.08)',
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
