package codex

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	linearGraphQLToolName = "linear_graphql"
)

type toolExecutor interface {
	ToolSpecs() []map[string]any
	Execute(tool string, arguments any) map[string]any
}

type dynamicTools struct {
	linearEndpoint string
	linearAPIKey   string
	timeout        time.Duration
	httpClient     *http.Client
}

type dynamicToolsOptions struct {
	LinearEndpoint string
	LinearAPIKey   string
	Timeout        time.Duration
	HTTPClient     *http.Client
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
	return &dynamicTools{
		linearEndpoint: strings.TrimSpace(opts.LinearEndpoint),
		linearAPIKey:   strings.TrimSpace(opts.LinearAPIKey),
		timeout:        timeout,
		httpClient:     hc,
	}
}

func (d *dynamicTools) ToolSpecs() []map[string]any {
	return []map[string]any{
		{
			"name":        linearGraphQLToolName,
			"description": "Execute a raw GraphQL query or mutation against Linear using Symphony's configured auth.",
			"inputSchema": map[string]any{
				"type":                 "object",
				"additionalProperties": false,
				"required":             []any{"query"},
				"properties": map[string]any{
					"query": map[string]any{
						"type":        "string",
						"description": "GraphQL query or mutation document to execute against Linear.",
					},
					"variables": map[string]any{
						"type":                 []any{"object", "null"},
						"description":          "Optional GraphQL variables object.",
						"additionalProperties": true,
					},
				},
			},
		},
	}
}

func (d *dynamicTools) Execute(tool string, arguments any) map[string]any {
	switch strings.TrimSpace(tool) {
	case linearGraphQLToolName:
		return d.executeLinearGraphQL(arguments)
	case "":
		return d.failure(map[string]any{
			"error": map[string]any{
				"message":        "Missing tool name for dynamic tool call.",
				"supportedTools": []any{linearGraphQLToolName},
			},
		})
	default:
		return d.failure(map[string]any{
			"error": map[string]any{
				"message":        fmt.Sprintf("Unsupported dynamic tool: %q.", tool),
				"supportedTools": []any{linearGraphQLToolName},
			},
		})
	}
}

func (d *dynamicTools) executeLinearGraphQL(arguments any) map[string]any {
	if strings.TrimSpace(d.linearAPIKey) == "" {
		return d.failure(map[string]any{
			"error": map[string]any{
				"message": "Symphony is missing Linear auth. Set `tracker.api_key` in `WORKFLOW.md` or export `LINEAR_API_KEY`.",
			},
		})
	}
	if strings.TrimSpace(d.linearEndpoint) == "" {
		return d.failure(map[string]any{
			"error": map[string]any{
				"message": "Symphony is missing Linear endpoint. Set `tracker.endpoint` in `WORKFLOW.md`.",
			},
		})
	}

	query, variables, err := normalizeLinearGraphQLArguments(arguments)
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
				"message": "Linear GraphQL tool execution failed.",
				"reason":  err.Error(),
			},
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), d.timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, d.linearEndpoint, bytes.NewReader(encoded))
	if err != nil {
		return d.failure(map[string]any{
			"error": map[string]any{
				"message": "Linear GraphQL request failed before receiving a successful response.",
				"reason":  err.Error(),
			},
		})
	}
	req.Header.Set("Authorization", d.linearAPIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return d.failure(map[string]any{
			"error": map[string]any{
				"message": "Linear GraphQL request failed before receiving a successful response.",
				"reason":  err.Error(),
			},
		})
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4*1024*1024))
	if err != nil {
		return d.failure(map[string]any{
			"error": map[string]any{
				"message": "Linear GraphQL request failed before receiving a successful response.",
				"reason":  err.Error(),
			},
		})
	}

	if resp.StatusCode != http.StatusOK {
		return d.failure(map[string]any{
			"error": map[string]any{
				"message": fmt.Sprintf("Linear GraphQL request failed with HTTP %d.", resp.StatusCode),
				"status":  resp.StatusCode,
				"body":    strings.TrimSpace(string(body)),
			},
		})
	}

	var decoded any
	if err := json.Unmarshal(body, &decoded); err != nil {
		return d.failure(map[string]any{
			"error": map[string]any{
				"message": "Linear GraphQL tool execution failed.",
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

func normalizeLinearGraphQLArguments(arguments any) (string, map[string]any, error) {
	switch v := arguments.(type) {
	case string:
		q := strings.TrimSpace(v)
		if q == "" {
			return "", nil, errors.New("`linear_graphql` requires a non-empty `query` string.")
		}
		return q, map[string]any{}, nil
	case map[string]any:
		return normalizeLinearGraphQLMap(v)
	default:
		if arguments == nil {
			return "", nil, errors.New("`linear_graphql` requires a non-empty `query` string.")
		}
		return "", nil, errors.New("`linear_graphql` expects either a GraphQL query string or an object with `query` and optional `variables`.")
	}
}

func normalizeLinearGraphQLMap(arguments map[string]any) (string, map[string]any, error) {
	queryAny, ok := arguments["query"]
	if !ok {
		// tolerate atom-style keys if present (best-effort); most decode paths use strings.
		if q, ok := arguments[":query"]; ok {
			queryAny = q
			ok = true
		}
	}
	query, ok := queryAny.(string)
	if !ok || strings.TrimSpace(query) == "" {
		return "", nil, errors.New("`linear_graphql` requires a non-empty `query` string.")
	}
	query = strings.TrimSpace(query)

	variablesAny, hasVars := arguments["variables"]
	if !hasVars {
		// default
		return query, map[string]any{}, nil
	}
	if variablesAny == nil {
		return query, map[string]any{}, nil
	}
	vars, ok := variablesAny.(map[string]any)
	if ok {
		return query, vars, nil
	}
	return "", nil, errors.New("`linear_graphql.variables` must be a JSON object when provided.")
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
