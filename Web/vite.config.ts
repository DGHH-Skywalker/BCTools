import { defineConfig } from "vite"
import vue from "@vitejs/plugin-vue"
import AutoImport from "unplugin-auto-import/vite"
import Components from "unplugin-vue-components/vite"
import { NaiveUiResolver } from "unplugin-vue-components/resolvers"
import viteCompression from "vite-plugin-compression"
import { resolve } from "path"

export default defineConfig({
  // 相对路径 base，让 dist 能在任意 base path 下静态部署
  // （anywhere、nginx 子路径、file:// 双击 index.html 都能跑）
  // 注意：base 必须放在 defineConfig 顶层，build.base 在 vite 5.4 上不生效
  base: "./",
  plugins: [
    vue(),
    AutoImport({
      imports: ["vue", "vue-router", "pinia"],
      dts: "src/auto-imports.d.ts",
    }),
    Components({
      resolvers: [NaiveUiResolver()],
      dts: "src/components.d.ts",
    }),
    viteCompression({ algorithm: "gzip" }),
  ],
  resolve: {
    alias: { "@": resolve(__dirname, "src") },
  },
  server: {
    proxy: {
      "/api": { target: "http://localhost:1743", changeOrigin: true },
    },
  },
  build: {
    outDir: "dist",
    assetsInlineLimit: 4096,
  },
})
