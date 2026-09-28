import { defineConfig } from 'tsup';

export default defineConfig({
  entry: ['src/index.ts'],
  format: ['esm', 'cjs'],
  sourcemap: true,
  clean: true,
  target: 'es2018',
  minify: true,
  splitting: false,
  treeshake: true,
  shims: true,
});
