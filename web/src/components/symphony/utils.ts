export const PHASES = [
  "PreparingWorkspace",
  "LaunchingAgentProcess",
  "InitializingSession",
  "BuildingPrompt",
  "StreamingTurn",
  "Finishing",
] as const;

export type Phase = (typeof PHASES)[number];

export const PHASE_LABELS: Record<Phase, string> = {
  PreparingWorkspace: "Preparing",
  LaunchingAgentProcess: "Launching",
  InitializingSession: "Init",
  BuildingPrompt: "Building",
  StreamingTurn: "Streaming",
  Finishing: "Finishing",
};

export function fmtTokens(n: number): string {
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`;
  if (n >= 1_000) return `${(n / 1_000).toFixed(1)}K`;
  return String(n);
}

export function fmtSeconds(s: number): string {
  if (s < 60) return `${Math.floor(s)}s`;
  if (s < 3600) return `${Math.floor(s / 60)}m ${Math.floor(s % 60)}s`;
  return `${Math.floor(s / 3600)}h ${Math.floor((s % 3600) / 60)}m`;
}

export function fmtAgo(iso: string): string {
  const secs = Math.floor((Date.now() - new Date(iso).getTime()) / 1000);
  if (secs < 5) return "just now";
  if (secs < 60) return `${secs}s ago`;
  if (secs < 3600) return `${Math.floor(secs / 60)}m ago`;
  return `${Math.floor(secs / 3600)}h ago`;
}

export function fmtDueIn(iso: string): string {
  const ms = new Date(iso).getTime() - Date.now();
  if (ms <= 0) return "now";
  const s = Math.floor(ms / 1000);
  if (s < 60) return `in ${s}s`;
  if (s < 3600) return `in ${Math.floor(s / 60)}m`;
  return `in ${Math.floor(s / 3600)}h`;
}

export function statusVariant(
  status: string,
): "success" | "error" | "warning" | "outline" | "secondary" | "info" {
  switch (status) {
    case "Succeeded":
      return "success";
    case "Failed":
      return "error";
    case "TimedOut":
    case "Stalled":
      return "warning";
    case "CanceledByReconciliation":
      return "outline";
    default:
      return "secondary";
  }
}

export type ActivityItem = {
  issueId: string;
  identifier: string;
  title: string;
  ts: string;
  event: string;
  message?: string | null;
};

import type { SymphonyRunningEntry } from "#/api-gen/types.gen";

export function collectActivity(running: Array<SymphonyRunningEntry>): Array<ActivityItem> {
  const out: Array<ActivityItem> = [];
  for (const entry of running) {
    const log = entry.live.event_log ?? [];
    for (const ev of log) {
      out.push({
        issueId: entry.issue_id,
        identifier: entry.issue.identifier,
        title: entry.issue.title,
        ts: ev.ts,
        event: ev.event,
        message: ev.message ?? null,
      });
    }
  }
  out.sort((a, b) => new Date(b.ts).getTime() - new Date(a.ts).getTime());
  return out.slice(0, 200);
}
