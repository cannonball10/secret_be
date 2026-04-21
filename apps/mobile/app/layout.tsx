// Mobile app root layout — same font loading strategy as the host so
// every RP* primitive renders identically. Paper surface by default
// (the mobile side of Replicant is the "delegate passport" form).

import type { Metadata, Viewport } from "next";
import type { ReactNode } from "react";
import { Oswald, Special_Elite, JetBrains_Mono, Inter_Tight } from "next/font/google";
import { ClerkProvider } from "@clerk/nextjs";
import "@replicant/tokens/styles.css";

const oswald = Oswald({
  subsets: ["latin"],
  weight: ["500", "600", "700"],
  variable: "--font-oswald",
  display: "swap",
});
const specialElite = Special_Elite({
  subsets: ["latin"],
  weight: ["400"],
  variable: "--font-special-elite",
  display: "swap",
});
const jetbrainsMono = JetBrains_Mono({
  subsets: ["latin"],
  weight: ["400", "500", "700"],
  variable: "--font-jetbrains-mono",
  display: "swap",
});
const interTight = Inter_Tight({
  subsets: ["latin"],
  weight: ["400", "500", "600", "700"],
  variable: "--font-inter-tight",
  display: "swap",
});

export const metadata: Metadata = {
  title: "REPLICANT · Passport",
  description: "Planetary Committee · Delegate Terminal",
};

export const viewport: Viewport = {
  themeColor: "#E8DFC9",
  width: "device-width",
  initialScale: 1,
  maximumScale: 1,
  userScalable: false,
};

export default function RootLayout({ children }: { children: ReactNode }) {
  const fontVars = [oswald.variable, specialElite.variable, jetbrainsMono.variable, interTight.variable].join(" ");
  // ClerkProvider mount is gated on the publishable key: v5 throws
  // without one, but this app MUST work without Clerk for guests.
  const clerkKey = process.env.NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY;
  const body = (
    <html lang="en" className={fontVars}>
      <body
        className="paper-tex"
        style={{
          background: "var(--paper)",
          color: "var(--ink)",
          fontFamily: "var(--font-sans)",
          minHeight: "100vh",
        }}
      >
        {children}
      </body>
    </html>
  );
  if (!clerkKey) return body;
  return <ClerkProvider>{body}</ClerkProvider>;
}
