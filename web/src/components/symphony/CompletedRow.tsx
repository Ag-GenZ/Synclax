import { memo } from "react";
import { ExternalLinkIcon } from "lucide-react";
import type { SymphonyCompletedEntry } from "#/api-gen/types.gen";
import { Badge } from "#/components/ui/badge";
import { TableRow, TableCell } from "#/components/ui/table";
import { Tooltip, TooltipTrigger, TooltipPopup } from "#/components/ui/tooltip";
import { fmtAgo, fmtSeconds, fmtTokens, statusVariant } from "./utils";

export const CompletedRow = memo(function CompletedRow({
  entry,
}: {
  entry: SymphonyCompletedEntry;
}) {
  const { issue } = entry;
  const v = statusVariant(entry.status);
  const title = issue.title?.trim() ? issue.title : "Untitled";

  return (
    <TableRow>
      <TableCell>
        <div className="flex items-start gap-2">
          <div className="min-w-0">
            <div className="flex items-center gap-2">
              <span className="font-mono text-xs font-semibold text-[var(--lagoon-deep)]">
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
            <div className="mt-1 text-sm font-medium leading-snug line-clamp-1">{title}</div>
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
      </TableCell>
      <TableCell>
        <Tooltip>
          <TooltipTrigger>
            <span className="text-sm font-medium tabular-nums text-muted-foreground">
              {fmtAgo(entry.ended_at)}
            </span>
          </TooltipTrigger>
          <TooltipPopup>{new Date(entry.ended_at).toLocaleString()}</TooltipPopup>
        </Tooltip>
      </TableCell>
      <TableCell className="tabular-nums text-sm text-muted-foreground">
        {fmtSeconds(entry.duration_secs)}
      </TableCell>
      <TableCell>
        <Tooltip>
          <TooltipTrigger>
            <span className="text-sm font-semibold tabular-nums">
              {fmtTokens(entry.codex_total_tokens)}
            </span>
          </TooltipTrigger>
          <TooltipPopup className="font-mono text-xs">
            in={entry.codex_input_tokens} out={entry.codex_output_tokens}{" "}
            total={entry.codex_total_tokens} turns={entry.turns_run}
          </TooltipPopup>
        </Tooltip>
      </TableCell>
      <TableCell className="max-w-xs">
        {entry.error ? (
          <Tooltip>
            <TooltipTrigger>
              <span className="block max-w-[260px] truncate font-mono text-xs text-destructive-foreground">
                {entry.error}
              </span>
            </TooltipTrigger>
            <TooltipPopup className="max-w-md break-words">{entry.error}</TooltipPopup>
          </Tooltip>
        ) : (
          <span className="text-muted-foreground text-sm">—</span>
        )}
      </TableCell>
    </TableRow>
  );
});
