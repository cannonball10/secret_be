// Mobile entry. A delegate arriving without a QR scan types the
// session code + their name here. Both feed into /join/CODE which is
// the shared join flow regardless of how they got there.

"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { RPButton } from "@replicant/ui";
import { loadSession } from "@/lib/session";

export default function MobileHomePage() {
  const router = useRouter();
  const [code, setCode] = useState("");

  // Bounce live sessions straight to the passport.
  useEffect(() => {
    const s = loadSession();
    if (s) router.replace("/game");
  }, [router]);

  const canContinue = code.trim().length >= 4;

  return (
    <main style={{ padding: "40px 24px", display: "flex", flexDirection: "column", gap: 20 }}>
      <div className="t-eyebrow" style={{ color: "var(--stamp-red)" }}>
        ◼ PLANETARY COMMITTEE · ACCORD R-07
      </div>
      <div
        style={{
          fontFamily: "var(--font-display)",
          fontWeight: 700,
          fontSize: 48,
          lineHeight: 0.95,
          letterSpacing: "-0.01em",
          color: "var(--ink)",
          position: "relative",
        }}
      >
        <span
          aria-hidden="true"
          style={{ position: "absolute", left: 2, top: 2, color: "var(--stamp-red)", opacity: 0.35 }}
        >
          REPLICANT
        </span>
        <span style={{ position: "relative" }}>REPLICANT</span>
      </div>
      <p className="t-memo" style={{ fontSize: 15, color: "var(--ink-soft)", margin: 0 }}>
        Submit the six-character credential provided by the Committee chair.
      </p>

      <label style={{ display: "flex", flexDirection: "column", gap: 6 }}>
        <span
          className="t-eyebrow"
          style={{ color: "var(--ink-faded)", fontSize: 10 }}
        >
          SESSION CODE
        </span>
        <input
          value={code}
          onChange={(e) =>
            setCode(e.target.value.toUpperCase().replace(/[^A-Z0-9]/g, ""))
          }
          maxLength={6}
          autoComplete="off"
          autoCapitalize="characters"
          spellCheck={false}
          placeholder="XXXXXX"
          style={{
            fontFamily: "var(--font-mono)",
            fontWeight: 700,
            fontSize: 28,
            letterSpacing: "0.3em",
            padding: "12px 14px",
            border: "2px solid var(--ink)",
            background: "var(--paper-3)",
            color: "var(--ink)",
            textAlign: "center",
            width: "100%",
          }}
        />
      </label>

      <RPButton
        variant="stamp"
        full
        disabled={!canContinue}
        onClick={() => router.push(`/join/${code.trim()}`)}
      >
        ▸ Proceed to intake
      </RPButton>

      <div
        className="t-memo"
        style={{ fontSize: 13, color: "var(--ink-faded)", marginTop: "auto" }}
      >
        &quot;The Committee appreciates your cooperation.&quot;
      </div>
    </main>
  );
}
