import type { Metadata } from "next";
import { IBM_Plex_Sans, Space_Grotesk } from "next/font/google";
import { RootProvider } from "fumadocs-ui/provider/next";

import "./global.css";

const bodyFont = IBM_Plex_Sans({
  subsets: ["latin"],
  variable: "--font-body",
  weight: ["400", "500", "600", "700"],
});

const displayFont = Space_Grotesk({
  subsets: ["latin"],
  variable: "--font-display",
  weight: ["500", "700"],
});

export const metadata: Metadata = {
  metadataBase: new URL("https://synclax.dev"),
  title: {
    default: "Synclax Docs",
    template: "%s | Synclax Docs",
  },
  description: "Full-feature documentation for Synclax, rebuilt on Fumadocs.",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="zh-CN" suppressHydrationWarning>
      <body className={`${bodyFont.variable} ${displayFont.variable}`}>
        <RootProvider>{children}</RootProvider>
      </body>
    </html>
  );
}
