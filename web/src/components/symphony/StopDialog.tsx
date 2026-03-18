import { PauseCircleIcon, AlertTriangleIcon } from "lucide-react";
import { Button } from "#/components/ui/button";
import {
  AlertDialog,
  AlertDialogTrigger,
  AlertDialogPopup,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogClose,
} from "#/components/ui/alert-dialog";
import { Spinner } from "#/components/ui/spinner";
import { Badge } from "#/components/ui/badge";
import { Separator } from "#/components/ui/separator";
import { Alert, AlertTitle, AlertDescription } from "#/components/ui/alert";

export function StopDialog({
  onStopSelected,
  onStopAll,
  isPending,
  selectedWorkflowPath,
  showStopAll,
}: {
  onStopSelected: () => void;
  onStopAll: () => void;
  isPending: boolean;
  selectedWorkflowPath: string | null;
  showStopAll: boolean;
}) {
  return (
    <AlertDialog>
      <AlertDialogTrigger
        render={
          <Button variant="destructive-outline" disabled={isPending} className="gap-2">
            {isPending ? <Spinner className="size-4" /> : <PauseCircleIcon className="size-4" />}
            Stop
          </Button>
        }
      />
      <AlertDialogPopup>
        <AlertDialogHeader>
          <AlertDialogTitle className="flex items-center gap-2">
            <AlertTriangleIcon className="size-4 text-destructive" />
            {showStopAll ? "Stop workflows?" : "Stop workflow?"}
          </AlertDialogTitle>
          <AlertDialogDescription>
            <Alert variant="warning" className="mt-2">
              <AlertTriangleIcon className="size-4" />
              <AlertTitle>This action will interrupt running agents</AlertTitle>
              <AlertDescription>
                Running agents will finish their current turn. Scheduled retries will be cleared.
              </AlertDescription>
            </Alert>
            {selectedWorkflowPath && (
              <div className="mt-3 flex items-center gap-2">
                <span className="text-xs text-muted-foreground">Selected:</span>
                <Badge variant="secondary" size="sm" className="font-mono max-w-[280px] truncate">
                  {selectedWorkflowPath}
                </Badge>
              </div>
            )}
          </AlertDialogDescription>
        </AlertDialogHeader>

        {showStopAll && <Separator className="my-2" />}

        <AlertDialogFooter>
          <AlertDialogClose render={<Button variant="ghost">Cancel</Button>} />
          <AlertDialogClose
            render={
              <Button variant="destructive-outline" onClick={onStopSelected} className="gap-1.5">
                <PauseCircleIcon className="size-3.5" />
                Stop Selected
              </Button>
            }
          />
          {showStopAll && (
            <AlertDialogClose
              render={
                <Button variant="destructive" onClick={onStopAll} className="gap-1.5">
                  <PauseCircleIcon className="size-3.5" />
                  Stop All
                </Button>
              }
            />
          )}
        </AlertDialogFooter>
      </AlertDialogPopup>
    </AlertDialog>
  );
}
