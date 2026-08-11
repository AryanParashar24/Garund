import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  reactCompiler: true,
  turbopack: {
    // This is a standalone Next.js project nested in the Go repository.
    root: process.cwd(),
  },
};

export default nextConfig;
