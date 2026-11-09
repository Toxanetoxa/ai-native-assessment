<script setup lang="ts">
withDefaults(defineProps<{ title: string; description?: string; showHeading?: boolean }>(), {
  showHeading: true,
});
</script>
<template>
  <section class="dashboard">
    <header v-if="showHeading !== false" class="page-heading">
      <div>
        <h1>{{ title }}</h1>
        <p v-if="description" class="muted">{{ description }}</p>
      </div>
      <slot name="actions" />
    </header>
    <main class="page-body"><slot /></main>
  </section>
</template>
