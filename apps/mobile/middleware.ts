// Clerk middleware for the mobile PWA. Same story as apps/host: mount
// only when a publishable key is configured; otherwise pass through.

import { clerkMiddleware } from "@clerk/nextjs/server";
import { NextResponse } from "next/server";

const CLERK_ENABLED = Boolean(process.env.NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY);

export default CLERK_ENABLED ? clerkMiddleware() : () => NextResponse.next();

export const config = {
  matcher: [
    "/((?!_next|[^?]*\\.(?:html?|css|js(?!on)|jpe?g|webp|png|gif|svg|ttf|woff2?|ico|csv|docx?|xlsx?|zip|webmanifest)).*)",
    "/(api|trpc)(.*)",
  ],
};
