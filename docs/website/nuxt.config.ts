import colors from 'tailwindcss/colors.js'
import { defineNuxtConfig } from 'nuxt'

// https://v3.nuxtjs.org/api/configuration/nuxt.config
export default defineNuxtConfig({
    ssr: true, // false: client-side rendering
    // components: {
    //     global: true,
    // },
    modules: [
        '@nuxt/content',
        '@nuxtjs/tailwindcss',
        '@nuxtjs/algolia',
    ],
    app: {
        // baseURL: process.env.NODE_ENV === 'production' ? '/couper-docs/' : '/'
    },
    algolia: {
        apiKey: '5551c3e4dfb61914988abf95fd9b762f',
        applicationId: 'MSIN2HU7WH',
        instantSearch: {
            theme: 'algolia'
        },
        crawler: {
            apiKey: '<WriteKey>',
            indexName: 'docs',
            include: undefined, // all routes
            meta: ['title', 'description', '_path']
        },

        // @ts-ignore
        indexer: {} // throws err if not set
    },
    css: [
        '@/assets/css/tailwind.css',
        '@/assets/css/font.css'
    ],
    vite: {
        define: {
            'process.env.FORCE_COLOR': {},
            'process.env.NODE_DISABLE_COLORS': {},
            'process.env.NO_COLOR': {},
            'process.env.FORCE_TERM': {}
        }
    },
    markdown: {
        toc: {
            depth: 3,
            searchDepth: 3,
        },
    },
    content: {
        // base: './content',
        // @ts-ignore
        documentDriven: {
            navigation: true,
            page: true,
            surround: true,
            injectPage: true,
        },
        // sources: [
        //     {
        //         name: 'edge',
        //         prefix: '/edge',
        //         driver: 'fs',
        //         base: resolve(__dirname, 'edge')
        //     },
        // ],
        highlight: {
            theme: 'slack-dark',
            preload: ['hcl', 'html', 'shell'],
        }
    },
    github: {
        owner: 'avenga',
        repo: 'couper-docs',
        branch: 'main',
        releases: false
    },
    tailwindcss: {
        config: {
            /* Extend the Tailwind config here */
            content: [
                'content/**/**.md'
            ],
            theme: {
                extend: {
                    colors: {
                        primary: colors.emerald
                    }
                }
            }
        }
    },
    build: {
        postcss: {
            postcssOptions: {
                plugins: {
                    tailwindcss: {},
                    autoprefixer: {},
                }
            }
        }
    },
})
