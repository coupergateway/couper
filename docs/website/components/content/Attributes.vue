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
          <code v-else-if="head.value === 'default' && value[head.value] != ''">{{ value[head.value] }}</code>
          <div v-else-if="head.value === 'description'" v-html="micromark(value[head.value])"/>
          <div v-else v-html="value[head.value] ? value[head.value] : '-'" />
        </td>
      </tr>
      </tbody>
    </table>
  </div>
</template>

<script setup>
import { micromark } from 'micromark'
const props = defineProps({
  header: {
    type: Array,
    required: false,
    default: [
      {
        name: "Name",
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
