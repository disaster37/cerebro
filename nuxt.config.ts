export default defineNuxtConfig({
  ssr: false,
  modules: ['@nuxt/ui'],
  css: ['~/assets/css/main.css'],
  ui: {
    // Preserve current light-only behavior and avoid build-time font fetching.
    colorMode: false,
    fonts: false,
  },
  nitro: {
    output: {
      publicDir: 'output/public',
    },
  },
  compatibilityDate: '2025-01-01',
})
