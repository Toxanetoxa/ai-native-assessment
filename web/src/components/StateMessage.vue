<script setup lang="ts">
import BaseButton from "./BaseButton.vue";
withDefaults(
  defineProps<{
    title: string;
    detail?: string;
    error?: boolean;
    loading?: boolean;
  }>(),
  { error: false, loading: false },
);
defineEmits<{ retry: [] }>();
</script>
<template>
  <div
    class="state"
    :class="{ 'state--error': error }"
    :role="error ? 'alert' : 'status'"
    :aria-busy="loading"
  >
    <span v-if="loading" class="spinner" /><strong>{{ title }}</strong>
    <p v-if="detail" class="small muted">{{ detail }}</p>
    <BaseButton v-if="error" @click="$emit('retry')">Повторить</BaseButton>
  </div>
</template>
