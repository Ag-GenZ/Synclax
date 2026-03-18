import { useEffect, useState, useCallback } from "react";
import { PlayCircleIcon, FolderOpenIcon } from "lucide-react";
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
import { Form } from "#/components/ui/form";
import { Field, FieldLabel, FieldDescription } from "#/components/ui/field";
import { Fieldset, FieldsetLegend } from "#/components/ui/fieldset";
import { InputGroup, InputGroupAddon, InputGroupInput } from "#/components/ui/input-group";
import {
  NumberField,
  NumberFieldGroup,
  NumberFieldDecrement,
  NumberFieldIncrement,
  NumberFieldInput,
} from "#/components/ui/number-field";
import { Separator } from "#/components/ui/separator";
import { Spinner } from "#/components/ui/spinner";
import { Badge } from "#/components/ui/badge";

export function StartDialog({
  onStart,
  isPending,
  defaultWorkflowPath,
}: {
  onStart: (path?: string, port?: number) => void;
  isPending: boolean;
  defaultWorkflowPath: string | null;
}) {
  const [open, setOpen] = useState(false);
  const [path, setPath] = useState("");
  const [port, setPort] = useState<number | null>(null);

  useEffect(() => {
    if (!open) return;
    if (typeof window === "undefined") return;

    if (!path.trim()) {
      const storedPath = window.localStorage.getItem("symphony.start.workflow_path") ?? "";
      const next = storedPath.trim() || defaultWorkflowPath?.trim() || "";
      if (next) setPath(next);
    }

    if (port == null) {
      const storedPort = window.localStorage.getItem("symphony.start.http_port") ?? "";
      const next = storedPort.trim();
      if (next) setPort(parseInt(next, 10));
    }
  }, [open, path, port, defaultWorkflowPath]);

  const submit = useCallback(
    (e: React.FormEvent) => {
      e.preventDefault();
      const pathValue = path.trim();

      if (typeof window !== "undefined") {
        if (pathValue) {
          window.localStorage.setItem("symphony.start.workflow_path", pathValue);
        } else {
          window.localStorage.removeItem("symphony.start.workflow_path");
        }
        if (port != null) {
          window.localStorage.setItem("symphony.start.http_port", String(port));
        } else {
          window.localStorage.removeItem("symphony.start.http_port");
        }
      }

      onStart(pathValue || undefined, port ?? undefined);
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
            Start
          </Button>
        }
      />
      <DialogPopup className="w-full max-w-md" showCloseButton={false}>
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <PlayCircleIcon className="size-4 text-success-foreground" />
            Start Workflow
          </DialogTitle>
          <DialogDescription>
            Launch a workflow. Leave blank to use{" "}
            {defaultWorkflowPath ? (
              <Badge variant="secondary" size="sm" className="font-mono">
                {defaultWorkflowPath}
              </Badge>
            ) : (
              "server defaults"
            )}
          </DialogDescription>
        </DialogHeader>

        <Form onSubmit={submit} className="px-6 pb-6">
          <Fieldset className="space-y-4">
            <FieldsetLegend className="sr-only">Workflow Configuration</FieldsetLegend>

            <Field>
              <FieldLabel htmlFor="wf-path">Workflow path</FieldLabel>
              <FieldDescription>Path to the workflow definition file</FieldDescription>
              <InputGroup className="mt-1.5">
                <InputGroupAddon align="inline-start">
                  <FolderOpenIcon className="size-3.5 text-muted-foreground" />
                </InputGroupAddon>
                <InputGroupInput
                  id="wf-path"
                  name="workflow_path"
                  autoComplete="off"
                  placeholder="./WORKFLOW.md"
                  value={path}
                  onChange={(e: React.ChangeEvent<HTMLInputElement>) => setPath(e.target.value)}
                  className="font-mono text-sm"
                />
              </InputGroup>
            </Field>

            <Separator />

            <Field>
              <FieldLabel>Debug HTTP port</FieldLabel>
              <FieldDescription>Set to -1 to disable the debug server</FieldDescription>
              <NumberField
                value={port ?? undefined}
                onValueChange={(val) => setPort(val ?? null)}
                min={-1}
                max={65535}
                size="sm"
              >
                <NumberFieldGroup className="mt-1.5">
                  <NumberFieldDecrement />
                  <NumberFieldInput placeholder="8089" />
                  <NumberFieldIncrement />
                </NumberFieldGroup>
              </NumberField>
            </Field>
          </Fieldset>

          <div className="flex justify-end gap-2 pt-4 mt-2 border-t">
            <DialogClose
              render={
                <Button variant="ghost" type="button">
                  Cancel
                </Button>
              }
            />
            <Button type="submit" className="gap-2">
              <PlayCircleIcon className="size-4" />
              Start Workflow
            </Button>
          </div>
        </Form>
      </DialogPopup>
    </Dialog>
  );
}
