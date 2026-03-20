import type { Metadata } from "next";
import { notFound } from "next/navigation";
import { DocsBody, DocsDescription, DocsPage, DocsTitle } from "fumadocs-ui/page";

import { source } from "@/lib/source";
import defaultMdxComponents from "@/mdx-components";

type DocsPageProps = {
  params: Promise<{
    slug?: string[];
  }>;
};

export async function generateStaticParams() {
  return source.generateParams();
}

export async function generateMetadata({ params }: DocsPageProps): Promise<Metadata> {
  const { slug } = await params;
  const page = source.getPage(slug);

  if (!page) {
    return {};
  }

  return {
    title: page.data.title,
    description: page.data.description,
  };
}

export default async function Page({ params }: DocsPageProps) {
  const { slug } = await params;
  const page = source.getPage(slug);

  if (!page) {
    notFound();
  }

  const Body = page.data.body;

  return (
    <DocsPage
      breadcrumb={{ enabled: true }}
      editOnGithub={{
        owner: "wibus-wee",
        repo: "synclax",
        path: `docs/content/docs/${page.path}`,
        sha: "main",
      }}
      full={page.data.full}
      lastUpdate={page.data.lastModified}
      toc={page.data.toc}
    >
      <DocsTitle>{page.data.title}</DocsTitle>
      <DocsDescription>{page.data.description}</DocsDescription>
      <DocsBody>
        <Body components={defaultMdxComponents} />
      </DocsBody>
    </DocsPage>
  );
}
