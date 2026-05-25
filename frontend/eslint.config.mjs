import { defineConfig } from 'eslint/config';
import js from '@eslint/js';
import tseslint from 'typescript-eslint';
import hooks from 'eslint-plugin-react-hooks';

export default defineConfig([
  js.configs.recommended,
  ...tseslint.configs.recommended,
  {
    plugins: {
      'react-hooks': hooks,
    },
    rules: {
      '@typescript-eslint/no-unused-vars': ['error', { argsIgnorePattern: '^_' }],
      '@typescript-eslint/no-explicit-any': 'warn',
      'no-console': ['warn', { allow: ['warn', 'error'] }],
      'react-hooks/exhaustive-deps': 'warn',
    },
  },
  {
    ignores: [
      'dist/**',
      'node_modules/**',
      '.rsbuild/**',
      'rsbuild.config.ts',
      'postcss.config.mjs',
      'eslint.config.mjs',
      '**/*.gen.ts',
      '**/routeTree.gen.ts',
    ],
  },
]);
