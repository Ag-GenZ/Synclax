import { PauseCircleIcon } from "lucide-react";
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

export function StopDialog({
  onStop,
  isPending,
}: {
  onStop: () => void;
  isPending: boolean;
}) {
  return (
    <AlertDialog>
      <AlertDialogTrigger
        render={
          <Button
            variant="outline"
            disabled={isPending}
            className="gap-2 border-destructive/30 text-destructive hover:bg-destructive/8 hover:border-destructive/50"
          >
            {isPending ? <Spinner className="size-4" /> : <PauseCircleIcon className="size-4" />}
            Stop
          </Button>
        }
      />
      <AlertDialogPopup>
        <AlertDialogHeader>
          <AlertDialogTitle>Stop Symphony?</AlertDialogTitle>
          <AlertDialogDescription>
            Signals the orchestrator to stop. Running agents finish their current turn. Scheduled
            retries are cleared.
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogClose render={<Button variant="ghost">Cancel</Button>} />
          <AlertDialogClose
            render={
              <Button
                variant="outline"
                className="border-destructive/30 text-destructive hover:bg-destructive/8"
                onClick={onStop}
              >
                Stop Symphony
              </Button>
            }
          />
        </AlertDialogFooter>
      </AlertDialogPopup>
    </AlertDialog>
  );
}
