import { createRouter, createWebHistory } from 'vue-router'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: "/",
      redirect: "/platform",
    },
    {
      path: "/knowledgeBase",
      name: "home",
      component: () => import("../views/knowledge/KnowledgeBase.vue"),
    },
    {
      path: "/platform",
      name: "Platform",
      redirect: "/platform/knowledgeBase",
      component: () => import("../views/platform/index.vue"),
      children: [
        {
          path: "knowledgeBase",
          name: "knowledgeBase",
          component: () => import("../views/knowledge/KnowledgeBase.vue"),
        },
        {
          path: "creatChat",
          name: "creatChat",
          component: () => import("../views/creatChat/creatChat.vue"),
        },
        {
          path: "chat/:chatid",
          name: "chat",
          component: () => import("../views/chat/index.vue"),
        },
        {
          path: "settings",
          name: "settings",
          component: () => import("../views/settings/Settings.vue"),
        },
      ],
    },
  ],
});

export default router
