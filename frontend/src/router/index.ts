import { createRouter, createWebHashHistory } from "vue-router"
import type { RouteRecordRaw } from "vue-router"
import AppLayout from "../views/AppLayout.vue"
const routes: RouteRecordRaw[] = [
  { path: "/", component: AppLayout, children: [
    { path: "", redirect: "/home" },
    { path: "home", component: () => import("../views/Home.vue"), meta: { title: "首页" } },
    { path: "song/import", component: () => import("../views/SongImport.vue"), meta: { title: "歌曲导入" } },
    { path: "dorm/manage", component: () => import("../views/DormManage.vue"), meta: { title: "宿舍歌单" } },
    { path: "broadcast", component: () => import("../views/BroadcastEdit.vue"), meta: { title: "播音歌单" } },
    { path: "export", component: () => import("../views/Export.vue"), meta: { title: "歌单导出" } },
    { path: "organize", component: () => import("../views/Organize.vue"), meta: { title: "换卡工具" } },
    { path: "settings", component: () => import("../views/Settings.vue"), meta: { title: "软件设置" } },
    { path: "guide", component: () => import("../views/Guide.vue"), meta: { title: "软件指南" } },
    { path: "about", component: () => import("../views/About.vue"), meta: { title: "软件作者" } },
    { path: "error", component: () => import("../views/Error.vue"), meta: { title: "出错了" } },
  ]},
  { path: "/advanced", component: () => import("../views/Advanced.vue"), meta: { title: "高级设置" } },
]
const router = createRouter({ history: createWebHashHistory(), routes })
router.beforeEach((to, _, next) => {
  if (to.meta?.title) document.title = `${to.meta.title} - 广播站工具`
  if (to.path === "/advanced" && !sessionStorage.getItem("advanced_authenticated")) { next("/settings"); return }
  next()
})
export default router
