<script setup lang="ts">
import DashboardLayout from "../layouts/DashboardLayout.vue";
import Panel from "../components/Panel.vue";
</script>
<template>
  <DashboardLayout title="Страница не найдена"
    ><Panel
      ><p>Такой страницы пока нет.</p>
      <RouterLink class="text-link" to="/operations"
        >Вернуться к показателям</RouterLink
      ></Panel
    ></DashboardLayout
  >
</template>
