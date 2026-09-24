/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        background: '#090D16',
        surface: '#0F172A',
        surfaceLight: '#1E293B',
        accentBlue: '#3B82F6',
        accentCyan: '#06B6D4',
        accentGreen: '#10B981',
        accentRed: '#EF4444',
        accentAmber: '#F59E0B',
      },
      fontFamily: {
        sans: ['Plus Jakarta Sans', 'system-ui', 'sans-serif'],
        mono: ['JetBrains Mono', 'monospace'],
      },
    },
  },
  plugins: [],
}
