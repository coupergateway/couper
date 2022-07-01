<template>
  <NuxtLayout>
    <NuxtPage />
  </NuxtLayout>
</template>

<script setup>
const { result, search } = useAlgoliaSearch("docs");

// onMounted(async () => {
//   await search({ query: "Samsung" });
// });
</script>
