<template>
  <div class="flex flex-col w-128 min-w-min prose prose-slate prose-lg prose-code:bg-sky-100 result">
    <div class="flex flex-col m-2">
      <h3 class="uppercase blockTitle pt-4 text-lg text-gray-800" v-html="name"></h3>
      <div v-html="description"></div>
      <table class="table-auto">
        <tbody>
        <tr class="align-top" v-for="attr in filtered" :key="attr.name+attr.desc">
          <td class="font-semibold text-purple-600" v-html="attr.name"></td>
          <td v-html="attr.desc"></td>
        </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<script>
export default {
  name: "SearchResultItem.vue",
  props: ['item'],
  data() {
    return {
      name: '',
      description: '',
      filtered: []
    }
  },
  watch: {
    item: { // content change as you type, e.g. highlight marks -> refresh filter
      handler: function (newItem) {
        this.name = this.blockName(newItem)
        this.attributes(newItem)
      },
      deep: true,
      immediate: true,
    },
  },
  methods: {
    attributes(newItem) {
      const attrResults = []
      for (const idx in newItem._highlightResult.attributes) {
        const attr = newItem._highlightResult.attributes[idx]
        if (attr.name.matchedWords.length > 0 ||
            attr.description.matchedWords.length > 0) {
          attrResults.push({
            name: attr.name.value,
            desc: attr.description.value,
          })
        }
      }
      this.filtered = attrResults
    },
    blockName(newItem) {
      if (newItem._highlightResult !== undefined) {
        return newItem._highlightResult.name.value
      }
      return newItem.name
    },
  },
}
</script>

<style scoped>
.blockTitle::before {
  content: "";
  position: absolute;
  top: 0.5em;
  left: 2.2em;
  height: 0.3em;
  width: 2.6em;
  background: rgb(101, 179, 46);
}

.result {
  max-width: 95%
}

.result >>> em {
  background-color: rgb(245 158 11 / var(--tw-text-opacity));
  font-style: normal;
}

.result >>> code {
  font-weight: normal;
  background: #eee
}

.result table {
  table-layout: fixed
}

.result table >>> td:first-child {
  width: 35%
}
</style>
