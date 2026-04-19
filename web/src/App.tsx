// Dead-simple React shell: one route per device role, a single
// reducer-backed state container, and raw REST + SSE plumbing. Good
// enough to demo the full game loop in a browser; style it later.

import { useCallback, useEffect, useMemo, useReducer, useState } from "react";
import { ApiClient, ApiError } from "./api";
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

// Backend origin. Override with VITE_API_URL at build time for prod.
const BASE_URL: string =
  (import.meta.env.VITE_API_URL as string | undefined) ?? "http://localhost:8080";

type Route =
  | { kind: "login" }
  | { kind: "lobby"; role: "board" | "player" }
  | { kind: "game"; role: "board" | "player"; gameId: string };

export default function App() {
  const [token, setToken] = useState<string>("");
  const [route, setRoute] = useState<Route>({ kind: "login" });
  const [state, dispatch] = useReducer(reducer, initialState);
  const [err, setErr] = useState<string>("");

  const api = useMemo(
    () => (token ? new ApiClient({ baseUrl: BASE_URL, token }) : null),
    [token],
  );

  // Open SSE once we have a gameId + token.
  useEffect(() => {
    if (route.kind !== "game" || !token) return;
    const handle = streamEnvelopes({
      baseUrl: BASE_URL,
      gameId: route.gameId,
      role: route.role,
      token,
      onEnvelope: (envelope) => dispatch({ type: "envelope", envelope }),
      onError: (e) => console.warn("stream error", e),
    });
    return () => handle.close();
  }, [route, token]);

  const handleLogin = (t: string) => {
    setToken(t);
    setRoute({ kind: "lobby", role: "player" });
  };

  const handleCreate = useCallback(
    async (joinCode: string, displayName: string) => {
      if (!api) return;
      try {
        const { game, player } = await api.createGame(joinCode, displayName);
        dispatch({
          type: "snapshot",
          game,
          players: [player],
          mePlayerId: player.playerId,
        });
        setRoute({ kind: "game", role: "board", gameId: game.gameId });
      } catch (e) {
        setErr(errMsg(e));
      }
    },
    [api],
  );

  const handleJoin = useCallback(
    async (joinCode: string, displayName: string) => {
      if (!api) return;
      try {
        const { game, player } = await api.joinGame(joinCode, displayName);
        dispatch({
          type: "snapshot",
          game,
          players: [player],
          mePlayerId: player.playerId,
        });
        setRoute({ kind: "game", role: "player", gameId: game.gameId });
      } catch (e) {
        setErr(errMsg(e));
      }
    },
    [api],
  );

  // ─── render ─────────────────────────────────────────────────────

  return (
    <div style={{ fontFamily: "system-ui, sans-serif", padding: 16 }}>
      <header style={{ marginBottom: 16, display: "flex", justifyContent: "space-between" }}>
        <h1 style={{ margin: 0, fontSize: 18 }}>Secret Hitler</h1>
        {state.game && <code>{state.game.joinCode}</code>}
      </header>

      {err && <div style={{ color: "crimson", marginBottom: 12 }}>Error: {err}</div>}

      {route.kind === "login" && <Login onLogin={handleLogin} />}

      {route.kind === "lobby" && api && (
        <Lobby onCreate={handleCreate} onJoin={handleJoin} />
      )}

      {route.kind === "game" && api && (
        <GameView
          route={route}
          state={state}
          api={api}
          onErr={(e) => setErr(errMsg(e))}
        />
      )}
    </div>
  );
}

// ─── login (dev-only: paste a token) ──────────────────────────────

function Login({ onLogin }: { onLogin: (token: string) => void }) {
  const [val, setVal] = useState("");
  return (
    <form
      onSubmit={(e) => {
        e.preventDefault();
        if (val.trim()) onLogin(val.trim());
      }}
    >
      <label>
        Token (Clerk JWT, or any string with <code>NopAuth</code> in dev)
        <br />
        <input
          value={val}
          onChange={(e) => setVal(e.target.value)}
          placeholder="user-alice"
          style={{ width: 320, padding: 6 }}
        />
      </label>
      <button type="submit" style={{ marginLeft: 8 }}>Continue</button>
    </form>
  );
}

