// Mobile token supplier — same pattern as apps/host/lib/useToken.ts.
//
// Default is guest (localStorage-backed deviceId). When the user has
// signed in through Clerk, return a fresh short-lived JWT per call.
// Party-game UX: sign-in is OPTIONAL — players should be able to
// type their name and join without any account setup.

"use client";

import { useAuth } from "@clerk/nextjs";
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
