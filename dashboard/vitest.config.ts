import { defineConfig } from "vitest/config";
import react from "@vitejs/plugin-react";
import path from "path";

export default defineConfig({
  plugins: [react()],
  test: {
    environment: "jsdom",
    setupFiles: ["./src/test/setup.ts"],
    globals: true,
    coverage: {
      provider: "v8",
      reporter: ["text", "lcov"],
    },
  },
  define: {
    "import.meta.env.VITE_API_URL": JSON.stringify("http://localhost:8080"),
    "import.meta.env.VITE_AUTH_URL": JSON.stringify("http://localhost:8081"),
    "import.meta.env.VITE_AUTH_CLIENT_ID": JSON.stringify("dago-dashboard"),
    "import.meta.env.VITE_AUTH_REDIRECT_URI": JSON.stringify("http://localhost:5173/auth/callback"),
  },
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
});