// ─── lobby (create/join) ──────────────────────────────────────────

function Lobby({
  onCreate,
  onJoin,
}: {
  onCreate: (joinCode: string, displayName: string) => void;
  onJoin: (joinCode: string, displayName: string) => void;
}) {
  const [joinCode, setJoinCode] = useState("");
  const [name, setName] = useState("");
  return (
    <div>
      <h2>Lobby</h2>
      <div style={{ display: "flex", gap: 8, marginBottom: 8 }}>
        <input
          placeholder="Join code"
          value={joinCode}
          onChange={(e) => setJoinCode(e.target.value.toUpperCase())}
        />
        <input
          placeholder="Display name"
          value={name}
          onChange={(e) => setName(e.target.value)}
        />
      </div>
      <button disabled={!joinCode || !name} onClick={() => onCreate(joinCode, name)}>
        Host new game (board device)
      </button>{" "}
      <button disabled={!joinCode || !name} onClick={() => onJoin(joinCode, name)}>
        Join as player
      </button>
    </div>
  );
}

// ─── game view (board or player) ──────────────────────────────────

function GameView({
  route,
  state,
  api,
  onErr,
}: {
  route: Extract<Route, { kind: "game" }>;
  state: ReturnType<typeof reducer>;
  api: ApiClient;
  onErr: (e: unknown) => void;
}) {
  const g = state.game;
  if (!g) return <p>Loading…</p>;

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
    <div>
      <h2>
        Round {g.round} · {g.phase}
      </h2>
      <BoardSummary state={state} />

      <h3>Seats</h3>
      <ul>
        {seated.map((p) => (
          <li key={p.playerId}>
            #{p.seat} {p.displayName}
            {pres?.playerId === p.playerId ? " · president" : ""}
            {chan?.playerId === p.playerId ? " · chancellor" : ""}
            {!p.isAlive ? " · ✖ executed" : ""}
            {state.votesCast[p.playerId] ? " · voted" : ""}
          </li>
        ))}
      </ul>

      {/* Host-only controls appear on the board device. */}
      {route.role === "board" && g.status === "lobby" && (
        <button onClick={() => call(() => api.startGame(g.gameId))}>Start game</button>
      )}
      {route.role === "board" && g.status === "in_progress" && (
        <button onClick={() => call(() => api.forceProgress(g.gameId))}>
          Force progress
        </button>
      )}

      {/* Player controls: only show the one the current phase permits. */}
      {route.role === "player" && (
        <PlayerControls
          state={state}
          api={api}
          iAmPresident={iAmPresident}
          iAmChancellor={iAmChancellor}
          onErr={onErr}
        />
      )}

      {state.me.role && (
        <Secret state={state} />
      )}

      {g.status === "completed" && (
        <h2 style={{ color: g.winner === "liberal" ? "dodgerblue" : "crimson" }}>
          {g.winner} wins ({g.winCondition})
        </h2>
      )}
    </div>
  );
}

