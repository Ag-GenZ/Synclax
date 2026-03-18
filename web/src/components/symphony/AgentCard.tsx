import { memo, useState, useRef, useEffect } from "react";
import {
  ActivityIcon,
  ChevronDownIcon,
  ChevronRightIcon,
  ExternalLinkIcon,
  FolderIcon,
  GitBranchIcon,
  TagIcon,
  TimerIcon,
} from "lucide-react";
import type { LiveEvent, SymphonyRunningEntry } from "#/api-gen/types.gen";
import { Card, CardHeader, CardPanel } from "#/components/ui/card";
import { Badge } from "#/components/ui/badge";
import { Collapsible, CollapsibleTrigger, CollapsiblePanel } from "#/components/ui/collapsible";
import { Tooltip, TooltipTrigger, TooltipPopup } from "#/components/ui/tooltip";
import { fmtAgo, fmtTokens } from "./utils";
import { PhaseBar } from "./PhaseBar";

type StreamBlock = { event: string; text: string };

function simplifyEvent(event: string): string {
  const last = event.split("/").at(-1) ?? event;
  return last.replace(/([A-Z])/g, (c) => `·${c.toLowerCase()}`);
}

function buildStreamBlocks(log: Array<LiveEvent>, maxBlocks = 30): Array<StreamBlock> {
  if (!log.length) return [];
  const runs: Array<StreamBlock> = [];
  let cur: StreamBlock | null = null;
  for (const ev of log) {
    const msg = ev.message ?? "";
    if (!cur || cur.event !== ev.event) {
      cur = { event: ev.event, text: msg };
      runs.push(cur);
    } else {
      cur.text += msg;
    }
  }
  return runs.filter((b) => b.text.trim()).slice(-maxBlocks);
}

export const AgentCard = memo(function AgentCard({ entry }: { entry: SymphonyRunningEntry }) {
  const [open, setOpen] = useState(false);
  const streamRef = useRef<HTMLDivElement>(null);
  const { issue, live, phase, started_at, workspace_path, attempt } = entry;

  const blocks = buildStreamBlocks(live.event_log ?? []);
  const hasLiveEvent = blocks.length > 0 || live.last_codex_message || live.last_codex_event;

  useEffect(() => {
    if (streamRef.current) {
      streamRef.current.scrollTop = streamRef.current.scrollHeight;
    }
  }, [blocks.at(-1)?.text, live.last_codex_message]);

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

        {/* Live event */}
        {hasLiveEvent && (
          <div className="rounded-md border bg-muted/30 px-3 py-2">
            <div className="flex items-center gap-1.5 text-[10px] text-muted-foreground mb-1.5">
              <ActivityIcon className="size-3" />
              <span className="font-mono">
                {simplifyEvent(blocks.at(-1)?.event ?? live.last_codex_event ?? "event")}
              </span>
              {live.last_codex_timestamp && (
                <span className="ml-auto tabular-nums">{fmtAgo(live.last_codex_timestamp)}</span>
              )}
            </div>
            {blocks.length > 0 ? (
              <div
                ref={streamRef}
                className="max-h-72 overflow-y-auto scroll-smooth space-y-1.5"
              >
                {blocks.map((block, i) => {
                  const isLast = i === blocks.length - 1;
                  return (
                    <p
                      key={i}
                      className={
                        isLast
                          ? "text-xs leading-relaxed"
                          : "text-xs leading-relaxed text-muted-foreground"
                      }
                    >
                      {block.text}
                    </p>
                  );
                })}
              </div>
            ) : live.last_codex_message ? (
              <p className="text-xs leading-relaxed">{live.last_codex_message}</p>
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
});
