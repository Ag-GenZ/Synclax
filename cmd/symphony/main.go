package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/wibus-wee/synclax/pkg/symphony/orchestrator"
)

func main() {
	port := flag.Int("port", -1, "HTTP server port (optional, -1 disables)")
	workflowFlag := flag.String("workflow", "", "Path to WORKFLOW.md (optional; positional arg also supported)")
	flag.Parse()

	workflowPath := *workflowFlag
	if workflowPath == "" {
		workflowPath = "WORKFLOW.md"
	}
	if flag.NArg() > 0 {
		workflowPath = flag.Arg(0)
	}

	if _, err := os.Stat(workflowPath); err != nil {
		log.Fatalf("workflow file not found: %s (error=%v)", workflowPath, err)
	}

	// Always pass the flag value through so `-1` can disable even when WORKFLOW.md sets server.port.
	portOverride := port

	orch, err := orchestrator.New(orchestrator.Options{WorkflowPath: workflowPath})
	if err != nil {
		log.Fatal(err)
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

	if err := orch.Run(ctx, portOverride); err != nil {
		log.Fatal(err)
	}
}
