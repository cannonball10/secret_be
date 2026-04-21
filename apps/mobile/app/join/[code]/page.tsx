// /join/CODE — reached either by QR scan from the host or by the
// manual entry flow on /. Code is locked (disabled) so the player
// only has to provide a display name.

"use client";

import { useEffect, useState } from "react";
import { useParams, useRouter, useSearchParams } from "next/navigation";
import { RPButton, RPSeal } from "@replicant/ui";
import { MobileApi, ApiError } from "@/lib/api";
import { deviceId, resetDeviceId } from "@/lib/deviceId";
import { clearSession, loadSession, saveSession } from "@/lib/session";
import { useSignOutSafe, useTokenSupplier } from "@/lib/useToken";

export default function MobileJoinPage() {
  const router = useRouter();
  const params = useParams<{ code: string }>();
  const search = useSearchParams();
  const code = (params?.code ?? "").toUpperCase();

  const [name, setName] = useState("");
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState<string | null>(null);
  const [ready, setReady] = useState(false);
  const getToken = useTokenSupplier();
  const signOutSafe = useSignOutSafe();

  // ?asNew=1 forces a completely fresh identity so one browser
  // profile can seat multiple players during local QA. That means:
  //   - mobile session (localStorage) cleared
  //   - deviceId re-rolled
  //   - Clerk signed out, since incognito WINDOWS share one cookie
  //     jar. Skipping this step means a Clerk-authed tester would
  //     keep resolving to their existing User row and the server
  //     would dedupe them to the same Player.
  // After the reset we replace() the URL to drop ?asNew=1 so a
  // refresh doesn't keep re-rolling.
  useEffect(() => {
    if (search?.get("asNew") === "1") {
      (async () => {
        clearSession();
        resetDeviceId();
        await signOutSafe();
        router.replace(`/join/${code}`);
      })();
      return;
    }
    const s = loadSession();
    if (s && s.joinCode === code) {
      router.replace("/game");
      return;
    }
    setReady(true);
  }, [code, router, search, signOutSafe]);

  const join = async () => {
    if (!name.trim() || busy) return;
    setBusy(true);
    setErr(null);
    try {
      // Resolve the token at click time: signed-in players get their
      // Clerk JWT and the server resolves them to their Clerk User
      // row; guests fall back to their persistent deviceId.
      const t = await getToken();
      const api = new MobileApi({ token: getToken });
      const { game, player } = await api.joinGame(code, name.trim());
      saveSession({
        token: t,
        gameId: game.gameId,
        joinCode: game.joinCode,
        playerId: player.playerId,
        displayName: player.displayName,
      });
      router.push("/game");
    } catch (e) {
      setBusy(false);
      setErr(e instanceof ApiError ? `${e.status}: ${e.message}` : String(e));
    }
  };

  // Hold the render until the reset/redirect effect settles. Avoids a
  // one-frame flash of the intake form when asNew=1 is being processed.
  if (!ready) return null;

  return (
    <main style={{ padding: "28px 22px 40px", display: "flex", flexDirection: "column", gap: 20 }}>
      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "flex-start" }}>
        <div>
          <div className="t-eyebrow" style={{ color: "var(--ink-faded)", marginBottom: 6 }}>
            ◼ INTAKE · ACCORD R-07
          </div>
          <div
            style={{
              fontFamily: "var(--font-display)",
              fontWeight: 700,
              fontSize: 36,
              color: "var(--ink)",
              lineHeight: 0.95,
            }}
          >
            IDENTIFY
            <br />
            YOURSELF
          </div>
        </div>
        <RPSeal size={72} />
      </div>

      <label style={{ display: "flex", flexDirection: "column", gap: 6 }}>
        <span className="t-eyebrow" style={{ color: "var(--ink-faded)", fontSize: 10 }}>
          SESSION CODE
        </span>
        <input
          value={code}
          disabled
          readOnly
          style={{
            fontFamily: "var(--font-mono)",
            fontWeight: 700,
            fontSize: 22,
            letterSpacing: "0.3em",
            padding: "10px 12px",
            border: "2px solid var(--ink-faded)",
            background: "var(--paper-2)",
            color: "var(--ink-soft)",
            textAlign: "center",
          }}
        />
      </label>

      <label style={{ display: "flex", flexDirection: "column", gap: 6 }}>
        <span className="t-eyebrow" style={{ color: "var(--ink-faded)", fontSize: 10 }}>
          DISPLAY NAME
        </span>
        <input
          value={name}
          onChange={(e) => setName(e.target.value)}
          maxLength={32}
          autoComplete="off"
          spellCheck={false}
          placeholder="How the Committee will address you"
          style={{
            fontFamily: "var(--font-typewriter)",
            fontSize: 18,
            padding: "10px 12px",
            border: "2px solid var(--ink)",
            background: "var(--paper-3)",
            color: "var(--ink)",
          }}
        />
      </label>

      {err && (
        <div
          style={{
            border: "1px solid var(--stamp-red)",
            color: "var(--stamp-red)",
            background: "rgba(179,39,36,0.08)",
            padding: "8px 12px",
            fontFamily: "var(--font-mono)",
            fontSize: 12,
          }}
        >
          {err}
        </div>
      )}

      <RPButton
        variant="stamp"
        full
        disabled={!name.trim() || busy}
        onClick={join}
      >
        ▸ Seat with the Committee
      </RPButton>

      <button
        onClick={() => {
          clearSession();
          router.push("/");
        }}
        style={{
          background: "transparent",
          border: "none",
          color: "var(--ink-faded)",
          fontFamily: "var(--font-mono)",
          fontSize: 11,
          letterSpacing: 1.4,
          cursor: "pointer",
          padding: 0,
          marginTop: "auto",
          textAlign: "center",
        }}
      >
        ← ENTER DIFFERENT CODE
      </button>
    </main>
  );
}
