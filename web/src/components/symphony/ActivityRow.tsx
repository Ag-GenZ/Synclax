import { memo } from "react";
import { Tooltip, TooltipTrigger, TooltipPopup } from "#/components/ui/tooltip";
import { type ActivityItem, fmtAgo } from "./utils";

function simplifyEvent(event: string): string {
  return event.split("/").at(-1) ?? event;
}

export const ActivityRow = memo(function ActivityRow({ entry }: { entry: ActivityItem }) {
  return (
    <div className="flex items-center gap-3 px-4 py-2 hover:bg-muted/30 transition-colors border-b border-border/30 last:border-0 group">
      {/* Pulse dot */}
      <div className="size-1.5 rounded-full bg-[var(--lagoon)]/60 shrink-0" />

      {/* Issue ID */}
      <span className="font-mono text-[11px] font-bold text-[var(--lagoon-deep)] shrink-0 w-14 truncate">
        {entry.identifier}
      </span>

      {/* Event type chip */}
      <span className="shrink-0 font-mono text-[10px] text-muted-foreground/60 bg-muted/50 rounded px-1.5 py-0.5 leading-none">
        {simplifyEvent(entry.event)}
      </span>

      {/* Message */}
      <div className="flex-1 min-w-0">
        {entry.message ? (
          <Tooltip>
            <TooltipTrigger>
              <span className="block text-[12px] text-foreground/75 truncate font-mono leading-none">
                {entry.message}
              </span>
            </TooltipTrigger>
            <TooltipPopup className="max-w-md break-words">{entry.message}</TooltipPopup>
          </Tooltip>
        ) : (
          <span className="text-muted-foreground/30 text-[11px]">—</span>
        )}
      </div>

      {/* Timestamp */}
      <Tooltip>
        <TooltipTrigger>
          <span className="text-[10px] tabular-nums text-muted-foreground/40 shrink-0 group-hover:text-muted-foreground/60 transition-colors">
            {fmtAgo(entry.ts)}
          </span>
        </TooltipTrigger>
        <TooltipPopup>{new Date(entry.ts).toLocaleString()}</TooltipPopup>
      </Tooltip>
    </div>
  );
});
