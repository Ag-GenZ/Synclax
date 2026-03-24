import type { Metadata } from "next";
import { RootProvider } from "fumadocs-ui/provider/next";

import "./global.css";

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
      <body className="flex min-h-screen flex-col">
        <RootProvider>{children}</RootProvider>
      </body>
    </html>
  );
}
