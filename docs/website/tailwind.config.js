/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [
      './components/**/*.vue',
      './layouts/**/*.vue',
  ],
  theme: {
    extend: {},
  },
  plugins: [
    require('@tailwindcss/typography'),
  ],
}
