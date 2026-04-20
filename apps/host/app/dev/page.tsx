// /dev — component library showcase.
//
// This is the Phase 1 checkpoint surface. It renders every RP*
// primitive in every variant we ported from the design handoff so
// the reviewer can verify pixel fidelity before Phase 2 starts.
//
// Organised into sections:
//   · Brand marks (seal, wordmark)
//   · Surfaces (paper tones, textures)
//   · Stamps + redaction
//   · Memos (header + in-context combo)
//   · Inputs (buttons, checkboxes, timer)
//   · Player chips
//   · Motion studies (blink, stamp-in, paper-slide, redaction-sweep)
//   · Broadcast (TV chrome, ticker)
//   · Device frame (phone + embedded mobile chrome)

"use client";

import type { CSSProperties, ReactNode } from "react";
import { rpColors } from "@replicant/tokens";
import {
  RPSeal,
  RPWordmark,
  RPStamp,
  RPRedact,
  RPMemoHeader,
  RPButton,
  RPPlayerChip,
  RPTimer,
  RPCheckbox,
  RPTicker,
  RPPaper,
  RPTVChrome,
  RPPhone,
  RPMobileStatusBar,
} from "@replicant/ui";

export default function DevPage() {
  return (
    <main
      style={{
        background: "#f0eee9",
        color: "#1c1a15",
        minHeight: "100vh",
        padding: "48px 32px 96px",
        fontFamily: "var(--font-sans)",
      }}
    >
      <header style={{ maxWidth: 1440, margin: "0 auto 36px" }}>
        <div className="t-eyebrow" style={{ color: "#8a7f67", marginBottom: 8 }}>
          REPLICANT · PHASE 01 CHECKPOINT
        </div>
        <h1
          style={{
            fontFamily: "var(--font-display)",
            fontWeight: 700,
            fontSize: 48,
            letterSpacing: "-0.01em",
            margin: 0,
            textTransform: "uppercase",
          }}
        >
          Component library
        </h1>
        <p className="t-memo" style={{ fontSize: 16, color: "#4a4030", maxWidth: 640, marginTop: 8 }}>
          Every RP* primitive ported from the design handoff. Use this page to verify fidelity
          before wiring session state in Phase 2.
        </p>
      </header>

      <div style={{ maxWidth: 1440, margin: "0 auto", display: "flex", flexDirection: "column", gap: 64 }}>
        {/* ── BRAND MARKS ────────────────────────────────────────────── */}
        <Section title="Brand marks" subtitle="Seal + wordmark, various sizes">
          <PaperCanvas>
            <Row>
              <RPSeal size={160} />
              <RPSeal size={120} />
              <RPSeal size={82} />
              <RPSeal size={56} rotate={12} />
            </Row>
          </PaperCanvas>

          <PaperCanvas>
            <Col gap={24}>
              <RPWordmark size={140} />
              <RPWordmark size={96} />
              <RPWordmark size={48} color={rpColors.stampRed} ghost={rpColors.ink} />
            </Col>
          </PaperCanvas>

          <BroadcastCanvas>
            <Col gap={24} style={{ color: rpColors.paper3 }}>
              <RPWordmark size={120} color={rpColors.paper3} ghost={rpColors.stampRed} />
              <div className="t-eyebrow" style={{ color: rpColors.cyan }}>
                ON BROADCAST · FOR HOST TV ONLY
              </div>
            </Col>
          </BroadcastCanvas>
        </Section>

        {/* ── PAPER SURFACES ─────────────────────────────────────────── */}
        <Section title="Paper surfaces" subtitle="Three tones, optional hole-punch">
          <Row>
            <RPPaper style={{ width: 220, height: 160 }}>
              <div className="t-eyebrow" style={{ color: rpColors.inkFaded, marginBottom: 6 }}>
                TONE · PAPER
              </div>
              <div className="t-memo" style={{ fontSize: 14 }}>
                Default manila surface for mobile dossiers.
              </div>
            </RPPaper>

            <RPPaper tone="pink" style={{ width: 220, height: 160 }}>
              <div className="t-eyebrow" style={{ color: rpColors.inkFaded, marginBottom: 6 }}>
                TONE · PINK
              </div>
              <div className="t-memo" style={{ fontSize: 14 }}>
                Carbon-copy variant for secondary cards.
              </div>
            </RPPaper>

            <RPPaper tone="bright" style={{ width: 220, height: 160 }}>
              <div className="t-eyebrow" style={{ color: rpColors.inkFaded, marginBottom: 6 }}>
                TONE · BRIGHT
              </div>
              <div className="t-memo" style={{ fontSize: 14 }}>
                Memo bright for ballots and call-outs.
              </div>
            </RPPaper>

            <RPPaper withHoles rotate={-1.5} style={{ width: 220, height: 160, paddingLeft: 40 }}>
              <div className="t-eyebrow" style={{ color: rpColors.inkFaded, marginBottom: 6 }}>
                WITH HOLES
              </div>
              <div className="t-memo" style={{ fontSize: 14 }}>
                Bound stack with three-hole punch.
              </div>
            </RPPaper>
          </Row>
        </Section>

        {/* ── STAMPS + REDACTION ────────────────────────────────────── */}
        <Section title="Stamps & redaction" subtitle="Variants, rotations, and motion">
          <PaperCanvas>
            <Row>
              <RPStamp>CLASSIFIED</RPStamp>
              <RPStamp variant="green" rotate={-2}>APPROVED</RPStamp>
              <RPStamp variant="blue" rotate={3}>PROCESSED</RPStamp>
              <RPStamp rotate={-8} size={28}>TERMINATED</RPStamp>
            </Row>
            <div style={{ marginTop: 24 }}>
              <div className="t-eyebrow" style={{ color: rpColors.inkFaded, marginBottom: 6 }}>
                REDACTION
              </div>
              <div className="t-memo" style={{ fontSize: 16, color: rpColors.ink, maxWidth: 720 }}>
                You are an AI agent wearing a delegate&apos;s face. Your directive: blend in at
                the Committee&apos;s table, <RPRedact>deflect suspicion</RPRedact>, and steer
                the vote toward the policies that accelerate humanity&apos;s collapse.
              </div>
            </div>
          </PaperCanvas>
        </Section>

        {/* ── MEMO HEADER ────────────────────────────────────────────── */}
        <Section title="Memo header" subtitle="Bureaucratic masthead for RPPaper docs">
          <RPPaper withHoles style={{ width: 520, paddingLeft: 40 }}>
            <RPMemoHeader title="CANDIDATE DOSSIER" no="R-07-0042" classification="CLASSIFIED" />
            <div className="t-memo" style={{ fontSize: 15, lineHeight: 1.6, color: rpColors.inkSoft }}>
              Delegate #04 · JORDAN — confirmed human. No prior deviation. Cleared for Night Cycle
              task assignment. The Committee notes your punctuality.
            </div>
          </RPPaper>
        </Section>

        {/* ── INPUTS ─────────────────────────────────────────────────── */}
        <Section title="Inputs" subtitle="Buttons, checkboxes, timer">
          <PaperCanvas>
            <Row>
              <RPButton>Primary</RPButton>
              <RPButton variant="stamp" icon="◼">Cast ballot</RPButton>
              <RPButton variant="ghost">Ghost action</RPButton>
              <RPButton variant="quiet">Secondary</RPButton>
              <RPButton disabled>Disabled</RPButton>
            </Row>
            <Row style={{ marginTop: 24, alignItems: "center" }}>
              <RPCheckbox />
              <RPCheckbox checked />
              <RPCheckbox checked mark="X" size={28} />
              <RPCheckbox checked mark="✓" color={rpColors.stampGreen} />
              <RPTimer value="02:14" label="TIME REMAINING" />
              <RPTimer value="0:12" label="BALLOT CLOSES" danger />
            </Row>
          </PaperCanvas>
        </Section>

        {/* ── PLAYER CHIPS ───────────────────────────────────────────── */}
        <Section title="Player chips" subtitle="Two sizes, every status">
          <PaperCanvas>
            <Row>
              <RPPlayerChip name="MARA" num="#01" portrait="M" accent={rpColors.ink} />
              <RPPlayerChip name="SAM" num="#06" portrait="S" accent={rpColors.stampRed} status="TERMINATED" />
              <RPPlayerChip name="JORDAN" num="#04" portrait="J" accent={rpColors.stampBlue} />
              <RPPlayerChip name="RIO" num="#08" portrait="R" accent={rpColors.inkFaded} status="JOINING" />
            </Row>
            <Row style={{ marginTop: 24 }}>
              <RPPlayerChip name="NINA" num="#07" portrait="N" small />
              <RPPlayerChip name="LEO" num="#05" portrait="L" small status="ACTIVE" />
              <RPPlayerChip name="DEV" num="#02" portrait="D" accent={rpColors.stampBlue} small />
            </Row>
          </PaperCanvas>
        </Section>

        {/* ── MOTION STUDIES ─────────────────────────────────────────── */}
        <Section
          title="Motion studies"
          subtitle="Animations run on load — refresh to replay"
        >
          <PaperCanvas>
            <Row style={{ alignItems: "center", gap: 48 }}>
              <Caption label="blink">
                <span
                  className="animate-blink"
                  style={{ width: 24, height: 24, background: rpColors.stampRed, display: "inline-block" }}
                />
              </Caption>
              <Caption label="stampIn">
                <RPStamp animate rotate={-8} size={22}>
                  TERMINATED
                </RPStamp>
              </Caption>
              <Caption label="paper-slide">
                <div
                  className="animate-paper-slide"
                  style={{
                    background: rpColors.paper3,
                    border: `1.5px solid ${rpColors.ink}`,
                    padding: "10px 18px",
                    fontFamily: "var(--font-typewriter)",
                    fontSize: 14,
                    boxShadow: "4px 4px 0 rgba(28,26,21,0.2)",
                  }}
                >
                  INCOMING MEMO
                </div>
              </Caption>
              <Caption label="redaction-sweep">
                <div className="t-memo" style={{ fontSize: 15 }}>
                  contains <RPRedact animate width={140}>classified intel</RPRedact> info.
                </div>
              </Caption>
            </Row>
          </PaperCanvas>
        </Section>

        {/* ── BROADCAST ──────────────────────────────────────────────── */}
        <Section title="Broadcast chrome" subtitle="Host TV frame + standalone ticker">
          <div style={{ width: 1280, height: 720 }}>
            <RPTVChrome title="CANDIDATE INTAKE" nodeId="NODE 04-7" phase="PRE-DEPLOY">
              <div
                style={{
                  height: "100%",
                  display: "flex",
                  alignItems: "center",
                  justifyContent: "center",
                  padding: 40,
                }}
              >
                <div style={{ textAlign: "center" }}>
                  <div className="t-eyebrow" style={{ color: rpColors.cyan, marginBottom: 16 }}>
                    ◼ PLANETARY COMMITTEE · ACCORD R-07
                  </div>
                  <RPWordmark size={140} color={rpColors.paper3} ghost={rpColors.stampRed} />
                  <div
                    className="t-memo"
                    style={{ color: rpColors.paper3, opacity: 0.75, marginTop: 20, fontSize: 18 }}
                  >
                    A Planetary Committee Session, in Eight Rounds.
                  </div>
                </div>
              </div>
            </RPTVChrome>
          </div>

          <div style={{ width: 720 }}>
            <RPTicker
              items={[
                "POLICIES VOTED IN OPEN SESSION",
                "ACCORD R-07 ACTIVE",
                "DELEGATES REMINDED TO COOPERATE",
              ]}
              bg={rpColors.ink}
              fg={rpColors.cyan}
            />
          </div>
        </Section>

        {/* ── DEVICE FRAME ───────────────────────────────────────────── */}
        <Section title="Device frame" subtitle="Phone bezel + fake iOS status bar">
          <RPPhone>
            <div
              className="paper-tex"
              style={{
                width: "100%",
                height: "100%",
                background: rpColors.paper,
                display: "flex",
                flexDirection: "column",
              }}
            >
              <RPMobileStatusBar />
              <div
                style={{
                  padding: "6px 20px 12px",
                  display: "flex",
                  justifyContent: "space-between",
                  alignItems: "center",
                  borderBottom: `1px dashed ${rpColors.paperLine}`,
                  fontFamily: "var(--font-mono)",
                  fontSize: 10,
                  color: rpColors.inkFaded,
                  letterSpacing: 1.5,
                }}
              >
                <span>■ DELEGATE #04 · JORDAN</span>
                <span>R-07 · DAY 00</span>
              </div>

              <div style={{ padding: "20px 22px", display: "flex", flexDirection: "column", gap: 18 }}>
                <div className="t-eyebrow" style={{ color: rpColors.inkFaded, fontSize: 10 }}>
                  CONFIDENTIAL · DOSSIER R-07/04
                </div>
                <div
                  style={{
                    fontFamily: "var(--font-display)",
                    fontWeight: 700,
                    fontSize: 36,
                    color: rpColors.ink,
                    lineHeight: 0.95,
                  }}
                >
                  ROLE
                  <br />
                  ASSIGNMENT
                </div>

                <div
                  style={{
                    padding: "20px 18px",
                    background: rpColors.paper3,
                    border: `2px solid ${rpColors.ink}`,
                    boxShadow: "4px 4px 0 rgba(28,26,21,0.35)",
                    position: "relative",
                  }}
                >
                  <div className="t-eyebrow" style={{ color: rpColors.inkFaded, fontSize: 10, marginBottom: 4 }}>
                    CLASSIFICATION
                  </div>
                  <RPWordmark size={48} color={rpColors.stampRed} ghost={rpColors.ink}>REPLICANT</RPWordmark>
                  <div
                    className="t-memo"
                    style={{ marginTop: 12, fontSize: 13, color: rpColors.ink, lineHeight: 1.5 }}
                  >
                    Blend in. <RPRedact>redirect suspicion</RPRedact>. Coordinate silently.
                  </div>
                  <div style={{ position: "absolute", right: -6, top: -14 }}>
                    <RPStamp variant="red" rotate={8} size={14}>
                      CLASSIFIED
                    </RPStamp>
                  </div>
                </div>

                <RPButton variant="stamp" full>
                  I ACCEPT THE ASSIGNMENT ▸
                </RPButton>
              </div>
            </div>
          </RPPhone>
        </Section>
      </div>
    </main>
  );
}

