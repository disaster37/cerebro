export default defineNuxtConfig({
  ssr: false,
  modules: ['@nuxt/ui'],
  css: ['~/assets/css/main.css'],
  colorMode: {
    preference: 'dark',
    fallback: 'dark',
    classSuffix: '',
    storageKey: 'cerebro-color-mode',
  },
  ui: {
    colorMode: true,
    fonts: false,
  },
  nitro: {
    output: {
      publicDir: 'output/public',
    },
  },
  compatibilityDate: '2025-01-01',
})
