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
          primary: '#7C3AED',
          secondary: '#9333EA',
          cyan: '#06B6D4',
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
        'glass': '0 8px 32px 0 rgba(124, 58, 237, 0.15)',
        'glass-hover': '0 8px 40px 0 rgba(124, 58, 237, 0.25)',
        'glow': '0 0 20px rgba(124, 58, 237, 0.4)',
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
          '0%, 100%': { boxShadow: '0 0 20px rgba(124, 58, 237, 0.4)' },
          '50%': { boxShadow: '0 0 30px rgba(124, 58, 237, 0.6)' },
        },
      },
    },
  },
  plugins: [],
}
