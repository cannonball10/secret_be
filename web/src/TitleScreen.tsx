// TitleScreen — plain wireframe landing page. A token prompt plus
// Host/Join. Mode toggle (choose → join) happens in place.

import { useState } from "react";

type Mode = "choose" | "join";

interface Props {
  onHost: (token: string) => void | Promise<void>;
  onJoin: (token: string, joinCode: string, displayName: string) => void | Promise<void>;
  error?: string;
  initialToken?: string;
}

export function TitleScreen({ onHost, onJoin, error, initialToken }: Props) {
  const [mode, setMode] = useState<Mode>("choose");
  const [token, setToken] = useState(initialToken ?? "");
  const [joinCode, setJoinCode] = useState("");
  const [displayName, setDisplayName] = useState("");
  const [busy, setBusy] = useState(false);

  const run = async (fn: () => Promise<void>) => {
    setBusy(true);
    try {
      await fn();
    } finally {
      setBusy(false);
    }
  };

  const canHost = !!token.trim() && !busy;
  const canJoin =
    !!token.trim() && joinCode.trim().length >= 4 && !!displayName.trim() && !busy;

  return (
    <div className="title stack">
      <h1>Replicant</h1>
      <div className="muted">Humans vs AI · find the replicants</div>

      {error && <div className="error">{error}</div>}

      <div className="stack">
        <label>
          Name
          <input
            value={token}
            onChange={(e) => setToken(e.target.value)}
            placeholder="your name"
            autoComplete="off"
            autoCapitalize="off"
            spellCheck={false}
            maxLength={64}
          />
        </label>

        {mode === "choose" && (
          <div className="row">
            <button
              className="primary"
              disabled={!canHost}
              onClick={() => run(async () => onHost(token.trim()))}
            >
              Host
            </button>
            <button
              disabled={!token.trim() || busy}
              onClick={() => setMode("join")}
            >
              Join
            </button>
          </div>
        )}

        {mode === "join" && (
          <>
            <label>
              Join code
              <input
                value={joinCode}
                onChange={(e) =>
                  setJoinCode(e.target.value.toUpperCase().replace(/\s/g, ""))
                }
                maxLength={6}
                autoCapitalize="characters"
                autoComplete="off"
                spellCheck={false}
                placeholder="XXXXXX"
              />
            </label>
            <label>
              Display name
              <input
                value={displayName}
                onChange={(e) => setDisplayName(e.target.value)}
                maxLength={32}
                autoComplete="off"
                placeholder="how you show up in seats"
              />
            </label>
            <div className="row">
              <button onClick={() => setMode("choose")} disabled={busy}>
                Back
              </button>
              <button
                className="primary"
                disabled={!canJoin}
                onClick={() =>
                  run(async () =>
                    onJoin(token.trim(), joinCode.trim(), displayName.trim()),
                  )
                }
              >
                Enter
              </button>
            </div>
          </>
        )}
      </div>
    </div>
  );
}
