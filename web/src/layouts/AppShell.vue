<script setup lang="ts">
import { onMounted, ref, watch } from "vue";
import { useRoute } from "vue-router";
import BaseIcon from "../components/BaseIcon.vue";
import logo from "../assets/froza.png";
import { useMeta } from "../composables/useMeta";
import { date } from "../utils/format";

const route = useRoute();
const { meta, load } = useMeta();
const minimized = ref(false);
const groupOpen = ref(true);
const navigation = [
  { name: "assignment", label: "Задание", icon: "document" },
  { name: "operations", label: "Операционные показатели", icon: "chart" },
  { name: "returns", label: "Возвраты", icon: "refresh" },
  { name: "returns-register", label: "Реестр возвратов", icon: "box" },
];
onMounted(() => {
  void load().catch(() => {});
});
watch(
  () => route.meta.title,
  (title) => {
    document.title = `${String(title ?? "Управление")} · FROZA`;
  },
  { immediate: true },
);
</script>

<template>
  <div class="workspace" :class="{ 'workspace--compact': minimized }">
    <aside class="sidebar" aria-label="Главное меню">
      <RouterLink to="/" class="brand" aria-label="FROZA — главная">
        <span class="brand__image"><img :src="logo" alt="" /></span>
        <span class="brand__caption">Тестовое задание</span>
      </RouterLink>
      <nav class="navigation" aria-label="Разделы">
        <button class="nav-group" :aria-expanded="groupOpen" aria-controls="section-links"
          @click="groupOpen = !groupOpen">
          <BaseIcon name="box" :size="16" />Склад и поставщики
          <BaseIcon name="chevron" :size="15" />
        </button>
        <div id="section-links" class="nav-items" :class="{ 'nav-items--closed': !groupOpen }">
          <RouterLink v-for="item in navigation" :key="item.name"
            :to="{ name: item.name }" :title="item.label" :aria-label="item.label" class="nav-link">
            <BaseIcon :name="item.icon" :size="16" /><span>{{ item.label }}</span>
          </RouterLink>
        </div>
      </nav>
      <div class="sidebar__bottom">
        <span class="avatar" aria-hidden="true"><BaseIcon name="user" :size="16" /></span>
        <span class="sidebar__account">Учебный доступ</span>
        <button class="sidebar__resize"
          :aria-label="minimized ? 'Развернуть меню' : 'Свернуть меню'"
          :title="minimized ? 'Развернуть меню' : 'Свернуть меню'"
          :aria-expanded="!minimized" @click="minimized = !minimized">
          <BaseIcon name="chevron" :size="16" />
        </button>
      </div>
    </aside>
    <div class="workspace__main">
      <slot />
      <footer class="workspace__footer">
        <span>Учебные данные<span v-if="meta"> · Срез {{ date(meta.manifest.as_of) }}</span></span>
        <span>{{ meta?.manifest.version ?? "Локальная версия" }}</span>
      </footer>
    </div>
  </div>
</template>
