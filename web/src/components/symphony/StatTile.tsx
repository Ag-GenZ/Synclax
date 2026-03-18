import { memo } from "react";
import { Skeleton } from "#/components/ui/skeleton";
import { cn } from "#/lib/utils";

const TINT_ICON = {
  success: "text-success-foreground",
  warning: "text-warning-foreground",
  info: "text-info-foreground",
} as const;

const TINT_BG = {
  success: "bg-success/10",
  warning: "bg-warning/10",
  info: "bg-info/10",
} as const;

const TINT_VAL = {
  success: "text-success-foreground",
  warning: "text-warning-foreground",
  info: "text-info-foreground",
} as const;

export const StatTile = memo(function StatTile({
  icon: Icon,
  label,
  value,
  tint,
  loading,
}: {
  icon: React.ElementType;
  label: string;
  value: string | number;
  tint?: "success" | "warning" | "info";
  loading?: boolean;
}) {
  return (
    <div className="rounded-xl border border-border/60 bg-card px-4 py-3.5 flex flex-col gap-3">
      <div className="flex items-center justify-between">
        <span className="text-[10.5px] font-semibold uppercase tracking-[0.1em] text-muted-foreground/80">
          {label}
        </span>
        <div
          className={cn(
            "rounded-lg p-1.5",
            tint ? cn(TINT_ICON[tint], TINT_BG[tint]) : "text-muted-foreground/50 bg-muted/40",
          )}
        >
          <Icon className="size-3.5" />
        </div>
      </div>
      {loading ? (
        <Skeleton className="h-7 w-16" />
      ) : (
        <div
          className={cn(
            "text-[28px] font-bold tabular-nums tracking-tight leading-none",
            tint ? TINT_VAL[tint] : "text-foreground",
          )}
        >
          {value}
        </div>
      )}
    </div>
  );
});
