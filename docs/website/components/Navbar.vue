<script setup lang="ts">
import Logo from "~/components/content/Logo.vue";

const indexName = 'docs'
const algoliaClient = useAlgoliaRef()
import { AisInstantSearch, AisSearchBox, AisHits, AisHighlight } from 'vue-instantsearch/vue3/es'

// const algoliaClient = useAlgoliaRef()
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
  <div class="flex flex-row border-b-2 border-lime-500">
    <div class="basis-1/4">
      <div class="flex flex-row p-4 gap-4">
        <Logo class="box-border h-16 w-16" />
        <h1 class="text-lg">Couper Documentation</h1>
        <div class="box-border text-base"><div class="rounded-md bg-lime-500 p-0.5">edge</div></div>
      </div>
    </div>
    <div class="basis-1/2 p-4">
      <ais-instant-search :index-name="indexName" :search-client="searchClient">
        <ais-search-box />
        <ais-hits>
          <template v-slot:item="{ item }">
            <NuxtLink :to="item.url">
<!--                <h2>{{ item.name }}</h2>-->
              <div class="hit-name">
                <ais-highlight attribute="name" :hit="item"></ais-highlight>
              </div>
            </NuxtLink>
          </template>
        </ais-hits>
      </ais-instant-search>
    </div>
  </div>
</template>

