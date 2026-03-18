package provider

import (
	"fmt"
	"strings"

	"github.com/wibus-wee/synclax/pkg/symphony/codex"
	symphonycfg "github.com/wibus-wee/synclax/pkg/symphony/config"
)

func Build(cfg symphonycfg.EffectiveConfig) (Provider, error) {
	kind := strings.TrimSpace(cfg.Provider.Kind)
	switch kind {
	case "", "codex":
		srv := codex.NewAppServer(codex.AppServerOptions{
			Command:        cfg.Codex.Command,
			ApprovalPolicy: cfg.Codex.ApprovalPolicy,
			ThreadSandbox:  cfg.Codex.ThreadSandbox,
			SandboxPolicy:  cfg.Codex.TurnSandboxPolicy,
			ReadTimeout:    cfg.Codex.ReadTimeout,
			TurnTimeout:    cfg.Codex.TurnTimeout,

			LinearEndpoint: cfg.Tracker.Endpoint,
			LinearAPIKey:   cfg.Tracker.APIKey,
			LinearTimeout:  cfg.Tracker.Timeout,
		})
		return newCodexProvider(srv), nil
	default:
		return nil, fmt.Errorf("unsupported provider kind: %s", kind)
	}
}

