// Host landing screen.
//
// Three states:
//   • Boot: check for a cached session — if the server still reports
//     it live, jump straight to /lobby/CODE (or /game/CODE if past
//     the lobby) so a refresh doesn't strand the board.
//   • Fresh: show the title card + Host button. Clicking Host creates
//     a board-only session on the Go engine and routes to the lobby.
//   • Error: show the error, offer retry.

"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { RPButton, RPWordmark } from "@replicant/ui";
import { HostApi, ApiError } from "@/lib/api";
import { deviceId } from "@/lib/deviceId";
import { clearSession, loadSession, saveSession } from "@/lib/session";
import { CLERK_ENABLED, useSignedIn, useTokenSupplier } from "@/lib/useToken";
import { SignInButton, UserButton } from "@clerk/nextjs";

type BootState = "boot" | "ready" | "working" | "error";

export default function HomePage() {
  const router = useRouter();
  const [state, setState] = useState<BootState>("boot");
  const [err, setErr] = useState<string | null>(null);
  const getToken = useTokenSupplier();
  const { isSignedIn } = useSignedIn();

  // Boot: restore a cached board session if still live.
  useEffect(() => {
    let cancelled = false;
    (async () => {
      const cached = loadSession();
      if (!cached) {
        setState("ready");
        return;
      }
      try {
        const api = new HostApi({ token: getToken });
        const { game } = await api.getGame(cached.gameId);
        if (cancelled) return;
        if (game.status === "completed" || game.status === "abandoned") {
          clearSession();
          setState("ready");
          return;
        }
        // Live — route back into the active lobby/game.
        router.replace(
          game.status === "lobby" ? `/lobby/${game.joinCode}` : `/game/${game.joinCode}`,
        );
      } catch {
        if (cancelled) return;
        clearSession();
        setState("ready");
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [router, getToken]);

  const host = async () => {
    setState("working");
    setErr(null);
    try {
      const api = new HostApi({ token: getToken });
      const { game } = await api.createGame();
      // Store deviceId as the session token for backwards compat
      // with code paths that still read session.token. Once every
      // caller moves to useTokenSupplier this field will be pruned.
      saveSession({
        token: deviceId(),
        gameId: game.gameId,
        role: "board",
        joinCode: game.joinCode,
      });
      router.push(`/lobby/${game.joinCode}`);
    } catch (e) {
      setState("error");
      setErr(e instanceof ApiError ? `${e.status}: ${e.message}` : String(e));
    }
  };

  return (
    <main
      style={{
        minHeight: "100vh",
        display: "flex",
        flexDirection: "column",
        alignItems: "center",
        justifyContent: "center",
        gap: 28,
        padding: 48,
        textAlign: "center",
        position: "relative",
      }}
    >
      {CLERK_ENABLED && (
        <div
          style={{
            position: "absolute",
            top: 20,
            right: 24,
            display: "flex",
            alignItems: "center",
            gap: 12,
          }}
        >
          {isSignedIn ? (
            <UserButton afterSignOutUrl="/" />
          ) : (
            <SignInButton mode="modal">
              <button
                className="t-eyebrow"
                style={{
                  background: "transparent",
                  color: "var(--paper-3)",
                  border: "1px solid var(--paper-3)",
                  padding: "6px 12px",
                  fontSize: 10,
                  letterSpacing: 1.4,
                  cursor: "pointer",
                }}
              >
                ▸ SIGN IN · SAVE PASSPORT
              </button>
            </SignInButton>
          )}
        </div>
      )}
      <div className="t-eyebrow" style={{ color: "var(--cyan)" }}>
        ◼ PLANETARY COMMITTEE · ACCORD R-07
      </div>
      <RPWordmark size={120} color="var(--paper-3)" ghost="var(--stamp-red)" />
      <div className="t-memo" style={{ color: "var(--paper-3)", opacity: 0.7, fontSize: 18 }}>
        A Planetary Committee Session, in Eight Rounds.
      </div>

      {state === "boot" && (
        <div className="t-eyebrow" style={{ color: "var(--ink-faded)" }}>
          Checking for an active session…
        </div>
      )}

      {(state === "ready" || state === "error") && (
        <div style={{ display: "flex", flexDirection: "column", gap: 16, alignItems: "center" }}>
          <RPButton variant="stamp" onClick={host}>
            ▸ OPEN NEW INTAKE
          </RPButton>
          {err && (
            <div
              style={{
                border: "1px solid var(--stamp-red)",
                color: "var(--stamp-red)",
                padding: "8px 14px",
                fontFamily: "var(--font-mono)",
                fontSize: 12,
                maxWidth: 520,
              }}
            >
              {err}
            </div>
          )}
        </div>
      )}

      {state === "working" && (
        <div className="t-eyebrow" style={{ color: "var(--cyan)" }}>
          Requisitioning session…
        </div>
      )}
    </main>
  );
}
