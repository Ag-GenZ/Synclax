import { memo } from "react";
import { TableRow, TableCell } from "#/components/ui/table";
import { Tooltip, TooltipTrigger, TooltipPopup } from "#/components/ui/tooltip";
import { type ActivityItem, fmtAgo } from "./utils";

export const ActivityRow = memo(function ActivityRow({ entry }: { entry: ActivityItem }) {
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
});
