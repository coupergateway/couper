<template>
  <div class="inline-flex flex-col justify-center relative text-gray-500">
    <ais-hits class="py-200">
      <template v-slot="{ items }">
        <ol v-for="item in filter(items)" :key="item.url" class="bg-gray-100 border border-gray-100 w-full mt-2">
          <NuxtLink :to="item.url.toLowerCase()" class="text-sky-600">
            <li class="pl-8 pr-2 py-1 border-b-2 border-gray-100 relative cursor-pointer hover:bg-sky-50 hover:text-gray-900">
<!--                <svg class="stroke-current absolute w-4 h-4 left-2 top-2" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17.657 16.657L13.414 20.9a1.998 1.998 0 01-2.827 0l-4.244-4.243a8 8 0 1111.314 0z"/>-->
<!--                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 11a3 3 0 11-6 0 3 3 0 016 0z"/>-->
<!--                </svg>-->
                <SearchResultItem :item="item" />
            </li>
          </NuxtLink>
        </ol>
      </template>
    </ais-hits>
  </div>
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
    }
  }
}
</script>
