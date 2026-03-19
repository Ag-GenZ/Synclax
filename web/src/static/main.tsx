import { StrictMode } from "react";
import { createRoot } from "react-dom/client";

import "#/styles.css";

import TanStackQueryProvider from "#/integrations/tanstack-query/root-provider";
import { DashboardPage } from "#/routes/index";

const rootEl = document.getElementById("root");
if (!rootEl) {
  throw new Error("Missing #root element");
}

createRoot(rootEl).render(
  <StrictMode>
    <TanStackQueryProvider>
      <DashboardPage />
    </TanStackQueryProvider>
  </StrictMode>,
);

