import type { CreateClientConfig } from "#/api-gen/client.gen";

export const createClientConfig: CreateClientConfig = (config) => ({
  ...config,
  baseUrl: "/api/v1",
});
