package runtime

import (
	"errors"

	symphonycfg "github.com/wibus-wee/synclax/pkg/symphony/config"
	"github.com/wibus-wee/synclax/pkg/symphony/template"
	"github.com/wibus-wee/synclax/pkg/symphony/workflow"
)

type EffectiveRuntime struct {
	Config   symphonycfg.EffectiveConfig
	Renderer *template.Renderer
}

var (
	ErrWorkflowInvalid = errors.New("workflow_invalid")
)

func Build(def *workflow.Definition) (*EffectiveRuntime, error) {
	if def == nil {
		return nil, ErrWorkflowInvalid
	}
	cfg, err := symphonycfg.FromWorkflowConfig(def.Config)
	if err != nil {
		return nil, err
	}
	renderer, err := template.Compile(def.PromptTemplate)
	if err != nil {
		return nil, err
	}
	return &EffectiveRuntime{
		Config:   cfg,
		Renderer: renderer,
	}, nil
}
