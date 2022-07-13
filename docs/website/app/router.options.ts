import type { RouterConfig } from '@nuxt/schema'

export default <RouterConfig>{
	scrollBehavior: (to, from, savedPosition) => {
		return new Promise((resolve, reject) => {
			setTimeout(() => {
				let position
				if (savedPosition) {
					position = savedPosition
				} else if (to.hash) {
					position = {
						el: to.hash,
						top: getOffset()
					}
				} else {
					position = { top: 0 }
				}

				resolve(position)
			}, 100)
		})
	}
}

function getOffset() {
	return document.getElementsByTagName("header")[0].offsetHeight + 20
}
