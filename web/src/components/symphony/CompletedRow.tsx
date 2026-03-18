import { memo } from "react";
import { ClockIcon, ExternalLinkIcon, ZapIcon } from "lucide-react";
import type { SymphonyCompletedEntry } from "#/api-gen/types.gen";
import { Badge } from "#/components/ui/badge";
import { Tooltip, TooltipTrigger, TooltipPopup } from "#/components/ui/tooltip";
import { cn } from "#/lib/utils";
import { fmtAgo, fmtSeconds, fmtTokens, statusVariant } from "./utils";

const STATUS_BAR: Record<string, string> = {
  success: "bg-success",
  error: "bg-destructive",
  warning: "bg-warning",
  outline: "bg-border",
  secondary: "bg-muted-foreground/40",
  info: "bg-info",
};

export const CompletedRow = memo(function CompletedRow({
  entry,
}: {
  entry: SymphonyCompletedEntry;
}) {
  const { issue } = entry;
  const v = statusVariant(entry.status);
  const title = issue.title?.trim() ? issue.title : "Untitled";

  return (
    <div className="flex items-center gap-3 px-4 py-2.5 hover:bg-muted/30 transition-colors group border-b border-border/30 last:border-0">
      {/* Status bar */}
      <div className={cn("w-0.5 h-6 rounded-full shrink-0", STATUS_BAR[v] ?? "bg-border")} />

      {/* Content */}
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-1.5 mb-0.5">
          <span className="font-mono text-[11px] font-bold text-info-foreground">
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
        <div className="text-[12.5px] font-medium text-foreground/80 truncate leading-tight">
          {title}
        </div>
        {entry.error && (
          <Tooltip>
            <TooltipTrigger>
              <span className="text-[10px] font-mono text-destructive-foreground truncate block max-w-[220px] mt-0.5 leading-none">
                {entry.error}
              </span>
            </TooltipTrigger>
            <TooltipPopup className="max-w-md break-words">{entry.error}</TooltipPopup>
          </Tooltip>
        )}
      </div>

      {/* Metadata chips */}
      <div className="flex items-center gap-3 shrink-0 text-[11px] text-muted-foreground/60">
        <span className="flex items-center gap-1">
          <ClockIcon className="size-3" />
          {fmtSeconds(entry.duration_secs)}
        </span>
        <Tooltip>
          <TooltipTrigger>
            <span className="flex items-center gap-1">
              <ZapIcon className="size-3" />
              {fmtTokens(entry.total_tokens)}
            </span>
          </TooltipTrigger>
          <TooltipPopup className="font-mono text-xs">
            in={entry.input_tokens} out={entry.output_tokens} turns={entry.turns_run}
          </TooltipPopup>
        </Tooltip>
        <Tooltip>
          <TooltipTrigger>
            <span className="tabular-nums">{fmtAgo(entry.ended_at)}</span>
          </TooltipTrigger>
          <TooltipPopup>{new Date(entry.ended_at).toLocaleString()}</TooltipPopup>
        </Tooltip>
        {issue.url && (
          <a
            href={issue.url}
            target="_blank"
            rel="noopener noreferrer"
            className="opacity-0 group-hover:opacity-100 text-muted-foreground/40 hover:text-foreground transition-all"
          >
            <ExternalLinkIcon className="size-3" />
          </a>
        )}
      </div>
    </div>
  );
});
