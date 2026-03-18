"use client";

import { useCallback, useEffect, useState } from "react";
import {
  BugIcon,
  LayersIcon,
  MoonIcon,
  PauseCircleIcon,
  PlayCircleIcon,
  RefreshCwIcon,
  SearchIcon,
  SettingsIcon,
  SunIcon,
  MonitorIcon,
  ZapIcon,
  ActivityIcon,
  RotateCcwIcon,
  CheckCircle2Icon,
} from "lucide-react";
import {
  CommandDialog,
  CommandDialogTrigger,
  CommandDialogPopup,
  CommandInput,
  CommandList,
  CommandEmpty,
  CommandGroup,
  CommandGroupLabel,
  CommandItem,
  CommandSeparator,
  CommandShortcut,
  CommandFooter,
} from "#/components/ui/command";
import { Button } from "#/components/ui/button";
import { Kbd, KbdGroup } from "#/components/ui/kbd";

interface CommandMenuProps {
  onNavigateTab: (tab: string) => void;
  onRefresh: () => void;
  onStartWorkflow: () => void;
  onStopWorkflow: () => void;
  onOpenSettings: () => void;
  onToggleTheme: () => void;
  isRunning: boolean;
  workflows: Array<{ id: string; workflow_path: string; running: boolean }>;
  onSelectWorkflow: (id: string) => void;
  currentTheme: string;
}

export function CommandMenu({
  onNavigateTab,
  onRefresh,
  onStartWorkflow,
  onStopWorkflow,
  onOpenSettings,
  onToggleTheme,
  isRunning,
  workflows,
  onSelectWorkflow,
  currentTheme,
}: CommandMenuProps) {
  const [open, setOpen] = useState(false);

  useEffect(() => {
    const down = (e: KeyboardEvent) => {
      if (e.key === "k" && (e.metaKey || e.ctrlKey)) {
        e.preventDefault();
        setOpen((o) => !o);
      }
    };
    document.addEventListener("keydown", down);
    return () => document.removeEventListener("keydown", down);
  }, []);

  const run = useCallback(
    (fn: () => void) => {
      setOpen(false);
      fn();
    },
    [],
  );

  return (
    <CommandDialog open={open} onOpenChange={setOpen}>
      <CommandDialogTrigger
        render={
          <Button
            variant="outline"
            size="sm"
            className="gap-2 text-muted-foreground font-normal min-w-[200px] justify-start"
          >
            <SearchIcon className="size-3.5" />
            <span className="flex-1 text-left">Search...</span>
            <KbdGroup>
              <Kbd>⌘</Kbd>
              <Kbd>K</Kbd>
            </KbdGroup>
          </Button>
        }
      />
      <CommandDialogPopup>
        <CommandInput placeholder="Type a command or search..." />
        <CommandList>
          <CommandEmpty>No results found.</CommandEmpty>

          <CommandGroup>
            <CommandGroupLabel>Actions</CommandGroupLabel>
            <CommandItem onSelect={() => run(onStartWorkflow)}>
              <PlayCircleIcon className="size-4" />
              Start Workflow
              <CommandShortcut>
                <KbdGroup><Kbd>⌘</Kbd><Kbd>S</Kbd></KbdGroup>
              </CommandShortcut>
            </CommandItem>
            {isRunning && (
              <CommandItem onSelect={() => run(onStopWorkflow)}>
                <PauseCircleIcon className="size-4" />
                Stop Workflow
              </CommandItem>
            )}
            <CommandItem onSelect={() => run(onRefresh)}>
              <RefreshCwIcon className="size-4" />
              Refresh Data
              <CommandShortcut>
                <KbdGroup><Kbd>⌘</Kbd><Kbd>R</Kbd></KbdGroup>
              </CommandShortcut>
            </CommandItem>
          </CommandGroup>

          <CommandSeparator />

          <CommandGroup>
            <CommandGroupLabel>Navigate</CommandGroupLabel>
            <CommandItem onSelect={() => run(() => onNavigateTab("running"))}>
              <LayersIcon className="size-4" />
              Running Agents
            </CommandItem>
            <CommandItem onSelect={() => run(() => onNavigateTab("activity"))}>
              <ActivityIcon className="size-4" />
              Activity Feed
            </CommandItem>
            <CommandItem onSelect={() => run(() => onNavigateTab("retrying"))}>
              <RotateCcwIcon className="size-4" />
              Retry Queue
            </CommandItem>
            <CommandItem onSelect={() => run(() => onNavigateTab("completed"))}>
              <CheckCircle2Icon className="size-4" />
              Completed
            </CommandItem>
            <CommandItem onSelect={() => run(() => onNavigateTab("tokens"))}>
              <ZapIcon className="size-4" />
              Token Usage
            </CommandItem>
            <CommandItem onSelect={() => run(() => onNavigateTab("debug"))}>
              <BugIcon className="size-4" />
              Debug Panel
            </CommandItem>
          </CommandGroup>

          {workflows.length > 0 && (
            <>
              <CommandSeparator />
              <CommandGroup>
                <CommandGroupLabel>Workflows</CommandGroupLabel>
                {workflows.map((w) => (
                  <CommandItem key={w.id} onSelect={() => run(() => onSelectWorkflow(w.id))}>
                    <span className={`size-1.5 rounded-full shrink-0 ${w.running ? "bg-success" : "bg-muted-foreground/40"}`} />
                    <span className="font-mono text-xs truncate">{w.workflow_path}</span>
                  </CommandItem>
                ))}
              </CommandGroup>
            </>
          )}

          <CommandSeparator />

          <CommandGroup>
            <CommandGroupLabel>Preferences</CommandGroupLabel>
            <CommandItem onSelect={() => run(onToggleTheme)}>
              {currentTheme === "dark" ? (
                <SunIcon className="size-4" />
              ) : currentTheme === "light" ? (
                <MoonIcon className="size-4" />
              ) : (
                <MonitorIcon className="size-4" />
              )}
              Toggle Theme
              <CommandShortcut>
                <span className="text-xs text-muted-foreground">{currentTheme}</span>
              </CommandShortcut>
            </CommandItem>
            <CommandItem onSelect={() => run(onOpenSettings)}>
              <SettingsIcon className="size-4" />
              Settings
            </CommandItem>
          </CommandGroup>
        </CommandList>

        <CommandFooter>
          <div className="flex items-center gap-3 text-xs text-muted-foreground">
            <span className="flex items-center gap-1">
              <Kbd>↑↓</Kbd> navigate
            </span>
            <span className="flex items-center gap-1">
              <Kbd>↵</Kbd> select
            </span>
            <span className="flex items-center gap-1">
              <Kbd>esc</Kbd> close
            </span>
          </div>
        </CommandFooter>
      </CommandDialogPopup>
    </CommandDialog>
  );
}
