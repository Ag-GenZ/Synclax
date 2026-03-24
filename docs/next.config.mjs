import { createMDX } from "fumadocs-mdx/next";

const withMDX = createMDX({
  configPath: "./source.config.ts",
});

const nextConfig = {
  reactStrictMode: true,
};

export default withMDX(nextConfig);
