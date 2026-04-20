const API_ORIGIN = process.env.REPLICANT_API_ORIGIN || "http://localhost:8080";

/** @type {import('next').NextConfig} */
const nextConfig = {
  reactStrictMode: true,
  // Let Next compile TS directly out of the workspace packages without
  // needing a build step in each.
  transpilePackages: ["@replicant/ui", "@replicant/tokens", "@replicant/schema"],
  // Proxy backend calls + SSE to the Go server so the browser stays
  // on the Next origin — sidesteps CORS entirely in dev.
  async rewrites() {
    return [
      { source: "/api/:path*", destination: `${API_ORIGIN}/api/:path*` },
      { source: "/healthz", destination: `${API_ORIGIN}/healthz` },
    ];
  },
};

module.exports = nextConfig;
