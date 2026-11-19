import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import wails from '@wailsio/runtime/plugins/vite'
import { fileURLToPath, URL } from 'node:url'
import process from 'node:process'

export default defineConfig({
  server: {
    host: '127.0.0.1',
    port: Number(process.env.WAILS_VITE_PORT) || 9245,
    strictPort: true,
  },
  plugins: [react(), tailwindcss(), wails('./bindings')],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  build: {
    outDir:'dist', emptyOutDir:true,
    rollupOptions:{ output:{ manualChunks(id) {
      if (id.includes('/node_modules/@tiptap/') || id.includes('/node_modules/prosemirror-')) return 'editor'
      if (id.includes('/node_modules/react') || id.includes('/node_modules/scheduler/')) return 'react'
    } } },
  },
  test: { environment: 'jsdom', setupFiles: './src/test/setup.ts', css: true, globals: true },
})
