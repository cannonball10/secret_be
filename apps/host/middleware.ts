// Clerk middleware — required for server-side auth helpers (auth(),
// currentUser(), etc.) to resolve the session from cookies. We don't
// protect any routes here since the UX is public-by-default (anyone
// can open the host page; the sign-in CTA is a choice).
//
// No-ops entirely when NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY is absent:
// clerkMiddleware would throw at module load if the key is missing,
// so we export a passthrough in that case to keep the dev-without-
// Clerk flow working.

import { clerkMiddleware } from "@clerk/nextjs/server";
import { NextResponse } from "next/server";

const CLERK_ENABLED = Boolean(process.env.NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY);

export default CLERK_ENABLED ? clerkMiddleware() : () => NextResponse.next();

export const config = {
  matcher: [
    // Everything except Next internals and static assets.
    "/((?!_next|[^?]*\\.(?:html?|css|js(?!on)|jpe?g|webp|png|gif|svg|ttf|woff2?|ico|csv|docx?|xlsx?|zip|webmanifest)).*)",
    "/(api|trpc)(.*)",
  ],
};
