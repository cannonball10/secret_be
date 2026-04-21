// DMDrawer — open-channel direct messages between any two delegates.
//
// Replaces the old ChatDrawer (AI-cabal-only). Two-state bottom sheet:
// peer picker (scrollable list of alive delegates, unread badge per
// peer) → selected thread (messages + composer). Back button returns
// to the picker.
//
// Messages arrive via the parent's SSE handler; this component only
// renders and emits sends. The parent partitions messages by peer.

"use client";

import { useEffect, useMemo, useRef, useState } from "react";
import type { ChatMessagePayload, Player } from "@replicant/schema";
import { rpColors } from "@replicant/tokens";

export interface DMDrawerProps {
  meId: string;
  players: Record<string, Player>;
  /** Keyed by peer playerId. Values are conversation history with that peer. */
  threads: Record<string, ChatMessagePayload[]>;
  /** Peer ids with at least one message the local device hasn't opened yet. */
  unread: Set<string>;
  /** Which thread is on screen. null = peer picker. Controlled by parent
   *  so the SSE handler can tell if the player is literally reading a
   *  thread right now (and skip marking it unread). */
  activePeer: string | null;
  onOpenThread: (peerId: string) => void;
  onBackToPicker: () => void;
  onSend: (peerId: string, body: string) => void | Promise<void>;
  onClose: () => void;
}

export function DMDrawer({
  meId,
  players,
  threads,
  unread,
  activePeer,
  onOpenThread,
  onBackToPicker,
  onSend,
  onClose,
}: DMDrawerProps) {
  // Alive peers, sorted by seat. Skip self + dead.
  const peers = useMemo(
    () =>
      Object.values(players)
        .filter((p) => p.playerId !== meId && p.isAlive)
        .sort((a, b) => a.seat - b.seat),
    [players, meId],
  );

  const openThread = (peerId: string) => {
    onOpenThread(peerId);
  };

  return (
    <div
      onClick={onClose}
      style={{
        position: "fixed",
        inset: 0,
        background: "rgba(6, 8, 10, 0.55)",
        zIndex: 40,
        display: "flex",
        alignItems: "flex-end",
      }}
    >
      <div
        onClick={(e) => e.stopPropagation()}
        style={{
          background: rpColors.paper3,
          borderTop: `2px solid ${rpColors.ink}`,
          width: "100%",
          maxHeight: "78vh",
          display: "flex",
          flexDirection: "column",
          boxShadow: "0 -6px 0 rgba(0,0,0,0.15)",
        }}
      >
        <Header
          label={activePeer ? peerLabel(players[activePeer]) : "DIRECT CHANNELS"}
          onBack={activePeer ? onBackToPicker : undefined}
          onClose={onClose}
        />

        {activePeer ? (
          <Thread
            thread={threads[activePeer] ?? []}
            meId={meId}
            onSend={(body) => onSend(activePeer, body)}
          />
        ) : (
          <PeerList
            peers={peers}
            unread={unread}
            lastMessages={threads}
            onPick={openThread}
          />
        )}
      </div>
    </div>
  );
}

// ─── header ──────────────────────────────────────────────────────

function Header({
  label,
  onBack,
  onClose,
}: {
  label: string;
  onBack?: () => void;
  onClose: () => void;
}) {
  return (
    <div
      style={{
        display: "flex",
        justifyContent: "space-between",
        alignItems: "center",
        padding: "12px 18px",
        borderBottom: `1px solid ${rpColors.paperLine}`,
      }}
    >
      <div style={{ display: "flex", alignItems: "center", gap: 10 }}>
        {onBack && (
          <button
            onClick={onBack}
            aria-label="Back to peers"
            style={{
              background: "transparent",
              border: "none",
              fontFamily: "var(--font-mono)",
              fontSize: 14,
              color: rpColors.ink,
              cursor: "pointer",
              padding: 0,
            }}
          >
            ◂
          </button>
        )}
        <div
          className="t-eyebrow"
          style={{ color: rpColors.stampRed, fontSize: 11, letterSpacing: 1.4 }}
        >
          ◼ {label}
        </div>
      </div>
      <button
        onClick={onClose}
        aria-label="Close DMs"
        style={{
          background: "transparent",
          border: "none",
          color: rpColors.ink,
          fontSize: 22,
          cursor: "pointer",
          lineHeight: 1,
        }}
      >
        ×
      </button>
    </div>
  );
}

// ─── peer list ───────────────────────────────────────────────────

