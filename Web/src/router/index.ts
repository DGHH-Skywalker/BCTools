import { createRouter, createWebHashHistory } from "vue-router"
import type { RouteRecordRaw } from "vue-router"
import AppLayout from "../views/AppLayout.vue"
import Home from "../views/Home.vue"

const routes: RouteRecordRaw[] = [
  { path: "/", redirect: "/home" },
  { path: "/home", component: Home, meta: { standalone: true } },
  {
    path: "/dorm/manage",
    component: AppLayout,
    children: [
      { path: "", component: () => import("../views/DormManage.vue"), meta: { title: "宿舍歌单" } },
      { path: "slots", component: () => import("../views/DormTimeSlots.vue"), meta: { title: "时段配置" } },
    ],
  },
  {
    path: "/broadcast",
    component: AppLayout,
    children: [{ path: "", component: () => import("../views/BroadcastEdit.vue"), meta: { title: "播音歌单" } }],
  },
  {
    path: "/export",
    component: AppLayout,
    children: [{ path: "", component: () => import("../views/Export.vue"), meta: { title: "歌单导出" } }],
  },
  {
    path: "/organize",
    component: AppLayout,
    children: [{ path: "", component: () => import("../views/Organize.vue"), meta: { title: "导出歌曲文件" } }],
  },
  {
    path: "/mobile",
    component: AppLayout,
    children: [{ path: "", component: () => import("../views/Mobile.vue"), meta: { title: "手机点歌" } }],
  },
  {
    path: "/settings",
    component: () => import("../views/Settings.vue"),
    children: [
      { path: "", component: () => import("../views/SettingsOverview.vue"), meta: { title: "软件设置" } },
      { path: "slots", component: () => import("../views/DormTimeSlots.vue"), meta: { title: "时段配置" } },
      { path: "broadcast", component: () => import("../views/BroadcastSettings.vue"), meta: { title: "播音栏目" } },
      { path: "export", component: () => import("../views/ExportSettings.vue"), meta: { title: "歌单背景设置" } },
      { path: "guide", component: () => import("../views/Guide.vue"), meta: { title: "软件指南" } },
    ],
  },
  { path: "/about-software", redirect: "/settings" },
  { path: "/guide", redirect: "/settings/guide" },
  {
    path: "/error",
    component: AppLayout,
    children: [{ path: "", component: () => import("../views/Error.vue"), meta: { title: "出错了" } }],
  },
]

const router = createRouter({ history: createWebHashHistory(), routes })

router.beforeEach((to, _, next) => {
  // 浏览器标签页统一使用应用名称，不再根据路由追加页面标题
  document.title = "小播点歌工具"
  next()
})

export default router
