import { defineConfig } from "astro/config";
import tailwindcss from "@tailwindcss/vite";

export default defineConfig({
  output: "static",
  trailingSlash: "always",
  site: process.env.SITE_URL || undefined,
  vite: { plugins: [tailwindcss()], build: { assetsInlineLimit: 0 } },
});
