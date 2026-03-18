import { memo } from "react";
import { Skeleton } from "#/components/ui/skeleton";
import { cn } from "#/lib/utils";

const TINT_CLS = {
  success: "text-success bg-success/8",
  warning: "text-warning-foreground bg-warning/8",
  info: "text-info bg-info/8",
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
    <div className="rounded-xl border bg-card p-4">
      <div className="flex items-center gap-2 mb-3">
        <div
          className={cn(
            "rounded-md p-1.5",
            tint ? TINT_CLS[tint] : "text-muted-foreground bg-muted/60",
          )}
        >
          <Icon className="size-3.5" />
        </div>
        <span className="text-xs font-medium text-muted-foreground">{label}</span>
      </div>
      {loading ? (
        <Skeleton className="h-7 w-16" />
      ) : (
        <div className="text-2xl font-semibold tabular-nums">{value}</div>
      )}
    </div>
  );
});
