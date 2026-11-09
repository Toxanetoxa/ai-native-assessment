<script setup lang="ts">
import BaseInput from "./BaseInput.vue";
import BaseSelect from "./BaseSelect.vue";
import BaseButton from "./BaseButton.vue";
import BaseIcon from "./BaseIcon.vue";
import type { Filters, Supplier } from "../api/types";
defineProps<{ suppliers: Supplier[]; busy?: boolean; search?: boolean }>();
const filters = defineModel<Filters>({ required: true });
defineEmits<{ apply: [] }>();
</script>
<template>
  <form class="filter-bar" @submit.prevent="$emit('apply')">
    <BaseInput
      v-model="filters.from"
      label="Период с"
      type="date"
      :max="filters.to"
    /><BaseInput
      v-model="filters.to"
      label="по"
      type="date"
      :min="filters.from"
    /><BaseSelect
      v-model="filters.supplier_id"
      label="Поставщик"
      :options="[
        { value: '', label: 'Все поставщики' },
        ...suppliers.map((item) => ({ value: item.id, label: item.name })),
      ]"
    /><BaseInput
      v-if="search"
      v-model="filters.q"
      label="Поиск"
      placeholder="Номер или товар"
    /><BaseButton type="submit" :disabled="busy"
      ><BaseIcon name="refresh" :size="16" />Применить</BaseButton
    >
  </form>
</template>
