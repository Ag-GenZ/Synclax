package workflow

import (
	"bytes"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type Definition struct {
	Config         map[string]any
	PromptTemplate string
}

func Load(path string) (*Definition, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrMissingWorkflowFile
		}
		return nil, ErrWorkflowParseError
	}

	config, prompt, err := parseWorkflow(raw)
	if err != nil {
		return nil, err
	}

	return &Definition{
		Config:         config,
		PromptTemplate: prompt,
	}, nil
}

func parseWorkflow(raw []byte) (map[string]any, string, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return map[string]any{}, "", nil
	}

	lines := strings.Split(string(trimmed), "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return map[string]any{}, strings.TrimSpace(string(raw)), nil
	}

	// Front matter is everything until the next --- delimiter.
	end := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			end = i
			break
		}
	}
	if end == -1 {
		return nil, "", ErrWorkflowParseError
	}

	frontMatter := strings.Join(lines[1:end], "\n")
	var decoded any
	if err := yaml.Unmarshal([]byte(frontMatter), &decoded); err != nil {
		return nil, "", ErrWorkflowParseError
	}

	var cfg map[string]any
	switch v := decoded.(type) {
	case nil:
		cfg = map[string]any{}
	case map[string]any:
		cfg = v
	case map[any]any:
		cfg = make(map[string]any, len(v))
		for k, vv := range v {
			ks, ok := k.(string)
			if !ok {
				continue
			}
			cfg[ks] = vv
		}
	default:
		return nil, "", ErrWorkflowFrontMatterNotAMap
	}

	body := ""
	if end+1 < len(lines) {
		body = strings.TrimSpace(strings.Join(lines[end+1:], "\n"))
	}
	return cfg, body, nil
}
