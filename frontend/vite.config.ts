import { defineConfig } from "vite";
import { svelte } from "@sveltejs/vite-plugin-svelte";
import wails from "@wailsio/runtime/plugins/vite";
import tailwindcss from "@tailwindcss/vite";
import path from "path";

// https://vitejs.dev/config/
export default defineConfig({
  server: {
    host: "localhost",
    port: Number(process.env.WAILS_VITE_PORT) || 9245,
    strictPort: true,
  },
  resolve: {
    alias: {
      "@bindings": path.resolve(
        import.meta.dirname,
        "./bindings/github.com/LibreDental/libredental/internal"
      ),
      $lib: path.resolve(import.meta.dirname, "./src/lib"),
    },
  },
  optimizeDeps: {
    include: ["@wailsio/runtime"],
  },
  plugins: [tailwindcss(), svelte(), wails("./bindings")],
});