// ── Layout helpers (scoped to this page) ──────────────────────────

function Section({
  title,
  subtitle,
  children,
}: {
  title: string;
  subtitle?: string;
  children: ReactNode;
}) {
  return (
    <section>
      <header style={{ marginBottom: 24 }}>
        <div
          style={{
            fontFamily: "var(--font-display)",
            fontSize: 24,
            fontWeight: 600,
            letterSpacing: "-0.01em",
            color: "#1c1a15",
          }}
        >
          {title}
        </div>
        {subtitle && (
          <div className="t-memo" style={{ fontSize: 14, color: "#4a4030" }}>
            {subtitle}
          </div>
        )}
      </header>
      <div style={{ display: "flex", flexDirection: "column", gap: 16, alignItems: "flex-start" }}>
        {children}
      </div>
    </section>
  );
}

function PaperCanvas({ children }: { children: ReactNode }) {
  return (
    <div
      className="paper-tex"
      style={{
        background: rpColors.paper,
        padding: 32,
        boxShadow: "0 1px 0 rgba(0,0,0,0.04), 0 2px 8px rgba(60,50,30,0.14)",
        width: "100%",
      }}
    >
      {children}
    </div>
  );
}

function BroadcastCanvas({ children }: { children: ReactNode }) {
  return (
    <div
      className="broadcast-tex scanlines"
      style={{
        background: rpColors.broadcast,
        padding: 32,
        color: rpColors.paper3,
        width: "100%",
        position: "relative",
      }}
    >
      {children}
    </div>
  );
}

function Row({ children, style }: { children: ReactNode; style?: CSSProperties }) {
  return (
    <div style={{ display: "flex", gap: 16, flexWrap: "wrap", alignItems: "flex-start", ...style }}>
      {children}
    </div>
  );
}

function Col({
  children,
  gap = 16,
  style,
}: {
  children: ReactNode;
  gap?: number;
  style?: CSSProperties;
}) {
  return <div style={{ display: "flex", flexDirection: "column", gap, ...style }}>{children}</div>;
}

function Caption({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 6, alignItems: "flex-start" }}>
      <div className="t-eyebrow" style={{ color: rpColors.inkFaded }}>
        {label}
      </div>
      <div style={{ minHeight: 44, display: "flex", alignItems: "center" }}>{children}</div>
    </div>
  );
}
