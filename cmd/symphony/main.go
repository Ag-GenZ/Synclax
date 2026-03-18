package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/wibus-wee/synclax/pkg/symphony/orchestrator"
)

type workflowPathsFlag []string

func (w *workflowPathsFlag) String() string {
	if w == nil || len(*w) == 0 {
		return ""
	}
	return strings.Join(*w, ",")
}

func (w *workflowPathsFlag) Set(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	*w = append(*w, value)
	return nil
}

func main() {
	port := flag.Int("port", -1, "HTTP server port (optional, -1 disables)")
	var workflowFlags workflowPathsFlag
	flag.Var(&workflowFlags, "workflow", "Path to WORKFLOW.md (repeatable; positional args also supported)")
	flag.Parse()

	if flag.NArg() > 0 {
		workflowFlags = append(workflowFlags[:0], flag.Args()...)
	}
	if len(workflowFlags) == 0 {
		workflowFlags = append(workflowFlags, "WORKFLOW.md")
	}

	for _, workflowPath := range workflowFlags {
		if _, err := os.Stat(workflowPath); err != nil {
			log.Fatalf("workflow file not found: %s (error=%v)", workflowPath, err)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		s := <-sig
		fmt.Printf("received signal %s, shutting down\n", s)
		cancel()
	}()

	orchs := make([]*orchestrator.Orchestrator, 0, len(workflowFlags))
	for _, workflowPath := range workflowFlags {
		orch, err := orchestrator.New(orchestrator.Options{WorkflowPath: workflowPath})
		if err != nil {
			log.Fatal(err)
		}
		orchs = append(orchs, orch)
	}

	errCh := make(chan error, len(orchs))
	for i, orch := range orchs {
		// Always pass the flag value through so `-1` can disable even when WORKFLOW.md sets server.port.
		override := *port
		if override >= 0 && len(orchs) > 1 {
			override = override + i
		}
		portOverride := &override

		go func(o *orchestrator.Orchestrator, po *int) {
			errCh <- o.Run(ctx, po)
		}(orch, portOverride)
	}

	var firstErr error
	for range orchs {
		err := <-errCh
		if err != nil && firstErr == nil {
			firstErr = err
			cancel()
		}
	}
	if firstErr != nil {
		log.Fatal(firstErr)
	}
}
