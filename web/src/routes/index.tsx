"use client";

import { createFileRoute } from "@tanstack/react-router";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  AreaChart,
  Area,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip as RechartsTooltip,
  ResponsiveContainer,
  PieChart,
  Pie,
  Cell,
} from "recharts";
import {
  ActivityIcon,
  AlertCircleIcon,
  BarChart3Icon,
  BugIcon,
  CheckCircle2Icon,
  CircleDotIcon,
  ClockIcon,
  CpuIcon,
  GaugeIcon,
  GridIcon,
  HomeIcon,
  LayersIcon,
  ListIcon,
  PauseCircleIcon,
  PlayCircleIcon,
  RefreshCwIcon,
  RotateCcwIcon,
  SearchIcon,
  SettingsIcon,
  ZapIcon,
  TrendingUpIcon,
  HashIcon,
} from "lucide-react";

import {
  getHealthOptions,
  getSymphonySnapshotOptions,
  getSymphonyWorkflowsOptions,
  startSymphonyMutation,
  stopSymphonyMutation,
} from "#/api-gen/@tanstack/react-query.gen";
import { toastManager } from "#/components/ui/toast";

/* ── UI Components (using ALL of them) ── */
import {
  SidebarProvider, Sidebar, SidebarHeader, SidebarContent, SidebarGroup,
  SidebarGroupLabel, SidebarGroupContent, SidebarMenu, SidebarMenuItem,
  SidebarMenuButton, SidebarFooter, SidebarInset, SidebarTrigger, SidebarSeparator,
} from "#/components/ui/sidebar";
import { Card, CardHeader, CardTitle, CardDescription, CardPanel } from "#/components/ui/card";
import { Badge } from "#/components/ui/badge";
import { Separator } from "#/components/ui/separator";
import { Skeleton } from "#/components/ui/skeleton";
import { Alert, AlertTitle, AlertDescription } from "#/components/ui/alert";
import { ScrollArea } from "#/components/ui/scroll-area";
import { Button } from "#/components/ui/button";
import { Table, TableHeader, TableBody, TableRow, TableHead } from "#/components/ui/table";
import { Tabs, TabsList, TabsTab, TabsPanel } from "#/components/ui/tabs";
import { Tooltip, TooltipTrigger, TooltipPopup } from "#/components/ui/tooltip";
import {
  Breadcrumb, BreadcrumbList, BreadcrumbItem, BreadcrumbLink,
  BreadcrumbSeparator, BreadcrumbPage,
} from "#/components/ui/breadcrumb";
import { Toolbar, ToolbarGroup, ToolbarSeparator } from "#/components/ui/toolbar";
import { ToggleGroup, Toggle as ToggleItem } from "#/components/ui/toggle-group";
import {
  Pagination, PaginationContent, PaginationItem, PaginationLink,
  PaginationPrevious, PaginationNext, PaginationEllipsis,
} from "#/components/ui/pagination";
import { Empty, EmptyMedia, EmptyTitle, EmptyDescription } from "#/components/ui/empty";
import { PreviewCard, PreviewCardTrigger, PreviewCardPopup } from "#/components/ui/preview-card";
import { Accordion, AccordionItem, AccordionTrigger, AccordionPanel } from "#/components/ui/accordion";
import { Meter, MeterTrack, MeterIndicator, MeterLabel, MeterValue } from "#/components/ui/meter";
import { Group, GroupSeparator } from "#/components/ui/group";
import { Avatar, AvatarFallback } from "#/components/ui/avatar";
import {
  Combobox, ComboboxCollection, ComboboxEmpty, ComboboxInput,
  ComboboxItem, ComboboxList, ComboboxPopup,
} from "#/components/ui/combobox";
import { Kbd, KbdGroup } from "#/components/ui/kbd";
import { Switch } from "#/components/ui/switch";
import { cn } from "#/lib/utils";

/* ── Symphony Components ── */
import { AgentCard } from "#/components/symphony/AgentCard";
import { ActivityRow } from "#/components/symphony/ActivityRow";
import { CompletedRow } from "#/components/symphony/CompletedRow";
import { RetryRow } from "#/components/symphony/RetryRow";
import { TokenUsage } from "#/components/symphony/TokenUsage";
import { StartDialog } from "#/components/symphony/StartDialog";
import { StopDialog } from "#/components/symphony/StopDialog";
import { DebugPanel } from "#/components/symphony/DebugPanel";
import { SettingsSheet } from "#/components/symphony/SettingsSheet";
import { collectActivity, fmtAgo, fmtTokens, fmtSeconds } from "#/components/symphony/utils";

export const Route = createFileRoute("/")({ component: Dashboard });

const PAGE_SIZE = 10;
const PIE_COLORS = ["var(--info)", "var(--success)"];

