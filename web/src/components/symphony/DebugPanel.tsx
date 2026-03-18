import { memo, useState, useCallback } from "react";
import {
  AlertCircleIcon,
  CopyIcon,
  FileTextIcon,
  RefreshCwIcon,
  ServerIcon,
  TerminalIcon,
  InfoIcon,
} from "lucide-react";
import { useDebugState, useDebugHealthz, useDebugRefresh } from "#/hooks/use-debug-server";
import { Card, CardHeader, CardTitle, CardDescription, CardPanel } from "#/components/ui/card";
import { Badge } from "#/components/ui/badge";
import { Button } from "#/components/ui/button";
import { Separator } from "#/components/ui/separator";
import { Skeleton } from "#/components/ui/skeleton";
import { Alert, AlertTitle, AlertDescription } from "#/components/ui/alert";
import { Collapsible, CollapsibleTrigger, CollapsiblePanel } from "#/components/ui/collapsible";
import { Tooltip, TooltipTrigger, TooltipPopup } from "#/components/ui/tooltip";
import { Popover, PopoverTrigger, PopoverPopup, PopoverTitle, PopoverDescription } from "#/components/ui/popover";
import { Frame, FramePanel, FrameTitle, FrameDescription } from "#/components/ui/frame";
import { Spinner } from "#/components/ui/spinner";
import { Meter, MeterTrack, MeterIndicator, MeterLabel, MeterValue } from "#/components/ui/meter";
import { toastManager } from "#/components/ui/toast";
import { cn } from "#/lib/utils";

