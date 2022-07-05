<template>
  <div>
    <h3 id="attributes"><a href="#attributes">Attributes</a></h3>
    <table class="table-auto">
      <thead>
      <tr>
        <th v-for="head in header"
        :key="head.value"
        >{{ head.name }}</th>
      </tr>
      </thead>
      <tbody>
      <tr v-for="value in values" :key="value.name">
        <td v-for="head in header" :key="head.value+value.name">
          <code v-if="head.value === 'name'">{{ value[head.value] ? value[head.value] : '-' }}</code>
          <div v-else>{{ value[head.value] ? value[head.value] : '-' }}</div>
        </td>
      </tr>
      </tbody>
    </table>
  </div>
</template>

<script setup>
const props = defineProps({
  header: {
    type: Array,
    required: false,
    default: [
      {
        name: "Attributes",
        value: "name",
      },
      {
        name: "Type",
        value: "type",
      },
      {
        name: "Default",
        value: "default",
      },
      {
        name: "Description",
        value: "description",
      },
    ],
  },
  values: {
    type: Array,
    required: true,
  }
})
</script>
