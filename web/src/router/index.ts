import { createRouter, createWebHashHistory, createWebHistory } from "vue-router";

export default createRouter({
  history: import.meta.env.VITE_API_MODE === "mock"
    ? createWebHashHistory(import.meta.env.BASE_URL)
    : createWebHistory(import.meta.env.BASE_URL),
  routes: [
    { path: "/", redirect: "/operations" },
    {
      path: "/assignment",
      name: "assignment",
      component: () => import("../pages/AssignmentPage.vue"),
      meta: { title: "О проекте" },
    },
    {
      path: "/operations",
      name: "operations",
      component: () => import("../pages/OperationsPage.vue"),
      meta: { title: "Операционные показатели" },
    },
    {
      path: "/returns",
      name: "returns",
      component: () => import("../pages/ReturnsPage.vue"),
      meta: { title: "Возвраты" },
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
