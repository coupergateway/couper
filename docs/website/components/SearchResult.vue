<template>
  <div class="flex flex-col w-128 min-w-min">
    <div class="flex flex-col">
      <h3 class="uppercase" v-html="blockName"></h3>
      <div v-html="blockDescription"></div>
      <table class="table-auto">
        <tbody>
        <tr class="prose align-top" v-for="attr in attributes()">
          <td class="font-semibold" v-html="attr.name"></td>
          <td v-html="attr.desc"></td>
        </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<script>
export default {
  name: "SearchResult.vue",
  props: ['item'],
  computed: {
    blockName() {
      if (this.item._highlightResult !== undefined) {
        return this.item._highlightResult.name.value
      }
      return this.item.name
    },
    blockDescription() {
      console.log(this.item)
    },
  },
  methods: {
    attributes() {
      const attrResults = []
      for (const idx in this.item._highlightResult.attributes) {
        const attr = this.item._highlightResult.attributes[idx]
        if (attr.name.matchLevel !== 'none') {
          attrResults.push({
            name: attr.name.value,
            desc: attr.description.value,
            match: attr.name.matchLevel,
          })
        }
      }
      return attrResults.sort(this.compareMatch)
    },
    compareMatch (left, right) {
      if (left.match !== 'full') {
        return 0
      } else if (left.match === 'full') {
        return 1
      }
      return -1
    }
  }
}
</script>

<style scoped>

</style>