function BoardSummary({ state }: { state: ReturnType<typeof reducer> }) {
  return (
    <div style={{ display: "flex", gap: 24 }}>
      <span>Liberal: {state.liberal} / 5</span>
      <span>Fascist: {state.fascist} / 6</span>
      <span>Tracker: {state.tracker} / 3</span>
      <span>Veto: {state.game?.vetoUnlocked ? "unlocked" : "locked"}</span>
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
      if (!iAmPresident) return <p>Waiting for the president to nominate…</p>;
      return (
        <div>
          <strong>You are President. Pick a Chancellor:</strong>
          <ul>
            {seatedPlayers(state)
              .filter((p) => p.isAlive && p.playerId !== state.me.playerId)
              .map((p) => (
                <li key={p.playerId}>
                  {p.displayName}{" "}
                  <button
                    onClick={() =>
                      call(() => api.nominateChancellor(g.gameId, p.playerId))
                    }
                  >
                    Nominate
                  </button>
                </li>
              ))}
          </ul>
        </div>
      );

    case "election":
      if (state.votesCast[state.me.playerId ?? ""]) return <p>Vote cast. Waiting…</p>;
      return (
        <div>
          <strong>Vote:</strong>{" "}
          <button onClick={() => vote("ja")}>Ja</button>{" "}
          <button onClick={() => vote("nein")}>Nein</button>
        </div>
      );

    case "legislative_president":
      if (!iAmPresident || !state.me.drawnPolicies) return <p>President is picking…</p>;
      return (
        <div>
          <strong>Discard one policy:</strong>
          <ul>
            {state.me.drawnPolicies.map((p, i) => (
              <li key={i}>
                {p}{" "}
                <button onClick={() => call(() => api.presidentDiscard(g.gameId, i))}>
                  Discard
                </button>
              </li>
            ))}
          </ul>
        </div>
      );

    case "legislative_chancellor":
      if (!iAmChancellor || !state.me.chancellorOptions) return <p>Chancellor is picking…</p>;
      return (
        <div>
          <strong>Enact one policy:</strong>
          <ul>
            {state.me.chancellorOptions.map((p, i) => (
              <li key={i}>
                {p}{" "}
                <button onClick={() => call(() => api.chancellorEnact(g.gameId, i))}>
                  Enact
                </button>
              </li>
            ))}
          </ul>
          {g.vetoUnlocked && (
            <button onClick={() => call(() => api.proposeVeto(g.gameId))}>Propose veto</button>
          )}
        </div>
      );

    case "veto_requested":
      if (!iAmPresident) return <p>Chancellor proposed a veto. Waiting on president…</p>;
      return (
        <div>
          <strong>Accept veto?</strong>{" "}
          <button onClick={() => call(() => api.resolveVeto(g.gameId, true))}>Accept</button>{" "}
          <button onClick={() => call(() => api.resolveVeto(g.gameId, false))}>Reject</button>
        </div>
      );

    case "executive_action":
      if (!iAmPresident) return <p>President is using their power…</p>;
      return (
        <div>
          <strong>Pick a target for {g.pendingActionType}:</strong>
          <ul>
            {seatedPlayers(state)
              .filter((p) => p.isAlive && p.playerId !== state.me.playerId)
              .map((p) => (
                <li key={p.playerId}>
                  {p.displayName}{" "}
                  <button onClick={() => call(() => api.executeAction(g.gameId, p.playerId))}>
                    Target
                  </button>
                </li>
              ))}
          </ul>
        </div>
      );

    default:
      return null;
  }
}

// Private panel: only the player who "owns" this device sees it.
function Secret({ state }: { state: ReturnType<typeof reducer> }) {
  const { me } = state;
  return (
    <aside style={{ marginTop: 24, padding: 12, background: "#f4f4f4", borderRadius: 8 }}>
      <h3 style={{ marginTop: 0 }}>Your secret</h3>
      <div>
        Role: <strong>{me.role}</strong> · Party: {me.party}
      </div>
      {me.teammates.length > 0 && (
        <div>
          Cabal: {me.teammates.map((t) => `${t.displayName} (${t.role})`).join(", ")}
        </div>
      )}
      {me.drawnPolicies && (
        <div>Drawn policies: {me.drawnPolicies.join(", ")}</div>
      )}
      {me.chancellorOptions && (
        <div>Remaining options: {me.chancellorOptions.join(", ")}</div>
      )}
      {me.peekedPolicies && (
        <div>Top of deck: {me.peekedPolicies.join(", ")}</div>
      )}
      {me.lastInvestigation && (
        <div>
          Investigation → {me.lastInvestigation.targetPlayerId}: {me.lastInvestigation.party}
        </div>
      )}
    </aside>
  );
}

// ─── helpers ─────────────────────────────────────────────────────

function errMsg(e: unknown): string {
  if (e instanceof ApiError) return `${e.status}: ${e.message}`;
  if (e instanceof Error) return e.message;
  return String(e);
}
