package codex

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"
)

const (
	githubGraphQLToolName = "github_graphql"
	linearGraphQLToolName = "linear_graphql"
)

type toolExecutor interface {
	ToolSpecs() []map[string]any
	Execute(tool string, arguments any) map[string]any
}

type dynamicTools struct {
	tools         map[string]graphQLTool
	order         []string
	supportedTool []any
	timeout       time.Duration
	httpClient    *http.Client
}

type dynamicToolsOptions struct {
	TrackerKind    string
	LinearEndpoint string
	LinearAPIKey   string
	GitHubEndpoint string
	GitHubToken    string
	Timeout        time.Duration
	HTTPClient     *http.Client
}

type graphQLTool struct {
	Name                   string
	Description            string
	QueryDescription       string
	Endpoint               string
	AuthorizationHeader    string
	AuthorizationValue     string
	MissingAuthMessage     string
	MissingEndpointMessage string
}

func newDynamicTools(opts dynamicToolsOptions) *dynamicTools {
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	hc := opts.HTTPClient
	if hc == nil {
		hc = &http.Client{Timeout: timeout}
	}

	tools := map[string]graphQLTool{}
	switch strings.TrimSpace(opts.TrackerKind) {
	case "github":
		token := strings.TrimSpace(opts.GitHubToken)
		authValue := ""
		if token != "" {
			authValue = "Bearer " + token
		}
		tools[githubGraphQLToolName] = graphQLTool{
			Name:                   githubGraphQLToolName,
			Description:            "Execute a raw GraphQL query or mutation against GitHub using Symphony's configured auth.",
			QueryDescription:       "GraphQL query or mutation document to execute against GitHub.",
			Endpoint:               strings.TrimSpace(opts.GitHubEndpoint),
			AuthorizationHeader:    "Authorization",
			AuthorizationValue:     authValue,
			MissingAuthMessage:     "Symphony is missing GitHub auth. Set `tracker.token` in `WORKFLOW.md` or export `GITHUB_TOKEN`.",
			MissingEndpointMessage: "Symphony is missing GitHub endpoint. Set `tracker.endpoint` in `WORKFLOW.md`.",
		}
	case "", "linear":
		tools[linearGraphQLToolName] = graphQLTool{
			Name:                   linearGraphQLToolName,
			Description:            "Execute a raw GraphQL query or mutation against Linear using Symphony's configured auth.",
			QueryDescription:       "GraphQL query or mutation document to execute against Linear.",
			Endpoint:               strings.TrimSpace(opts.LinearEndpoint),
			AuthorizationHeader:    "Authorization",
			AuthorizationValue:     strings.TrimSpace(opts.LinearAPIKey),
			MissingAuthMessage:     "Symphony is missing Linear auth. Set `tracker.api_key` in `WORKFLOW.md` or export `LINEAR_API_KEY`.",
			MissingEndpointMessage: "Symphony is missing Linear endpoint. Set `tracker.endpoint` in `WORKFLOW.md`.",
		}
	}

	supported := make([]any, 0, len(tools))
	order := make([]string, 0, len(tools))
	for name := range tools {
		order = append(order, name)
	}
	sort.Strings(order)
	for _, name := range order {
		supported = append(supported, name)
	}

	return &dynamicTools{
		tools:         tools,
		order:         order,
		supportedTool: supported,
		timeout:       timeout,
		httpClient:    hc,
	}
}

func (d *dynamicTools) ToolSpecs() []map[string]any {
	specs := make([]map[string]any, 0, len(d.order))
	for _, name := range d.order {
		tool := d.tools[name]
		specs = append(specs, graphQLToolSpec(tool))
	}
	return specs
}

func graphQLToolSpec(tool graphQLTool) map[string]any {
	return map[string]any{
		"name":        tool.Name,
		"description": tool.Description,
		"inputSchema": map[string]any{
			"type":                 "object",
			"additionalProperties": false,
			"required":             []any{"query"},
			"properties": map[string]any{
				"query": map[string]any{
					"type":        "string",
					"description": tool.QueryDescription,
				},
				"variables": map[string]any{
					"type":                 []any{"object", "null"},
					"description":          "Optional GraphQL variables object.",
					"additionalProperties": true,
				},
			},
		},
	}
}

func (d *dynamicTools) Execute(tool string, arguments any) map[string]any {
	name := strings.TrimSpace(tool)
	if name == "" {
		return d.failure(map[string]any{
			"error": map[string]any{
				"message":        "Missing tool name for dynamic tool call.",
				"supportedTools": d.supportedTools(),
			},
		})
	}
	cfg, ok := d.tools[name]
	if !ok {
		return d.failure(map[string]any{
			"error": map[string]any{
				"message":        fmt.Sprintf("Unsupported dynamic tool: %q.", tool),
				"supportedTools": d.supportedTools(),
			},
		})
	}
	return d.executeGraphQL(cfg, arguments)
}

