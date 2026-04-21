// Clerk-hosted sign-in flow. Catch-all route so Clerk's
// internal links (factor-one, factor-two, SSO callbacks) all resolve
// here. On success, Clerk bounces back to afterSignInUrl which is
// configured via NEXT_PUBLIC_CLERK_SIGN_IN_FALLBACK_REDIRECT_URL env.

"use client";

import { SignIn } from "@clerk/nextjs";

export default function SignInPage() {
  return (
    <main
      style={{
        minHeight: "100vh",
        display: "flex",
        alignItems: "center",
        justifyContent: "center",
        padding: 48,
      }}
    >
      <SignIn
        appearance={{
          elements: {
            card: { background: "var(--paper-3)" },
          },
        }}
      />
    </main>
  );
}
