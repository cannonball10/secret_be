// env.ts — single source of truth for the API origin the mobile app
// hits directly. Same story as the host's env.ts: Next rewrites buffer
// SSE in dev, so we bypass them with a direct :8080 connection and
// rely on CORS.

export const API_ORIGIN: string =
  (process.env.NEXT_PUBLIC_API_ORIGIN as string | undefined) ??
  (typeof window !== "undefined" && window.location.hostname === "localhost"
    ? "http://localhost:8080"
    : "");
