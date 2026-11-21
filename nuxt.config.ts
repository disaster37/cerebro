export default defineNuxtConfig({
  ssr: false,
  app: {
    head: {
      title: 'cerebro',
      link: [
        { rel: 'icon', type: 'image/x-icon', href: '/favicon.ico' },
        { rel: 'icon', type: 'image/png', sizes: '512x512', href: '/favicon.png' },
        { rel: 'apple-touch-icon', href: '/favicon.png' },
      ],
    },
  },
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
  icon: {
    clientBundle: {
      scan: true,
    },
  },
  nitro: {
    output: {
      publicDir: 'output/public',
    },
  },
  compatibilityDate: '2025-01-01',
})
