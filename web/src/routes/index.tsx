"use client";

import { createFileRoute, Link } from "@tanstack/react-router";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState, useCallback } from "react";
import {
  ActivityIcon,
  AlertCircleIcon,
  BarChart3Icon,
  CheckCircle2Icon,
  ChevronDownIcon,
  ChevronRightIcon,
  CircleDotIcon,
  ClockIcon,
  CpuIcon,
  ExternalLinkIcon,
  FolderIcon,
  GitBranchIcon,
  LayersIcon,
  PauseCircleIcon,
  PlayCircleIcon,
  RefreshCwIcon,
  RotateCcwIcon,
  ServerIcon,
  SettingsIcon,
  TagIcon,
  TimerIcon,
  ZapIcon,
} from "lucide-react";

import {
  getHealthOptions,
  getSymphonySnapshotOptions,
  startSymphonyMutation,
  stopSymphonyMutation,
} from "#/api-gen/@tanstack/react-query.gen";
import { toastManager } from "#/components/ui/toast";
import type {
  SymphonyRunningEntry,
  SymphonyRetryEntry,
  SymphonyCompletedEntry,
  CodexTotals,
} from "#/api-gen/types.gen";

import {
  SidebarProvider,
  Sidebar,
  SidebarHeader,
  SidebarContent,
  SidebarGroup,
  SidebarGroupLabel,
  SidebarGroupContent,
  SidebarMenu,
  SidebarMenuItem,
  SidebarMenuButton,
  SidebarFooter,
  SidebarInset,
  SidebarTrigger,
  SidebarSeparator,
} from "#/components/ui/sidebar";
import { Card, CardHeader, CardTitle, CardDescription, CardPanel } from "#/components/ui/card";
import { Badge } from "#/components/ui/badge";
import { Button } from "#/components/ui/button";
import {
  Dialog,
  DialogTrigger,
  DialogPopup,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogClose,
} from "#/components/ui/dialog";
import {
  AlertDialog,
  AlertDialogTrigger,
  AlertDialogPopup,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogClose,
} from "#/components/ui/alert-dialog";
import { Input } from "#/components/ui/input";
import { Label } from "#/components/ui/label";
import { Separator } from "#/components/ui/separator";
import { Skeleton } from "#/components/ui/skeleton";
import { Spinner } from "#/components/ui/spinner";
import { Alert, AlertTitle, AlertDescription } from "#/components/ui/alert";
import {
  Progress,
  ProgressLabel,
  ProgressTrack,
  ProgressIndicator,
} from "#/components/ui/progress";
import { Collapsible, CollapsibleTrigger, CollapsiblePanel } from "#/components/ui/collapsible";
import { ScrollArea } from "#/components/ui/scroll-area";
import {
  Table,
  TableHeader,
  TableBody,
  TableRow,
  TableHead,
  TableCell,
} from "#/components/ui/table";
import { Tooltip, TooltipTrigger, TooltipPopup } from "#/components/ui/tooltip";
import { Tabs, TabsList, TabsTab, TabsPanel } from "#/components/ui/tabs";
import ThemeToggle from "#/components/ThemeToggle";
import { cn } from "#/lib/utils";

export const Route = createFileRoute("/")({ component: SymphonyDashboard });

// ─── Phase config ────────────────────────────────────────────────────────────
const PHASES = [
  "PreparingWorkspace",
  "LaunchingAgentProcess",
  "InitializingSession",
  "BuildingPrompt",
  "StreamingTurn",
  "Finishing",
] as const;

type Phase = (typeof PHASES)[number];

const PHASE_LABELS: Record<Phase, string> = {
  PreparingWorkspace: "Preparing",
  LaunchingAgentProcess: "Launching",
  InitializingSession: "Init",
  BuildingPrompt: "Building",
  StreamingTurn: "Streaming",
  Finishing: "Finishing",
};

