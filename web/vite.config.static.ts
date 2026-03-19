import { defineConfig } from "vite-plus";
import tsconfigPaths from "vite-tsconfig-paths";

import viteReact from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";

import { fileURLToPath } from "node:url";

const staticIndexHtml = fileURLToPath(new URL("./index.static.html", import.meta.url));

export default defineConfig({
  lint: { options: { typeAware: true, typeCheck: true } },
  plugins: [
    tsconfigPaths({ projects: ["./tsconfig.json"] }),
    tailwindcss(),
    viteReact({
      babel: {
        plugins: ["babel-plugin-react-compiler"],
      },
    }),
  ],
  build: {
    outDir: "dist/static",
    emptyOutDir: true,
    rollupOptions: {
      input: staticIndexHtml,
    },
  },
});

