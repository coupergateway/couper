import type { Config } from 'tailwindcss'
import colors from 'tailwindcss/colors'
import defaultTheme from 'tailwindcss/defaultTheme'

export default <Partial<Config>>{
    content: [
        'content/**/**.md'
    ],
    theme: {
        extend: {
            colors: {
                primary: colors.emerald
            },
            fontFamily: {
                'sans': ['couper', ...defaultTheme.fontFamily.sans],
            },
        }
    },
    plugins: [
        require('@tailwindcss/typography'),
    ],
}