// ─── Helpers ─────────────────────────────────────────────────────────────────
function fmtTokens(n: number): string {
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`;
  if (n >= 1_000) return `${(n / 1_000).toFixed(1)}K`;
  return String(n);
}

function fmtSeconds(s: number): string {
  if (s < 60) return `${Math.floor(s)}s`;
  if (s < 3600) return `${Math.floor(s / 60)}m ${Math.floor(s % 60)}s`;
  return `${Math.floor(s / 3600)}h ${Math.floor((s % 3600) / 60)}m`;
}

function fmtAgo(iso: string): string {
  const secs = Math.floor((Date.now() - new Date(iso).getTime()) / 1000);
  if (secs < 5) return "just now";
  if (secs < 60) return `${secs}s ago`;
  if (secs < 3600) return `${Math.floor(secs / 60)}m ago`;
  return `${Math.floor(secs / 3600)}h ago`;
}

function fmtDueIn(iso: string): string {
  const ms = new Date(iso).getTime() - Date.now();
  if (ms <= 0) return "now";
  const s = Math.floor(ms / 1000);
  if (s < 60) return `in ${s}s`;
  if (s < 3600) return `in ${Math.floor(s / 60)}m`;
  return `in ${Math.floor(s / 3600)}h`;
}

type ActivityItem = {
  issueId: string;
  identifier: string;
  title: string;
  ts: string;
  event: string;
  message?: string | null;
};

function collectActivity(running: Array<SymphonyRunningEntry>): Array<ActivityItem> {
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

function statusVariant(status: string):
  | "success"
  | "error"
  | "warning"
  | "outline"
  | "secondary"
  | "info" {
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

// ─── Phase progress bar ───────────────────────────────────────────────────────
function PhaseBar({ phase }: { phase: string }) {
  const idx = PHASES.indexOf(phase as Phase);
  return (
    <div className="flex items-center gap-0.5">
      {PHASES.map((p, i) => (
        <Tooltip key={p}>
          <TooltipTrigger>
            <div
              className={cn(
                "h-1 w-5 rounded-full transition-all",
                i < idx
                  ? "bg-success/60"
                  : i === idx
                    ? "bg-[var(--lagoon-deep)] animate-pulse"
                    : "bg-border",
              )}
            />
          </TooltipTrigger>
          <TooltipPopup>{PHASE_LABELS[p] ?? p}</TooltipPopup>
        </Tooltip>
      ))}
      <span className="ml-2 font-mono text-xs text-muted-foreground">
        {PHASE_LABELS[phase as Phase] ?? phase}
      </span>
    </div>
  );
}

// ─── Running agent card ───────────────────────────────────────────────────────
function AgentCard({ entry }: { entry: SymphonyRunningEntry }) {
  const [open, setOpen] = useState(false);
  const { issue, live, phase, started_at, workspace_path, attempt } = entry;

  const stateVariant =
    issue.state === "In Progress"
      ? ("success" as const)
      : issue.state === "Todo"
        ? ("info" as const)
        : issue.state === "Done" || issue.state === "Canceled"
          ? ("outline" as const)
          : ("secondary" as const);

  return (
    <Card className="overflow-hidden">
      <CardHeader className="pb-3">
        {/* Title row */}
        <div className="flex items-start gap-2">
          <div className="flex-1 min-w-0">
            <div className="flex flex-wrap items-center gap-1.5 mb-1.5">
              <span className="font-mono text-xs font-semibold text-[var(--lagoon-deep)]">
                {issue.identifier}
              </span>
              <Badge variant={stateVariant} size="sm">
                {issue.state}
              </Badge>
              {attempt != null && (
                <Badge variant="outline" size="sm">
                  attempt #{attempt}
                </Badge>
              )}
              {issue.labels?.map((l) => (
                <Badge key={l} variant="secondary" size="sm">
                  <TagIcon className="size-2.5" />
                  {l}
                </Badge>
              ))}
            </div>
            <p className="text-sm font-medium leading-snug line-clamp-2">{issue.title}</p>
          </div>
          {issue.url && (
            <Tooltip>
              <TooltipTrigger>
                <a
                  href={issue.url}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="mt-0.5 text-muted-foreground hover:text-foreground transition-colors"
                >
                  <ExternalLinkIcon className="size-3.5" />
                </a>
              </TooltipTrigger>
              <TooltipPopup>Open in Linear</TooltipPopup>
            </Tooltip>
          )}
        </div>
        {/* Phase */}
        <div className="mt-3">
          <PhaseBar phase={phase} />
        </div>
      </CardHeader>

      <CardPanel className="pt-0 space-y-3">
        {/* Metrics */}
        <div className="grid grid-cols-3 gap-2">
          <div className="rounded-lg bg-muted/50 p-2.5 text-center">
            <div className="text-[10px] text-muted-foreground mb-0.5">Turns</div>
            <div className="text-sm font-semibold tabular-nums">{live.turn_count}</div>
          </div>
          <div className="rounded-lg bg-muted/50 p-2.5 text-center">
            <div className="text-[10px] text-muted-foreground mb-0.5">In</div>
            <div className="text-sm font-semibold tabular-nums text-info-foreground">
              {fmtTokens(live.codex_input_tokens)}
            </div>
          </div>
          <div className="rounded-lg bg-muted/50 p-2.5 text-center">
            <div className="text-[10px] text-muted-foreground mb-0.5">Out</div>
            <div className="text-sm font-semibold tabular-nums text-success-foreground">
              {fmtTokens(live.codex_output_tokens)}
            </div>
          </div>
        </div>

        {/* Last event */}
        {(live.last_codex_message || live.last_codex_event || live.last_codex_timestamp) && (
          <div className="rounded-md border bg-muted/30 px-3 py-2">
            <div className="flex items-center gap-1.5 text-[10px] text-muted-foreground mb-1">
              <ActivityIcon className="size-3" />
              <span>{live.last_codex_event ?? "event"}</span>
              {live.last_codex_timestamp && (
                <span className="ml-auto tabular-nums">{fmtAgo(live.last_codex_timestamp)}</span>
              )}
            </div>
            {live.last_codex_message ? (
              <p className="text-xs font-mono leading-relaxed line-clamp-2">
                {live.last_codex_message}
              </p>
            ) : (
              <p className="text-xs text-muted-foreground">—</p>
            )}
          </div>
        )}

        {/* Meta */}
        <div className="flex flex-wrap items-center gap-3 text-xs text-muted-foreground">
          <span className="flex items-center gap-1">
            <TimerIcon className="size-3" />
            {fmtAgo(started_at)}
          </span>
          {issue.branch_name && (
            <span className="flex items-center gap-1 min-w-0">
              <GitBranchIcon className="size-3 shrink-0" />
              <span className="font-mono truncate">{issue.branch_name}</span>
            </span>
          )}
        </div>
        <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
          <FolderIcon className="size-3 shrink-0" />
          <span className="font-mono truncate">{workspace_path}</span>
        </div>

        {/* Expandable details */}
        {(issue.description || (issue.blocked_by && issue.blocked_by.length > 0)) && (
          <Collapsible open={open} onOpenChange={setOpen}>
            <CollapsibleTrigger className="flex w-full items-center gap-1.5 py-1 text-xs text-muted-foreground hover:text-foreground transition-colors">
              {open ? (
                <ChevronDownIcon className="size-3.5" />
              ) : (
                <ChevronRightIcon className="size-3.5" />
              )}
              {open ? "Hide details" : "Show details"}
            </CollapsibleTrigger>
            <CollapsiblePanel>
              <div className="pt-2 space-y-2">
                {issue.description && (
                  <p className="text-xs text-muted-foreground leading-relaxed">
                    {issue.description}
                  </p>
                )}
                {issue.blocked_by && issue.blocked_by.length > 0 && (
                  <div>
                    <p className="text-xs font-medium mb-1">Blocked by</p>
                    <div className="flex flex-wrap gap-1.5">
                      {issue.blocked_by.map((b) => (
                        <Badge key={b.id ?? b.identifier} variant="warning" size="sm">
                          {b.identifier ?? b.id}
                          {b.state && <span className="opacity-60 ml-1">· {b.state}</span>}
                        </Badge>
                      ))}
                    </div>
                  </div>
                )}
              </div>
            </CollapsiblePanel>
          </Collapsible>
        )}
      </CardPanel>
    </Card>
  );
}

// ─── Retry row ────────────────────────────────────────────────────────────────
function RetryRow({ entry }: { entry: SymphonyRetryEntry }) {
  return (
    <TableRow>
      <TableCell className="font-mono text-xs font-semibold text-[var(--lagoon-deep)]">
        {entry.identifier}
      </TableCell>
      <TableCell className="tabular-nums text-sm text-muted-foreground">#{entry.attempt}</TableCell>
      <TableCell>
        <Tooltip>
          <TooltipTrigger>
            <span className="text-sm font-medium text-warning-foreground">
              {fmtDueIn(entry.due_at)}
            </span>
          </TooltipTrigger>
          <TooltipPopup>{new Date(entry.due_at).toLocaleString()}</TooltipPopup>
        </Tooltip>
      </TableCell>
      <TableCell>
        {entry.delay_type ? (
          <Badge variant="outline" size="sm">
            {entry.delay_type}
          </Badge>
        ) : (
          <span className="text-muted-foreground text-sm">—</span>
        )}
      </TableCell>
      <TableCell className="max-w-xs">
        {entry.error ? (
          <Tooltip>
            <TooltipTrigger>
              <span className="block max-w-[200px] truncate font-mono text-xs text-destructive-foreground">
                {entry.error}
              </span>
            </TooltipTrigger>
            <TooltipPopup className="max-w-sm break-words">{entry.error}</TooltipPopup>
          </Tooltip>
        ) : (
          <span className="text-muted-foreground text-sm">—</span>
        )}
      </TableCell>
    </TableRow>
  );
}

function ActivityRow({ entry }: { entry: ActivityItem }) {
  return (
    <TableRow>
      <TableCell className="font-mono text-xs font-semibold text-[var(--lagoon-deep)]">
        {entry.identifier}
      </TableCell>
      <TableCell className="tabular-nums text-sm text-muted-foreground">
        <Tooltip>
          <TooltipTrigger>{fmtAgo(entry.ts)}</TooltipTrigger>
          <TooltipPopup>{new Date(entry.ts).toLocaleString()}</TooltipPopup>
        </Tooltip>
      </TableCell>
      <TableCell className="font-mono text-xs text-muted-foreground">{entry.event}</TableCell>
      <TableCell className="max-w-xs">
        {entry.message ? (
          <Tooltip>
            <TooltipTrigger>
              <span className="block max-w-[360px] truncate font-mono text-xs">
                {entry.message}
              </span>
            </TooltipTrigger>
            <TooltipPopup className="max-w-md break-words">{entry.message}</TooltipPopup>
          </Tooltip>
        ) : (
          <span className="text-muted-foreground text-sm">—</span>
        )}
      </TableCell>
    </TableRow>
  );
}

// ─── Completed row ────────────────────────────────────────────────────────────
function CompletedRow({ entry }: { entry: SymphonyCompletedEntry }) {
  const { issue } = entry;
  const v = statusVariant(entry.status);
  const title = issue.title?.trim() ? issue.title : "Untitled";
  return (
    <TableRow>
      <TableCell>
        <div className="flex items-start gap-2">
          <div className="min-w-0">
            <div className="flex items-center gap-2">
              <span className="font-mono text-xs font-semibold text-[var(--lagoon-deep)]">
                {issue.identifier}
              </span>
              <Badge variant={v} size="sm">
                {entry.status}
              </Badge>
              {entry.attempt != null && (
                <Badge variant="outline" size="sm">
                  #{entry.attempt}
                </Badge>
              )}
            </div>
            <div className="mt-1 text-sm font-medium leading-snug line-clamp-1">{title}</div>
          </div>
          {issue.url && (
            <Tooltip>
              <TooltipTrigger>
                <a
                  href={issue.url}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="mt-0.5 text-muted-foreground hover:text-foreground transition-colors"
                >
                  <ExternalLinkIcon className="size-3.5" />
                </a>
              </TooltipTrigger>
              <TooltipPopup>Open in Linear</TooltipPopup>
            </Tooltip>
          )}
        </div>
      </TableCell>
      <TableCell>
        <Tooltip>
          <TooltipTrigger>
            <span className="text-sm font-medium tabular-nums text-muted-foreground">
              {fmtAgo(entry.ended_at)}
            </span>
          </TooltipTrigger>
          <TooltipPopup>{new Date(entry.ended_at).toLocaleString()}</TooltipPopup>
        </Tooltip>
      </TableCell>
      <TableCell className="tabular-nums text-sm text-muted-foreground">
        {fmtSeconds(entry.duration_secs)}
      </TableCell>
      <TableCell>
        <Tooltip>
          <TooltipTrigger>
            <span className="text-sm font-semibold tabular-nums">{fmtTokens(entry.codex_total_tokens)}</span>
          </TooltipTrigger>
          <TooltipPopup className="font-mono text-xs">
            in={entry.codex_input_tokens} out={entry.codex_output_tokens} total={entry.codex_total_tokens} turns={entry.turns_run}
          </TooltipPopup>
        </Tooltip>
      </TableCell>
      <TableCell className="max-w-xs">
        {entry.error ? (
          <Tooltip>
            <TooltipTrigger>
              <span className="block max-w-[260px] truncate font-mono text-xs text-destructive-foreground">
                {entry.error}
              </span>
            </TooltipTrigger>
            <TooltipPopup className="max-w-md break-words">{entry.error}</TooltipPopup>
          </Tooltip>
        ) : (
          <span className="text-muted-foreground text-sm">—</span>
        )}
      </TableCell>
    </TableRow>
  );
}

// ─── Token usage ──────────────────────────────────────────────────────────────
function TokenUsage({ totals }: { totals: CodexTotals }) {
  const inputPct = totals.total_tokens > 0 ? (totals.input_tokens / totals.total_tokens) * 100 : 0;
  const outputPct =
    totals.total_tokens > 0 ? (totals.output_tokens / totals.total_tokens) * 100 : 0;
  return (
    <div className="grid gap-4 sm:grid-cols-2">
      <div className="space-y-3">
        <Progress value={inputPct} max={100}>
          <div className="flex items-center justify-between text-xs mb-1.5">
            <ProgressLabel>Input Tokens</ProgressLabel>
            <span className="tabular-nums">{fmtTokens(totals.input_tokens)}</span>
          </div>
          <ProgressTrack>
            <ProgressIndicator className="bg-info/70" />
          </ProgressTrack>
        </Progress>
        <Progress value={outputPct} max={100}>
          <div className="flex items-center justify-between text-xs mb-1.5">
            <ProgressLabel>Output Tokens</ProgressLabel>
            <span className="tabular-nums">{fmtTokens(totals.output_tokens)}</span>
          </div>
          <ProgressTrack>
            <ProgressIndicator className="bg-success/70" />
          </ProgressTrack>
        </Progress>
      </div>
      <div className="grid grid-cols-2 gap-3">
        <div className="rounded-xl border bg-muted/30 p-4">
          <div className="text-xs text-muted-foreground mb-1">Total</div>
          <div className="text-xl font-semibold tabular-nums">{fmtTokens(totals.total_tokens)}</div>
        </div>
        <div className="rounded-xl border bg-muted/30 p-4">
          <div className="text-xs text-muted-foreground mb-1">Uptime</div>
          <div className="text-xl font-semibold tabular-nums">
            {fmtSeconds(totals.seconds_running)}
          </div>
        </div>
      </div>
    </div>
  );
}

// ─── Stat tile ────────────────────────────────────────────────────────────────
function StatTile({
  icon: Icon,
  label,
  value,
  tint,
  loading,
}: {
  icon: React.ElementType;
  label: string;
  value: string | number;
  tint?: "success" | "warning" | "info";
  loading?: boolean;
}) {
  const tintCls = {
    success: "text-success bg-success/8",
    warning: "text-warning-foreground bg-warning/8",
    info: "text-info bg-info/8",
  };
  return (
    <div className="rounded-xl border bg-card p-4">
      <div className="flex items-center gap-2 mb-3">
        <div
          className={cn(
            "rounded-md p-1.5",
            tint ? tintCls[tint] : "text-muted-foreground bg-muted/60",
          )}
        >
          <Icon className="size-3.5" />
        </div>
        <span className="text-xs font-medium text-muted-foreground">{label}</span>
      </div>
      {loading ? (
        <Skeleton className="h-7 w-16" />
      ) : (
        <div className="text-2xl font-semibold tabular-nums">{value}</div>
      )}
    </div>
  );
}

// ─── Start dialog ─────────────────────────────────────────────────────────────
function StartDialog({
  onStart,
  isPending,
}: {
  onStart: (path?: string, port?: number) => void;
  isPending: boolean;
}) {
  const [open, setOpen] = useState(false);
  const [path, setPath] = useState("");
  const [port, setPort] = useState("");

  const submit = useCallback(
    (e: React.FormEvent) => {
      e.preventDefault();
      onStart(path.trim() || undefined, port.trim() ? parseInt(port.trim(), 10) : undefined);
      setOpen(false);
    },
    [onStart, path, port],
  );

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger
        render={
          <Button disabled={isPending} className="gap-2">
            {isPending ? <Spinner className="size-4" /> : <PlayCircleIcon className="size-4" />}
            Start Symphony
          </Button>
        }
      />
      <DialogPopup className="w-full max-w-md" showCloseButton={false}>
        <DialogHeader>
          <DialogTitle>Start Symphony</DialogTitle>
          <DialogDescription>
            Launch the orchestrator. Both fields are optional — leave blank to use server defaults.
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={submit} className="space-y-4 px-6 pb-6">
          <div className="space-y-1.5">
            <Label htmlFor="wf-path">Workflow path</Label>
            <Input
              id="wf-path"
              placeholder="./WORKFLOW.md"
              value={path}
              onChange={(e) => setPath(e.target.value)}
              className="font-mono text-sm"
            />
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="dbg-port">Debug HTTP port</Label>
            <Input
              id="dbg-port"
              type="number"
              placeholder="e.g. 8089  ·  -1 to disable"
              value={port}
              onChange={(e) => setPort(e.target.value)}
              className="font-mono text-sm"
            />
          </div>
          <div className="flex justify-end gap-2 pt-2">
            <DialogClose
              render={
                <Button variant="ghost" type="button">
                  Cancel
                </Button>
              }
            />
            <Button type="submit">
              <PlayCircleIcon className="size-4" />
              Start
            </Button>
          </div>
        </form>
      </DialogPopup>
    </Dialog>
  );
}

// ─── Stop dialog ──────────────────────────────────────────────────────────────
function StopDialog({ onStop, isPending }: { onStop: () => void; isPending: boolean }) {
  return (
    <AlertDialog>
      <AlertDialogTrigger
        render={
          <Button
            variant="outline"
            disabled={isPending}
            className="gap-2 border-destructive/30 text-destructive hover:bg-destructive/8 hover:border-destructive/50"
          >
            {isPending ? <Spinner className="size-4" /> : <PauseCircleIcon className="size-4" />}
            Stop
          </Button>
        }
      />
      <AlertDialogPopup>
        <AlertDialogHeader>
          <AlertDialogTitle>Stop Symphony?</AlertDialogTitle>
          <AlertDialogDescription>
            Signals the orchestrator to stop. Running agents finish their current turn. Scheduled
            retries are cleared.
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogClose render={<Button variant="ghost">Cancel</Button>} />
          <AlertDialogClose
            render={
              <Button
                variant="outline"
                className="border-destructive/30 text-destructive hover:bg-destructive/8"
                onClick={onStop}
              >
                Stop Symphony
              </Button>
            }
          />
        </AlertDialogFooter>
      </AlertDialogPopup>
    </AlertDialog>
  );
}

// ─── Main ─────────────────────────────────────────────────────────────────────
function SymphonyDashboard() {
  const qc = useQueryClient();

  const {
    data: health,
    isLoading: healthLoading,
    error: healthError,
  } = useQuery({
    ...getHealthOptions(),
    refetchInterval: 4000,
  });

  const {
    data: snapshot,
    isLoading: snapLoading,
    error: snapError,
    dataUpdatedAt,
  } = useQuery({
    ...getSymphonySnapshotOptions(),
    refetchInterval: 1000,
  });

  const { mutate: doStart, isPending: isStarting } = useMutation({
    ...startSymphonyMutation(),
    onSuccess: () => {
      toastManager.add({ title: "Symphony started", type: "success" });
      qc.invalidateQueries();
    },
    onError: (e) =>
      toastManager.add({ title: "Failed to start", description: String(e), type: "error" }),
  });

  const { mutate: doStop, isPending: isStopping } = useMutation({
    ...stopSymphonyMutation(),
    onSuccess: () => {
      toastManager.add({ title: "Symphony stopped", type: "success" });
      qc.invalidateQueries();
    },
    onError: (e) =>
      toastManager.add({ title: "Failed to stop", description: String(e), type: "error" }),
  });

  const handleStart = useCallback(
    (path?: string, port?: number) => {
      doStart({ body: { workflow_path: path, http_port: port } });
    },
    [doStart],
  );

  const handleStop = useCallback(() => doStop({}), [doStop]);

  const isRunning = health?.symphony_running ?? false;
  const runningCount = snapshot?.running.length ?? 0;
  const retryingCount = snapshot?.retrying.length ?? 0;
  const completedCount = snapshot?.completed?.length ?? 0;
  const totalTokens = snapshot?.codex_totals.total_tokens ?? 0;
  const secondsRunning = snapshot?.codex_totals.seconds_running ?? 0;
  const rateLimits = snapshot?.rate_limits;
  const hasRateLimits = rateLimits && Object.keys(rateLimits).length > 0;
  const isLoading = healthLoading || snapLoading;
  const activity = snapshot ? collectActivity(snapshot.running) : [];
  const activityCount = activity.length;

  return (
    <SidebarProvider defaultOpen>
      <div className="flex min-h-screen w-full bg-[var(--bg-base)]">
        {/* ── Sidebar ── */}
        <Sidebar>
          <SidebarHeader className="p-4">
            <div className="flex items-center gap-2.5">
              <div className="relative flex size-7 items-center justify-center rounded-lg bg-[var(--lagoon-deep)] text-white shadow-sm">
                <ZapIcon className="size-3.5" />
                {isRunning && (
                  <span className="absolute -right-0.5 -top-0.5 size-2 rounded-full bg-success ring-2 ring-sidebar" />
                )}
              </div>
              <div>
                <div className="text-sm font-semibold text-sidebar-foreground">Synclax</div>
                <div className="text-xs text-muted-foreground">Symphony Orchestrator</div>
              </div>
            </div>
          </SidebarHeader>

          <SidebarSeparator />

          <SidebarContent className="px-2 py-3">
            <SidebarGroup>
              <SidebarGroupLabel>Navigation</SidebarGroupLabel>
              <SidebarGroupContent>
                <SidebarMenu>
                  <SidebarMenuItem>
                    <SidebarMenuButton
                      isActive
                      render={
                        <Link to="/">
                          <LayersIcon className="size-4" />
                          Dashboard
                        </Link>
                      }
                    />
                  </SidebarMenuItem>
                  <SidebarMenuItem>
                    <SidebarMenuButton
                      render={
                        <Link to="/about">
                          <ServerIcon className="size-4" />
                          About
                        </Link>
                      }
                    />
                  </SidebarMenuItem>
                </SidebarMenu>
              </SidebarGroupContent>
            </SidebarGroup>

            <SidebarGroup className="mt-2">
              <SidebarGroupLabel>Status</SidebarGroupLabel>
              <SidebarGroupContent>
                <div className="rounded-lg border bg-sidebar-accent/40 p-3 space-y-2.5">
                  <div className="flex items-center justify-between">
                    <span className="text-xs text-muted-foreground">Orchestrator</span>
                    {healthLoading ? (
                      <Skeleton className="h-4 w-16" />
                    ) : (
                      <Badge variant={isRunning ? "success" : "outline"} size="sm">
                        {isRunning ? "Running" : "Stopped"}
                      </Badge>
                    )}
                  </div>
                  {health?.symphony_workflow_path && (
                    <div>
                      <div className="text-xs text-muted-foreground mb-0.5">Workflow</div>
                      <div className="font-mono text-xs truncate">
                        {health.symphony_workflow_path}
                      </div>
                    </div>
                  )}
                  {health?.symphony_last_error && (
                    <div className="font-mono text-xs text-destructive-foreground line-clamp-2">
                      {health.symphony_last_error}
                    </div>
                  )}
                  <Separator />
                  <div className="flex items-center justify-between">
                    <span className="text-xs text-muted-foreground">Agents</span>
                    <span className="text-xs font-semibold tabular-nums">
                      {runningCount} running
                    </span>
                  </div>
                  <div className="flex items-center justify-between">
                    <span className="text-xs text-muted-foreground">Retry queue</span>
                    <span
                      className={cn(
                        "text-xs font-semibold tabular-nums",
                        retryingCount > 0 && "text-warning-foreground",
                      )}
                    >
                      {retryingCount}
                    </span>
                  </div>
                  <div className="flex items-center justify-between">
                    <span className="text-xs text-muted-foreground">Recent completed</span>
                    <span className="text-xs font-semibold tabular-nums">{completedCount}</span>
                  </div>
                </div>
              </SidebarGroupContent>
            </SidebarGroup>
          </SidebarContent>

          <SidebarFooter className="p-4">
            <Separator className="mb-3" />
            <div className="flex items-center justify-between">
              <span className="text-xs text-muted-foreground">Theme</span>
              <ThemeToggle />
            </div>
          </SidebarFooter>
        </Sidebar>

        {/* ── Main ── */}
        <SidebarInset className="flex flex-col min-w-0">
          {/* Topbar */}
          <header className="sticky top-0 z-20 flex items-center gap-3 border-b border-[var(--line)] bg-[var(--header-bg)] px-4 py-3 backdrop-blur-md">
            <SidebarTrigger className="-ml-1" />
            <Separator orientation="vertical" className="h-5" />
            <div className="flex-1 min-w-0">
              <h1 className="text-sm font-semibold leading-none">Symphony Dashboard</h1>
              {dataUpdatedAt > 0 && (
                <p className="mt-0.5 text-xs text-muted-foreground">
                  Updated {fmtAgo(new Date(dataUpdatedAt).toISOString())}
                </p>
              )}
            </div>
            <Tooltip>
              <TooltipTrigger>
                <button
                  onClick={() => qc.invalidateQueries()}
                  className="rounded-md p-1.5 text-muted-foreground hover:text-foreground hover:bg-muted/60 transition-colors"
                  aria-label="Refresh"
                >
                  <RefreshCwIcon className="size-3.5" />
                </button>
              </TooltipTrigger>
              <TooltipPopup>Refresh</TooltipPopup>
            </Tooltip>
          </header>

          <ScrollArea className="flex-1">
            <div className="p-6 space-y-6 max-w-5xl mx-auto">
              {/* Error */}
              {(healthError || snapError) && (
                <Alert variant="error">
                  <AlertCircleIcon className="size-4" />
                  <AlertTitle>Connection error</AlertTitle>
                  <AlertDescription>
                    {String(healthError ?? snapError)}. Is the Synclax server running on{" "}
                    <code className="rounded bg-muted px-1 text-xs">localhost:2910</code>?
                  </AlertDescription>
                </Alert>
              )}

              {/* Control bar */}
              <section className="flex flex-wrap items-center gap-3">
                <div className="flex-1">
                  {isLoading ? (
                    <Skeleton className="h-6 w-40" />
                  ) : (
                    <Badge
                      variant={isRunning ? "success" : "outline"}
                      className="gap-1.5 h-7 px-3 text-sm"
                    >
                      <span
                        className={cn(
                          "size-1.5 rounded-full",
                          isRunning ? "bg-success animate-pulse" : "bg-muted-foreground",
                        )}
                      />
                      {isRunning ? "Orchestrator running" : "Orchestrator stopped"}
                    </Badge>
                  )}
                </div>
                {isRunning ? (
                  <StopDialog onStop={handleStop} isPending={isStopping} />
                ) : (
                  <StartDialog onStart={handleStart} isPending={isStarting} />
                )}
              </section>

              {/* Stat tiles */}
              <section className="grid grid-cols-2 gap-3 sm:grid-cols-4">
                <StatTile
                  icon={CpuIcon}
                  label="Active Agents"
                  value={runningCount}
                  tint={runningCount > 0 ? "success" : undefined}
                  loading={isLoading}
                />
                <StatTile
                  icon={RotateCcwIcon}
                  label="Retry Queue"
                  value={retryingCount}
                  tint={retryingCount > 0 ? "warning" : undefined}
                  loading={isLoading}
                />
                <StatTile
                  icon={ZapIcon}
                  label="Total Tokens"
                  value={fmtTokens(totalTokens)}
                  tint="info"
                  loading={isLoading}
                />
                <StatTile
                  icon={ClockIcon}
                  label="Uptime"
                  value={fmtSeconds(secondsRunning)}
                  loading={isLoading}
                />
              </section>

              {/* Tabs */}
              <Tabs defaultValue="running">
                <TabsList>
                  <TabsTab value="running" className="gap-1.5">
                    <CircleDotIcon className="size-3.5" />
                    Running
                    {runningCount > 0 && (
                      <Badge variant="success" size="sm" className="ml-1 tabular-nums">
                        {runningCount}
                      </Badge>
                    )}
                  </TabsTab>
                  <TabsTab value="activity" className="gap-1.5">
                    <BarChart3Icon className="size-3.5" />
                    Activity
                    {activityCount > 0 && (
                      <Badge variant="secondary" size="sm" className="ml-1 tabular-nums">
                        {activityCount}
                      </Badge>
                    )}
                  </TabsTab>
                  <TabsTab value="retrying" className="gap-1.5">
                    <RotateCcwIcon className="size-3.5" />
                    Retry Queue
                    {retryingCount > 0 && (
                      <Badge variant="warning" size="sm" className="ml-1 tabular-nums">
                        {retryingCount}
                      </Badge>
                    )}
                  </TabsTab>
                  <TabsTab value="completed" className="gap-1.5">
                    <CheckCircle2Icon className="size-3.5" />
                    Completed
                    {completedCount > 0 && (
                      <Badge variant="secondary" size="sm" className="ml-1 tabular-nums">
                        {completedCount}
                      </Badge>
                    )}
                  </TabsTab>
                  <TabsTab value="tokens" className="gap-1.5">
                    <ZapIcon className="size-3.5" />
                    Tokens
                  </TabsTab>
                  {hasRateLimits && (
                    <TabsTab value="rate-limits" className="gap-1.5">
                      <SettingsIcon className="size-3.5" />
                      Rate Limits
                    </TabsTab>
                  )}
                </TabsList>

                {/* Running */}
                <TabsPanel value="running" className="mt-4">
                  {snapLoading ? (
                    <div className="grid gap-3 sm:grid-cols-2">
                      <Skeleton className="h-64 rounded-2xl" />
                      <Skeleton className="h-64 rounded-2xl" />
                    </div>
                  ) : !snapshot?.running.length ? (
                    <div className="rounded-xl border border-dashed p-10 text-center">
                      <div className="mx-auto mb-3 flex size-10 items-center justify-center rounded-full bg-muted">
                        <CpuIcon className="size-5 text-muted-foreground" />
                      </div>
                      <p className="text-sm font-medium mb-1">No agents running</p>
                      <p className="text-xs text-muted-foreground max-w-xs mx-auto leading-relaxed">
                        {isRunning
                          ? "Symphony is running but no issues are being processed. Waiting for candidates from the tracker."
                          : "Start Symphony to begin processing issues from your tracker."}
                      </p>
                    </div>
                  ) : (
                    <div className="grid gap-3 sm:grid-cols-2">
                      {snapshot.running.map((e) => (
                        <AgentCard key={e.issue_id} entry={e} />
                      ))}
                    </div>
                  )}
                </TabsPanel>

                {/* Activity */}
                <TabsPanel value="activity" className="mt-4">
                  {snapLoading ? (
                    <Skeleton className="h-40 rounded-xl" />
                  ) : activity.length === 0 ? (
                    <div className="rounded-xl border border-dashed p-10 text-center">
                      <div className="mx-auto mb-3 flex size-10 items-center justify-center rounded-full bg-muted">
                        <ActivityIcon className="size-5 text-muted-foreground" />
                      </div>
                      <p className="text-sm font-medium mb-1">No activity yet</p>
                      <p className="text-xs text-muted-foreground max-w-sm mx-auto leading-relaxed">
                        Activity is built from live Codex events while agents are running. If
                        you don't see anything, check that there is at least one running issue and
                        that Codex is emitting events.
                      </p>
                    </div>
                  ) : (
                    <Card className="max-h-120 overflow-y-auto">
                      <Table>
                        <TableHeader>
                          <TableRow>
                            <TableHead>Issue</TableHead>
                            <TableHead>When</TableHead>
                            <TableHead>Event</TableHead>
                            <TableHead>Message</TableHead>
                          </TableRow>
                        </TableHeader>
                        <TableBody>
                          {activity.map((e) => (
                            <ActivityRow key={`${e.issueId}-${e.ts}-${e.event}`} entry={e} />
                          ))}
                        </TableBody>
                      </Table>
                    </Card>
                  )}
                </TabsPanel>

                {/* Retrying */}
                <TabsPanel value="retrying" className="mt-4">
                  {snapLoading ? (
                    <Skeleton className="h-40 rounded-xl" />
                  ) : !snapshot?.retrying.length ? (
                    <div className="rounded-xl border border-dashed p-10 text-center">
                      <div className="mx-auto mb-3 flex size-10 items-center justify-center rounded-full bg-muted">
                        <CheckCircle2Icon className="size-5 text-muted-foreground" />
                      </div>
                      <p className="text-sm font-medium mb-1">Retry queue empty</p>
                      <p className="text-xs text-muted-foreground">
                        No issues scheduled for retry.
                      </p>
                    </div>
                  ) : (
                    <Card>
                      <Table>
                        <TableHeader>
                          <TableRow>
                            <TableHead>Issue</TableHead>
                            <TableHead>Attempt</TableHead>
                            <TableHead>Due</TableHead>
                            <TableHead>Type</TableHead>
                            <TableHead>Error</TableHead>
                          </TableRow>
                        </TableHeader>
                        <TableBody>
                          {snapshot.retrying.map((e) => (
                            <RetryRow key={`${e.issue_id}-${e.attempt}`} entry={e} />
                          ))}
                        </TableBody>
                      </Table>
                    </Card>
                  )}
                </TabsPanel>

                {/* Completed */}
                <TabsPanel value="completed" className="mt-4">
                  {snapLoading ? (
                    <Skeleton className="h-40 rounded-xl" />
                  ) : !snapshot?.completed?.length ? (
                    <div className="rounded-xl border border-dashed p-10 text-center">
                      <div className="mx-auto mb-3 flex size-10 items-center justify-center rounded-full bg-muted">
                        <CheckCircle2Icon className="size-5 text-muted-foreground" />
                      </div>
                      <p className="text-sm font-medium mb-1">No completed attempts yet</p>
                      <p className="text-xs text-muted-foreground max-w-sm mx-auto leading-relaxed">
                        Completed attempts appear here after an agent run finishes. This is a
                        best-effort recent history (persisted under{" "}
                        <span className="font-mono">.symphony_state/attempts.jsonl</span>).
                      </p>
                    </div>
                  ) : (
                    <Card>
                      <Table>
                        <TableHeader>
                          <TableRow>
                            <TableHead>Issue</TableHead>
                            <TableHead>Ended</TableHead>
                            <TableHead>Duration</TableHead>
                            <TableHead>Tokens</TableHead>
                            <TableHead>Error</TableHead>
                          </TableRow>
                        </TableHeader>
                        <TableBody>
                          {snapshot.completed.map((e) => (
                            <CompletedRow
                              key={`${e.issue_id}-${e.ended_at}-${e.attempt ?? 0}`}
                              entry={e}
                            />
                          ))}
                        </TableBody>
                      </Table>
                    </Card>
                  )}
                </TabsPanel>

                {/* Tokens */}
                <TabsPanel value="tokens" className="mt-4">
                  <Card>
                    <CardHeader>
                      <CardTitle>Token Usage</CardTitle>
                      <CardDescription>
                        Cumulative Codex token consumption across all agent sessions.
                      </CardDescription>
                    </CardHeader>
                    <CardPanel>
                      {snapLoading ? (
                        <div className="space-y-3">
                          <Skeleton className="h-8 w-full" />
                          <Skeleton className="h-8 w-full" />
                        </div>
                      ) : snapshot ? (
                        <TokenUsage totals={snapshot.codex_totals} />
                      ) : (
                        <p className="text-sm text-muted-foreground">No data.</p>
                      )}
                    </CardPanel>
                  </Card>
                </TabsPanel>

                {/* Rate limits */}
                {hasRateLimits && (
                  <TabsPanel value="rate-limits" className="mt-4">
                    <Card>
                      <CardHeader>
                        <CardTitle>Rate Limits</CardTitle>
                        <CardDescription>
                          Active rate limiting state reported by Symphony.
                        </CardDescription>
                      </CardHeader>
                      <CardPanel>
                        <div className="grid gap-2 sm:grid-cols-2">
                          {Object.entries(rateLimits!).map(([k, v]) => (
                            <div
                              key={k}
                              className="flex items-center justify-between rounded-lg border bg-muted/30 px-3 py-2.5"
                            >
                              <span className="font-mono text-xs text-muted-foreground">{k}</span>
                              <span className="text-xs font-semibold tabular-nums">
                                {String(v)}
                              </span>
                            </div>
                          ))}
                        </div>
                      </CardPanel>
                    </Card>
                  </TabsPanel>
                )}
              </Tabs>
            </div>
          </ScrollArea>
        </SidebarInset>
      </div>
    </SidebarProvider>
  );
}
