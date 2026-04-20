// App.tsx — composition root.
//
// Flow:
//   1. Boot: try to restore a cached session via localStorage.
//   2. Title screen: enter a name, Host or Join.
//   3. Game view: plain wireframe of seats, phase, and controls.

import { useCallback, useEffect, useMemo, useReducer, useRef, useState } from "react";
import { ApiClient, ApiError } from "./api";
import { TitleScreen } from "./TitleScreen";
import { clearSession, loadSession, saveSession } from "./session";
import {
  currentChancellor,
  currentPresident,
  initialState,
  isMeChancellor,
  isMePresident,
  reducer,
  seatedPlayers,
} from "./store";
import { streamEnvelopes } from "./stream";
import type { Game, VoteChoice } from "./types";

const BASE_URL: string =
  (import.meta.env.VITE_API_URL as string | undefined) ?? "";

type Route =
  | { kind: "boot" }
  | { kind: "title" }
  | { kind: "game"; role: "board" | "player"; gameId: string };

export default function App() {
  const [token, setToken] = useState<string>("");
  const [route, setRoute] = useState<Route>({ kind: "boot" });
  const [state, dispatch] = useReducer(reducer, initialState);
  const [err, setErr] = useState<string>("");
  const bootedRef = useRef(false);

  const api = useMemo(
    () => (token ? new ApiClient({ baseUrl: BASE_URL, token }) : null),
    [token],
  );

  // ── Boot: restore session if the server still has a live game. ──
  useEffect(() => {
    if (bootedRef.current) return;
    bootedRef.current = true;
    (async () => {
      const cached = loadSession();
      if (!cached) {
        setRoute({ kind: "title" });
        return;
      }
      const client = new ApiClient({ baseUrl: BASE_URL, token: cached.token });
      try {
        const { game, players } = await client.getGame(cached.gameId);
        if (game.status === "completed") {
          clearSession();
          setRoute({ kind: "title" });
          return;
        }
        const me = players.find(
          (p) => p.userId === cached.token || p.playerId === cached.token,
        );
        dispatch({
          type: "snapshot",
          game,
          players,
          mePlayerId: me?.playerId,
        });
        setToken(cached.token);
        setRoute({ kind: "game", role: cached.role, gameId: game.gameId });
      } catch {
        clearSession();
        setRoute({ kind: "title" });
      }
    })();
  }, []);

  // ── SSE subscription, scoped to the active game. ───────────────
  useEffect(() => {
    if (route.kind !== "game" || !token) return;
    const handle = streamEnvelopes({
      baseUrl: BASE_URL,
      gameId: route.gameId,
      role: route.role,
      token,
      onEnvelope: (envelope) => {
        dispatch({ type: "envelope", envelope });
        if (envelope.event.type === "game_ended") {
          clearSession();
        }
      },
      onError: (e) => console.warn("stream error", e),
    });
    return () => handle.close();
  }, [route, token]);

  // ── Title-screen actions. ──────────────────────────────────────

  const handleHost = useCallback(async (enteredToken: string) => {
    const client = new ApiClient({ baseUrl: BASE_URL, token: enteredToken });
    try {
      const { game, player } = await client.createGame();
      setToken(enteredToken);
      dispatch({
        type: "snapshot",
        game,
        players: player ? [player] : [],
        mePlayerId: player?.playerId,
      });
      saveSession({ token: enteredToken, gameId: game.gameId, role: "board" });
      setRoute({ kind: "game", role: "board", gameId: game.gameId });
      setErr("");
    } catch (e) {
      setErr(errMsg(e));
    }
  }, []);

  const handleJoin = useCallback(
    async (enteredToken: string, joinCode: string, displayName: string) => {
      const client = new ApiClient({ baseUrl: BASE_URL, token: enteredToken });
      try {
        const { game, player, players } = await client.joinGame(
          joinCode,
          displayName,
        );
        setToken(enteredToken);
        dispatch({
          type: "snapshot",
          game,
          players,
          mePlayerId: player.playerId,
        });
        saveSession({ token: enteredToken, gameId: game.gameId, role: "player" });
        setRoute({ kind: "game", role: "player", gameId: game.gameId });
        setErr("");
      } catch (e) {
        setErr(errMsg(e));
      }
    },
    [],
  );

  const handleLeave = useCallback(() => {
    clearSession();
    setToken("");
    setRoute({ kind: "title" });
    dispatch({ type: "reset" });
    setErr("");
  }, []);

  return (
    <div className="page">
      {route.kind === "boot" && <div className="muted">Loading…</div>}

      {route.kind === "title" && (
        <TitleScreen onHost={handleHost} onJoin={handleJoin} error={err} />
      )}

      {route.kind === "game" && api && (
        <GameView
          route={route}
          state={state}
          api={api}
          onErr={(e) => setErr(errMsg(e))}
          onLeave={handleLeave}
          err={err}
        />
      )}
    </div>
  );
}

