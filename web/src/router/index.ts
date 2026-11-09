import { createRouter, createWebHistory } from "vue-router";

export default createRouter({
  history: createWebHistory(),
  routes: [
    { path: "/", redirect: "/operations" },
    {
      path: "/assignment",
      name: "assignment",
      component: () => import("../pages/AssignmentPage.vue"),
      meta: { title: "Задание" },
    },
    {
      path: "/operations",
      name: "operations",
      component: () => import("../pages/OperationsPage.vue"),
      meta: { title: "Операционные показатели" },
    },
    {
      path: "/returns-register",
      name: "returns-register",
      component: () => import("../pages/RegisterPage.vue"),
      meta: { title: "Реестр возвратов" },
    },
    {
      path: "/:pathMatch(.*)*",
      name: "not-found",
      component: () => import("../pages/NotFoundPage.vue"),
      meta: { title: "Страница не найдена" },
    },
  ],
});
