// Root layout — loads the four Google Fonts via next/font and exposes
// them as CSS variables the token stylesheet references. Imports the
// global token stylesheet so every descendant has access to the vars
// and utility classes (.paper-tex, .t-eyebrow, etc.).

import type { Metadata } from "next";
import type { ReactNode } from "react";
import { Oswald, Special_Elite, JetBrains_Mono, Inter_Tight } from "next/font/google";
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
  description: "Department of Human Affairs · Form R-07",
};

export default function RootLayout({ children }: { children: ReactNode }) {
  const fontVars = [oswald.variable, specialElite.variable, jetbrainsMono.variable, interTight.variable].join(" ");
  return (
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
}
