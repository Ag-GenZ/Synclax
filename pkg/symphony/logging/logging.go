package logging

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	symphonycfg "github.com/wibus-wee/synclax/pkg/symphony/config"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	mu         sync.Mutex
	curFile    string
	curCloser  io.Closer
	configured bool
)

// Configure applies a process-wide logger sink for Symphony by configuring the
// standard library's `log` package output.
//
// If cfg.File is blank, logging remains on the default stderr sink.
// This function is safe to call multiple times (for example, on WORKFLOW reload).
func Configure(cfg symphonycfg.LoggingConfig) {
	file := strings.TrimSpace(cfg.File)

	mu.Lock()
	defer mu.Unlock()

	// First call: keep existing sink unless a file path is provided.
	if !configured && file == "" {
		configured = true
		return
	}

	if file == "" {
		// Reset to stderr.
		if curCloser != nil {
			_ = curCloser.Close()
		}
		curCloser = nil
		curFile = ""
		log.SetOutput(os.Stderr)
		configured = true
		return
	}

	expanded := filepath.Clean(file)
	if filepath.IsAbs(expanded) == false {
		// Interpret relative paths from the current working directory.
		if cwd, err := os.Getwd(); err == nil {
			expanded = filepath.Join(cwd, expanded)
		}
	}

	// No-op if unchanged.
	if configured && curFile == expanded {
		return
	}

	_ = os.MkdirAll(filepath.Dir(expanded), 0o755)

	if curCloser != nil {
		_ = curCloser.Close()
	}

	sink := &lumberjack.Logger{
		Filename:   expanded,
		MaxSize:    clampInt(cfg.MaxSizeMB, 1, 1024), // MB
		MaxBackups: clampInt(cfg.MaxBackups, 1, 100),
		MaxAge:     clampInt(cfg.MaxAgeDays, 0, 3650), // days
		Compress:   cfg.Compress,
	}
	curCloser = sink
	curFile = expanded
	log.SetOutput(sink)

	// Ensure logs are timestamped and UTC for cross-host consistency.
	log.SetFlags(log.LstdFlags | log.LUTC)
	configured = true
}

func clampInt(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

