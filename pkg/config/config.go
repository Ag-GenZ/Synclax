package config

import (
	"os"

	"github.com/cloudcarver/anclax/lib/conf"
	anclax_config "github.com/cloudcarver/anclax/pkg/config"
)

type Config struct {
	Anclax   anclax_config.Config `yaml:"anclax,omitempty"`
	Symphony SymphonyConfig       `yaml:"symphony,omitempty"`
}

type SymphonyConfig struct {
	// WorkflowPath is the path to WORKFLOW.md used by Symphony. If empty, defaults to "WORKFLOW.md".
	WorkflowPath string `yaml:"workflow_path,omitempty"`
	// HTTPPort enables Symphony's internal debug HTTP server when set. If nil, it is disabled.
	HTTPPort *int `yaml:"http_port,omitempty"`
}

const (
	envPrefix  = "MYAPP_"
	configFile = "app.yaml"
)

func NewConfig() (*Config, error) {
	c := &Config{}
	if err := conf.FetchConfig((func() string {
		if _, err := os.Stat(configFile); err != nil {
			return ""
		}
		return configFile
	})(), envPrefix, c); err != nil {
		return nil, err
	}

	return c, nil
}
