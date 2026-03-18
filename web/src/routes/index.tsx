"use client";

import { createFileRoute, Link } from "@tanstack/react-router";
import { useCallback } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  ActivityIcon,
  AlertCircleIcon,
  BarChart3Icon,
  BugIcon,
  CheckCircle2Icon,
  CircleDotIcon,
  ClockIcon,
  CpuIcon,
  LayersIcon,
  RefreshCwIcon,
  RotateCcwIcon,
  ServerIcon,
  SettingsIcon,
  ZapIcon,
} from "lucide-react";

import {
  getHealthOptions,
  getSymphonySnapshotOptions,
  startSymphonyMutation,
  stopSymphonyMutation,
} from "#/api-gen/@tanstack/react-query.gen";
import { toastManager } from "#/components/ui/toast";

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
import { Separator } from "#/components/ui/separator";
import { Skeleton } from "#/components/ui/skeleton";
import { Alert, AlertTitle, AlertDescription } from "#/components/ui/alert";
import { ScrollArea } from "#/components/ui/scroll-area";
import {
  Table,
  TableHeader,
  TableBody,
  TableRow,
  TableHead,
} from "#/components/ui/table";
import { Tabs, TabsList, TabsTab, TabsPanel } from "#/components/ui/tabs";
import { Tooltip, TooltipTrigger, TooltipPopup } from "#/components/ui/tooltip";
import ThemeToggle from "#/components/ThemeToggle";
import { cn } from "#/lib/utils";

import { AgentCard } from "#/components/symphony/AgentCard";
import { ActivityRow } from "#/components/symphony/ActivityRow";
import { CompletedRow } from "#/components/symphony/CompletedRow";
import { RetryRow } from "#/components/symphony/RetryRow";
import { StatTile } from "#/components/symphony/StatTile";
import { TokenUsage } from "#/components/symphony/TokenUsage";
import { StartDialog } from "#/components/symphony/StartDialog";
import { StopDialog } from "#/components/symphony/StopDialog";
import { DebugPanel } from "#/components/symphony/DebugPanel";
import { collectActivity, fmtAgo, fmtTokens, fmtSeconds } from "#/components/symphony/utils";

export const Route = createFileRoute("/")({ component: SymphonyDashboard });

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
  const handleRefresh = useCallback(() => qc.invalidateQueries(), [qc]);

  const isRunning = health?.symphony_running ?? false;
  const debugPort = health?.symphony_http_port;
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
                  onClick={handleRefresh}
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
            <div className="p-6 space-y-6 max-w-7xl mx-auto">
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
                  <TabsTab value="debug" className="gap-1.5">
                    <BugIcon className="size-3.5" />
                    Debug
                    {debugPort && (
                      <Badge variant="outline" size="sm" className="ml-1 tabular-nums font-mono">
                        :{debugPort}
                      </Badge>
                    )}
                  </TabsTab>
                </TabsList>

                {/* Running */}
                <TabsPanel value="running" className="mt-4">
                  {snapLoading ? (
                    <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
                      <Skeleton className="h-64 rounded-2xl" />
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
                    <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
                      {[...snapshot.running]
                        .sort((a, b) => a.started_at.localeCompare(b.started_at))
                        .map((e) => (
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
                        Activity is built from live Codex events while agents are running. If you
                        don't see anything, check that there is at least one running issue and that
                        Codex is emitting events.
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

                {/* Debug */}
                <TabsPanel value="debug" className="mt-4">
                  <DebugPanel port={debugPort} />
                </TabsPanel>
              </Tabs>
            </div>
          </ScrollArea>
        </SidebarInset>
      </div>
    </SidebarProvider>
  );
}
