<script setup lang="ts" generic="T extends { id: string }">
import StateMessage from "./StateMessage.vue";
defineProps<{
  rows: T[];
  columns: { key: string; label: string; numeric?: boolean }[];
  caption: string;
}>();
function cell(row: T, key: string) {
  return (row as Record<string, unknown>)[key];
}
</script>
<template>
  <div class="table-scroll">
    <table class="data-table">
      <caption class="sr-only">
        {{
          caption
        }}
      </caption>
      <thead>
        <tr>
          <th
            v-for="column in columns"
            :key="column.key"
            :class="{ numeric: column.numeric }"
            scope="col"
          >
            {{ column.label }}
          </th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="row in rows" :key="row.id">
          <td
            v-for="column in columns"
            :key="column.key"
            :class="{ numeric: column.numeric }"
          >
            <slot :name="column.key" :row="row">{{
              cell(row, column.key)
            }}</slot>
          </td>
        </tr>
      </tbody>
    </table>
    <StateMessage
      v-if="!rows.length"
      title="Ничего не найдено"
      detail="Измените период или условия поиска."
    />
  </div>
</template>
