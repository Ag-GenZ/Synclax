"use client";

import { useCallback, useState, useEffect } from "react";
import { SettingsIcon, Volume2Icon, BellIcon, GlobeIcon } from "lucide-react";
import {
  Sheet,
  SheetPopup,
  SheetHeader,
  SheetTitle,
  SheetDescription,
  SheetPanel,
  SheetFooter,
} from "#/components/ui/sheet";
import { Button } from "#/components/ui/button";
import { Form } from "#/components/ui/form";
import { Field, FieldLabel, FieldDescription } from "#/components/ui/field";
import { Fieldset, FieldsetLegend } from "#/components/ui/fieldset";
import { Switch } from "#/components/ui/switch";
import { Slider, SliderValue } from "#/components/ui/slider";
import { RadioGroup, Radio } from "#/components/ui/radio-group";
import {
  Select,
  SelectTrigger,
  SelectValue,
  SelectPopup,
  SelectItem,
} from "#/components/ui/select";
import {
  NumberField,
  NumberFieldGroup,
  NumberFieldDecrement,
  NumberFieldIncrement,
  NumberFieldInput,
} from "#/components/ui/number-field";
import { Checkbox } from "#/components/ui/checkbox";
import { CheckboxGroup } from "#/components/ui/checkbox-group";
import { Textarea } from "#/components/ui/textarea";
import { InputGroup, InputGroupAddon, InputGroupInput } from "#/components/ui/input-group";
import { Separator } from "#/components/ui/separator";
import { Badge } from "#/components/ui/badge";
import { Tooltip, TooltipTrigger, TooltipPopup } from "#/components/ui/tooltip";

interface SettingsSheetProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  workflows: Array<{ id: string; workflow_path: string; running: boolean }>;
  selectedWorkflowId: string;
  onSelectWorkflow: (id: string) => void;
  currentTheme: string;
  onToggleTheme: (theme: string) => void;
}

function getStoredNumber(key: string, fallback: number): number {
  if (typeof window === "undefined") return fallback;
  const v = window.localStorage.getItem(key);
  return v ? parseInt(v, 10) : fallback;
}

function getStoredBool(key: string, fallback: boolean): boolean {
  if (typeof window === "undefined") return fallback;
  const v = window.localStorage.getItem(key);
  return v === null ? fallback : v === "true";
}

function getStoredArray(key: string, fallback: string[]): string[] {
  if (typeof window === "undefined") return fallback;
  try {
    const v = window.localStorage.getItem(key);
    return v ? JSON.parse(v) : fallback;
  } catch {
    return fallback;
  }
}

