import type { Metadata } from "next";
import { headers } from "next/headers";
import {
  JetBrains_Mono,
  Manrope,
  Press_Start_2P,
  Unbounded,
} from "next/font/google";
import "./globals.css";

// Variable fonts (one file per family). Self-hosted at build time: no render-blocking request to Google, fonts are
// preloaded from our origin, and fallbacks are metric-adjusted (no reflow).
const display = Unbounded({
  subsets: ["latin", "cyrillic"],
  variable: "--font-unbounded",
  display: "swap",
});
const sans = Manrope({
  subsets: ["latin", "cyrillic"],
  variable: "--font-manrope",
  display: "swap",
});
const mono = JetBrains_Mono({
  subsets: ["latin", "cyrillic"],
  variable: "--font-jetbrains",
  display: "swap",
});
const pixel = Press_Start_2P({
  subsets: ["latin", "cyrillic"],
  weight: "400",
  variable: "--font-press-start",
  display: "swap",
  preload: false,
});

export const metadata: Metadata = {
  title: "Accord — координация агентов",
  description: "Задачи, агенты и работа проекта в одном месте",
};
export default async function RootLayout({
  children,
}: Readonly<{ children: React.ReactNode }>) {
  await headers();
  return (
    <html
      lang="ru"
      suppressHydrationWarning
      className={`${display.variable} ${sans.variable} ${mono.variable} ${pixel.variable}`}
    >
      <body>{children}</body>
    </html>
  );
}
