<script setup lang="ts">
defineProps<{ title?: string; subtitle?: string }>();
</script>
<template>
  <section class="panel">
    <header v-if="title || $slots.actions" class="panel__header">
      <div>
        <h2 v-if="title">{{ title }}</h2>
        <p v-if="subtitle" class="muted small">{{ subtitle }}</p>
      </div>
      <slot name="actions" />
    </header>
    <div class="panel__body"><slot /></div>
    <footer v-if="$slots.footer" class="panel__footer">
      <slot name="footer" />
    </footer>
  </section>
</template>
