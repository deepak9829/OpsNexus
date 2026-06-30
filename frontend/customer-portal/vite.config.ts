import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import path from 'path'

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  server: {
    port: 3000,
    proxy: {
      '/api/v1/auth': 'http://localhost:8081',
      '/api/v1/tenants': 'http://localhost:8082',
      '/api/v1/profiles': 'http://localhost:8082',
      '/api/v1/organizations': 'http://localhost:8082',
      '/api/v1/cases': 'http://localhost:8083',
      '/api/v1/tasks': 'http://localhost:8083',
      '/api/v1/workflows': 'http://localhost:8083',
      '/api/v1/forms': 'http://localhost:8084',
      '/api/v1/documents': 'http://localhost:8084',
      '/api/v1/submissions': 'http://localhost:8084',
      '/api/v1/notifications': 'http://localhost:8085',
      '/api/v1/audit-events': 'http://localhost:8085',
    },
  },
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: ['./src/test-setup.ts'],
    coverage: {
      reporter: ['text', 'lcov'],
    },
  },
})
