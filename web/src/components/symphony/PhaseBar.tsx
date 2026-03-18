import { memo } from "react";
import { Tooltip, TooltipTrigger, TooltipPopup } from "#/components/ui/tooltip";
import { cn } from "#/lib/utils";
import { PHASES, PHASE_LABELS, type Phase } from "./utils";

export const PhaseBar = memo(function PhaseBar({ phase }: { phase: string }) {
  const idx = PHASES.indexOf(phase as Phase);
  return (
    <div className="flex items-center gap-0.5">
      {PHASES.map((p, i) => (
        <Tooltip key={p}>
          <TooltipTrigger>
            <div
              className={cn(
                "h-1 w-5 rounded-full transition-all",
                i < idx
                  ? "bg-success/60"
                  : i === idx
                    ? "bg-[var(--lagoon-deep)] animate-pulse"
                    : "bg-border",
              )}
            />
          </TooltipTrigger>
          <TooltipPopup>{PHASE_LABELS[p] ?? p}</TooltipPopup>
        </Tooltip>
      ))}
      <span className="ml-2 font-mono text-xs text-muted-foreground">
        {PHASE_LABELS[phase as Phase] ?? phase}
      </span>
    </div>
  );
});