export const DebugPanel = memo(function DebugPanel({
  port,
}: {
  port: number | null | undefined;
}) {
  const {
    data: state,
    isLoading: stateLoading,
    error: stateError,
  } = useDebugState(port);

  const {
    data: healthz,
    error: healthzError,
  } = useDebugHealthz(port);

  const { mutate: doRefresh, isPending: isRefreshing } = useDebugRefresh(port);

  const [rawOpen, setRawOpen] = useState(false);

  const handleCopyRaw = useCallback(() => {
    if (state) {
      navigator.clipboard.writeText(JSON.stringify(state, null, 2));
      toastManager.add({ title: "Copied to clipboard", type: "success" });
    }
  }, [state]);

  if (port == null || port <= 0) {
    return (
      <Frame className="border-dashed">
        <FramePanel className="p-10 text-center">
          <div className="mx-auto mb-3 flex size-10 items-center justify-center rounded-full bg-muted">
            <ServerIcon className="size-5 text-muted-foreground" />
          </div>
          <FrameTitle className="text-sm font-medium mb-1">Debug server not configured</FrameTitle>
          <FrameDescription className="text-xs text-muted-foreground max-w-sm mx-auto leading-relaxed">
            Set <code className="rounded bg-muted px-1 font-mono text-xs">server.port</code> in your{" "}
            <code className="rounded bg-muted px-1 font-mono text-xs">WORKFLOW.md</code> front matter,
            or pass a debug HTTP port when starting Symphony.
          </FrameDescription>
        </FramePanel>
      </Frame>
    );
  }

  const isOnline = healthz === "ok";
  const isError = !!stateError || !!healthzError;

  return (
    <div className="space-y-4">
      {/* Connection status & actions */}
      <div className="flex flex-wrap items-center gap-3">
        <div className="flex items-center gap-2 flex-1 min-w-0">
          <Badge
            variant={isError ? "error" : isOnline ? "success" : "outline"}
            className="gap-1.5"
          >
            <span
              className={cn(
                "size-1.5 rounded-full",
                isError
                  ? "bg-destructive"
                  : isOnline
                    ? "bg-success animate-pulse"
                    : "bg-muted-foreground",
              )}
            />
            {isError ? "Unreachable" : isOnline ? "Online" : "Checking…"}
          </Badge>
          <span className="font-mono text-xs text-muted-foreground">
            127.0.0.1:{port}
          </span>

          <Popover>
            <PopoverTrigger
              render={
                <button className="rounded-md p-1 text-muted-foreground/40 hover:text-muted-foreground transition-colors">
                  <InfoIcon className="size-3.5" />
                </button>
              }
            />
            <PopoverPopup sideOffset={8}>
              <PopoverTitle>Debug Server</PopoverTitle>
              <PopoverDescription>
                The debug HTTP server runs alongside Symphony and exposes internal state for
                development. It provides real-time snapshots of running agents, retry queues,
                and token consumption metrics.
              </PopoverDescription>
            </PopoverPopup>
          </Popover>
        </div>
        <Tooltip>
          <TooltipTrigger>
            <Button
              size="sm"
              variant="outline"
              disabled={isRefreshing || !isOnline}
              onClick={() => doRefresh()}
              className="gap-1.5"
            >
              {isRefreshing ? (
                <Spinner className="size-3.5" />
              ) : (
                <RefreshCwIcon className="size-3.5" />
              )}
              Force Refresh
            </Button>
          </TooltipTrigger>
          <TooltipPopup>POST /api/v1/refresh — triggers a poll+dispatch cycle</TooltipPopup>
        </Tooltip>
      </div>

      {/* Error */}
      {isError && (
        <Alert variant="error">
          <AlertCircleIcon className="size-4" />
          <AlertTitle>Cannot reach debug server</AlertTitle>
          <AlertDescription>
            {String(stateError ?? healthzError)}. Ensure the debug server is running on{" "}
            <code className="rounded bg-muted px-1 font-mono text-xs">
              127.0.0.1:{port}
            </code>
            .
          </AlertDescription>
        </Alert>
      )}

      {/* Meta + State */}
      {stateLoading ? (
        <div className="grid gap-3 sm:grid-cols-2">
          <Skeleton className="h-40 rounded-xl" />
          <Skeleton className="h-40 rounded-xl" />
        </div>
      ) : state ? (
        <div className="grid gap-3 sm:grid-cols-2">
          {/* Meta card */}
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2 text-base">
                <FileTextIcon className="size-4" />
                Runtime Meta
              </CardTitle>
            </CardHeader>
            <CardPanel>
              <dl className="space-y-2.5">
                <div className="flex items-center justify-between">
                  <dt className="text-xs text-muted-foreground">Workflow Path</dt>
                  <dd className="font-mono text-xs truncate max-w-[60%] text-right">
                    {state.meta.workflow_path || "—"}
                  </dd>
                </div>
                <Separator />
                <div className="flex items-center justify-between">
                  <dt className="text-xs text-muted-foreground">Config Revision</dt>
                  <dd className="font-mono text-xs font-semibold tabular-nums">
                    {state.meta.revision}
                  </dd>
                </div>
              </dl>
            </CardPanel>
          </Card>

          {/* Quick stats with meters */}
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2 text-base">
                <TerminalIcon className="size-4" />
                Debug Snapshot
              </CardTitle>
              <CardDescription>
                Live data from the debug HTTP server.
              </CardDescription>
            </CardHeader>
            <CardPanel className="space-y-3">
              <div className="grid grid-cols-2 gap-2">
                <StatBlock
                  label="Running"
                  value={state.snapshot.running.length}
                  variant={state.snapshot.running.length > 0 ? "success" : undefined}
                />
                <StatBlock
                  label="Retrying"
                  value={state.snapshot.retrying.length}
                  variant={state.snapshot.retrying.length > 0 ? "warning" : undefined}
                />
                <StatBlock
                  label="Completed"
                  value={state.snapshot.completed?.length ?? 0}
                />
                <StatBlock
                  label="Total Tokens"
                  value={state.snapshot.agent_totals.total_tokens.toLocaleString()}
                />
              </div>

              {/* Capacity meter */}
              {state.snapshot.running.length > 0 && (
                <Meter
                  value={state.snapshot.running.length}
                  min={0}
                  max={Math.max(state.snapshot.running.length + 2, 5)}
                >
                  <div className="flex items-center justify-between mb-1">
                    <MeterLabel className="text-[10px] text-muted-foreground">Agent Capacity</MeterLabel>
                    <MeterValue className="text-[10px] font-mono tabular-nums">
                      {() => `${state.snapshot.running.length} active`}
                    </MeterValue>
                  </div>
                  <MeterTrack className="h-1.5 rounded-full">
                    <MeterIndicator className="bg-success/60 rounded-full" />
                  </MeterTrack>
                </Meter>
              )}
            </CardPanel>
          </Card>
        </div>
      ) : null}

      {/* Raw JSON viewer */}
      {state && (
        <Collapsible open={rawOpen} onOpenChange={setRawOpen}>
          <div className="flex items-center gap-2">
            <CollapsibleTrigger className="flex items-center gap-1.5 text-xs text-muted-foreground hover:text-foreground transition-colors">
              <TerminalIcon className="size-3.5" />
              {rawOpen ? "Hide" : "Show"} raw JSON
            </CollapsibleTrigger>
            {rawOpen && (
              <Tooltip>
                <TooltipTrigger>
                  <button
                    onClick={handleCopyRaw}
                    className="rounded-md p-1 text-muted-foreground hover:text-foreground hover:bg-muted/60 transition-colors"
                    aria-label="Copy JSON"
                  >
                    <CopyIcon className="size-3" />
                  </button>
                </TooltipTrigger>
                <TooltipPopup>Copy to clipboard</TooltipPopup>
              </Tooltip>
            )}
          </div>
          <CollapsiblePanel>
            <pre className="mt-2 max-h-96 overflow-auto rounded-lg border bg-muted/30 p-4 font-mono text-xs leading-relaxed">
              {JSON.stringify(state, null, 2)}
            </pre>
          </CollapsiblePanel>
        </Collapsible>
      )}
    </div>
  );
});

function StatBlock({
  label,
  value,
  variant,
}: {
  label: string;
  value: string | number;
  variant?: "success" | "warning";
}) {
  return (
    <div className="rounded-lg border bg-muted/30 px-3 py-2.5">
      <div className="text-[10px] text-muted-foreground mb-0.5">{label}</div>
      <div
        className={cn(
          "text-sm font-semibold tabular-nums",
          variant === "success" && "text-success-foreground",
          variant === "warning" && "text-warning-foreground",
        )}
      >
        {value}
      </div>
    </div>
  );
}