// ─── game view ───────────────────────────────────────────────────

function GameView({
  route,
  state,
  api,
  onErr,
  onLeave,
  err,
}: {
  route: Extract<Route, { kind: "game" }>;
  state: ReturnType<typeof reducer>;
  api: ApiClient;
  onErr: (e: unknown) => void;
  onLeave: () => void;
  err: string;
}) {
  const g = state.game;
  if (!g) return <div className="muted">Loading…</div>;

  const seated = seatedPlayers(state);
  const pres = currentPresident(state);
  const chan = currentChancellor(state);
  const iAmPresident = isMePresident(state);
  const iAmChancellor = isMeChancellor(state);

  const call = async <T,>(fn: () => Promise<T>) => {
    try {
      await fn();
    } catch (e) {
      onErr(e);
    }
  };

  return (
    <div className="stack">
      <div className="row" style={{ alignItems: "baseline" }}>
        <h1>Replicant</h1>
        <div style={{ flex: "0 0 auto", textAlign: "right" }}>
          <div>
            Code: <strong>{g.joinCode}</strong>
          </div>
          <button className="link" onClick={onLeave}>
            Leave game
          </button>
        </div>
      </div>

      <div className="muted">
        {route.role === "board" ? "Board device" : "Player device"} · Round{" "}
        {g.round} · {formatPhase(g.phase)}
      </div>

      {err && <div className="error">{err}</div>}

      <BoardSummary state={state} />

      <Seats state={state} seated={seated} pres={pres} chan={chan} />

      {route.role === "board" && g.status === "lobby" && (
        <div>
          <button className="primary" onClick={() => call(() => api.startGame(g.gameId))}>
            Start game
          </button>
        </div>
      )}
      {route.role === "board" && g.status === "in_progress" && (
        <div>
          <button onClick={() => call(() => api.forceProgress(g.gameId))}>
            Force progress
          </button>
        </div>
      )}

      {route.role === "player" && (
        <PlayerControls
          state={state}
          api={api}
          iAmPresident={iAmPresident}
          iAmChancellor={iAmChancellor}
          onErr={onErr}
        />
      )}

      {state.me.role && <Secret state={state} />}

      {g.status === "completed" && (
        <div className="banner">
          {g.winner === "human" ? "Humans prevail" : "AI prevails"}
          <div className="muted" style={{ fontWeight: 400, fontSize: "0.9rem", marginTop: 6 }}>
            {winConditionLabel(g.winCondition ?? "")}
          </div>
        </div>
      )}
    </div>
  );
}

function BoardSummary({ state }: { state: ReturnType<typeof reducer> }) {
  return (
    <div className="stack" style={{ gap: 6 }}>
      <div className="meters">
        <span className="meter">Humans {state.humans}/5</span>
        <span className="meter">AI {state.ai}/6</span>
        <span className="meter">Tracker {state.tracker}/3</span>
        <span className="meter">
          Veto {state.game?.vetoUnlocked ? "unlocked" : "locked"}
        </span>
      </div>
      <PowerTrack state={state} />
    </div>
  );
}

function PowerTrack({ state }: { state: ReturnType<typeof reducer> }) {
  const playerCount = state.game?.playerCount ?? 0;
  const enacted = state.ai;
  if (playerCount < 5) return null;
  return (
    <div className="powertrack">
      {[1, 2, 3, 4, 5].map((n) => {
        const power = powerAt(playerCount, n);
        const done = n <= enacted;
        const upcoming = n === enacted + 1;
        return (
          <div
            key={n}
            className={`slot${done ? " done" : ""}${upcoming ? " upcoming" : ""}`}
          >
            <div>AI #{n}</div>
            <div>{powerLabel(power)}</div>
          </div>
        );
      })}
      <div className={`slot win${enacted >= 6 ? " done" : ""}${enacted === 5 ? " upcoming" : ""}`}>
        <div>AI #6</div>
        <div>AI wins</div>
      </div>
    </div>
  );
}

