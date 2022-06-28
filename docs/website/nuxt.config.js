import { createServerDirective } from '@chakra-ui/vue/src/directives'
import { defaultTheme } from '@chakra-ui/vue'

export default {
  // Target: https://go.nuxtjs.dev/config-target
  mode: 'universal',
  options: {
    target: 'static'
  },

  // Global page headers: https://go.nuxtjs.dev/config-head
  head: {
    title: 'Couper Documentation',
    htmlAttrs: {
      lang: 'en'
    },
    meta: [
      { charset: 'utf-8' },
      { name: 'viewport', content: 'width=device-width, initial-scale=1' },
      { hid: 'description', name: 'description', content: 'Couper documentation' },
      { name: 'format-detection', content: 'telephone=no' }
    ],
    link: [
      { rel: 'icon', type: 'image/x-icon', href: '/favicon.ico' }
    ]
  },

  // Global CSS: https://go.nuxtjs.dev/config-css
  css: [
  ],

  // Plugins to run before rendering page: https://go.nuxtjs.dev/config-plugins
  plugins: [
  ],

  // Auto import components: https://go.nuxtjs.dev/config-components
  components: true,

  // Modules for dev and build (recommended): https://go.nuxtjs.dev/config-modules
  buildModules: [
    '@nuxtjs/mdx'
  ],

  // Modules: https://go.nuxtjs.dev/config-modules
  modules: [
    '@chakra-ui/nuxt',
    '@nuxtjs/emotion'
  ],

  router: {
    prefetchLinks: true
  },

  extensions: [
    'mdx'
  ],

  render: {
    bundleRenderer: {
      directives: {
        chakra: createServerDirective(defaultTheme)
      }
    }
  },

  // Build Configuration: https://go.nuxtjs.dev/config-build
  build: {
    transpile: [
      '@chakra-ui/vue',
      '@chakra-ui/theme-vue'
    ],
    additionalExtensions: [
      '.mdx'
    ],
    extend (config, ctx) {
      config.resolve.alias.vue = 'vue/dist/vue.common'
    }
  }
}
