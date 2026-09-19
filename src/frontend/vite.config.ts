import { defineConfig } from "vitest/config";
import react from "@vitejs/plugin-react";

export default defineConfig({
  plugins: [react()],
  server: {
    host: "0.0.0.0",
    port: 5173,
    proxy: {
      "/healthz": "http://localhost:8080",
      "/api": "http://localhost:8080",
    },
  },
  test: {
    environment: "node",
  },
});