function Seats({
  state,
  seated,
  pres,
  chan,
}: {
  state: ReturnType<typeof reducer>;
  seated: ReturnType<typeof seatedPlayers>;
  pres: ReturnType<typeof currentPresident>;
  chan: ReturnType<typeof currentChancellor>;
}) {
  return (
    <div className="seats">
      {seated.map((p) => {
        const isMe = p.playerId === state.me.playerId;
        const isPres = pres?.playerId === p.playerId;
        const isChan = chan?.playerId === p.playerId;
        return (
          <div key={p.playerId} className={`seat${isMe ? " me" : ""}`}>
            <span className="muted">#{p.seat + 1}</span>
            <span style={{ flex: 1 }}>
              {p.displayName}
              {isMe ? " (you)" : ""}
            </span>
            {isPres && <span className="tag pres">President</span>}
            {isChan && <span className="tag chan">Chancellor</span>}
            {state.votesCast[p.playerId] && !isPres && !isChan && (
              <span className="tag">Voted</span>
            )}
            {!p.isAlive && <span className="tag dead">Executed</span>}
          </div>
        );
      })}
    </div>
  );
}

function PlayerControls({
  state,
  api,
  iAmPresident,
  iAmChancellor,
  onErr,
}: {
  state: ReturnType<typeof reducer>;
  api: ApiClient;
  iAmPresident: boolean;
  iAmChancellor: boolean;
  onErr: (e: unknown) => void;
}) {
  const g: Game | null = state.game;
  if (!g) return null;
  const call = async (fn: () => Promise<unknown>) => {
    try {
      await fn();
    } catch (e) {
      onErr(e);
    }
  };
  const vote = (c: VoteChoice) => call(() => api.castVote(g.gameId, c));

  switch (g.phase) {
    case "nomination":
      if (!iAmPresident)
        return <div className="muted">Waiting for President to nominate.</div>;
      return (
        <div className="callout">
          <div>
            <strong>You are President.</strong> Nominate a Chancellor:
          </div>
          <div className="choices">
            {seatedPlayers(state)
              .filter((p) => p.isAlive && p.playerId !== state.me.playerId)
              .map((p) => (
                <button
                  key={p.playerId}
                  onClick={() => call(() => api.nominateChancellor(g.gameId, p.playerId))}
                >
                  {p.displayName}
                </button>
              ))}
          </div>
        </div>
      );

    case "election":
      if (state.votesCast[state.me.playerId ?? ""])
        return <div className="muted">Vote recorded. Waiting on the rest.</div>;
      return (
        <div className="callout">
          <div><strong>Vote on the government:</strong></div>
          <div className="choices">
            <button onClick={() => vote("ja")}>Ja</button>
            <button onClick={() => vote("nein")}>Nein</button>
          </div>
        </div>
      );

    case "legislative_president":
      if (!iAmPresident || !state.me.drawnPolicies)
        return <div className="muted">President is choosing which policy to discard.</div>;
      return (
        <div className="callout">
          <div><strong>Discard one policy.</strong> The other two go to the Chancellor.</div>
          <div className="choices">
            {state.me.drawnPolicies.map((p, i) => (
              <button
                key={i}
                onClick={() => call(() => api.presidentDiscard(g.gameId, i))}
              >
                Discard {p.toUpperCase()}
              </button>
            ))}
          </div>
        </div>
      );

    case "legislative_chancellor":
      if (!iAmChancellor || !state.me.chancellorOptions)
        return <div className="muted">Chancellor is enacting a policy.</div>;
      return (
        <div className="callout">
          <div><strong>Enact one policy.</strong></div>
          <div className="choices">
            {state.me.chancellorOptions.map((p, i) => (
              <button
                key={i}
                onClick={() => call(() => api.chancellorEnact(g.gameId, i))}
              >
                Enact {p.toUpperCase()}
              </button>
            ))}
            {g.vetoUnlocked && (
              <button onClick={() => call(() => api.proposeVeto(g.gameId))}>
                Propose veto
              </button>
            )}
          </div>
        </div>
      );

    case "veto_requested":
      if (!iAmPresident)
        return <div className="muted">Chancellor proposed a veto. Waiting on President.</div>;
      return (
        <div className="callout">
          <div><strong>Accept the veto?</strong></div>
          <div className="choices">
            <button onClick={() => call(() => api.resolveVeto(g.gameId, true))}>Accept</button>
            <button onClick={() => call(() => api.resolveVeto(g.gameId, false))}>Reject</button>
          </div>
        </div>
      );

    case "executive_action":
      if (!iAmPresident)
        return <div className="muted">President is using a power.</div>;
      return (
        <div className="callout">
          <div>
            <strong>Power: {formatPower(g.pendingActionType ?? "")}.</strong> Pick a target:
          </div>
          <div className="choices">
            {seatedPlayers(state)
              .filter((p) => p.isAlive && p.playerId !== state.me.playerId)
              .map((p) => (
                <button
                  key={p.playerId}
                  onClick={() => call(() => api.executeAction(g.gameId, p.playerId))}
                >
                  {p.displayName}
                </button>
              ))}
          </div>
        </div>
      );

    default:
      return null;
  }
}

