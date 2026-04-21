// Mobile token supplier — same pattern as apps/host/lib/useToken.ts.
//
// Default is guest (localStorage-backed deviceId). When the user has
// signed in through Clerk, return a fresh short-lived JWT per call.
// Party-game UX: sign-in is OPTIONAL — players should be able to
// type their name and join without any account setup.

"use client";

import { useAuth, useClerk } from "@clerk/nextjs";
import { useCallback } from "react";
import { deviceId } from "./deviceId";

export type GetToken = () => Promise<string>;

export const CLERK_ENABLED = Boolean(
  process.env.NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY,
);

export function useTokenSupplier(): GetToken {
  // eslint-disable-next-line react-hooks/rules-of-hooks
  return CLERK_ENABLED ? useClerkToken() : useGuestToken();
}

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

export function useSignedIn(): { isSignedIn: boolean; isLoaded: boolean } {
  // eslint-disable-next-line react-hooks/rules-of-hooks
  return CLERK_ENABLED ? useClerkSignedIn() : { isSignedIn: false, isLoaded: true };
}

function useClerkSignedIn() {
  const { isLoaded, isSignedIn } = useAuth();
  return { isSignedIn: Boolean(isSignedIn), isLoaded: Boolean(isLoaded) };
}

/** Safely returns a sign-out function regardless of whether Clerk is
 *  mounted. When Clerk is disabled or the user is a guest, this is
 *  a no-op — callers can fire it unconditionally without checking
 *  CLERK_ENABLED themselves. */
export function useSignOutSafe(): () => Promise<void> {
  // eslint-disable-next-line react-hooks/rules-of-hooks
  return CLERK_ENABLED ? useClerkSignOut() : useNoopSignOut();
}

function useClerkSignOut() {
  const { signOut } = useClerk();
  return useCallback(async () => {
    try {
      await signOut();
    } catch {
      /* already signed out or Clerk failed — not fatal */
    }
  }, [signOut]);
}

function useNoopSignOut() {
  return useCallback(async () => {}, []);
}
