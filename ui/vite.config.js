import vue from "@vitejs/plugin-vue2"
import autoprefixer from "autoprefixer"
import { fileURLToPath, URL } from "node:url"

export default {
  plugins: [vue()],
  resolve: {
    alias: {
      "@": fileURLToPath(new URL("./src", import.meta.url)),
    },
  },
  css: {
    preprocessorOptions: {
      scss: {
        additionalData: `@import "src/common/global.scss"; @import "src/common/style.scss";`,
      },
    },
    postcss: {
      plugins: [
        autoprefixer({}) // add options if needed
      ],
    }
  },
  server: {
    port: 3000,
    strictPort: true,
    proxy: {
      "/collection": {
        target: process.env.ENDURO_API_ADDRESS || "http://127.0.0.1:9000",
        changeOrigin: true,
      },
      "^/collection/monitor": {
        target: process.env.ENDURO_API_ADDRESS || "http://127.0.0.1:9000",
        changeOrigin: false,
        ws: true,
      },
    },
  },
}
