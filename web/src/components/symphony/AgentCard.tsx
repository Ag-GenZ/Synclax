import { memo, useState, useRef, useEffect } from "react";
import {
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
import { cn } from "#/lib/utils";
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
      <CardHeader className="px-4 pt-4 pb-3 space-y-2.5">
        {/* ID + badges row */}
        <div className="flex items-start gap-2">
          <div className="flex-1 min-w-0">
            <div className="flex flex-wrap items-center gap-1.5 mb-1.5">
              <span className="font-mono text-[11px] font-bold text-[var(--lagoon-deep)]">
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
            <p className="text-[13px] font-semibold leading-snug line-clamp-2">{issue.title}</p>
          </div>
          {issue.url && (
            <Tooltip>
              <TooltipTrigger>
                <a
                  href={issue.url}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="mt-0.5 text-muted-foreground/40 hover:text-foreground transition-colors"
                >
                  <ExternalLinkIcon className="size-3.5" />
                </a>
              </TooltipTrigger>
              <TooltipPopup>Open in Linear</TooltipPopup>
            </Tooltip>
          )}
        </div>

        {/* Phase progress */}
        <PhaseBar phase={phase} />

        {/* Inline metrics */}
        <div className="flex items-center gap-2.5 text-[11px] text-muted-foreground flex-wrap">
          <span className="tabular-nums">
            <span className="font-semibold text-foreground">{live.turn_count}</span>{" "}
            <span className="text-muted-foreground/60">turns</span>
          </span>
          <span className="text-border/60">·</span>
          <span className="tabular-nums text-info-foreground font-medium">
            {fmtTokens(live.codex_input_tokens)} in
          </span>
          <span className="text-border/60">·</span>
          <span className="tabular-nums text-success-foreground font-medium">
            {fmtTokens(live.codex_output_tokens)} out
          </span>
          <span className="text-border/60">·</span>
          <span className="flex items-center gap-1 text-muted-foreground/70">
            <TimerIcon className="size-3" />
            {fmtAgo(started_at)}
          </span>
        </div>
      </CardHeader>

      <CardPanel className="pt-0 px-4 pb-4 space-y-3">
        {/* Terminal stream */}
        {hasLiveEvent && (
          <div className="rounded-lg border border-border/40 overflow-hidden">
            {/* macOS-style title bar */}
            <div className="flex items-center gap-2 px-3 py-1.5 bg-muted/30 border-b border-border/30">
              <div className="flex items-center gap-1">
                <div className="size-2 rounded-full bg-destructive/40" />
                <div className="size-2 rounded-full bg-warning/40" />
                <div className="size-2 rounded-full bg-success/40" />
              </div>
              <span className="font-mono text-[10px] text-muted-foreground/50 ml-1 flex-1 truncate">
                {simplifyEvent(blocks.at(-1)?.event ?? live.last_codex_event ?? "event")}
              </span>
              {live.last_codex_timestamp && (
                <span className="font-mono text-[10px] text-muted-foreground/40 tabular-nums shrink-0">
                  {fmtAgo(live.last_codex_timestamp)}
                </span>
              )}
            </div>
            {/* Stream content */}
            <div
              ref={streamRef}
              className="bg-black/[0.03] dark:bg-black/30 p-3 max-h-44 overflow-y-auto scroll-smooth space-y-1"
            >
              {blocks.length > 0 ? (
                blocks.map((block, i) => {
                  const isLast = i === blocks.length - 1;
                  return (
                    <p
                      key={i}
                      className={cn(
                        "font-mono text-[11px] leading-relaxed whitespace-pre-wrap break-all",
                        isLast ? "text-foreground/90" : "text-muted-foreground/50",
                      )}
                    >
                      {block.text}
                    </p>
                  );
                })
              ) : live.last_codex_message ? (
                <p className="font-mono text-[11px] leading-relaxed text-foreground/90">
                  {live.last_codex_message}
                </p>
              ) : (
                <p className="font-mono text-[11px] text-muted-foreground/40">—</p>
              )}
            </div>
          </div>
        )}

        {/* Meta */}
        <div className="space-y-1 text-[11px] text-muted-foreground/60">
          {issue.branch_name && (
            <div className="flex items-center gap-1.5">
              <GitBranchIcon className="size-3 shrink-0" />
              <span className="font-mono truncate">{issue.branch_name}</span>
            </div>
          )}
          <div className="flex items-center gap-1.5">
            <FolderIcon className="size-3 shrink-0" />
            <span className="font-mono truncate">{workspace_path}</span>
          </div>
        </div>

        {/* Expandable details */}
        {(issue.description || (issue.blocked_by && issue.blocked_by.length > 0)) && (
          <Collapsible open={open} onOpenChange={setOpen}>
            <CollapsibleTrigger className="flex w-full items-center gap-1.5 py-0.5 text-[11px] text-muted-foreground/50 hover:text-muted-foreground transition-colors">
              {open ? (
                <ChevronDownIcon className="size-3" />
              ) : (
                <ChevronRightIcon className="size-3" />
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
