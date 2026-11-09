<script setup lang="ts">
withDefaults(
  defineProps<{
    variant?: "primary" | "secondary" | "ghost";
    type?: "button" | "submit";
    disabled?: boolean;
  }>(),
  { variant: "secondary", type: "button" },
);
</script>
<template>
  <button
    :type="type"
    :disabled="disabled"
    class="button"
    :class="`button--${variant}`"
  >
    <slot />
  </button>
</template>
