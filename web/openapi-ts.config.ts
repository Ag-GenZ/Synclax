import { defineConfig } from "@hey-api/openapi-ts";

export default defineConfig({
  input: "../api/v1.yaml",
  output: {
    path: "./src/api-gen",
    clean: true,
    preferExportAll: true,
  },
  plugins: [
    {
      name: "@hey-api/client-ofetch",
      runtimeConfigPath: "@/lib/client.config",
      exportFromIndex: true,
    },
    {
      name: "@tanstack/react-query",
    },
    {
      name: "zod",
      responses: false,
    },
    {
      name: "@hey-api/sdk",
      validator: true,
    },
  ],
});
