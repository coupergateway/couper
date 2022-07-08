<script setup lang="ts">
const { prev, next } = useContent();
</script>
<template>
  <div>
    <ContentDoc :path="`${$route.path}`" />
    <div class="flex items-center justify-between ">
      <div>
        <NuxtLink v-if="prev" :to="prev._path">&lt; {{ prev.title }}</NuxtLink>
      </div>
      <div>
        <NuxtLink v-if="next" :to="next._path">{{ next.title }} &gt;</NuxtLink>
      </div>
    </div>
  </div>
</template>
