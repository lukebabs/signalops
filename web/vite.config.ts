import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

// Server-side proxy target. The browser calls same-origin paths (/healthz,
// /readyz, /v1) and Vite forwards them to the gateway, avoiding CORS (the
// gateway has no CORS middleware). Override with SIGNALOPS_GATEWAY_URL.
const GATEWAY = process.env.SIGNALOPS_GATEWAY_URL ?? 'http://localhost:18000';

export default defineConfig({
  plugins: [react()],
  server: {
    host: '0.0.0.0',
    port: 5173,
    proxy: {
      '/healthz': { target: GATEWAY, changeOrigin: true },
      '/readyz': { target: GATEWAY, changeOrigin: true },
      '/v1': { target: GATEWAY, changeOrigin: true },
    },
  },
  preview: {
    host: '0.0.0.0',
    port: 5173,
  },
  build: {
    chunkSizeWarningLimit: 1500,
    rollupOptions: {
      output: {
        manualChunks: {
          router: ['@tanstack/react-router', '@tanstack/react-query', 'zustand'],
          echarts: ['echarts', 'echarts-for-react'],
          aggrid: ['ag-grid-react', 'ag-grid-community'],
        },
      },
    },
  },
});
