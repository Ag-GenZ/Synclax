import type { ReactNode } from "react";
import { Blocks, BookOpenText, DatabaseZap, MonitorSmartphone, Wrench } from "lucide-react";
import { DocsLayout } from "fumadocs-ui/layouts/docs";

import { source } from "@/lib/source";

const sectionIcons: Record<string, ReactNode> = {
  "/docs/getting-started": <BookOpenText size={16} />,
  "/docs/concepts": <Blocks size={16} />,
  "/docs/console": <MonitorSmartphone size={16} />,
  "/docs/developer": <Wrench size={16} />,
  "/docs/reference": <DatabaseZap size={16} />,
};

export default function DocsRootLayout({
  children,
}: Readonly<{
  children: ReactNode;
}>) {
  return (
    <DocsLayout
      tree={source.pageTree}
      githubUrl="https://github.com/wibus-wee/synclax"
      nav={{
        title: "Synclax Docs",
        url: "/",
      }}
      links={[
        {
          type: "main",
          text: "Home",
          url: "/",
          on: "nav",
        },
        {
          type: "main",
          text: "Quickstart",
          url: "/docs/getting-started/quickstart",
          on: "nav",
        },
      ]}
      tabs={{
        transform(option) {
          return {
            ...option,
            icon: sectionIcons[option.url] ?? option.icon,
          };
        },
      }}
    >
      {children}
    </DocsLayout>
  );
}
