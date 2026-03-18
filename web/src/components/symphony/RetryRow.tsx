import { memo } from "react";
import { RotateCcwIcon } from "lucide-react";
import type { SymphonyRetryEntry } from "#/api-gen/types.gen";
import { Badge } from "#/components/ui/badge";
import { Tooltip, TooltipTrigger, TooltipPopup } from "#/components/ui/tooltip";
import { fmtDueIn } from "./utils";

export const RetryRow = memo(function RetryRow({ entry }: { entry: SymphonyRetryEntry }) {
  return (
    <div className="flex items-center gap-3 px-4 py-2.5 hover:bg-muted/30 transition-colors border-b border-border/30 last:border-0">
      {/* Warning indicator */}
      <div className="size-1.5 rounded-full bg-warning shrink-0" />

      {/* Content */}
      <div className="flex items-center gap-2 flex-1 min-w-0 overflow-hidden">
        <span className="font-mono text-[11px] font-bold text-info-foreground shrink-0">
          {entry.identifier}
        </span>
        <Badge variant="outline" size="sm">
          #{entry.attempt}
        </Badge>
        {entry.delay_type && (
          <Badge variant="secondary" size="sm">
            {entry.delay_type}
          </Badge>
        )}
        {entry.error && (
          <Tooltip>
            <TooltipTrigger>
              <span className="max-w-[200px] truncate font-mono text-[10px] text-destructive-foreground block">
                {entry.error}
              </span>
            </TooltipTrigger>
            <TooltipPopup className="max-w-sm break-words">{entry.error}</TooltipPopup>
          </Tooltip>
        )}
      </div>

      {/* Due countdown */}
      <Tooltip>
        <TooltipTrigger>
          <div className="flex items-center gap-1.5 text-[11px] font-medium text-warning-foreground shrink-0">
            <RotateCcwIcon className="size-3" />
            {fmtDueIn(entry.due_at)}
          </div>
        </TooltipTrigger>
        <TooltipPopup>{new Date(entry.due_at).toLocaleString()}</TooltipPopup>
      </Tooltip>
    </div>
  );
});
