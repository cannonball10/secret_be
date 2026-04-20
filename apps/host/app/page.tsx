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

type BootState = "boot" | "ready" | "working" | "error";

export default function HomePage() {
  const router = useRouter();
  const [state, setState] = useState<BootState>("boot");
  const [err, setErr] = useState<string | null>(null);

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
        const api = new HostApi({ token: cached.token });
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
  }, [router]);

  const host = async () => {
    setState("working");
    setErr(null);
    try {
      const token = deviceId();
      const api = new HostApi({ token });
      const { game } = await api.createGame();
      saveSession({
        token,
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
      }}
    >
      <div className="t-eyebrow" style={{ color: "var(--cyan)" }}>
        ◼ DEPT. OF HUMAN AFFAIRS · FORM R-07
      </div>
      <RPWordmark size={120} color="var(--paper-3)" ghost="var(--stamp-red)" />
      <div className="t-memo" style={{ color: "var(--paper-3)", opacity: 0.7, fontSize: 18 }}>
        A Human Verification Procedure, in Eight Rounds.
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
