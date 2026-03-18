import { memo } from "react";
import { Tooltip, TooltipTrigger, TooltipPopup } from "#/components/ui/tooltip";
import { cn } from "#/lib/utils";
import { PHASES, PHASE_LABELS, type Phase } from "./utils";

export const PhaseBar = memo(function PhaseBar({ phase }: { phase: string }) {
  const idx = PHASES.indexOf(phase as Phase);
  return (
    <div className="space-y-1.5">
      <div className="flex items-center h-1 gap-px">
        {PHASES.map((p, i) => (
          <Tooltip key={p}>
            <TooltipTrigger className="flex-1 h-full">
              <div
                className={cn(
                  "h-full w-full rounded-full transition-all duration-300",
                  i < idx
                    ? "bg-success/50"
                    : i === idx
                      ? "bg-info animate-pulse"
                      : "bg-border/50",
                )}
              />
            </TooltipTrigger>
            <TooltipPopup>{PHASE_LABELS[p] ?? p}</TooltipPopup>
          </Tooltip>
        ))}
      </div>
      <div className="font-mono text-[10px] text-muted-foreground/60">
        {PHASE_LABELS[phase as Phase] ?? phase}
      </div>
    </div>
  );
});
