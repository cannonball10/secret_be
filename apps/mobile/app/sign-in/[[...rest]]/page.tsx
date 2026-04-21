// Clerk sign-in for the mobile PWA. Optional — players can skip
// and join as guests. Same catch-all route shape as the host's.

"use client";

import { SignIn } from "@clerk/nextjs";

export default function MobileSignInPage() {
  return (
    <main
      style={{
        minHeight: "100vh",
        display: "flex",
        alignItems: "center",
        justifyContent: "center",
        padding: 24,
      }}
    >
      <SignIn />
    </main>
  );
}