function Secret({ state }: { state: ReturnType<typeof reducer> }) {
  const { me } = state;
  const lbl: React.CSSProperties = { fontSize: "0.75rem", textTransform: "uppercase", color: "#666" };
  return (
    <div className="secret stack">
      <div>
        <div style={lbl}>Your role (private)</div>
        <div>
          <strong>{me.role?.toUpperCase()}</strong> · party: {me.party}
        </div>
      </div>
      {me.teammates.length > 0 && (
        <div>
          <div style={lbl}>Teammates</div>
          <div>{me.teammates.map((t) => `${t.displayName} (${t.role})`).join(", ")}</div>
        </div>
      )}
      {me.drawnPolicies && (
        <div>
          <div style={lbl}>Drawn policies</div>
          <div>{me.drawnPolicies.join(", ")}</div>
        </div>
      )}
      {me.chancellorOptions && (
        <div>
          <div style={lbl}>Remaining options</div>
          <div>{me.chancellorOptions.join(", ")}</div>
        </div>
      )}
      {me.peekedPolicies && (
        <div>
          <div style={lbl}>Top of deck</div>
          <div>{me.peekedPolicies.join(", ")}</div>
        </div>
      )}
      {me.lastInvestigation && (
        <div>
          <div style={lbl}>Investigation</div>
          <div>
            {me.lastInvestigation.targetPlayerId}:{" "}
            <strong>{me.lastInvestigation.party}</strong>
          </div>
        </div>
      )}
    </div>
  );
}

// ─── helpers ─────────────────────────────────────────────────────

function formatPhase(p: string): string {
  switch (p) {
    case "lobby":
      return "Awaiting players";
    case "nomination":
      return "Nomination";
    case "election":
      return "Election";
    case "legislative_president":
      return "President chooses";
    case "legislative_chancellor":
      return "Chancellor chooses";
    case "veto_requested":
      return "Veto proposed";
    case "executive_action":
      return "Executive power";
    case "game_over":
      return "Game over";
    default:
      return p;
  }
}

function formatPower(p: string): string {
  switch (p) {
    case "investigate_loyalty":
      return "Investigate Loyalty";
    case "special_election":
      return "Special Election";
    case "policy_peek":
      return "Policy Peek";
    case "execution":
      return "Execution";
    default:
      return p || "—";
  }
}

function winConditionLabel(c: string): string {
  switch (c) {
    case "human_policies":
      return "Five human policies passed";
    case "ai_policies":
      return "Six AI policies passed";
    case "rogue_elected_chancellor":
      return "The rogue AI was elected Chancellor with three AI policies on the board";
    case "rogue_executed":
      return "The rogue AI was executed by a presidential power";
    default:
      return c;
  }
}

// Mirrors schemas/replicant/powers.go:PowerFor (schedule is fixed).
function powerAt(
  playerCount: number,
  slot: number,
): "" | "investigate_loyalty" | "special_election" | "policy_peek" | "execution" {
  if (slot < 1 || slot > 5) return "";
  const table9_10 = [
    "investigate_loyalty",
    "investigate_loyalty",
    "special_election",
    "execution",
    "execution",
  ] as const;
  const table7_8 = ["", "investigate_loyalty", "special_election", "execution", "execution"] as const;
  const table5_6 = ["", "", "policy_peek", "execution", "execution"] as const;
  if (playerCount >= 9) return table9_10[slot - 1];
  if (playerCount >= 7) return table7_8[slot - 1];
  if (playerCount >= 5) return table5_6[slot - 1];
  return "";
}

function powerLabel(p: string): string {
  switch (p) {
    case "investigate_loyalty":
      return "Investigate";
    case "special_election":
      return "Special Election";
    case "policy_peek":
      return "Policy Peek";
    case "execution":
      return "Execution";
    default:
      return "—";
  }
}

function errMsg(e: unknown): string {
  if (e instanceof ApiError) return `${e.status}: ${e.message}`;
  if (e instanceof Error) return e.message;
  return String(e);
}
