import type { CreateClientConfig } from "#/api-gen/client.gen";

export const createClientConfig: CreateClientConfig = (config) => ({
  ...config,
  baseUrl: "http://localhost:2910/api/v1",
});
