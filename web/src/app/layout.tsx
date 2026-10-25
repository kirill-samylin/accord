import type { Metadata } from "next";
import { headers } from "next/headers";
import "./globals.css";
export const metadata: Metadata = {
  title: "Accord — координация агентов",
  description: "Задачи, агенты и работа проекта в одном месте",
};
export default async function RootLayout({
  children,
}: Readonly<{ children: React.ReactNode }>) {
  await headers();
  return (
    <html lang="ru" suppressHydrationWarning>
      <head>
        {/* Шрифты neo-brutalism UI: Unbounded (заголовки), Manrope (текст),
            JetBrains Mono (ключи/время), Press Start 2P (пиксельные кикеры).
            При недоступности Google Fonts работают локальные фолбэки. */}
        <link rel="preconnect" href="https://fonts.googleapis.com" />
        <link
          rel="preconnect"
          href="https://fonts.gstatic.com"
          crossOrigin="anonymous"
        />
        <link
          rel="stylesheet"
          href="https://fonts.googleapis.com/css2?family=Unbounded:wght@700;900&family=Manrope:wght@500;600;700;800&family=JetBrains+Mono:wght@500;600;700;800&family=Press+Start+2P&display=swap"
        />
      </head>
      <body>{children}</body>
    </html>
  );
}