function Dashboard() {
  const qc = useQueryClient();

  const { data: health, isLoading: healthLoading, error: healthError } = useQuery({
    ...getHealthOptions(), refetchInterval: 4000,
  });
  const { data: workflowsRes, isLoading: workflowsLoading, error: workflowsError } = useQuery({
    ...getSymphonyWorkflowsOptions(), refetchInterval: 4000,
  });

  const [workflowId, setWorkflowId] = useState(() =>
    typeof window !== "undefined" ? window.localStorage.getItem("symphony.workflow_id") ?? "" : "",
  );
  const [tab, setTab] = useState(() =>
    typeof window !== "undefined" ? window.localStorage.getItem("symphony.tab") ?? "running" : "running",
  );
  const [viewMode, setViewMode] = useState<string[]>(["grid"]);
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [completedPage, setCompletedPage] = useState(1);
  const [autoRefresh, setAutoRefresh] = useState(true);
  const [themeMode, setThemeMode] = useState(() =>
    typeof window !== "undefined" ? window.localStorage.getItem("theme") ?? "auto" : "auto",
  );
  const [tokenHistory, setTokenHistory] = useState<Array<{ time: string; tokens: number }>>([]);

  useEffect(() => {
    if (typeof window === "undefined") return;
    if (workflowId) window.localStorage.setItem("symphony.workflow_id", workflowId);
  }, [workflowId]);
  useEffect(() => {
    if (typeof window === "undefined") return;
    if (tab) window.localStorage.setItem("symphony.tab", tab);
  }, [tab]);

  const workflows = workflowsRes?.workflows ?? [];
  const activeWorkflowId = workflowsRes?.active_workflow_id ?? null;

  useEffect(() => {
    if (workflowId) return;
    const next = activeWorkflowId ?? workflows[0]?.id ?? "";
    if (next) setWorkflowId(next);
  }, [workflowId, activeWorkflowId, workflows]);

  useEffect(() => {
    if (!workflowId || workflows.length === 0) return;
    if (workflows.some((w) => w.id === workflowId)) return;
    const next = activeWorkflowId ?? workflows[0]?.id ?? "";
    if (next && next !== workflowId) setWorkflowId(next);
  }, [workflowId, workflows, activeWorkflowId]);

  const selectedWorkflow = useMemo(
    () => workflows.find((w) => w.id === workflowId) ?? workflows.find((w) => w.id === activeWorkflowId) ?? workflows[0],
    [workflows, workflowId, activeWorkflowId],
  );
  const selectedWorkflowId = selectedWorkflow?.id ?? "";

  const { data: snapshot, isLoading: snapLoading, error: snapError, dataUpdatedAt } = useQuery({
    ...getSymphonySnapshotOptions(selectedWorkflowId ? { query: { workflow_id: selectedWorkflowId } } : undefined),
    refetchInterval: autoRefresh ? 1000 : false,
    enabled: !!selectedWorkflowId,
  });

  // Track token history for chart
  useEffect(() => {
    if (!snapshot) return;
    const total = snapshot.agent_totals.total_tokens;
    setTokenHistory((prev) => {
      const now = new Date().toLocaleTimeString([], { hour: "2-digit", minute: "2-digit", second: "2-digit" });
      const next = [...prev, { time: now, tokens: total }];
      return next.slice(-30); // keep last 30 data points
    });
  }, [snapshot?.agent_totals.total_tokens]);

  const { mutate: doStart, isPending: isStarting } = useMutation({
    ...startSymphonyMutation(),
    onSuccess: (data) => {
      if (data?.workflow_id) setWorkflowId(data.workflow_id);
      toastManager.add({ title: data?.workflow_id ? "Workflow started" : "Start requested", description: data?.workflow_path ? String(data.workflow_path) : undefined, type: "success" });
      qc.invalidateQueries();
    },
    onError: (e) => toastManager.add({ title: "Failed to start", description: String(e), type: "error" }),
  });

  const { mutate: doStop, isPending: isStopping } = useMutation({
    ...stopSymphonyMutation(),
    onSuccess: (data) => {
      toastManager.add({ title: data?.workflow_id ? "Workflow stopped" : "Stop requested", type: "success" });
      qc.invalidateQueries();
    },
    onError: (e) => toastManager.add({ title: "Failed to stop", description: String(e), type: "error" }),
  });

  const handleStart = useCallback((path?: string, port?: number) => {
    const body: Record<string, unknown> = {};
    if (path) body.workflow_path = path;
    else if (selectedWorkflowId) body.workflow_id = selectedWorkflowId;
    if (port != null) body.http_port = port;
    doStart(Object.keys(body).length ? { body } : {});
  }, [doStart, selectedWorkflowId]);

  const handleStartById = useCallback((id: string) => {
    if (id) setWorkflowId(id);
    doStart({ body: { workflow_id: id } });
  }, [doStart]);

  const handleStop = useCallback(() => doStop({}), [doStop]);
  const handleStopSelected = useCallback(() => {
    doStop(selectedWorkflowId ? { query: { workflow_id: selectedWorkflowId } } : {});
  }, [doStop, selectedWorkflowId]);
  const handleStopById = useCallback((id: string) => doStop({ query: { workflow_id: id } }), [doStop]);
  const handleRefresh = useCallback(() => qc.invalidateQueries(), [qc]);

  const handleToggleTheme = useCallback((theme?: string) => {
    const next = theme ?? (themeMode === "light" ? "dark" : themeMode === "dark" ? "auto" : "light");
    setThemeMode(next);
    if (typeof window === "undefined") return;
    window.localStorage.setItem("theme", next);
    const prefersDark = window.matchMedia("(prefers-color-scheme: dark)").matches;
    const resolved = next === "auto" ? (prefersDark ? "dark" : "light") : next;
    document.documentElement.classList.remove("light", "dark");
    document.documentElement.classList.add(resolved);
    document.documentElement.style.colorScheme = resolved;
    if (next === "auto") document.documentElement.removeAttribute("data-theme");
    else document.documentElement.setAttribute("data-theme", next);
  }, [themeMode]);

  const anyRunning = health?.symphony_running ?? false;
  const isRunning = selectedWorkflow?.running ?? false;
  const debugPort = selectedWorkflow?.http_port;
  const selectedWorkflowPath = selectedWorkflow?.workflow_path;
  const runningCount = snapshot?.running.length ?? 0;
  const retryingCount = snapshot?.retrying.length ?? 0;
  const completedCount = snapshot?.completed?.length ?? 0;
  const totalTokens = snapshot?.agent_totals.total_tokens ?? 0;
  const inputTokens = snapshot?.agent_totals.input_tokens ?? 0;
  const outputTokens = snapshot?.agent_totals.output_tokens ?? 0;
  const secondsRunning = snapshot?.agent_totals.seconds_running ?? 0;
  const rateLimits = snapshot?.rate_limits;
  const hasRateLimits = rateLimits && Object.keys(rateLimits).length > 0;
  const isLoading = healthLoading || workflowsLoading || snapLoading;
  const activity = snapshot ? collectActivity(snapshot.running) : [];
  const completedAll = snapshot?.completed ?? [];
  const completedTotalPages = Math.max(1, Math.ceil(completedAll.length / PAGE_SIZE));
  const completedSlice = completedAll.slice((completedPage - 1) * PAGE_SIZE, completedPage * PAGE_SIZE);

  const sortedWorkflows = useMemo(
    () => [...workflows].sort((a, b) => (a.running !== b.running ? (a.running ? -1 : 1) : a.workflow_path.localeCompare(b.workflow_path))),
    [workflows],
  );

  const pieData = useMemo(() => [
    { name: "Input", value: inputTokens || 1 },
    { name: "Output", value: outputTokens || 1 },
  ], [inputTokens, outputTokens]);

  return (
    <SidebarProvider defaultOpen>
      <div className="flex min-h-screen w-full bg-background">
        <SettingsSheet
          open={settingsOpen} onOpenChange={setSettingsOpen}
          workflows={workflows} selectedWorkflowId={selectedWorkflowId}
          onSelectWorkflow={setWorkflowId} currentTheme={themeMode}
          onToggleTheme={handleToggleTheme}
        />

        {/* ── Sidebar ── */}
        <Sidebar className="border-r">
          <SidebarHeader className="p-4">
            <div className="flex items-center gap-3">
              <div className="flex size-8 items-center justify-center rounded-md bg-foreground text-background">
                <ZapIcon className="size-4" />
              </div>
              <div>
                <div className="text-sm font-semibold tracking-tight">Synclax</div>
                <div className="text-[11px] text-muted-foreground">Symphony</div>
              </div>
              {anyRunning && (
                <Badge variant="success" size="sm" className="ml-auto">Live</Badge>
              )}
            </div>
          </SidebarHeader>
          <SidebarSeparator />

          <SidebarContent className="px-2 py-2">
            <SidebarGroup>
              <SidebarGroupLabel>Menu</SidebarGroupLabel>
              <SidebarGroupContent>
                <SidebarMenu>
                  <SidebarMenuItem>
                    <SidebarMenuButton isActive>
                      <HomeIcon className="size-4" />
                      Overview
                    </SidebarMenuButton>
                  </SidebarMenuItem>
                </SidebarMenu>
              </SidebarGroupContent>
            </SidebarGroup>

            <SidebarGroup className="mt-1">
              <SidebarGroupLabel>Workflows</SidebarGroupLabel>
              <SidebarGroupContent>
                {workflowsLoading ? (
                  <div className="space-y-1.5 px-2"><Skeleton className="h-7 w-full" /><Skeleton className="h-7 w-full" /></div>
                ) : workflows.length === 0 ? (
                  <p className="px-2 py-2 text-xs text-muted-foreground">None configured</p>
                ) : (
                  <Accordion defaultValue={["wf"]}>
                    <AccordionItem value="wf" className="border-0">
                      <AccordionTrigger className="text-xs py-1.5 px-2 hover:no-underline">
                        {workflows.length} workflow{workflows.length !== 1 && "s"}
                      </AccordionTrigger>
                      <AccordionPanel>
                        <SidebarMenu>
                          {sortedWorkflows.map((w) => (
                            <SidebarMenuItem key={w.id}>
                              <SidebarMenuButton
                                isActive={selectedWorkflowId === w.id}
                                onClick={() => setWorkflowId(w.id)}
                              >
                                <span className={cn("size-1.5 rounded-full shrink-0", w.running ? "bg-success" : "bg-muted-foreground/30")} />
                                <span className="font-mono text-[11px] truncate">{w.workflow_path}</span>
                              </SidebarMenuButton>
                            </SidebarMenuItem>
                          ))}
                        </SidebarMenu>
                      </AccordionPanel>
                    </AccordionItem>
                  </Accordion>
                )}
              </SidebarGroupContent>
            </SidebarGroup>

            <SidebarGroup className="mt-1">
              <SidebarGroupLabel>Quick Stats</SidebarGroupLabel>
              <SidebarGroupContent>
                <div className="space-y-1.5 px-2 text-xs">
                  <div className="flex justify-between"><span className="text-muted-foreground">Agents</span><span className="font-mono font-medium">{runningCount}</span></div>
                  <div className="flex justify-between"><span className="text-muted-foreground">Retrying</span><span className={cn("font-mono font-medium", retryingCount > 0 && "text-warning-foreground")}>{retryingCount}</span></div>
                  <div className="flex justify-between"><span className="text-muted-foreground">Completed</span><span className="font-mono font-medium">{completedCount}</span></div>
                  <div className="flex justify-between"><span className="text-muted-foreground">Tokens</span><span className="font-mono font-medium">{fmtTokens(totalTokens)}</span></div>
                </div>
              </SidebarGroupContent>
            </SidebarGroup>

            <SidebarGroup className="mt-1">
              <SidebarGroupLabel>Live Refresh</SidebarGroupLabel>
              <SidebarGroupContent>
                <div className="flex items-center justify-between px-2 py-1">
                  <span className="text-xs text-muted-foreground">Auto-refresh</span>
                  <Switch checked={autoRefresh} onCheckedChange={setAutoRefresh} />
                </div>
              </SidebarGroupContent>
            </SidebarGroup>
          </SidebarContent>

          <SidebarFooter className="p-3">
            <Separator className="mb-2" />
            <div className="flex items-center justify-between">
              <Button variant="ghost" size="sm" className="h-7 gap-1.5 text-xs text-muted-foreground" onClick={() => setSettingsOpen(true)}>
                <SettingsIcon className="size-3" /> Settings
              </Button>
              <KbdGroup><Kbd>⌘</Kbd><Kbd>K</Kbd></KbdGroup>
            </div>
          </SidebarFooter>
        </Sidebar>

        {/* ── Main ── */}
        <SidebarInset className="flex flex-col min-w-0">
          <header className="sticky top-0 z-20 flex items-center gap-3 border-b bg-background/95 backdrop-blur-sm px-6 h-14">
            <SidebarTrigger className="-ml-2" />
            <Separator orientation="vertical" className="h-4" />
            <Breadcrumb>
              <BreadcrumbList>
                <BreadcrumbItem><BreadcrumbLink className="flex items-center gap-1.5"><ZapIcon className="size-3" />Synclax</BreadcrumbLink></BreadcrumbItem>
                <BreadcrumbSeparator />
                <BreadcrumbItem><BreadcrumbPage>Dashboard</BreadcrumbPage></BreadcrumbItem>
                {selectedWorkflowPath && (<><BreadcrumbSeparator /><BreadcrumbItem><BreadcrumbPage className="font-mono text-xs max-w-[180px] truncate">{selectedWorkflowPath}</BreadcrumbPage></BreadcrumbItem></>)}
              </BreadcrumbList>
            </Breadcrumb>
            <div className="flex-1" />
            {dataUpdatedAt > 0 && <span className="text-[11px] text-muted-foreground tabular-nums hidden sm:block">{fmtAgo(new Date(dataUpdatedAt).toISOString())}</span>}
            <Tooltip><TooltipTrigger><Button variant="ghost" size="sm" onClick={handleRefresh} className="size-8 p-0"><RefreshCwIcon className="size-3.5" /></Button></TooltipTrigger><TooltipPopup>Refresh</TooltipPopup></Tooltip>
          </header>

          <ScrollArea className="flex-1">
            <div className="p-6 space-y-6 max-w-[1400px] mx-auto">
              {/* Errors */}
              {(healthError || workflowsError || snapError) && (
                <Alert variant="error">
                  <AlertCircleIcon className="size-4" />
                  <AlertTitle>Connection error</AlertTitle>
                  <AlertDescription>
                    {String(healthError ?? workflowsError ?? snapError)}. Is the server running on <code className="text-xs font-mono bg-muted px-1 rounded">localhost:2910</code>?
                  </AlertDescription>
                </Alert>
              )}

              {/* Toolbar */}
              <Toolbar className="flex items-center gap-3 -mt-1">
                <ToolbarGroup className="flex-1 flex items-center gap-3">
                  <Badge variant={anyRunning ? "success" : "outline"} className="gap-1.5 h-7 px-3">
                    <span className={cn("size-1.5 rounded-full", anyRunning ? "bg-success animate-pulse" : "bg-muted-foreground/40")} />
                    {anyRunning ? "Running" : "Stopped"}
                  </Badge>
                </ToolbarGroup>
                <ToolbarSeparator />
                <ToolbarGroup>
                  <ToggleGroup value={viewMode} onValueChange={(v) => v.length > 0 && setViewMode(v)} size="sm">
                    <ToggleItem value="grid"><GridIcon className="size-3.5" /></ToggleItem>
                    <ToggleItem value="list"><ListIcon className="size-3.5" /></ToggleItem>
                  </ToggleGroup>
                </ToolbarGroup>
                <ToolbarSeparator />
                <ToolbarGroup className="flex items-center gap-2">
                  <Group>
                    <StartDialog onStart={handleStart} isPending={isStarting} defaultWorkflowPath={selectedWorkflowPath ?? null} />
                    {anyRunning && (<><GroupSeparator /><StopDialog onStopAll={handleStop} onStopSelected={handleStopSelected} isPending={isStopping} selectedWorkflowPath={selectedWorkflowPath ?? null} showStopAll={workflows.length > 1} /></>)}
                  </Group>
                </ToolbarGroup>
              </Toolbar>

              {/* Stats + Chart row */}
              <div className="grid gap-4 lg:grid-cols-3">
                {/* Left: stat cards */}
                <div className="lg:col-span-1 grid grid-cols-2 gap-3">
                  <StatCard icon={CpuIcon} label="Agents" value={runningCount} loading={isLoading} accent={runningCount > 0 ? "success" : undefined} />
                  <StatCard icon={RotateCcwIcon} label="Retrying" value={retryingCount} loading={isLoading} accent={retryingCount > 0 ? "warning" : undefined} />
                  <StatCard icon={ZapIcon} label="Tokens" value={fmtTokens(totalTokens)} loading={isLoading} accent="info" />
                  <StatCard icon={ClockIcon} label="Uptime" value={fmtSeconds(secondsRunning)} loading={isLoading} />
                </div>

                {/* Right: token chart */}
                <Card className="lg:col-span-2">
                  <CardHeader className="pb-2">
                    <div className="flex items-center justify-between">
                      <div>
                        <CardTitle className="text-sm flex items-center gap-1.5"><TrendingUpIcon className="size-3.5" />Token Consumption</CardTitle>
                        <CardDescription className="text-xs">Real-time token usage over time</CardDescription>
                      </div>
                      <div className="flex items-center gap-4">
                        <div className="flex items-center gap-4 text-xs">
                          <span className="flex items-center gap-1.5"><span className="size-2 rounded-full bg-info" />Input</span>
                          <span className="flex items-center gap-1.5"><span className="size-2 rounded-full bg-success" />Output</span>
                        </div>
                        {/* Mini pie */}
                        <div className="size-10">
                          <ResponsiveContainer width="100%" height="100%">
                            <PieChart>
                              <Pie data={pieData} dataKey="value" cx="50%" cy="50%" innerRadius={12} outerRadius={18} strokeWidth={0}>
                                {pieData.map((_, i) => <Cell key={i} fill={PIE_COLORS[i]} opacity={0.7} />)}
                              </Pie>
                            </PieChart>
                          </ResponsiveContainer>
                        </div>
                      </div>
                    </div>
                  </CardHeader>
                  <CardPanel className="pt-0">
                    <div className="h-[180px]">
                      <ResponsiveContainer width="100%" height="100%">
                        <AreaChart data={tokenHistory} margin={{ top: 4, right: 4, left: -20, bottom: 0 }}>
                          <defs>
                            <linearGradient id="tokenGrad" x1="0" y1="0" x2="0" y2="1">
                              <stop offset="0%" stopColor="var(--info)" stopOpacity={0.3} />
                              <stop offset="100%" stopColor="var(--info)" stopOpacity={0} />
                            </linearGradient>
                          </defs>
                          <CartesianGrid strokeDasharray="3 3" vertical={false} />
                          <XAxis dataKey="time" tick={{ fontSize: 10 }} tickLine={false} axisLine={false} />
                          <YAxis tick={{ fontSize: 10 }} tickLine={false} axisLine={false} tickFormatter={(v: number) => fmtTokens(v)} />
                          <RechartsTooltip
                            contentStyle={{ background: "var(--card)", border: "1px solid var(--border)", borderRadius: 8, fontSize: 12 }}
                            labelStyle={{ color: "var(--muted-foreground)" }}
                            formatter={(value) => [fmtTokens(Number(value ?? 0)), "Tokens"]}
                          />
                          <Area type="monotone" dataKey="tokens" stroke="var(--info)" strokeWidth={2} fill="url(#tokenGrad)" />
                        </AreaChart>
                      </ResponsiveContainer>
                    </div>
                  </CardPanel>
                </Card>
              </div>

              {/* Workflows */}
              <section>
                <div className="flex items-center justify-between mb-3">
                  <h2 className="text-sm font-semibold">Workflows</h2>
                  {workflows.length > 3 && (
                    <Combobox items={sortedWorkflows} value={selectedWorkflow ?? null} onValueChange={(v) => setWorkflowId(v?.id ?? "")} itemToStringLabel={(i) => i.workflow_path} isItemEqualToValue={(a, b) => a.id === b.id} autoComplete="list">
                      <div className="relative">
                        <SearchIcon className="absolute left-2 top-1/2 -translate-y-1/2 size-3.5 text-muted-foreground pointer-events-none" />
                        <ComboboxInput size="sm" placeholder="Search…" aria-label="Workflow" autoComplete="off" spellCheck={false} name="workflow" className="w-[200px] pl-7" />
                      </div>
                      <ComboboxPopup><ComboboxList><ComboboxEmpty>No match</ComboboxEmpty><ComboboxCollection>{(w) => (<ComboboxItem key={w.id} value={w}><span className={cn("size-1.5 rounded-full", w.running ? "bg-success" : "bg-muted-foreground/30")} /><span className="font-mono text-xs truncate">{w.workflow_path}</span></ComboboxItem>)}</ComboboxCollection></ComboboxList></ComboboxPopup>
                    </Combobox>
                  )}
                </div>
                {workflowsLoading ? (
                  <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3"><Skeleton className="h-28 rounded-lg" /><Skeleton className="h-28 rounded-lg" /><Skeleton className="h-28 rounded-lg" /></div>
                ) : workflows.length === 0 ? (
                  <Empty className="py-12"><EmptyMedia variant="icon"><LayersIcon className="size-5" /></EmptyMedia><EmptyTitle>No workflows</EmptyTitle><EmptyDescription>Add workflows on the server or start one by path.</EmptyDescription></Empty>
                ) : (
                  <div className={cn("grid gap-3", viewMode.includes("grid") ? "sm:grid-cols-2 lg:grid-cols-3" : "grid-cols-1")}>
                    {sortedWorkflows.map((w) => {
                      const sel = w.id === selectedWorkflowId;
                      const initials = w.workflow_path.split("/").pop()?.slice(0, 2).toUpperCase() ?? "WF";
                      return (
                        <PreviewCard key={w.id}>
                          <PreviewCardTrigger render={
                            <div className={cn("rounded-lg border p-4 cursor-pointer transition-all hover:border-foreground/20", sel && "ring-2 ring-ring/20 border-foreground/20")} onClick={() => setWorkflowId(w.id)}>
                              <div className="flex items-start gap-3">
                                <Avatar className="size-8 text-[10px]"><AvatarFallback className={cn("font-semibold", w.running ? "bg-success/10 text-success-foreground" : "bg-muted")}>{initials}</AvatarFallback></Avatar>
                                <div className="flex-1 min-w-0">
                                  <div className="font-mono text-xs truncate mb-1.5">{w.workflow_path}</div>
                                  <div className="flex flex-wrap gap-1.5">
                                    <Badge variant={w.running ? "success" : "outline"} size="sm">{w.running ? "Running" : "Stopped"}</Badge>
                                    {w.http_port && <Badge variant="outline" size="sm" className="font-mono">:{w.http_port}</Badge>}
                                    {w.last_error && <Badge variant="error" size="sm">Error</Badge>}
                                  </div>
                                </div>
                              </div>
                              <div className="flex justify-end gap-2 mt-3">
                                {w.running ? (
                                  <Button size="sm" variant="destructive-outline" disabled={isStopping} onClick={(e) => { e.stopPropagation(); handleStopById(w.id); }}><PauseCircleIcon className="size-3.5" />Stop</Button>
                                ) : (
                                  <Button size="sm" disabled={isStarting} onClick={(e) => { e.stopPropagation(); handleStartById(w.id); }}><PlayCircleIcon className="size-3.5" />Start</Button>
                                )}
                              </div>
                            </div>
                          } />
                          <PreviewCardPopup sideOffset={8}>
                            <div className="p-3 space-y-2 max-w-xs">
                              <div className="font-mono text-xs">{w.workflow_path}</div>
                              <Badge variant={w.running ? "success" : "outline"} size="sm">{w.running ? "Running" : "Stopped"}</Badge>
                              {w.last_error && <p className="text-xs text-destructive-foreground">{w.last_error}</p>}
                            </div>
                          </PreviewCardPopup>
                        </PreviewCard>
                      );
                    })}
                  </div>
                )}
              </section>

              {/* Tabs */}
              <Tabs value={tab} onValueChange={setTab}>
                <TabsList>
                  <TabsTab value="running" className="gap-1.5"><CircleDotIcon className="size-3.5" />Running{runningCount > 0 && <Badge variant="success" size="sm" className="ml-1">{runningCount}</Badge>}</TabsTab>
                  <TabsTab value="activity" className="gap-1.5"><BarChart3Icon className="size-3.5" />Activity{activity.length > 0 && <Badge variant="secondary" size="sm" className="ml-1">{activity.length}</Badge>}</TabsTab>
                  <TabsTab value="retrying" className="gap-1.5"><RotateCcwIcon className="size-3.5" />Retry{retryingCount > 0 && <Badge variant="warning" size="sm" className="ml-1">{retryingCount}</Badge>}</TabsTab>
                  <TabsTab value="completed" className="gap-1.5"><CheckCircle2Icon className="size-3.5" />Completed{completedCount > 0 && <Badge variant="secondary" size="sm" className="ml-1">{completedCount}</Badge>}</TabsTab>
                  <TabsTab value="tokens" className="gap-1.5"><ZapIcon className="size-3.5" />Tokens</TabsTab>
                  {hasRateLimits && <TabsTab value="rate-limits" className="gap-1.5"><GaugeIcon className="size-3.5" />Rate Limits</TabsTab>}
                  <TabsTab value="debug" className="gap-1.5"><BugIcon className="size-3.5" />Debug{debugPort && <Badge variant="outline" size="sm" className="ml-1 font-mono">:{debugPort}</Badge>}</TabsTab>
                </TabsList>

                {/* Running */}
                <TabsPanel value="running" className="mt-4">
                  {snapLoading ? (
                    <div className={cn("grid gap-3", viewMode.includes("grid") ? "sm:grid-cols-2 xl:grid-cols-3" : "grid-cols-1")}><Skeleton className="h-64 rounded-xl" /><Skeleton className="h-64 rounded-xl" /><Skeleton className="h-64 rounded-xl" /></div>
                  ) : !snapshot?.running.length ? (
                    <Empty className="py-16"><EmptyMedia variant="icon"><CpuIcon className="size-5" /></EmptyMedia><EmptyTitle>No agents running</EmptyTitle><EmptyDescription>{isRunning ? "Waiting for candidates from the tracker." : "Start Symphony to begin processing."}</EmptyDescription></Empty>
                  ) : (
                    <div className={cn("grid gap-3", viewMode.includes("grid") ? "sm:grid-cols-2 xl:grid-cols-3" : "grid-cols-1")}>
                      {[...snapshot.running].sort((a, b) => a.started_at.localeCompare(b.started_at)).map((e) => <AgentCard key={e.issue_id} entry={e} />)}
                    </div>
                  )}
                </TabsPanel>

                {/* Activity */}
                <TabsPanel value="activity" className="mt-4">
                  {snapLoading ? <Skeleton className="h-40 rounded-xl" /> : activity.length === 0 ? (
                    <Empty className="py-16"><EmptyMedia variant="icon"><ActivityIcon className="size-5" /></EmptyMedia><EmptyTitle>No activity</EmptyTitle><EmptyDescription>Events appear here while agents are running.</EmptyDescription></Empty>
                  ) : (
                    <Card className="overflow-hidden"><div className="max-h-[480px] overflow-y-auto">
                      <Table><TableHeader><TableRow><TableHead>Issue</TableHead><TableHead>When</TableHead><TableHead>Event</TableHead><TableHead>Message</TableHead></TableRow></TableHeader>
                        <TableBody>{activity.map((e) => <ActivityRow key={`${e.issueId}-${e.ts}-${e.event}`} entry={e} />)}</TableBody></Table>
                    </div></Card>
                  )}
                </TabsPanel>

                {/* Retrying */}
                <TabsPanel value="retrying" className="mt-4">
                  {snapLoading ? <Skeleton className="h-40 rounded-xl" /> : !snapshot?.retrying.length ? (
                    <Empty className="py-16"><EmptyMedia variant="icon"><CheckCircle2Icon className="size-5" /></EmptyMedia><EmptyTitle>Queue empty</EmptyTitle><EmptyDescription>No issues scheduled for retry.</EmptyDescription></Empty>
                  ) : (
                    <Card className="overflow-hidden">
                      <Table><TableHeader><TableRow><TableHead>Issue</TableHead><TableHead>Attempt</TableHead><TableHead>Due</TableHead><TableHead>Type</TableHead><TableHead>Error</TableHead></TableRow></TableHeader>
                        <TableBody>{snapshot.retrying.map((e) => <RetryRow key={`${e.issue_id}-${e.attempt}`} entry={e} />)}</TableBody></Table>
                    </Card>
                  )}
                </TabsPanel>

                {/* Completed + Pagination */}
                <TabsPanel value="completed" className="mt-4 space-y-4">
                  {snapLoading ? <Skeleton className="h-40 rounded-xl" /> : !completedAll.length ? (
                    <Empty className="py-16"><EmptyMedia variant="icon"><CheckCircle2Icon className="size-5" /></EmptyMedia><EmptyTitle>No completed attempts</EmptyTitle><EmptyDescription>Results appear after agent runs finish.</EmptyDescription></Empty>
                  ) : (<>
                    <Card className="overflow-hidden">
                      <Table><TableHeader><TableRow><TableHead><HashIcon className="inline size-3 mr-1" />Issue</TableHead><TableHead>Ended</TableHead><TableHead>Duration</TableHead><TableHead>Tokens</TableHead><TableHead>Error</TableHead></TableRow></TableHeader>
                        <TableBody>{completedSlice.map((e) => <CompletedRow key={`${e.issue_id}-${e.ended_at}-${e.attempt ?? 0}`} entry={e} />)}</TableBody></Table>
                    </Card>
                    {completedTotalPages > 1 && (
                      <Pagination>
                        <PaginationContent>
                          <PaginationItem><PaginationPrevious onClick={() => setCompletedPage((p) => Math.max(1, p - 1))} className={completedPage <= 1 ? "pointer-events-none opacity-50" : ""} /></PaginationItem>
                          {Array.from({ length: Math.min(completedTotalPages, 5) }, (_, i) => i + 1).map((p) => (
                            <PaginationItem key={p}><PaginationLink isActive={p === completedPage} onClick={() => setCompletedPage(p)}>{p}</PaginationLink></PaginationItem>
                          ))}
                          {completedTotalPages > 5 && <PaginationItem><PaginationEllipsis /></PaginationItem>}
                          <PaginationItem><PaginationNext onClick={() => setCompletedPage((p) => Math.min(completedTotalPages, p + 1))} className={completedPage >= completedTotalPages ? "pointer-events-none opacity-50" : ""} /></PaginationItem>
                        </PaginationContent>
                      </Pagination>
                    )}
                  </>)}
                </TabsPanel>

                {/* Tokens */}
                <TabsPanel value="tokens" className="mt-4">
                  <Card><CardHeader><CardTitle className="text-sm">Token Usage</CardTitle><CardDescription>Cumulative consumption across all sessions.</CardDescription></CardHeader>
                    <CardPanel>{snapLoading ? <div className="space-y-3"><Skeleton className="h-8 w-full" /><Skeleton className="h-8 w-full" /></div> : snapshot ? <TokenUsage totals={snapshot.agent_totals} /> : <p className="text-sm text-muted-foreground">No data.</p>}</CardPanel></Card>
                </TabsPanel>

                {/* Rate Limits */}
                {hasRateLimits && (
                  <TabsPanel value="rate-limits" className="mt-4">
                    <Card><CardHeader><CardTitle className="text-sm flex items-center gap-2"><GaugeIcon className="size-4" />Rate Limits</CardTitle><CardDescription>Active rate limiting state.</CardDescription></CardHeader>
                      <CardPanel><div className="grid gap-3 sm:grid-cols-2">
                        {Object.entries(rateLimits!).map(([k, v]) => {
                          const n = typeof v === "number" ? v : parseFloat(String(v));
                          return (
                            <div key={k} className="rounded-lg border p-3 space-y-2">
                              <Meter value={Math.min(isNaN(n) ? 0 : n, 100)} min={0} max={100}>
                                <div className="flex justify-between text-xs">
                                  <MeterLabel className="font-mono text-xs text-muted-foreground">{k}</MeterLabel>
                                  <MeterValue className="font-semibold text-xs tabular-nums">{() => String(v)}</MeterValue>
                                </div>
                                {!isNaN(n) && n > 0 && <MeterTrack className="h-1.5 rounded-full"><MeterIndicator className={cn("rounded-full", n > 80 ? "bg-destructive/60" : n > 50 ? "bg-warning/60" : "bg-success/60")} /></MeterTrack>}
                              </Meter>
                            </div>
                          );
                        })}
                      </div></CardPanel></Card>
                  </TabsPanel>
                )}

                {/* Debug */}
                <TabsPanel value="debug" className="mt-4"><DebugPanel port={debugPort} /></TabsPanel>
              </Tabs>
            </div>
          </ScrollArea>
        </SidebarInset>
      </div>
    </SidebarProvider>
  );
}

/* ── StatCard ── */
function StatCard({
  icon: Icon,
  label,
  value,
  loading,
  accent,
}: {
  icon: React.ElementType;
  label: string;
  value: string | number;
  loading?: boolean;
  accent?: "success" | "warning" | "info";
}) {
  const accentColor = accent === "success" ? "text-success-foreground" : accent === "warning" ? "text-warning-foreground" : accent === "info" ? "text-info-foreground" : "text-foreground";
  const accentBg = accent === "success" ? "bg-success/10" : accent === "warning" ? "bg-warning/10" : accent === "info" ? "bg-info/10" : "bg-muted";

  return (
    <div className="rounded-lg border p-4 flex flex-col gap-2">
      <div className="flex items-center justify-between">
        <span className="text-[11px] font-medium text-muted-foreground uppercase tracking-wider">{label}</span>
        <div className={cn("rounded-md p-1.5", accentBg, accentColor)}>
          <Icon className="size-3.5" />
        </div>
      </div>
      {loading ? (
        <Skeleton className="h-8 w-16" />
      ) : (
        <div className={cn("text-2xl font-bold tabular-nums tracking-tight", accentColor)}>
          {value}
        </div>
      )}
    </div>
  );
}