func (d *dynamicTools) executeGraphQL(tool graphQLTool, arguments any) map[string]any {
	if strings.TrimSpace(tool.AuthorizationValue) == "" {
		return d.failure(map[string]any{
			"error": map[string]any{
				"message": tool.MissingAuthMessage,
			},
		})
	}
	if strings.TrimSpace(tool.Endpoint) == "" {
		return d.failure(map[string]any{
			"error": map[string]any{
				"message": tool.MissingEndpointMessage,
			},
		})
	}

	query, variables, err := normalizeGraphQLArguments(tool.Name, arguments)
	if err != nil {
		return d.failure(map[string]any{
			"error": map[string]any{
				"message": err.Error(),
			},
		})
	}

	payload := map[string]any{
		"query":     query,
		"variables": variables,
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return d.failure(map[string]any{
			"error": map[string]any{
				"message": fmt.Sprintf("%s tool execution failed.", toolLabel(tool.Name)),
				"reason":  err.Error(),
			},
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), d.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tool.Endpoint, bytes.NewReader(encoded))
	if err != nil {
		return d.failure(map[string]any{
			"error": map[string]any{
				"message": fmt.Sprintf("%s request failed before receiving a successful response.", toolLabel(tool.Name)),
				"reason":  err.Error(),
			},
		})
	}
	req.Header.Set(tool.AuthorizationHeader, tool.AuthorizationValue)
	req.Header.Set("Content-Type", "application/json")

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return d.failure(map[string]any{
			"error": map[string]any{
				"message": fmt.Sprintf("%s request failed before receiving a successful response.", toolLabel(tool.Name)),
				"reason":  err.Error(),
			},
		})
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4*1024*1024))
	if err != nil {
		return d.failure(map[string]any{
			"error": map[string]any{
				"message": fmt.Sprintf("%s request failed before receiving a successful response.", toolLabel(tool.Name)),
				"reason":  err.Error(),
			},
		})
	}

	if resp.StatusCode != http.StatusOK {
		return d.failure(map[string]any{
			"error": map[string]any{
				"message": fmt.Sprintf("%s request failed with HTTP %d.", toolLabel(tool.Name), resp.StatusCode),
				"status":  resp.StatusCode,
				"body":    strings.TrimSpace(string(body)),
			},
		})
	}

	var decoded any
	if err := json.Unmarshal(body, &decoded); err != nil {
		return d.failure(map[string]any{
			"error": map[string]any{
				"message": fmt.Sprintf("%s tool execution failed.", toolLabel(tool.Name)),
				"reason":  err.Error(),
			},
		})
	}

	success := true
	if m, ok := decoded.(map[string]any); ok {
		if errs, ok := m["errors"].([]any); ok && len(errs) > 0 {
			success = false
		}
	}
	return dynamicToolResponse(success, decoded)
}

func (d *dynamicTools) failure(payload any) map[string]any {
	return dynamicToolResponse(false, payload)
}

func (d *dynamicTools) supportedTools() []any {
	return append([]any(nil), d.supportedTool...)
}

func toolLabel(name string) string {
	switch name {
	case githubGraphQLToolName:
		return "GitHub GraphQL"
	default:
		return "Linear GraphQL"
	}
}

func normalizeGraphQLArguments(toolName string, arguments any) (string, map[string]any, error) {
	switch v := arguments.(type) {
	case string:
		q := strings.TrimSpace(v)
		if q == "" {
			return "", nil, fmt.Errorf("`%s` requires a non-empty `query` string.", toolName)
		}
		return q, map[string]any{}, nil
	case map[string]any:
		return normalizeGraphQLMap(toolName, v)
	default:
		if arguments == nil {
			return "", nil, fmt.Errorf("`%s` requires a non-empty `query` string.", toolName)
		}
		return "", nil, fmt.Errorf("`%s` expects either a GraphQL query string or an object with `query` and optional `variables`.", toolName)
	}
}

func normalizeGraphQLMap(toolName string, arguments map[string]any) (string, map[string]any, error) {
	queryAny, ok := arguments["query"]
	if !ok {
		if q, ok := arguments[":query"]; ok {
			queryAny = q
			ok = true
		}
	}
	query, ok := queryAny.(string)
	if !ok || strings.TrimSpace(query) == "" {
		return "", nil, fmt.Errorf("`%s` requires a non-empty `query` string.", toolName)
	}
	query = strings.TrimSpace(query)

	variablesAny, hasVars := arguments["variables"]
	if !hasVars || variablesAny == nil {
		return query, map[string]any{}, nil
	}
	vars, ok := variablesAny.(map[string]any)
	if ok {
		return query, vars, nil
	}
	return "", nil, fmt.Errorf("`%s.variables` must be a JSON object when provided.", toolName)
}

func dynamicToolResponse(success bool, payload any) map[string]any {
	output := encodeToolPayload(payload)
	return map[string]any{
		"success": success,
		"output":  output,
		"contentItems": []any{
			map[string]any{
				"type": "inputText",
				"text": output,
			},
		},
	}
}

func encodeToolPayload(payload any) string {
	switch payload.(type) {
	case map[string]any, []any:
		if b, err := json.MarshalIndent(payload, "", "  "); err == nil {
			return string(b)
		}
	}
	b, err := json.Marshal(payload)
	if err == nil {
		return string(b)
	}
	return fmt.Sprintf("%v", payload)
}

func toolSpecNames(specs []map[string]any) []any {
	out := make([]any, 0, len(specs))
	for _, spec := range specs {
		name, _ := spec["name"].(string)
		if strings.TrimSpace(name) == "" {
			continue
		}
		out = append(out, name)
	}
	return out
}
