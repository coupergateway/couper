<template>
  <nav>
    <ContentNavigation v-slot="{ navigation }">
      <div v-for="parent of navigation" :key="parent._path" class="space-y-2">
        <div class="font-semibold text-lg text-gray-400">{{parent.title}}</div>
          <div v-for="link of parent.children" :key="link._path" class="space-y-1">
            <div v-if="link.children" class="pl-2">
              <div class="font-semibold text-gray-400">{{link.title}}</div>
              <div v-for="l of link.children" :key="l._path" class="space-y-1">
                <NuxtLink :to="l._path" class="text-base pl-2">{{ l.title }}</NuxtLink>
              </div>
            </div>
            <NuxtLink v-else :to="link._path" class="text-base pl-2">{{ link.title }}</NuxtLink>
          </div>
      </div>
    </ContentNavigation>
  </nav>
</template>
<!--<script setup>-->
<!--import NuxtLink from '#imports'-->
<!--</script>-->
