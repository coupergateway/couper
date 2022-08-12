<template>
  <NuxtLayout>
    <NuxtPage />
  </NuxtLayout>
</template>

<script setup>
const { result, search } = useAlgoliaSearch("docs")

onMounted(async () => {
	setInterval(() => {
	for (const pre of document.getElementsByTagName("pre")) {
		pre.title = "Click to copy to clipboard"
		pre.onclick = (event) => {
			const target = event.target
			navigator.clipboard.writeText(target.innerText)
			target.classList.add("copied")
		    setTimeout(() => target.classList.remove("copied"), 1500)
		}
	}
	}, 1000)
})
</script>

<style>
.prose pre {
	position: relative;
	pointer-events: none;
	line-height: 1.5rem
}

pre::before {
	content: "📋";
	position: absolute;
	right: 1rem;
	font-size: 1.5rem;
	cursor: pointer;
	pointer-events: auto;
}

pre:hover::before {
	filter: brightness(1.2);
}

pre.copied::after {
	font-family: sans-serif;
	content: "Copied!  ✅";
	position: absolute;
	background: #475569;
	top: 0.5rem;
	right: 1rem;
	padding: 2px 10px;
	border-radius: 5px;
}
</style>
