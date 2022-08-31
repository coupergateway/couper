<script setup lang="ts">
import SearchResult from "@/components/SearchResult";

const indexName = 'docs'
const algoliaClient = useAlgoliaRef()

import { AisInstantSearch, AisSearchBox } from 'vue-instantsearch/vue3/es'

const searchClient = {
  ...algoliaClient,
  search(requests) {
    if (requests.every(({ params }) => !params.query)) {
      return Promise.resolve({
        results: requests.map(() => ({
          hits: [],
          nbHits: 0,
          nbPages: 0,
          page: 0,
          processingTimeMS: 0,
        })),
      });
    }

    return algoliaClient.search(requests);
  },
};
</script>

<template>
    <div class="w-4/5 py-8">
      <div class="mb-2 text-sm font-medium text-gray-900 dark:text-gray-500">
          <ais-instant-search class="relative" :index-name="indexName" :search-client="searchClient" :stalled-search-delay="150">
            <ais-search-box @blur="onBlur"
                class="block mt-1 p-4 pl-10 w-full text-sm text-gray-900 rounded-lg border-2 border-lime-500 focus:ring-blue-500 focus:border-blue-500"
            />
            <SearchResult class="my-5 absolute top-15 z-50 auto-rows-auto gap-4" />
          </ais-instant-search>
      </div>
    </div>
</template>

<script lang="ts">
export default {
  name: 'SearchBar',
  data() {
    return {
      needle: '',
    }
  },
  methods: {
    reset() {
      this.needle = ''
    },
    onBlur() {
      setTimeout(this.reset, 75)
    }
  }
}
</script>

<style>
  .ais-SearchBox-form {
    width: 100%;
  }
  .ais-SearchBox-input {
    background-color: white;
  }
</style>
