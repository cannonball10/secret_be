// Root layout — loads the four Google Fonts via next/font and exposes
// them as CSS variables the token stylesheet references. Imports the
// global token stylesheet so every descendant has access to the vars
// and utility classes (.paper-tex, .t-eyebrow, etc.).

import type { Metadata } from "next";
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
  title: "REPLICANT · Host",
  description: "Planetary Committee · Accord R-07",
};

export default function RootLayout({ children }: { children: ReactNode }) {
  const fontVars = [oswald.variable, specialElite.variable, jetbrainsMono.variable, interTight.variable].join(" ");
  // Only mount ClerkProvider when a publishable key is present — v5
  // throws at runtime if the key is missing, which would kill the
  // dev-without-Clerk path. The child components gate Clerk hooks
  // behind CLERK_ENABLED (lib/useToken.ts) so they no-op when the
  // provider isn't mounted.
  const clerkKey = process.env.NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY;
  const body = (
    <html lang="en" className={fontVars}>
      <body
        style={{
          background: "var(--broadcast)",
          color: "var(--paper-3)",
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
