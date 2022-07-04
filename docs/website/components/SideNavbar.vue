<template>
  <nav>
    <ContentNavigation v-slot="{ navigation }">
      <div v-for="parent of navigation" :key="parent._path">
        <div class="font-semibold text-lg text-gray-50" v-if="parent.children !== undefined">{{parent.title}}</div>
          <div v-for="link of parent.children" :key="link._path" class="border-l-2">
            <div v-if="link.children" class="pl-2">
              <div class="font-semibold text-gray-50">{{link.title}}</div>
              <div v-for="l of link.children" :key="l._path">
                <NuxtLink :to="l._path" exact-active-class="border-sky-500" class="text-base pl-2">{{ l.title }}</NuxtLink>
              </div>
            </div>
            <NuxtLink v-else :to="link._path" exact-active-class="border-sky-500" class="text-base pl-2">{{ link.title }}</NuxtLink>
          </div>
      </div>
    </ContentNavigation>
  </nav>
</template>
