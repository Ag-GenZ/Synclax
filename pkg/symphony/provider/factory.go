package provider

import (
	"fmt"
	"os"
	"strings"

	"github.com/wibus-wee/synclax/pkg/symphony/codex"
	symphonycfg "github.com/wibus-wee/synclax/pkg/symphony/config"
	"github.com/wibus-wee/synclax/pkg/symphony/tracker"
)

func Build(cfg symphonycfg.EffectiveConfig) (Provider, error) {
	kind := strings.TrimSpace(cfg.Provider.Kind)
	switch kind {
	case "", "codex":
		trackerKind := strings.TrimSpace(cfg.Tracker.Kind)
		appOpts := codex.AppServerOptions{
			Command:            cfg.Codex.Command,
			ApprovalPolicy:     cfg.Codex.ApprovalPolicy,
			ThreadSandbox:      cfg.Codex.ThreadSandbox,
			SandboxPolicy:      cfg.Codex.TurnSandboxPolicy,
			ReadTimeout:        cfg.Codex.ReadTimeout,
			TurnTimeout:        cfg.Codex.TurnTimeout,
			TrackerKind:        trackerKind,
			DynamicToolTimeout: cfg.Tracker.Timeout,
		}
		switch trackerKind {
		case "", "linear":
			appOpts.LinearEndpoint = tracker.StringParam(cfg.Tracker.Params, "endpoint", "")
			appOpts.LinearAPIKey = tracker.StringParam(cfg.Tracker.Params, "api_key", "")
			if strings.TrimSpace(appOpts.LinearAPIKey) == "" {
				appOpts.LinearAPIKey = os.Getenv("LINEAR_API_KEY")
			}
		case "github":
			appOpts.GitHubEndpoint = tracker.StringParam(cfg.Tracker.Params, "endpoint", "https://api.github.com/graphql")
			appOpts.GitHubToken = tracker.StringParam(cfg.Tracker.Params, "token", "")
			if strings.TrimSpace(appOpts.GitHubToken) == "" {
				appOpts.GitHubToken = os.Getenv("GITHUB_TOKEN")
			}
		}
		srv := codex.NewAppServer(appOpts)
		return newCodexProvider(srv), nil
	default:
		return nil, fmt.Errorf("unsupported provider kind: %s", kind)
	}
}
