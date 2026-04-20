// env.ts — resolve the API origin the browser should talk to.
//
// In dev the Next rewrite in next.config.js proxies /api → :8080 for
// REST, but the Next dev server is known to buffer long-lived SSE
// responses through that proxy, so the board stream never delivers
// envelopes. To avoid that we point the browser DIRECTLY at the Go
// server for both REST and SSE, and turn on CORS on the Go side.
//
// Production: set NEXT_PUBLIC_API_ORIGIN to your deployed API, e.g.
// "https://api.replicant.example.com". Leave empty and requests stay
// same-origin via the Next rewrite.

export const API_ORIGIN: string =
  (process.env.NEXT_PUBLIC_API_ORIGIN as string | undefined) ??
  (typeof window !== "undefined" && window.location.hostname === "localhost"
    ? "http://localhost:8080"
    : "");
