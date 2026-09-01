import tailwindcss from '@tailwindcss/vite'

// https://nuxt.com/docs/api/configuration/nuxt-config
export default defineNuxtConfig({
  compatibilityDate: '2025-07-15',
  devtools: { enabled: false },
  css: ['~/assets/css/main.css'],
  vite: {
    plugins: [tailwindcss()],
  },
  runtimeConfig: {
    // Server-only. The Nitro proxy (web/server/routes/v1/[...].ts) forwards
    // same-origin /v1/* requests here. The browser bundle never sees this
    // value, so the frontend never hardcodes an API host.
    apiBase: process.env.NUXT_API_BASE || 'http://localhost:8080',
  },
  app: {
    head: {
      title: 'Cyto AI — Risk Dashboard',
      meta: [
        { charset: 'utf-8' },
        { name: 'viewport', content: 'width=device-width, initial-scale=1' },
      ],
      link: [
        {
          rel: 'stylesheet',
          href: 'https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&family=JetBrains+Mono:wght@400;600;700&display=swap',
        },
      ],
    },
  },
})

