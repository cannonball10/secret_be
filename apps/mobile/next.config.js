const API_ORIGIN = process.env.REPLICANT_API_ORIGIN || "http://localhost:8080";

/** @type {import('next').NextConfig} */
const nextConfig = {
  reactStrictMode: true,
  transpilePackages: ["@replicant/ui", "@replicant/tokens", "@replicant/schema"],
  async rewrites() {
    return [
      { source: "/api/:path*", destination: `${API_ORIGIN}/api/:path*` },
      { source: "/healthz", destination: `${API_ORIGIN}/healthz` },
    ];
  },
};

module.exports = nextConfig;
