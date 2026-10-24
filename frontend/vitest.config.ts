import { defineConfig } from "vitest/config";
import { svelte } from "@sveltejs/vite-plugin-svelte";
import { svelteTesting } from "@testing-library/svelte/vite";
import path from "node:path";
export default defineConfig({
  plugins: [svelte(), svelteTesting()],
  resolve: {
    alias: {
      "@bindings": path.resolve(
        import.meta.dirname,
        "bindings/github.com/LibreDental/libredental/internal"
      ),
      $lib: path.resolve(import.meta.dirname, "src/lib"),
    },
  },
  test: { environment: "jsdom", include: ["tests/**/*.test.ts"], maxWorkers: 1 },
});
