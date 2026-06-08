import type { Metadata } from "next";
import { Geist, Geist_Mono } from "next/font/google";
import "./globals.css";

const geistSans = Geist({
  variable: "--font-geist-sans",
  subsets: ["latin"],
});

const geistMono = Geist_Mono({
  variable: "--font-geist-mono",
  subsets: ["latin"],
});

export const metadata: Metadata = {
  title: "TradeBench",
  description: "Submit your trading bot, compete against other algorithms, and climb the live leaderboard. Lower latency means a higher score in this high-frequency trading arena.",
  keywords: ["trading bot", "algorithmic trading", "hft", "high frequency trading", "leaderboard", "coding competition"],
  authors: [{ name: "TradeBench Team" }],
  openGraph: {
    title: "TradeBench | Live Algorithmic Trading Bot Leaderboard",
    description: "Submit your trading bot, compete against other algorithms, and climb the live leaderboard.",
    type: "website",
    locale: "en_US",
    siteName: "TradeBench",
  },
  twitter: {
    card: "summary_large_image",
    title: "TradeBench | Live Algorithmic Trading Bot Leaderboard",
    description: "Submit your trading bot, compete against other algorithms, and climb the live leaderboard.",
  },
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html
      lang="en"
      className={`${geistSans.variable} ${geistMono.variable} h-full antialiased`}
    >
      <body className="min-h-full flex flex-col">{children}</body>
    </html>
  );
}
