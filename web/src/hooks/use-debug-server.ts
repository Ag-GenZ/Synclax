import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { ofetch } from "ofetch";

export type DebugState = {
  meta: {
    workflow_path: string;
    revision: number;
  };
  snapshot: {
    running: Array<Record<string, unknown>>;
    retrying: Array<Record<string, unknown>>;
    completed?: Array<Record<string, unknown>>;
    codex_totals: {
      input_tokens: number;
      output_tokens: number;
      total_tokens: number;
      seconds_running: number;
    };
    rate_limits?: Record<string, unknown>;
  };
};

function debugBaseUrl(port: number): string {
  return `http://127.0.0.1:${port}`;
}

export function useDebugState(port: number | null | undefined) {
  return useQuery<DebugState>({
    queryKey: ["debug-state", port],
    queryFn: () => ofetch<DebugState>(`${debugBaseUrl(port!)}/api/v1/state`),
    enabled: port != null && port > 0,
    refetchInterval: 2000,
    retry: 1,
  });
}

export function useDebugHealthz(port: number | null | undefined) {
  return useQuery<string>({
    queryKey: ["debug-healthz", port],
    queryFn: async () => {
      const res = await fetch(`${debugBaseUrl(port!)}/healthz`);
      return res.text();
    },
    enabled: port != null && port > 0,
    refetchInterval: 5000,
    retry: 1,
  });
}

export function useDebugRefresh(port: number | null | undefined) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: () =>
      ofetch(`${debugBaseUrl(port!)}/api/v1/refresh`, { method: "POST" }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["debug-state", port] });
    },
  });
}
