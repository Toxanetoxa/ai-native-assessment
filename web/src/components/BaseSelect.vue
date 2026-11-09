<script setup lang="ts">
defineProps<{ label: string; options: { value: string; label: string }[] }>();
const value = defineModel<string>({ default: "" });
</script>
<template>
  <label class="field"
    ><span class="field__label">{{ label }}</span
    ><select v-model="value" class="input">
      <option
        v-for="option in options"
        :key="option.value"
        :value="option.value"
      >
        {{ option.label }}
      </option>
    </select></label
  >
</template>
