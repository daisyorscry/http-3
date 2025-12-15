import { sveltekit } from "@sveltejs/kit/vite";
import { defineConfig } from "vite";

export default defineConfig({
  plugins: [sveltekit()],
  server: { host: true, port: 5000 },
  preview: { host: true, port: 5000 },
  optimizeDeps: {
    exclude: ["better-sqlite3", "@mikro-orm/better-sqlite", "@mikro-orm/core"],
  },
  ssr: { noExternal: [] },
});
