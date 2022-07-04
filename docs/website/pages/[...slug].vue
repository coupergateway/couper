<script setup lang="ts">
const { prev, next } = useContent();
</script>
<template>
  <div>
    <ContentDoc :path="`${$route.path}`" />
    <div class="flex flex-row">
      <div class="basis-5/6">
        <NuxtLink v-if="prev" :to="prev._path" class=" bg-amber-400">&lt; {{ prev.title }}</NuxtLink>
      </div>
      <div class="basis-1/6">
        <NuxtLink v-if="next" :to="next._path" class="bg-amber-400">{{ next.title }} &gt;</NuxtLink>
      </div>
    </div>
  </div>
</template>
