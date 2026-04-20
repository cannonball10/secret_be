// RulesHelper — bottom-sheet chat with the Committee's rules bot.
// Shows a few canned starter prompts on open, free-text compose, and
// a scrolling Q&A log. Non-streaming; one submit = one answer block.

"use client";

import { useEffect, useRef, useState } from "react";
import { rpColors } from "@replicant/tokens";
import { MobileApi } from "@/lib/api";

interface QA {
  id: number;
  q: string;
  a: string; // empty while pending
  pending: boolean;
  error?: string;
}

export function RulesHelper({
  api,
  gameId,
  onClose,
}: {
  api: MobileApi;
  gameId: string;
  onClose: () => void;
}) {
  const [faq, setFaq] = useState<string[]>([]);
  const [log, setLog] = useState<QA[]>([]);
  const [draft, setDraft] = useState("");
  const nextId = useRef(1);
  const scrollRef = useRef<HTMLDivElement | null>(null);

  // Fetch FAQ on open so the list is personalised to the player's
  // role + current game's feature flags.
  useEffect(() => {
    let cancelled = false;
    api
      .faq(gameId)
      .then((r) => {
        if (!cancelled) setFaq(r.faq ?? []);
      })
      .catch(() => {
        /* non-fatal */
      });
    return () => {
      cancelled = true;
    };
  }, [api, gameId]);

  // Pin to the newest entry on log growth.
  useEffect(() => {
    const el = scrollRef.current;
    if (el) el.scrollTop = el.scrollHeight;
  }, [log.length]);

  const ask = async (q: string) => {
    const body = q.trim();
    if (!body) return;
    const id = nextId.current++;
    setLog((prev) => [...prev, { id, q: body, a: "", pending: true }]);
    setDraft("");
    try {
      const res = await api.ask(gameId, body);
      setLog((prev) =>
        prev.map((row) => (row.id === id ? { ...row, a: res.answer, pending: false } : row)),
      );
    } catch (e) {
      setLog((prev) =>
        prev.map((row) =>
          row.id === id
            ? { ...row, a: "", pending: false, error: e instanceof Error ? e.message : String(e) }
            : row,
        ),
      );
    }
  };

  return (
    <div
      onClick={onClose}
      style={{
        position: "fixed",
        inset: 0,
        background: "rgba(6, 8, 10, 0.62)",
        zIndex: 55,
        display: "flex",
        alignItems: "flex-end",
      }}
    >
      <div
        onClick={(e) => e.stopPropagation()}
        style={{
          background: rpColors.paper3,
          borderTop: `2px solid ${rpColors.stampBlue}`,
          width: "100%",
          maxHeight: "86vh",
          display: "flex",
          flexDirection: "column",
        }}
      >
        <div
          style={{
            display: "flex",
            justifyContent: "space-between",
            alignItems: "center",
            padding: "12px 18px",
            borderBottom: `1px solid ${rpColors.paperLine}`,
          }}
        >
          <div className="t-eyebrow" style={{ color: rpColors.stampBlue, fontSize: 11 }}>
            ◼ COMMITTEE HELP DESK
          </div>
          <button
            onClick={onClose}
            aria-label="Close rules helper"
            style={{
              background: "transparent",
              border: "none",
              color: rpColors.ink,
              fontSize: 22,
              cursor: "pointer",
            }}
          >
            ×
          </button>
        </div>

        <div ref={scrollRef} style={{ overflowY: "auto", padding: "12px 18px" }}>
          {log.length === 0 && (
            <div
              className="t-memo"
              style={{ fontSize: 13, color: rpColors.inkSoft, lineHeight: 1.45, marginBottom: 10 }}
            >
              &quot;The Committee fields enquiries. Ask anything about the
              rules, your role, or what the board is doing. I won&apos;t
              speculate about other delegates.&quot;
            </div>
          )}

          {faq.length > 0 && log.length === 0 && (
            <div style={{ display: "flex", flexDirection: "column", gap: 6, marginTop: 8 }}>
              <div
                className="t-eyebrow"
                style={{ color: rpColors.inkFaded, fontSize: 9, marginBottom: 2 }}
              >
                COMMON QUESTIONS
              </div>
              {faq.map((q) => (
                <button
                  key={q}
                  onClick={() => ask(q)}
                  style={{
                    textAlign: "left",
                    background: rpColors.paper2,
                    border: `1px solid ${rpColors.paperLine}`,
                    borderLeft: `3px solid ${rpColors.stampBlue}`,
                    color: rpColors.ink,
                    padding: "8px 12px",
                    fontFamily: "var(--font-typewriter)",
                    fontSize: 13,
                    lineHeight: 1.35,
                    cursor: "pointer",
                  }}
                >
                  {q}
                </button>
              ))}
            </div>
          )}

          {log.map((row) => (
            <div
              key={row.id}
              style={{ display: "flex", flexDirection: "column", gap: 6, marginBottom: 14 }}
            >
              <div
                style={{
                  alignSelf: "flex-end",
                  maxWidth: "82%",
                  padding: "6px 10px",
                  background: rpColors.stampBlue,
                  color: rpColors.paper3,
                  fontFamily: "var(--font-typewriter)",
                  fontSize: 13,
                  lineHeight: 1.4,
                }}
              >
                {row.q}
              </div>
              {row.pending ? (
                <div
                  style={{
                    alignSelf: "flex-start",
                    fontFamily: "var(--font-mono)",
                    fontSize: 11,
                    color: rpColors.inkFaded,
                    letterSpacing: 1.3,
                    padding: "4px 8px",
                  }}
                >
                  … the Committee is consulting the manual
                </div>
              ) : row.error ? (
                <div
                  style={{
                    alignSelf: "flex-start",
                    fontFamily: "var(--font-mono)",
                    fontSize: 11,
                    color: rpColors.stampRed,
                    padding: "4px 8px",
                  }}
                >
                  Help desk is unavailable: {row.error}
                </div>
              ) : (
                <div
                  style={{
                    alignSelf: "flex-start",
                    maxWidth: "90%",
                    padding: "8px 12px",
                    background: rpColors.paper2,
                    border: `1px solid ${rpColors.paperLine}`,
                    borderLeft: `3px solid ${rpColors.ink}`,
                    color: rpColors.ink,
                    fontFamily: "var(--font-typewriter)",
                    fontSize: 13,
                    lineHeight: 1.45,
                    whiteSpace: "pre-wrap",
                  }}
                >
                  {row.a}
                </div>
              )}
            </div>
          ))}
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
            placeholder="Ask about the rules…"
            onKeyDown={(e) => {
              if (e.key === "Enter" && !e.shiftKey) {
                e.preventDefault();
                void ask(draft);
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
            onClick={() => void ask(draft)}
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
            Ask
          </button>
        </div>
      </div>
    </div>
  );
}
