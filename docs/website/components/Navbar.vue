<script setup lang="ts">
import Logo from "~/components/content/Logo.vue";

const indexName = 'docs'
const algolia = useAlgoliaRef()
import { AisInstantSearch, AisSearchBox, AisHits } from 'vue-instantsearch/vue3/es'

// const { result, search } = useAlgoliaSearch("docs");
</script>

<template>
  <div>
    <Logo class="box-border h-32 w-32 p-4" />
    <div>
      <div>Version</div>
      <div>
        <ais-instant-search :index-name="indexName" :search-client="algolia">
          <ais-search-box />
          <ais-hits />
        </ais-instant-search>
      </div>
    </div>
  </div>
</template>

