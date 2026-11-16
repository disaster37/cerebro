export default defineNuxtConfig({
  ssr: false,
  nitro: {
    output: {
      publicDir: 'output/public',
    },
  },
  css: ['bootstrap/dist/css/bootstrap.min.css', '~/assets/css/main.css'],
  compatibilityDate: '2025-01-01',
})
