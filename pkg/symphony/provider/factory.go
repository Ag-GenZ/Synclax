package provider

import (
	"fmt"
	"os"
	"strings"

	"github.com/wibus-wee/synclax/pkg/symphony/codex"
	symphonycfg "github.com/wibus-wee/synclax/pkg/symphony/config"
	"github.com/wibus-wee/synclax/pkg/symphony/tracker/linear"
)

func Build(cfg symphonycfg.EffectiveConfig) (Provider, error) {
	kind := strings.TrimSpace(cfg.Provider.Kind)
	switch kind {
	case "", "codex":
		linearEndpoint := linear.StringParam(cfg.Tracker.Params, "endpoint", "")
		linearAPIKey := linear.StringParam(cfg.Tracker.Params, "api_key", "")
		if strings.TrimSpace(linearAPIKey) == "" {
			linearAPIKey = os.Getenv("LINEAR_API_KEY")
		}
		srv := codex.NewAppServer(codex.AppServerOptions{
			Command:        cfg.Codex.Command,
			ApprovalPolicy: cfg.Codex.ApprovalPolicy,
			ThreadSandbox:  cfg.Codex.ThreadSandbox,
			SandboxPolicy:  cfg.Codex.TurnSandboxPolicy,
			ReadTimeout:    cfg.Codex.ReadTimeout,
			TurnTimeout:    cfg.Codex.TurnTimeout,

			LinearEndpoint: linearEndpoint,
			LinearAPIKey:   linearAPIKey,
			LinearTimeout:  cfg.Tracker.Timeout,
		})
		return newCodexProvider(srv), nil
	default:
		return nil, fmt.Errorf("unsupported provider kind: %s", kind)
	}
}
