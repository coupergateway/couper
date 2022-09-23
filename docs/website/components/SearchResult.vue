<template>
    <ais-hits :escapeHTML="false" ref="searchHits" :transform-items="transform">
      <template v-slot="{ items }">
        <ol v-for="item in filter(items)" :key="item.url" class="bg-white border-4 border-gray-200 rounded-lg w-full mt-2">
          <NuxtLink :to="item.url.toLowerCase()" class="text-sky-600" @click="reset()">
            <li class="pl-8 pr-2 py-1 relative cursor-pointer hover:bg-sky-50 hover:text-gray-900">
<!--                <svg class="stroke-current absolute w-4 h-4 left-2 top-2" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17.657 16.657L13.414 20.9a1.998 1.998 0 01-2.827 0l-4.244-4.243a8 8 0 1111.314 0z"/>-->
<!--                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 11a3 3 0 11-6 0 3 3 0 016 0z"/>-->
<!--                </svg>-->
                <SearchResultItem :item="item" />
            </li>
          </NuxtLink>
        </ol>
      </template>
    </ais-hits>
</template>

<script>
import SearchResultItem from '@/components/SearchResultItem'
import {  AisHits } from 'vue-instantsearch/vue3/es'
export default {
  name: "SearchResult.vue",
  components: {
    AisHits,
    SearchResultItem,
  },
  methods: {
    filter(items) {
      const filteredItems = []
      for (const idx in items) {
        const item = items[idx]
        if (item._highlightResult !== undefined) {
          filteredItems.push(item)
        }
      }

      filteredItems.sort((left, right) => left.__position > right.__position ? 1 : 0)
      return filteredItems
    },
    transform(items) {
      for (const i in items) {
        const item = items[i]
        for (const j in item._highlightResult.attributes) {
          const attribute = item._highlightResult.attributes[j]
          if (attribute.transformed) {
            continue
          }
          let description = attribute.description.value
          // do not mark markdown _highlights_ as search hits
          description = description.replace(/\b_(.*?)_\b/g, "$1")
          // drop link markdown
          description = description.replace(/\[(.*?)\]\(.*?\)/g, "$1")
          // render code markdown
          description = description.replace(/`(.*?)`/g, "<code>$1</code>")
          attribute.description.value = description
          attribute.transformed = true
        }
      }
      return items
    },
    reset() {
      document
          .querySelectorAll('.ais-SearchBox-input')
          .forEach((e) => (e.value = ''))
      this.$refs.searchHits.state.hits = []
    }
  }
}
</script>

<style>
.ais-SearchBox-input {
  background-color: #475569;
}
</style>