function PeerList({
  peers,
  unread,
  lastMessages,
  onPick,
}: {
  peers: Player[];
  unread: Set<string>;
  lastMessages: Record<string, ChatMessagePayload[]>;
  onPick: (peerId: string) => void;
}) {
  if (peers.length === 0) {
    return (
      <div
        style={{
          padding: 24,
          fontFamily: "var(--font-mono)",
          fontSize: 12,
          color: rpColors.inkFaded,
          textAlign: "center",
        }}
      >
        No other delegates seated.
      </div>
    );
  }
  return (
    <div
      style={{
        overflowY: "auto",
        display: "flex",
        flexDirection: "column",
      }}
    >
      {peers.map((p) => {
        const thread = lastMessages[p.playerId] ?? [];
        const last = thread[thread.length - 1];
        const hasUnread = unread.has(p.playerId);
        return (
          <button
            key={p.playerId}
            onClick={() => onPick(p.playerId)}
            style={{
              textAlign: "left",
              background: hasUnread ? "rgba(179,39,36,0.06)" : "transparent",
              border: "none",
              borderBottom: `1px solid ${rpColors.paperLine}`,
              padding: "12px 18px",
              cursor: "pointer",
              display: "flex",
              gap: 12,
              alignItems: "center",
            }}
          >
            <div
              style={{
                width: 36,
                height: 36,
                flexShrink: 0,
                background: hasUnread ? rpColors.stampRed : rpColors.ink,
                color: rpColors.paper3,
                display: "flex",
                alignItems: "center",
                justifyContent: "center",
                fontFamily: "var(--font-display)",
                fontWeight: 700,
                fontSize: 16,
              }}
            >
              {p.displayName.trim().slice(0, 1).toUpperCase() || "?"}
            </div>
            <div style={{ flex: 1, minWidth: 0 }}>
              <div
                style={{
                  fontFamily: "var(--font-mono)",
                  fontSize: 9,
                  color: rpColors.inkFaded,
                  letterSpacing: 1.3,
                }}
              >
                SEAT {String(p.seat + 1).padStart(2, "0")}
                {p.countryCode && ` · ${p.countryCode}`}
              </div>
              <div
                style={{
                  fontFamily: "var(--font-display)",
                  fontWeight: 600,
                  fontSize: 15,
                  color: rpColors.ink,
                  textTransform: "uppercase",
                }}
              >
                {p.displayName}
              </div>
              {last && (
                <div
                  style={{
                    fontFamily: "var(--font-typewriter)",
                    fontSize: 12,
                    color: rpColors.inkSoft,
                    overflow: "hidden",
                    textOverflow: "ellipsis",
                    whiteSpace: "nowrap",
                    maxWidth: 240,
                  }}
                >
                  {last.body}
                </div>
              )}
            </div>
            {hasUnread && (
              <span
                style={{
                  width: 8,
                  height: 8,
                  background: rpColors.stampRed,
                  borderRadius: 999,
                  flexShrink: 0,
                }}
              />
            )}
          </button>
        );
      })}
    </div>
  );
}

// ─── thread ──────────────────────────────────────────────────────

function Thread({
  thread,
  meId,
  onSend,
}: {
  thread: ChatMessagePayload[];
  meId: string;
  onSend: (body: string) => void | Promise<void>;
}) {
  const [draft, setDraft] = useState("");
  const scrollRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    const el = scrollRef.current;
    if (el) el.scrollTop = el.scrollHeight;
  }, [thread.length]);

  const send = () => {
    const body = draft.trim();
    if (!body) return;
    setDraft("");
    void onSend(body);
  };

  return (
    <>
      <div
        ref={scrollRef}
        style={{
          flex: 1,
          overflowY: "auto",
          padding: "14px 18px",
          display: "flex",
          flexDirection: "column",
          gap: 8,
        }}
      >
        {thread.length === 0 ? (
          <div
            style={{
              fontFamily: "var(--font-mono)",
              fontSize: 11,
              color: rpColors.inkFaded,
              textAlign: "center",
              padding: 30,
            }}
          >
            No prior correspondence.
          </div>
        ) : (
          thread.map((m) => {
            const mine = m.authorPlayerId === meId;
            return (
              <div
                key={m.messageId}
                style={{
                  alignSelf: mine ? "flex-end" : "flex-start",
                  maxWidth: "80%",
                  padding: "8px 12px",
                  background: mine ? rpColors.stampBlue : rpColors.paper2,
                  color: mine ? rpColors.paper3 : rpColors.ink,
                  border: `1px solid ${mine ? rpColors.stampBlue : rpColors.paperLine}`,
                  fontFamily: "var(--font-typewriter)",
                  fontSize: 14,
                  lineHeight: 1.4,
                  boxShadow: mine ? "2px 2px 0 rgba(28,26,21,0.2)" : "none",
                }}
              >
                {m.body}
              </div>
            );
          })
        )}
      </div>
      <div
        style={{
          display: "flex",
          gap: 8,
          padding: "10px 14px",
          borderTop: `1px solid ${rpColors.paperLine}`,
          background: rpColors.paper2,
        }}
      >
        <input
          value={draft}
          onChange={(e) => setDraft(e.target.value)}
          placeholder="Compose a message…"
          onKeyDown={(e) => {
            if (e.key === "Enter" && !e.shiftKey) {
              e.preventDefault();
              send();
            }
          }}
          style={{
            flex: 1,
            padding: "8px 10px",
            border: `1px solid ${rpColors.ink}`,
            background: rpColors.paper3,
            fontFamily: "var(--font-typewriter)",
            fontSize: 14,
            color: rpColors.ink,
            outline: "none",
          }}
        />
        <button
          onClick={send}
          disabled={!draft.trim()}
          style={{
            background: rpColors.stampBlue,
            color: rpColors.paper3,
            border: `2px solid ${rpColors.stampBlue}`,
            padding: "6px 12px",
            fontFamily: "var(--font-display)",
            fontWeight: 700,
            fontSize: 12,
            letterSpacing: "0.12em",
            textTransform: "uppercase",
            cursor: draft.trim() ? "pointer" : "not-allowed",
            opacity: draft.trim() ? 1 : 0.4,
          }}
        >
          Send
        </button>
      </div>
    </>
  );
}

function peerLabel(p: Player | undefined): string {
  if (!p) return "DELEGATE";
  if (p.countryName) return p.countryName.toUpperCase();
  return p.displayName.toUpperCase();
}
