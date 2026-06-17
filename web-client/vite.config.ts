import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  server: {
    host: '0.0.0.0',
    port: 5173,
  },
  build: {
    target: 'es2020',
    sourcemap: false,
    chunkSizeWarningLimit: 500,
    rollupOptions: {
      output: {
        manualChunks: {
          rnnoise: ['@timephy/rnnoise-wasm'],
          react: ['react', 'react-dom'],
          livekit: ['livekit-client'],
        },
      },
    },
  },
})
