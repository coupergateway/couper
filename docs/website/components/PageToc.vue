<template>
  <nav class="w-1/5 grow-0 p-2">
    <h2 class="text-amber-500">On this Page</h2>
    <ul v-if="toc && toc.links">
      <li v-for="link in toc.links" :key="link.text">
        <NuxtLink :href="`#${link.id}`">
          {{ link.text }}
        </NuxtLink>
        <ul v-if="link.children">
          <li v-for="child in link.children" :key="child.id">
            <NuxtLink class="pl-2" :href="`#${child.id}`">
              {{ child.text }}
            </NuxtLink>
          </li>
        </ul>
      </li>
    </ul>
  </nav>
</template>

<script>
const headlineTags = ["h2", "h3", "h4", "attributes", "duration", "blocks"]

function getText(element) {
	let text = element.value ?? ""
	for (const child of element.children ?? []) {
		text += getText(child)
	}
	return text
}

async function createPageToC() {
	const { path } = useRoute()
	const result = await queryContent(path).findOne()

	const tocEntries = result.body.children.filter((element) => {
		return headlineTags.includes(element.tag)
	}).map((element) => {
		let id, text
		if (element.tag === "attributes" || element.tag === "duration" || element.tag === "blocks") {
			id = element.tag
			text = element.tag.substring(0, 1).toUpperCase() + element.tag.substring(1)
		} else {
			id = element.props.id
			text = getText(element)
		}

		return { id: id, text: text , children: [] }
	})

	return {links: tocEntries}
}

export default {
	data() {
		return {
			toc: {links:[]}
		}
	},
	async mounted() {
		this.toc = await createPageToC()
	},
	watch: {
		async $route(to, from) {
			this.toc = await createPageToC()
		}
	}
}
</script>
