import { defineConfig } from 'vite'
import { svelte } from '@sveltejs/vite-plugin-svelte'

export default defineConfig({
  plugins: [svelte()],
  // base relatif agar aset tetap resolve saat di-embed & dilayani di "/"
  base: './',
  build: {
    outDir: 'dist',
    emptyOutDir: true,
  },
  server: {
    // proxy API ke backend Go saat `yarn dev`
    proxy: {
      '/api': 'http://localhost:8080',
      '/berkas': 'http://localhost:8080',
    },
  },
})
