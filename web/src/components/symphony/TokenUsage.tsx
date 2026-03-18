import { memo } from "react";
import type { CodexTotals } from "#/api-gen/types.gen";
import { cn } from "#/lib/utils";
import { fmtTokens, fmtSeconds } from "./utils";

export const TokenUsage = memo(function TokenUsage({ totals }: { totals: CodexTotals }) {
  const inputPct =
    totals.total_tokens > 0 ? (totals.input_tokens / totals.total_tokens) * 100 : 50;

  return (
    <div className="space-y-6">
      {/* Hero number */}
      <div>
        <p className="text-[10.5px] font-semibold uppercase tracking-[0.1em] text-muted-foreground/70 mb-1.5">
          Total Consumed
        </p>
        <div className="text-4xl font-bold tabular-nums tracking-tight leading-none">
          {fmtTokens(totals.total_tokens)}
        </div>
        <p className="text-xs text-muted-foreground/60 mt-2">
          across {fmtSeconds(totals.seconds_running)} of runtime
        </p>
      </div>

      {/* Stacked proportional bar */}
      <div className="space-y-2">
        <div className="flex h-2.5 rounded-full overflow-hidden bg-muted/40 gap-px">
          <div
            className="bg-info/60 rounded-l-full transition-[width] duration-700"
            style={{ width: `${inputPct}%` }}
          />
          <div className="bg-success/60 rounded-r-full flex-1 transition-[width] duration-700" />
        </div>
        <div className="flex items-center justify-between text-[11px]">
          <div className="flex items-center gap-2">
            <div className="size-2 rounded-full bg-info/60 shrink-0" />
            <span className="text-muted-foreground">
              Input{" "}
              <span className="font-semibold text-info-foreground tabular-nums">
                {fmtTokens(totals.input_tokens)}
              </span>
            </span>
          </div>
          <div className="flex items-center gap-2">
            <span className="text-muted-foreground">
              Output{" "}
              <span className="font-semibold text-success-foreground tabular-nums">
                {fmtTokens(totals.output_tokens)}
              </span>
            </span>
            <div className="size-2 rounded-full bg-success/60 shrink-0" />
          </div>
        </div>
      </div>

      {/* Stats grid */}
      <div className="grid grid-cols-2 gap-2 sm:grid-cols-4">
        <TokenStat label="Total" value={fmtTokens(totals.total_tokens)} />
        <TokenStat label="Uptime" value={fmtSeconds(totals.seconds_running)} />
        <TokenStat label="Input" value={fmtTokens(totals.input_tokens)} color="text-info-foreground" />
        <TokenStat
          label="Output"
          value={fmtTokens(totals.output_tokens)}
          color="text-success-foreground"
        />
      </div>
    </div>
  );
});

function TokenStat({
  label,
  value,
  color,
}: {
  label: string;
  value: string;
  color?: string;
}) {
  return (
    <div className="rounded-xl border border-border/60 bg-card/60 px-3.5 py-3">
      <div className="text-[10px] font-semibold uppercase tracking-[0.08em] text-muted-foreground/60 mb-1.5">
        {label}
      </div>
      <div className={cn("text-base font-bold tabular-nums leading-none", color ?? "")}>
        {value}
      </div>
    </div>
  );
}
