import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  server: {
    port: 3000,
    host: true,
  },
  build: {
    outDir: "dist",
    rollupOptions: {
      input: "./index.html",
    },
  },
  define: {
    // バックエンドベースURL（開発時・本番時で切り替え可能）
    __BACKEND_BASE_URL__: JSON.stringify(
      process.env.VITE_BACKEND_BASE_URL || "http://localhost:8080",
    ),
  },
});
