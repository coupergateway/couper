<script setup lang="ts">
import Logo from "~/components/content/Logo.vue";

const indexName = 'docs'
const algoliaClient = useAlgoliaRef()

import { AisInstantSearch, AisSearchBox, AisHits, AisHighlight } from 'vue-instantsearch/vue3/es'

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
  <div class="max-w-8xl mx-auto">
    <div class="py-4 border-b border-slate-900/10 lg:px-8 lg:border-0 dark:border-slate-300/10 mx-4 lg:mx-0">
      <div class="relative flex items-center">
        <Logo class="box-border h-16 w-16 md:w-18 md:h-18 xl:w-22 xl:h-22" />
        <h1 class="text-lg font-medium text-slate-50 align-top p-4">Couper Documentation</h1>
        <div class="box-border text-base">
          <div class="rounded-md bg-lime-500 p-0.5">edge</div>
        </div>
        <div class="hidden w-full lg:flex items-center text-sm pl-4">
          <ais-instant-search :index-name="indexName" :search-client="searchClient">
            <ais-search-box class="leading-6 rounded-md shadow-sm py-1.5 pl-2 pr-3" />
            <ais-hits>
              <template v-slot:item="{ item }">
                <NuxtLink :to="item.url" class="text-lime-400">
                  {{item.name}}
                  {{item.description}}
<!--                  &lt;!&ndash;                <h2>{{ item.name }}</h2>&ndash;&gt;-->
<!--                  <ais-highlight attribute="name" :hit="item" />-->
<!--                  <ais-highlight attribute="description" :hit="item" />-->
                </NuxtLink>
              </template>
            </ais-hits>
          </ais-instant-search>
        </div>
      </div>
    </div>
  </div>
</template>

<style>
  .ais-InstantSearch {
    width: 50%;
  }
  .ais-SearchBox-input {
    background-color: unset;
    color: #e3e4e4;
  }
</style>
