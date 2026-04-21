// Unified token supplier for the host UI.
//
// Produces a Clerk JWT when the user is signed in, or falls back to
// the localStorage-backed deviceId so the existing guest flow keeps
// working. Returns a stable *function* (not a resolved string)
// because Clerk JWTs are short-lived (~60s) — every API call should
// pull a fresh token so in-flight requests don't 401 after a minute
// of idle time.
//
// Clerk is optional: when NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY is not
// set we never mount ClerkProvider (see app/layout.tsx) and never
// call Clerk hooks, so this file works in bundles where @clerk/nextjs
// isn't on the runtime path at all.

"use client";

import { useAuth } from "@clerk/nextjs";
import { useCallback } from "react";
import { deviceId } from "./deviceId";

/** The return type of useTokenSupplier — an async function that
 *  resolves to the bearer token to send. Always a function (never a
 *  raw string) so callers know to await it, and so Clerk's per-call
 *  JWT refresh works without a rewrite. HostApi's TokenSupplier
 *  accepts this shape. */
export type GetToken = () => Promise<string>;

/** Whether Clerk is configured for this deployment. Constant at
 *  build time — Next inlines the env ref. The parent ClerkProvider
 *  uses the same check to decide whether to mount, so this flag is
 *  the single source of truth for "is Clerk in the tree?". */
export const CLERK_ENABLED = Boolean(
  process.env.NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY,
);

export function useTokenSupplier(): GetToken {
  // eslint-disable-next-line react-hooks/rules-of-hooks
  return CLERK_ENABLED ? useClerkToken() : useGuestToken();
}

// When Clerk isn't mounted, useAuth would throw "useAuth must be
// called inside ClerkProvider." The branch here is constant for any
// given build, so React never sees the hook count change across a
// single component's renders.
function useClerkToken(): GetToken {
  const { isLoaded, isSignedIn, getToken } = useAuth();

  return useCallback(async () => {
    if (!isLoaded || !isSignedIn) {
      return deviceId();
    }
    try {
      const jwt = await getToken();
      return jwt ?? deviceId();
    } catch {
      return deviceId();
    }
  }, [isLoaded, isSignedIn, getToken]);
}

function useGuestToken(): GetToken {
  return useCallback(async () => deviceId(), []);
}

/** Safe wrapper around Clerk's useAuth that degrades cleanly when
 *  Clerk isn't mounted. Components can call this anywhere — signed-
 *  in state will just always be false in dev-without-Clerk builds. */
export function useSignedIn(): { isSignedIn: boolean; isLoaded: boolean } {
  // eslint-disable-next-line react-hooks/rules-of-hooks
  return CLERK_ENABLED ? useClerkSignedIn() : { isSignedIn: false, isLoaded: true };
}

function useClerkSignedIn(): { isSignedIn: boolean; isLoaded: boolean } {
  const { isLoaded, isSignedIn } = useAuth();
  return { isSignedIn: Boolean(isSignedIn), isLoaded: Boolean(isLoaded) };
}
