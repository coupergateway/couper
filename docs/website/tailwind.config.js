/** @type {import('tailwindcss').Config} */
import defaultTheme from 'tailwindcss/defaultTheme'

module.exports = {
  content: [
    `components/**/*.{vue,js,ts}`,
    `layouts/**/*.vue`,
    `app.vue`,
    `plugins/**/*.{js,ts}`,
  ],
  theme: {
    extend: {
      fontFamily: {
        'sans': ['couper', ...defaultTheme.fontFamily.sans],
      },
      fontSize: {
        'base': '1.05rem',
      }
    },
  },
  plugins: [
    require('@tailwindcss/typography'),
  ],
}
