<script setup lang="ts">
import { computed } from "vue";
import MarkdownIt from "markdown-it";

const props = withDefaults(defineProps<{ source: string; headingOffset?: number }>(), {
  headingOffset: 1,
});
const markdown = new MarkdownIt({ html: false, linkify: false });
markdown.core.ruler.push("content-headings", (state) => {
  for (const token of state.tokens) {
    if (token.type === "heading_open" || token.type === "heading_close") {
      token.tag = `h${Math.min(6, Number(token.tag.slice(1)) + props.headingOffset)}`;
    }
  }
});
markdown.renderer.rules.table_open = () =>
  '<div class="markdown-table"><table>';
markdown.renderer.rules.table_close = () => "</table></div>";
const rendered = computed(() => markdown.render(props.source));
</script>

<template><article class="markdown-content" v-html="rendered" /></template>

<style scoped>
.markdown-content {
  color: #374151;
  font-size: 14px;
  line-height: 1.75;
  overflow-wrap: anywhere;
}
.markdown-content :deep(> :first-child) {
  margin-top: 0;
}
.markdown-content :deep(> :last-child) {
  margin-bottom: 0;
}
.markdown-content :deep(h2) {
  margin: 28px 0 16px;
  font-size: 22px;
  line-height: 1.4;
}
.markdown-content :deep(h3) {
  margin: 28px 0 12px;
  font-size: 17px;
}
.markdown-content :deep(p),
.markdown-content :deep(ul),
.markdown-content :deep(ol) {
  margin: 0 0 16px;
}
.markdown-content :deep(ul),
.markdown-content :deep(ol) {
  padding-left: 24px;
}
.markdown-content :deep(li + li) {
  margin-top: 6px;
}
.markdown-content :deep(a) {
  color: var(--primary);
  text-decoration: underline;
}
.markdown-content :deep(code) {
  background: var(--subtle);
  border-radius: 3px;
  padding: 2px 5px;
  font-size: 12px;
}
.markdown-content :deep(pre) {
  margin: 0 0 20px;
  padding: 14px 16px;
  border: 1px solid var(--border);
  border-radius: var(--radius);
  background: var(--subtle);
  overflow-x: auto;
}
.markdown-content :deep(pre code) {
  padding: 0;
  white-space: pre;
}
.markdown-content :deep(.markdown-table) {
  max-width: 100%;
  overflow-x: auto;
  margin: 0 0 20px;
}
.markdown-content :deep(table) {
  width: 100%;
  border-collapse: collapse;
  font-size: 13px;
}
.markdown-content :deep(th),
.markdown-content :deep(td) {
  border: 1px solid var(--border);
  padding: 10px 12px;
  text-align: left;
  vertical-align: top;
  min-width: 90px;
}
.markdown-content :deep(th) {
  background: var(--subtle);
  font-weight: 600;
}
.markdown-content :deep(td:last-child) {
  min-width: 260px;
}
@media (max-width: 600px) {
  .markdown-content {
    font-size: 13px;
  }
  .markdown-content :deep(h2) {
    font-size: 19px;
  }
}
</style>
