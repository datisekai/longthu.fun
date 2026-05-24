import { defineConfig } from '@rsbuild/core';
import { pluginReact } from '@rsbuild/plugin-react';
import { TanStackRouterRspack } from '@tanstack/router-plugin/rspack';
import path from 'node:path';

// Docs: https://rsbuild.rs/config/
export default defineConfig({
  plugins: [pluginReact()],
  source: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  html: {
    template: './index.html',
    title: 'Longthu.fun',
    meta: {
      description: 'Chia bill cầu lông cho nhóm chơi — tự động khớp thanh toán.',
    },
  },
  server: {
    port: 5173,
  },
  tools: {
    rspack: {
      plugins: [
        TanStackRouterRspack({
          target: 'react',
          autoCodeSplitting: true,
          routesDirectory: './src/routes',
          generatedRouteTree: './src/routeTree.gen.ts',
        }),
      ],
    },
  },
});
