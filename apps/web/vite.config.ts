import { defineConfig, loadEnv } from 'vite';
import react from '@vitejs/plugin-react';

function vendorChunk(id: string): string | undefined {
  if (!id.includes('node_modules')) {
    return undefined;
  }
  if (id.includes('react-dom') || id.includes('/react/')) {
    return 'react-vendor';
  }
  if (id.includes('@tanstack/react-query')) {
    return 'query-vendor';
  }
  if (id.includes('lucide-react')) {
    return 'icons-vendor';
  }
  return undefined;
}

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, '.', 'LEOTIME_');

  return {
    plugins: [react()],
    build: {
      rollupOptions: {
        output: {
          manualChunks(id) {
            return vendorChunk(id);
          },
        },
      },
    },
    server: {
      port: 5173,
      proxy: {
        '/api': env.LEOTIME_API_PROXY_TARGET ?? 'http://127.0.0.1:8080',
      },
    },
  };
});
