import { useState, useCallback } from "react";
import { PlayCircleIcon } from "lucide-react";
import { Button } from "#/components/ui/button";
import {
  Dialog,
  DialogTrigger,
  DialogPopup,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogClose,
} from "#/components/ui/dialog";
import { Input } from "#/components/ui/input";
import { Label } from "#/components/ui/label";
import { Spinner } from "#/components/ui/spinner";

export function StartDialog({
  onStart,
  isPending,
}: {
  onStart: (path?: string, port?: number) => void;
  isPending: boolean;
}) {
  const [open, setOpen] = useState(false);
  const [path, setPath] = useState("");
  const [port, setPort] = useState("");

  const submit = useCallback(
    (e: React.FormEvent) => {
      e.preventDefault();
      onStart(path.trim() || undefined, port.trim() ? parseInt(port.trim(), 10) : undefined);
      setOpen(false);
    },
    [onStart, path, port],
  );

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger
        render={
          <Button disabled={isPending} className="gap-2">
            {isPending ? <Spinner className="size-4" /> : <PlayCircleIcon className="size-4" />}
            Start Symphony
          </Button>
        }
      />
      <DialogPopup className="w-full max-w-md" showCloseButton={false}>
        <DialogHeader>
          <DialogTitle>Start Symphony</DialogTitle>
          <DialogDescription>
            Launch the orchestrator. Both fields are optional — leave blank to use server defaults.
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={submit} className="space-y-4 px-6 pb-6">
          <div className="space-y-1.5">
            <Label htmlFor="wf-path">Workflow path</Label>
            <Input
              id="wf-path"
              placeholder="./WORKFLOW.md"
              value={path}
              onChange={(e) => setPath(e.target.value)}
              className="font-mono text-sm"
            />
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="dbg-port">Debug HTTP port</Label>
            <Input
              id="dbg-port"
              type="number"
              placeholder="e.g. 8089  ·  -1 to disable"
              value={port}
              onChange={(e) => setPort(e.target.value)}
              className="font-mono text-sm"
            />
          </div>
          <div className="flex justify-end gap-2 pt-2">
            <DialogClose
              render={
                <Button variant="ghost" type="button">
                  Cancel
                </Button>
              }
            />
            <Button type="submit">
              <PlayCircleIcon className="size-4" />
              Start
            </Button>
          </div>
        </form>
      </DialogPopup>
    </Dialog>
  );
}
