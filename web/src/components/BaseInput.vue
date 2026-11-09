<script setup lang="ts">
withDefaults(
  defineProps<{
    label: string;
    type?: string;
    placeholder?: string;
    min?: string;
    max?: string;
  }>(),
  { type: "text" },
);
const value = defineModel<string>({ default: "" });
</script>
<template>
  <label class="field"
    ><span class="field__label">{{ label }}</span
    ><input
      v-model="value"
      class="input"
      :type="type"
      :placeholder="placeholder"
      :min="min"
      :max="max"
  /></label>
</template>
