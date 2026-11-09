<script setup lang="ts">
defineProps<{ label: string; value: string; note: string }>();
</script>
<template>
  <section class="stat-card">
    <p class="muted small">{{ label }}</p>
    <strong class="stat-card__value">{{ value }}</strong>
    <p class="small muted">{{ note }}</p>
  </section>
</template>
