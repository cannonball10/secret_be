// HostLobby — live roster + join code/QR + DEPLOY.
//
// Layout ports host-screens.jsx:HostLobby. Two-column grid inside an
// RPTVChrome. Left column: instructions + session code + QR. Right
// column: player roster grid + DEPLOY button. Roster updates as
// `player_joined` envelopes come through SSE.
//
// When the host clicks DEPLOY, we POST /host/start and then route to
// /game/CODE. The game_started / roles_assigned envelopes fire after
// that; mobile devices pick those up from their own streams.

"use client";

import { useEffect, useMemo, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { QRCodeSVG } from "qrcode.react";
import { RPTVChrome } from "@replicant/ui";
import { countryForSeat, type Envelope, type Game, type Player } from "@replicant/schema";
import { rpColors } from "@replicant/tokens";
import { HostApi, ApiError } from "@/lib/api";
import { deviceId } from "@/lib/deviceId";
import { clearSession, loadSession, saveSession } from "@/lib/session";
import { useStream } from "@/lib/useStream";

const MIN_PLAYERS = 5;
const MAX_PLAYERS = 10;

// Maps a delegate number to a consistent accent color so every seat
// keeps the same portrait color between re-renders.
const ACCENT_PALETTE = [
  rpColors.ink,
  rpColors.stampBlue,
  rpColors.stampRed,
  rpColors.cyanSoft,
  rpColors.amber,
  rpColors.stampGreen,
];

function accentFor(seat: number): string {
  return ACCENT_PALETTE[seat % ACCENT_PALETTE.length] ?? rpColors.ink;
}

function portraitFor(name: string): string {
  return name.trim().slice(0, 1).toUpperCase() || "?";
}

export default function HostLobbyPage() {
  const router = useRouter();
  const params = useParams<{ code: string }>();
  const code = (params?.code ?? "").toUpperCase();

  const [token, setToken] = useState<string | null>(null);
  const [game, setGame] = useState<Game | null>(null);
  const [players, setPlayers] = useState<Record<string, Player>>({});
  const [err, setErr] = useState<string | null>(null);

  // ── Boot: resolve the session's gameId from the code via GET. ──
  useEffect(() => {
    const cached = loadSession();
    if (!cached || cached.joinCode !== code) {
      // Host opened a stale URL without a matching session cache —
      // boot them back to the title so they can either resume from
      // the cached code or create a fresh lobby.
      router.replace("/");
      return;
    }
    setToken(cached.token);

    (async () => {
      try {
        const api = new HostApi({ token: cached.token });
        const snap = await api.getGame(cached.gameId);
        setGame(snap.game);
        const ix: Record<string, Player> = {};
        for (const p of snap.players) ix[p.playerId] = p;
        setPlayers(ix);
        // If the game has already left the lobby, bounce to the game
        // screen — happens after a refresh post-deploy.
        if (snap.game.status !== "lobby") {
          router.replace(`/game/${code}`);
        }
      } catch (e) {
        setErr(e instanceof ApiError ? `${e.status}: ${e.message}` : String(e));
      }
    })();
  }, [code, router]);

  // ── SSE: keep the roster live. ─────────────────────────────────
  useStream({
    gameId: game?.gameId ?? null,
    role: "board",
    token,
    onEnvelope: (env: Envelope) => {
      const type = env.event.type;
      if (type === "player_joined") {
        const p = env.payload as { playerId: string; displayName: string; seat: number } | undefined;
        if (!p) return;
        setPlayers((prev) => ({
          ...prev,
          [p.playerId]: {
            playerId: p.playerId,
            gameId: env.gameId,
            userId: "",
            displayName: p.displayName,
            seat: p.seat,
            isHost: false,
            isAlive: true,
            isConnected: true,
            createdAt: "",
            updatedAt: "",
          },
        }));
      }
      if (type === "game_started") {
        // Host started the game — navigate. Mobile clients do the
        // same on their side via their own stream.
        router.push(`/game/${code}`);
      }
    },
  });

  // ── Derived state ──────────────────────────────────────────────
  const seated = useMemo(
    () => Object.values(players).sort((a, b) => a.seat - b.seat),
    [players],
  );
  const ready = seated.length >= MIN_PLAYERS;

  // URL mobile devices point their browser at. In dev Next runs the
  // mobile app on :3001; override at build with NEXT_PUBLIC_MOBILE_URL.
  const mobileUrl = useMemo(() => {
    const envUrl = process.env.NEXT_PUBLIC_MOBILE_URL;
    if (envUrl) return `${envUrl.replace(/\/$/, "")}/join/${code}`;
    if (typeof window === "undefined") return `/join/${code}`;
    const { hostname, protocol } = window.location;
    return `${protocol}//${hostname}:3001/join/${code}`;
  }, [code]);

  const deploy = async () => {
    if (!game || !token || !ready) return;
    try {
      const api = new HostApi({ token });
      await api.startGame(game.gameId);
      // game_started envelope will arrive via SSE and route us.
    } catch (e) {
      setErr(e instanceof ApiError ? `${e.status}: ${e.message}` : String(e));
    }
  };

  const leave = () => {
    clearSession();
    router.push("/");
  };

  return (
    <div style={{ width: "100vw", height: "100vh" }}>
      <RPTVChrome title="CANDIDATE INTAKE" nodeId={`NODE ${code.slice(0, 6)}`} phase="PRE-DEPLOY">
        {err && (
          <div
            style={{
              position: "absolute",
              top: 12,
              right: 12,
              border: "1px solid var(--stamp-red)",
              color: "var(--stamp-red)",
              background: "rgba(179,39,36,0.08)",
              padding: "6px 12px",
              fontFamily: "var(--font-mono)",
              fontSize: 11,
              zIndex: 20,
            }}
          >
            {err}
          </div>
        )}

        <div
          style={{
            display: "grid",
            gridTemplateColumns: "1fr 1fr",
            height: "100%",
            gap: 0,
          }}
        >
          {/* LEFT — join instructions + code + QR */}
          <div
            style={{
              padding: "56px 72px",
              display: "flex",
              flexDirection: "column",
              gap: 36,
              borderRight: `1px solid ${rpColors.broadcastRule}`,
            }}
          >
            <div>
              <div className="t-eyebrow" style={{ color: rpColors.cyan, marginBottom: 14 }}>
                ◼ PLANETARY COMMITTEE · ACCORD R-07
              </div>
              <div
                style={{
                  fontFamily: "var(--font-display)",
                  fontWeight: 700,
                  fontSize: 120,
                  lineHeight: 0.9,
                  letterSpacing: "-0.02em",
                  position: "relative",
                }}
              >
                <span
                  aria-hidden="true"
                  style={{ position: "absolute", left: 6, top: 6, color: rpColors.stampRed, opacity: 0.5 }}
                >
                  REPLICANT
                </span>
                <span style={{ color: rpColors.paper3, position: "relative" }}>REPLICANT</span>
              </div>
              <div
                className="t-memo"
                style={{ color: rpColors.paper3, marginTop: 16, fontSize: 18, opacity: 0.8, maxWidth: 520 }}
              >
                A Planetary Committee Session, in Eight Rounds.
              </div>
            </div>

            <div
              style={{
                border: `1px solid ${rpColors.broadcastRule}`,
                padding: "24px 28px",
                background: "rgba(95,211,204,0.04)",
              }}
            >
              <div className="t-eyebrow" style={{ color: rpColors.cyan, marginBottom: 12 }}>
                ▸ INTAKE INSTRUCTIONS
              </div>
              <ol
                style={{
                  margin: 0,
                  padding: "0 0 0 22px",
                  fontFamily: "var(--font-typewriter)",
                  fontSize: 17,
                  lineHeight: 1.9,
                  color: rpColors.paper3,
                }}
              >
                <li>
                  Scan the code, or open <span style={{ color: rpColors.cyan }}>{displayMobileHost(mobileUrl)}</span> on your device.
                </li>
                <li>Submit the six-character session code below.</li>
                <li>Await role allocation. Do not discuss your assignment.</li>
              </ol>
            </div>

            <div style={{ display: "flex", alignItems: "center", gap: 24 }}>
              <div>
                <div className="t-eyebrow" style={{ color: rpColors.inkFaded, fontSize: 11, marginBottom: 6 }}>
                  SESSION CODE
                </div>
                <div
                  style={{
                    fontFamily: "var(--font-mono)",
                    fontWeight: 700,
                    fontSize: 64,
                    letterSpacing: "0.15em",
                    color: rpColors.cyan,
                    lineHeight: 1,
                    textShadow: `0 0 20px ${rpColors.cyan}66`,
                  }}
                >
                  {code}
                </div>
              </div>
              <div style={{ background: rpColors.paper3, padding: 8 }}>
                <QRCodeSVG value={mobileUrl} size={104} bgColor={rpColors.paper3} fgColor={rpColors.ink} />
              </div>
            </div>

            <div style={{ marginTop: "auto" }}>
              <button
                onClick={leave}
                style={{
                  background: "transparent",
                  border: `1px solid ${rpColors.broadcastRule}`,
                  color: rpColors.inkFaded,
                  padding: "6px 12px",
                  fontFamily: "var(--font-mono)",
                  fontSize: 10,
                  letterSpacing: 1.5,
                  cursor: "pointer",
                }}
              >
                ABANDON INTAKE
              </button>
            </div>
          </div>

          {/* RIGHT — roster + DEPLOY */}
          <div style={{ padding: "56px 72px", display: "flex", flexDirection: "column", gap: 28 }}>
            <div style={{ display: "flex", justifyContent: "space-between", alignItems: "flex-end" }}>
              <div>
                <div className="t-eyebrow" style={{ color: rpColors.cyan, marginBottom: 6 }}>
                  CANDIDATES REGISTERED
                </div>
                <div
                  style={{
                    fontFamily: "var(--font-display)",
                    fontWeight: 700,
                    fontSize: 72,
                    color: rpColors.paper3,
                    lineHeight: 0.9,
                  }}
                >
                  {String(seated.length).padStart(2, "0")}
                  <span style={{ color: rpColors.inkFaded, fontSize: 40 }}>/{String(MAX_PLAYERS).padStart(2, "0")}</span>
                </div>
              </div>
              <div
                style={{
                  fontFamily: "var(--font-mono)",
                  fontSize: 13,
                  color: rpColors.inkFaded,
                  textAlign: "right",
                  lineHeight: 1.6,
                }}
              >
                MIN. THRESHOLD: {String(MIN_PLAYERS).padStart(2, "0")}
                <br />
                MAX. CAPACITY: {String(MAX_PLAYERS).padStart(2, "0")}
                <br />
                <span style={{ color: ready ? rpColors.stampGreen : rpColors.inkFaded }}>
                  {ready ? "● QUORUM ACHIEVED" : "○ QUORUM PENDING"}
                </span>
              </div>
            </div>

            <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 10 }}>
              {seated.map((p) => (
                <div
                  key={p.playerId}
                  style={{
                    display: "flex",
                    alignItems: "center",
                    gap: 12,
                    border: `1px solid ${rpColors.cyan}`,
                    background: "rgba(95,211,204,0.06)",
                    padding: "10px 14px",
                  }}
                >
                  <div
                    style={{
                      width: 44,
                      height: 44,
                      flexShrink: 0,
                      background: accentFor(p.seat),
                      color: rpColors.broadcast,
                      display: "flex",
                      alignItems: "center",
                      justifyContent: "center",
                      fontFamily: "var(--font-display)",
                      fontWeight: 700,
                      fontSize: 22,
                    }}
                  >
                    {portraitFor(p.displayName)}
                  </div>
                  <div style={{ flex: 1, minWidth: 0 }}>
                    <div
                      style={{
                        fontFamily: "var(--font-mono)",
                        fontSize: 10,
                        color: rpColors.inkFaded,
                        letterSpacing: 1.4,
                      }}
                    >
                      {(() => {
                        const c = p.countryCode
                          ? { code: p.countryCode, name: p.countryName ?? "" }
                          : countryForSeat(p.seat);
                        return c
                          ? `SEAT ${String(p.seat + 1).padStart(2, "0")} · ${c.code}`
                          : `SEAT ${String(p.seat + 1).padStart(2, "0")}`;
                      })()}
                    </div>
                    <div
                      style={{
                        fontFamily: "var(--font-display)",
                        fontWeight: 600,
                        fontSize: 18,
                        color: rpColors.paper3,
                        letterSpacing: "0.02em",
                        textTransform: "uppercase",
                        overflow: "hidden",
                        textOverflow: "ellipsis",
                        whiteSpace: "nowrap",
                      }}
                    >
                      {p.displayName}
                    </div>
                    {(() => {
                      const c = p.countryCode
                        ? { code: p.countryCode, name: p.countryName ?? "" }
                        : countryForSeat(p.seat);
                      if (!c?.name) return null;
                      return (
                        <div
                          style={{
                            fontFamily: "var(--font-mono)",
                            fontSize: 10,
                            color: rpColors.cyanSoft,
                            letterSpacing: 1.2,
                            textTransform: "uppercase",
                            marginTop: 2,
                            overflow: "hidden",
                            textOverflow: "ellipsis",
                            whiteSpace: "nowrap",
                          }}
                        >
                          DELEGATE OF {c.name}
                        </div>
                      );
                    })()}
                  </div>
                  <div
                    style={{
                      fontFamily: "var(--font-mono)",
                      fontSize: 10,
                      color: rpColors.cyan,
                    }}
                  >
                    ● READY
                  </div>
                </div>
              ))}
              {/* Empty-slot placeholders so the grid feels bounded. */}
              {Array.from({ length: Math.max(0, MIN_PLAYERS - seated.length) }).map((_, i) => (
                <div
                  key={`empty-${i}`}
                  style={{
                    display: "flex",
                    alignItems: "center",
                    gap: 12,
                    border: `1px dashed ${rpColors.broadcastRule}`,
                    padding: "10px 14px",
                    opacity: 0.4,
                  }}
                >
                  <div
                    style={{
                      width: 44,
                      height: 44,
                      flexShrink: 0,
                      border: `1px dashed ${rpColors.broadcastRule}`,
                    }}
                  />
                  <div
                    className="t-eyebrow"
                    style={{ color: rpColors.inkFaded, fontSize: 11 }}
                  >
                    AWAITING CANDIDATE…
                  </div>
                </div>
              ))}
            </div>

            <div
              style={{
                marginTop: "auto",
                borderTop: `1px solid ${rpColors.broadcastRule}`,
                paddingTop: 20,
                display: "flex",
                justifyContent: "space-between",
                alignItems: "center",
              }}
            >
              <div
                style={{
                  fontFamily: "var(--font-typewriter)",
                  fontSize: 15,
                  color: rpColors.paper3,
                  opacity: 0.7,
                  maxWidth: "60%",
                }}
              >
                {ready
                  ? '"The Committee accepts your convened delegation. Deploy when ready."'
                  : `"Awaiting ${MIN_PLAYERS - seated.length} further delegate${
                      MIN_PLAYERS - seated.length === 1 ? "" : "s"
                    }. The Committee does not tolerate tardiness."`}
              </div>
              <button
                onClick={deploy}
                disabled={!ready}
                style={{
                  background: ready ? rpColors.cyan : "transparent",
                  color: ready ? rpColors.broadcast : rpColors.inkFaded,
                  border: `1px solid ${ready ? rpColors.cyan : rpColors.broadcastRule}`,
                  padding: "10px 22px",
                  fontFamily: "var(--font-display)",
                  fontSize: 16,
                  letterSpacing: "0.1em",
                  fontWeight: 700,
                  cursor: ready ? "pointer" : "not-allowed",
                }}
              >
                DEPLOY ▸
              </button>
            </div>
          </div>
        </div>
      </RPTVChrome>
    </div>
  );
}

/** Strips scheme + trailing slash for clean display inside the memo. */
function displayMobileHost(url: string): string {
  try {
    const u = new URL(url);
    const port = u.port && u.port !== "80" && u.port !== "443" ? `:${u.port}` : "";
    return `${u.hostname}${port}`;
  } catch {
    return url.replace(/^https?:\/\//, "").replace(/\/.*$/, "");
  }
}
