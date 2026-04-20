// Chat drawer — AI faction private channel.
//
// Fixed bottom-sheet overlay covering the lower 70% of the viewport.
// Messages scroll; composer stays pinned to the bottom. Received
// messages are pushed into the parent's state from the SSE handler;
// this component only renders + emits sends.

"use client";

import { useEffect, useRef, useState } from "react";
import type { ChatMessagePayload } from "@replicant/schema";
import { rpColors } from "@replicant/tokens";

export function ChatDrawer({
  messages,
  meId,
  onClose,
  onSend,
}: {
  messages: ChatMessagePayload[];
  meId: string;
  onClose: () => void;
  onSend: (body: string) => void | Promise<void>;
}) {
  const [draft, setDraft] = useState("");
  const scrollRef = useRef<HTMLDivElement | null>(null);

  // Pin to the newest message whenever the list grows.
  useEffect(() => {
    const el = scrollRef.current;
    if (el) el.scrollTop = el.scrollHeight;
  }, [messages.length]);

  const send = () => {
    const body = draft.trim();
    if (!body) return;
    setDraft("");
    void onSend(body);
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
        justifyContent: "stretch",
      }}
    >
      <div
        onClick={(e) => e.stopPropagation()}
        className="paper-tex"
        style={{
          width: "100%",
          height: "72vh",
          maxHeight: "720px",
          background: rpColors.paper,
          borderTop: `3px solid ${rpColors.ink}`,
          display: "flex",
          flexDirection: "column",
          boxShadow: "0 -12px 40px rgba(0,0,0,0.5)",
          animation: "paperSlide 0.35s cubic-bezier(.2,.8,.2,1) both",
        }}
      >
        {/* banner */}
        <div
          style={{
            background: rpColors.stampRed,
            color: rpColors.paper3,
            padding: "8px 16px",
            display: "flex",
            justifyContent: "space-between",
            alignItems: "center",
            fontFamily: "var(--font-mono)",
            fontSize: 10,
            letterSpacing: 1.4,
          }}
        >
          <span>▸ REPLICANT · PRIVATE · ENC. AES-R7</span>
          <button
            onClick={onClose}
            style={{
              background: "transparent",
              border: `1px solid ${rpColors.paper3}`,
              color: rpColors.paper3,
              padding: "2px 8px",
              fontFamily: "var(--font-mono)",
              fontSize: 10,
              letterSpacing: 1.4,
              cursor: "pointer",
            }}
          >
            CLOSE
          </button>
        </div>

        {/* messages */}
        <div
          ref={scrollRef}
          style={{
            flex: 1,
            overflow: "auto",
            padding: "16px 16px 8px",
            display: "flex",
            flexDirection: "column",
            gap: 12,
          }}
        >
          {messages.length === 0 && (
            <div
              className="t-memo"
              style={{
                color: rpColors.inkFaded,
                fontSize: 13,
                textAlign: "center",
                padding: "28px 12px",
              }}
            >
              &quot;This channel is empty. The Committee has not yet intercepted any
              transmissions among kin.&quot;
            </div>
          )}

          {messages.map((m) => {
            const mine = m.authorPlayerId === meId;
            return (
              <div
                key={m.messageId}
                style={{
                  display: "flex",
                  flexDirection: mine ? "row-reverse" : "row",
                  gap: 8,
                  alignItems: "flex-start",
                }}
              >
                <div
                  style={{
                    width: 30,
                    height: 30,
                    flexShrink: 0,
                    background: mine ? rpColors.stampBlue : rpColors.stampRed,
                    color: rpColors.paper3,
                    border: `1.5px solid ${rpColors.ink}`,
                    display: "flex",
                    alignItems: "center",
                    justifyContent: "center",
                    fontFamily: "var(--font-display)",
                    fontWeight: 700,
                    fontSize: 14,
                  }}
                >
                  {m.authorDisplayName.trim().slice(0, 1).toUpperCase() || "?"}
                </div>
                <div style={{ maxWidth: "78%" }}>
                  <div
                    style={{
                      display: "flex",
                      gap: 8,
                      alignItems: "baseline",
                      marginBottom: 3,
                      flexDirection: mine ? "row-reverse" : "row",
                    }}
                  >
                    <span
                      style={{
                        fontFamily: "var(--font-display)",
                        fontWeight: 600,
                        fontSize: 12,
                        color: rpColors.ink,
                        letterSpacing: "0.06em",
                      }}
                    >
                      {mine ? "YOU" : m.authorDisplayName.toUpperCase()}
                    </span>
                    <span
                      style={{
                        fontFamily: "var(--font-mono)",
                        fontSize: 9,
                        color: rpColors.inkFaded,
                      }}
                    >
                      {formatTime(m.sentAt)}
                    </span>
                  </div>
                  <div
                    style={{
                      background: mine ? rpColors.ink : rpColors.paper3,
                      color: mine ? rpColors.paper3 : rpColors.ink,
                      border: mine ? "none" : `1.5px solid ${rpColors.ink}`,
                      padding: "10px 13px",
                      fontFamily: "var(--font-typewriter)",
                      fontSize: 14,
                      lineHeight: 1.45,
                      boxShadow: mine ? "none" : "2px 2px 0 rgba(28,26,21,0.2)",
                      wordBreak: "break-word",
                    }}
                  >
                    {m.body}
                  </div>
                </div>
              </div>
            );
          })}
        </div>

        {/* composer */}
        <div
          style={{
            padding: "10px 12px 14px",
            borderTop: `1px dashed ${rpColors.paperLine}`,
            background: rpColors.paper2,
            display: "flex",
            flexDirection: "column",
            gap: 6,
          }}
        >
          <div
            style={{
              display: "flex",
              gap: 6,
              alignItems: "stretch",
              background: rpColors.paper3,
              border: `2px solid ${rpColors.ink}`,
              padding: "4px 6px 4px 10px",
            }}
          >
            <span
              style={{
                fontFamily: "var(--font-mono)",
                fontSize: 14,
                color: rpColors.inkFaded,
                alignSelf: "center",
              }}
            >
              ▸
            </span>
            <textarea
              value={draft}
              onChange={(e) => setDraft(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter" && !e.shiftKey) {
                  e.preventDefault();
                  send();
                }
              }}
              rows={1}
              placeholder="transmit to kin"
              style={{
                flex: 1,
                background: "transparent",
                border: "none",
                outline: "none",
                fontFamily: "var(--font-typewriter)",
                fontSize: 14,
                color: rpColors.ink,
                resize: "none",
                padding: "6px 0",
                lineHeight: 1.4,
                maxHeight: 120,
              }}
            />
            <button
              onClick={send}
              disabled={!draft.trim()}
              style={{
                background: draft.trim() ? rpColors.ink : rpColors.inkFaded,
                color: rpColors.paper3,
                padding: "6px 12px",
                border: "none",
                fontFamily: "var(--font-display)",
                fontSize: 12,
                fontWeight: 700,
                letterSpacing: "0.1em",
                cursor: draft.trim() ? "pointer" : "not-allowed",
              }}
            >
              SEND ◼
            </button>
          </div>
          <div
            style={{
              textAlign: "center",
              fontFamily: "var(--font-mono)",
              fontSize: 9,
              color: rpColors.inkFaded,
              letterSpacing: 1.3,
            }}
          >
            MESSAGES ARE ARCHIVED BY THE COMMITTEE AFTER CYCLE END
          </div>
        </div>
      </div>
    </div>
  );
}

function formatTime(iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "";
  return d.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit", hour12: false });
}
