import tailwindcss from '@tailwindcss/vite'
import react from '@vitejs/plugin-react'
import { defineConfig } from 'vite'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react(), tailwindcss()],
  server: {
    // During `npm run dev`, forward API calls to the Go backend.
    proxy: {
      '/api': 'http://localhost:8080',
    },
  },
  build: {
    chunkSizeWarningLimit: 1200,
  },
})
