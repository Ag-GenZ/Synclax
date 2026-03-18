import { memo } from "react";
import type { SymphonyRetryEntry } from "#/api-gen/types.gen";
import { Badge } from "#/components/ui/badge";
import { TableRow, TableCell } from "#/components/ui/table";
import { Tooltip, TooltipTrigger, TooltipPopup } from "#/components/ui/tooltip";
import { fmtDueIn } from "./utils";

export const RetryRow = memo(function RetryRow({ entry }: { entry: SymphonyRetryEntry }) {
  return (
    <TableRow>
      <TableCell className="font-mono text-xs font-semibold text-[var(--lagoon-deep)]">
        {entry.identifier}
      </TableCell>
      <TableCell className="tabular-nums text-sm text-muted-foreground">
        #{entry.attempt}
      </TableCell>
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
});
