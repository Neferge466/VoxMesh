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
        manualChunks(id: string) {
          if (id.includes('node_modules/@timephy/rnnoise-wasm')) return 'rnnoise';
          if (id.includes('node_modules/react') || id.includes('node_modules/react-dom')) return 'react';
          if (id.includes('node_modules/livekit-client')) return 'livekit';
        },
      },
    },
  },
})
