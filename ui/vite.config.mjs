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
        // Silence deprecation warnings emitted by bootstrap 4.x or our own code.
        // We won't be able to migrate to Dart Sass 3.x unless we deal with these.
        quietDeps: true,
        silenceDeprecations: ["legacy-js-api", "import", "color-functions", "global-builtin"],
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
      "/pipeline": {
        target: process.env.ENDURO_API_ADDRESS || "http://127.0.0.1:9000",
        changeOrigin: true,
      },
      "/batch": {
        target: process.env.ENDURO_API_ADDRESS || "http://127.0.0.1:9000",
        changeOrigin: true,
      },
    },
  },
}
