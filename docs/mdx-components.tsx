import type { MDXComponents } from "mdx/types";
import defaultMdxComponents from "fumadocs-ui/mdx";
import { Banner } from "fumadocs-ui/components/banner";
import { Callout } from "fumadocs-ui/components/callout";
import { Card, Cards } from "fumadocs-ui/components/card";
import { File, Files, Folder } from "fumadocs-ui/components/files";
import { ImageZoom } from "fumadocs-ui/components/image-zoom";
import { Step, Steps } from "fumadocs-ui/components/steps";
import { Tab, Tabs, TabsContent, TabsList, TabsTrigger } from "fumadocs-ui/components/tabs";

const components: MDXComponents = {
  ...defaultMdxComponents,
  Banner,
  Callout,
  Card,
  Cards,
  File,
  Files,
  Folder,
  ImageZoom,
  Step,
  Steps,
  Tab,
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
};

export default components;
