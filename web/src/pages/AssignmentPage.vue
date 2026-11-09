<script setup lang="ts">
import assignment from "../../../README.md?raw";
import DashboardLayout from "../layouts/DashboardLayout.vue";
import Panel from "../components/Panel.vue";
import MarkdownContent from "../components/MarkdownContent.vue";
</script>

<template>
  <DashboardLayout title="Задание" :show-heading="false">
    <div class="assignment-page">
      <Panel><MarkdownContent :source="assignment" :heading-offset="0" /></Panel>
    </div>
  </DashboardLayout>
</template>

<style scoped>
.assignment-page {
  max-width: 1000px;
}
.assignment-page :deep(.markdown-content) {
  font-size: 15px;
  line-height: 1.7;
}
.assignment-page :deep(.markdown-content h1) {
  margin: 0 0 18px;
  font-size: 24px;
  line-height: 1.3;
}
.assignment-page :deep(.markdown-content h2) {
  margin: 36px 0 16px;
  padding-top: 24px;
  border-top: 1px solid var(--border);
  font-size: 19px;
  line-height: 1.4;
}
.assignment-page :deep(.markdown-content h3) {
  margin: 24px 0 8px;
  font-size: 15px;
  line-height: 1.5;
}
.assignment-page :deep(.markdown-content li + li) {
  margin-top: 8px;
}
@media (max-width: 600px) {
  .assignment-page :deep(.markdown-content) {
    font-size: 14px;
  }
  .assignment-page :deep(.markdown-content h1) {
    font-size: 21px;
  }
  .assignment-page :deep(.markdown-content h2) {
    margin-top: 28px;
    padding-top: 20px;
    font-size: 18px;
  }
}
</style>
