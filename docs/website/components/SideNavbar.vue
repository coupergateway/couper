<script lang="ts" setup>
const route = useRoute();

const activeStyle = function(link:string) {
  const base = 'pl-2 border-l-2'
  return route.path === link ? base+' border-sky-400 text-sky-500 rounded-r-md' : base
}
</script>

<template>
  <nav>
    <ContentNavigation v-slot="{ navigation }">
      <div v-for="parent of navigation" :key="parent._path">
        <div class="font-semibold text-lg text-amber-500" v-if="parent.children !== undefined">{{parent.title}}</div>
          <div v-for="link of parent.children" :key="link._path" class="text-base pl-2">
            <div v-if="link.children" class="pl-2">
              <div class="font-semibold text-amber-500">{{link.title}}</div>
              <div v-for="l of link.children" :key="l._path" :class="activeStyle(l._path)">
                <NuxtLink :to="l._path">{{ l.title }}</NuxtLink>
              </div>
            </div>
            <div v-else class="text-base">
              <NuxtLink :to="link._path" :class="activeStyle(link._path)">{{ link.title }}</NuxtLink>
            </div>
          </div>
      </div>
    </ContentNavigation>
  </nav>
</template>
