import { memo } from "react";
import type { CodexTotals } from "#/api-gen/types.gen";
import {
  Progress,
  ProgressLabel,
  ProgressTrack,
  ProgressIndicator,
} from "#/components/ui/progress";
import { fmtTokens, fmtSeconds } from "./utils";

export const TokenUsage = memo(function TokenUsage({ totals }: { totals: CodexTotals }) {
  const inputPct = totals.total_tokens > 0 ? (totals.input_tokens / totals.total_tokens) * 100 : 0;
  const outputPct =
    totals.total_tokens > 0 ? (totals.output_tokens / totals.total_tokens) * 100 : 0;

  return (
    <div className="grid gap-4 sm:grid-cols-2">
      <div className="space-y-3">
        <Progress value={inputPct} max={100}>
          <div className="flex items-center justify-between text-xs mb-1.5">
            <ProgressLabel>Input Tokens</ProgressLabel>
            <span className="tabular-nums">{fmtTokens(totals.input_tokens)}</span>
          </div>
          <ProgressTrack>
            <ProgressIndicator className="bg-info/70" />
          </ProgressTrack>
        </Progress>
        <Progress value={outputPct} max={100}>
          <div className="flex items-center justify-between text-xs mb-1.5">
            <ProgressLabel>Output Tokens</ProgressLabel>
            <span className="tabular-nums">{fmtTokens(totals.output_tokens)}</span>
          </div>
          <ProgressTrack>
            <ProgressIndicator className="bg-success/70" />
          </ProgressTrack>
        </Progress>
      </div>
      <div className="grid grid-cols-2 gap-3">
        <div className="rounded-xl border bg-muted/30 p-4">
          <div className="text-xs text-muted-foreground mb-1">Total</div>
          <div className="text-xl font-semibold tabular-nums">{fmtTokens(totals.total_tokens)}</div>
        </div>
        <div className="rounded-xl border bg-muted/30 p-4">
          <div className="text-xs text-muted-foreground mb-1">Uptime</div>
          <div className="text-xl font-semibold tabular-nums">
            {fmtSeconds(totals.seconds_running)}
          </div>
        </div>
        <div className="rounded-xl border bg-muted/30 p-4">
          <div className="text-xs text-muted-foreground mb-1">In</div>
          <div className="text-xl font-semibold tabular-nums text-info-foreground">
            {fmtTokens(totals.input_tokens)}
          </div>
        </div>
        <div className="rounded-xl border bg-muted/30 p-4">
          <div className="text-xs text-muted-foreground mb-1">Out</div>
          <div className="text-xl font-semibold tabular-nums text-success-foreground">
            {fmtTokens(totals.output_tokens)}
          </div>
        </div>
      </div>
    </div>
  );
});