export function SettingsSheet({
  open,
  onOpenChange,
  workflows,
  selectedWorkflowId,
  onSelectWorkflow,
  currentTheme,
  onToggleTheme,
}: SettingsSheetProps) {
  const [autoRefresh, setAutoRefresh] = useState(() => getStoredBool("settings.autoRefresh", true));
  const [refreshInterval, setRefreshInterval] = useState(() =>
    getStoredNumber("settings.refreshInterval", 4),
  );
  const [debugPort, setDebugPort] = useState(() =>
    getStoredNumber("settings.debugPort", 8089),
  );
  const [notifications, setNotifications] = useState<string[]>(() =>
    getStoredArray("settings.notifications", ["errors", "completions"]),
  );
  const [notes, setNotes] = useState(() => {
    if (typeof window === "undefined") return "";
    return window.localStorage.getItem("settings.notes") ?? "";
  });
  const [apiBase, setApiBase] = useState(() => {
    if (typeof window === "undefined") return "localhost:2910";
    return window.localStorage.getItem("settings.apiBase") ?? "localhost:2910";
  });

  const persist = useCallback((key: string, value: unknown) => {
    if (typeof window === "undefined") return;
    window.localStorage.setItem(key, typeof value === "string" ? value : JSON.stringify(value));
  }, []);

  useEffect(() => { persist("settings.autoRefresh", String(autoRefresh)); }, [autoRefresh, persist]);
  useEffect(() => { persist("settings.refreshInterval", String(refreshInterval)); }, [refreshInterval, persist]);
  useEffect(() => { persist("settings.debugPort", String(debugPort)); }, [debugPort, persist]);
  useEffect(() => { persist("settings.notifications", JSON.stringify(notifications)); }, [notifications, persist]);

  const handleNotificationToggle = useCallback((key: string, checked: boolean) => {
    setNotifications((prev) =>
      checked ? [...prev, key] : prev.filter((n) => n !== key),
    );
  }, []);

  const handleSaveNotes = useCallback(() => {
    persist("settings.notes", notes);
  }, [notes, persist]);

  const handleSaveApiBase = useCallback(() => {
    persist("settings.apiBase", apiBase);
  }, [apiBase, persist]);

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetPopup side="right" showCloseButton>
        <SheetHeader>
          <SheetTitle className="flex items-center gap-2">
            <SettingsIcon className="size-4" />
            Settings
          </SheetTitle>
          <SheetDescription>
            Configure dashboard preferences. Changes are saved to local storage.
          </SheetDescription>
        </SheetHeader>

        <SheetPanel scrollFade>
          <Form className="space-y-6">
            {/* ── Data Refresh ── */}
            <Fieldset>
              <FieldsetLegend>Data Refresh</FieldsetLegend>
              <div className="space-y-4 mt-3">
                <Field>
                  <div className="flex items-center justify-between">
                    <div>
                      <FieldLabel>Auto-refresh</FieldLabel>
                      <FieldDescription>Automatically poll for new data</FieldDescription>
                    </div>
                    <Switch
                      checked={autoRefresh}
                      onCheckedChange={setAutoRefresh}
                    />
                  </div>
                </Field>

                <Field>
                  <FieldLabel>Refresh interval</FieldLabel>
                  <FieldDescription>How often to poll (in seconds)</FieldDescription>
                  <div className="mt-2">
                    <Slider
                      value={refreshInterval}
                      onValueChange={(value) => setRefreshInterval(typeof value === "number" ? value : value[0])}
                      min={1}
                      max={30}
                      disabled={!autoRefresh}
                    >
                      <SliderValue />
                    </Slider>
                  </div>
                  <div className="flex justify-between text-[10px] text-muted-foreground mt-1">
                    <span>1s</span>
                    <span>30s</span>
                  </div>
                </Field>
              </div>
            </Fieldset>

            <Separator />

            {/* ── Appearance ── */}
            <Fieldset>
              <FieldsetLegend>Appearance</FieldsetLegend>
              <div className="space-y-4 mt-3">
                <Field>
                  <FieldLabel>Theme</FieldLabel>
                  <RadioGroup
                    value={currentTheme}
                    onValueChange={onToggleTheme}
                    className="mt-2 flex gap-3"
                  >
                    <label className="flex items-center gap-2 text-sm cursor-pointer">
                      <Radio value="light" />
                      Light
                    </label>
                    <label className="flex items-center gap-2 text-sm cursor-pointer">
                      <Radio value="dark" />
                      Dark
                    </label>
                    <label className="flex items-center gap-2 text-sm cursor-pointer">
                      <Radio value="auto" />
                      System
                    </label>
                  </RadioGroup>
                </Field>
              </div>
            </Fieldset>

            <Separator />

            {/* ── Connection ── */}
            <Fieldset>
              <FieldsetLegend>Connection</FieldsetLegend>
              <div className="space-y-4 mt-3">
                <Field>
                  <FieldLabel>Default workflow</FieldLabel>
                  <FieldDescription>Workflow to select on load</FieldDescription>
                  <Select
                    value={selectedWorkflowId}
                    onValueChange={(value) => { if (value !== null) onSelectWorkflow(value); }}
                  >
                    <SelectTrigger size="sm" className="mt-1.5">
                      <SelectValue placeholder="Select workflow..." />
                    </SelectTrigger>
                    <SelectPopup>
                      {workflows.map((w) => (
                        <SelectItem key={w.id} value={w.id}>
                          <span className="flex items-center gap-2">
                            <span
                              className={`size-1.5 rounded-full ${w.running ? "bg-success" : "bg-muted-foreground/40"}`}
                            />
                            <span className="font-mono text-xs truncate">{w.workflow_path}</span>
                          </span>
                        </SelectItem>
                      ))}
                    </SelectPopup>
                  </Select>
                </Field>

                <Field>
                  <FieldLabel>API base URL</FieldLabel>
                  <InputGroup className="mt-1.5">
                    <InputGroupAddon align="inline-start">
                      <GlobeIcon className="size-3.5 text-muted-foreground" />
                    </InputGroupAddon>
                    <InputGroupInput
                      value={apiBase}
                      onChange={(e: React.ChangeEvent<HTMLInputElement>) => setApiBase(e.target.value)}
                      onBlur={handleSaveApiBase}
                      placeholder="localhost:2910"
                      className="font-mono text-xs"
                    />
                  </InputGroup>
                </Field>

                <Field>
                  <FieldLabel>Debug HTTP port</FieldLabel>
                  <FieldDescription>Port for the debug server</FieldDescription>
                  <NumberField
                    value={debugPort}
                    onValueChange={(val) => {
                      if (val != null) {
                        setDebugPort(val);
                        persist("settings.debugPort", String(val));
                      }
                    }}
                    min={-1}
                    max={65535}
                    size="sm"
                  >
                    <NumberFieldGroup>
                      <NumberFieldDecrement />
                      <NumberFieldInput />
                      <NumberFieldIncrement />
                    </NumberFieldGroup>
                  </NumberField>
                </Field>
              </div>
            </Fieldset>

            <Separator />

            {/* ── Notifications ── */}
            <Fieldset>
              <FieldsetLegend className="flex items-center gap-2">
                <BellIcon className="size-3.5" />
                Notifications
              </FieldsetLegend>
              <FieldDescription className="mt-1">
                Choose which events trigger toast notifications
              </FieldDescription>
              <CheckboxGroup className="mt-3 space-y-2.5">
                <label className="flex items-center gap-2.5 text-sm cursor-pointer">
                  <Checkbox
                    checked={notifications.includes("errors")}
                    onCheckedChange={(c) => handleNotificationToggle("errors", !!c)}
                  />
                  Error alerts
                </label>
                <label className="flex items-center gap-2.5 text-sm cursor-pointer">
                  <Checkbox
                    checked={notifications.includes("completions")}
                    onCheckedChange={(c) => handleNotificationToggle("completions", !!c)}
                  />
                  Completion events
                </label>
                <label className="flex items-center gap-2.5 text-sm cursor-pointer">
                  <Checkbox
                    checked={notifications.includes("retries")}
                    onCheckedChange={(c) => handleNotificationToggle("retries", !!c)}
                  />
                  Retry events
                </label>
                <label className="flex items-center gap-2.5 text-sm cursor-pointer">
                  <Checkbox
                    checked={notifications.includes("ratelimits")}
                    onCheckedChange={(c) => handleNotificationToggle("ratelimits", !!c)}
                  />
                  Rate limit warnings
                </label>
              </CheckboxGroup>
            </Fieldset>

            <Separator />

            {/* ── Notes ── */}
            <Fieldset>
              <FieldsetLegend>Notes</FieldsetLegend>
              <FieldDescription className="mt-1">
                Personal notes about this deployment
              </FieldDescription>
              <Textarea
                value={notes}
                onChange={(e) => setNotes(e.target.value)}
                onBlur={handleSaveNotes}
                placeholder="Add notes about this deployment..."
                size="sm"
                className="mt-2"
                rows={3}
              />
            </Fieldset>
          </Form>
        </SheetPanel>

        <SheetFooter>
          <div className="flex items-center justify-between w-full">
            <Badge variant="outline" size="sm">
              <Volume2Icon className="size-3" />
              {notifications.length} active
            </Badge>
            <Tooltip>
              <TooltipTrigger>
                <Button
                  size="sm"
                  variant="outline"
                  onClick={() => onOpenChange(false)}
                >
                  Done
                </Button>
              </TooltipTrigger>
              <TooltipPopup>Close settings panel</TooltipPopup>
            </Tooltip>
          </div>
        </SheetFooter>
      </SheetPopup>
    </Sheet>
  );
}
