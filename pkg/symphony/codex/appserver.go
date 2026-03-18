package codex

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"os/exec"
	"strings"
	"sync"
	"time"
)

type AppServer struct {
	command string

	approvalPolicy any
	threadSandbox  any
	sandboxPolicy  any

	readTimeout time.Duration
	turnTimeout time.Duration

	tools toolExecutor
}

type AppServerOptions struct {
	Command        string
	ApprovalPolicy any
	ThreadSandbox  any
	SandboxPolicy  any
	ReadTimeout    time.Duration
	TurnTimeout    time.Duration

	LinearEndpoint string
	LinearAPIKey   string
	LinearTimeout  time.Duration
}

func NewAppServer(opts AppServerOptions) *AppServer {
	readTimeout := opts.ReadTimeout
	if readTimeout <= 0 {
		readTimeout = 5 * time.Second
	}
	turnTimeout := opts.TurnTimeout
	if turnTimeout <= 0 {
		turnTimeout = time.Hour
	}

	tools := newDynamicTools(dynamicToolsOptions{
		LinearEndpoint: opts.LinearEndpoint,
		LinearAPIKey:   opts.LinearAPIKey,
		Timeout:        opts.LinearTimeout,
	})
	return &AppServer{
		command:        strings.TrimSpace(opts.Command),
		approvalPolicy: opts.ApprovalPolicy,
		threadSandbox:  opts.ThreadSandbox,
		sandboxPolicy:  opts.SandboxPolicy,
		readTimeout:    readTimeout,
		turnTimeout:    turnTimeout,
		tools:          tools,
	}
}

type Session struct {
	threadID string
	proc     *process
	client   *rpcClient
}

func (s *Session) ThreadID() string { return s.threadID }

func (s *Session) PID() *int {
	if s == nil || s.proc == nil || s.proc.cmd == nil || s.proc.cmd.Process == nil {
		return nil
	}
	pid := s.proc.cmd.Process.Pid
	return &pid
}

func (s *Session) Close() error {
	if s == nil || s.proc == nil {
		return nil
	}
	return s.proc.close()
}

func (a *AppServer) StartSession(ctx context.Context, workspacePath string) (*Session, error) {
	if a == nil || a.command == "" {
		return nil, &Error{Category: ErrCodexNotFound.Error(), Err: errors.New("codex.command is empty")}
	}

	proc, err := startProcess(ctx, a.command, workspacePath)
	if err != nil {
		return nil, err
	}

	client := newRPCClient(proc, a.readTimeout, a.tools)
	if err := client.start(ctx); err != nil {
		_ = proc.close()
		return nil, err
	}

	threadID, err := client.startThread(ctx, workspacePath, a.approvalPolicy, a.threadSandbox)
	if err != nil {
		_ = proc.close()
		return nil, err
	}

	return &Session{threadID: threadID, proc: proc, client: client}, nil
}

type TurnResult struct {
	TurnID         string
	LastEvent      string
	LastMessage    string
	InputTokens    int
	OutputTokens   int
	TotalTokens    int
	RateLimits     map[string]any
	EndedWithError error
}

func (a *AppServer) RunTurn(ctx context.Context, session *Session, workspacePath string, title string, inputText string, onEvent func(event string, payload map[string]any)) (*TurnResult, error) {
	if session == nil || session.proc == nil {
		return nil, &Error{Category: ErrResponseError.Error(), Err: errors.New("nil session")}
	}
	if session.client == nil {
		return nil, &Error{Category: ErrResponseError.Error(), Err: errors.New("nil rpc client")}
	}

	tctx, cancel := context.WithTimeout(ctx, a.turnTimeout)
	defer cancel()

	turnID, err := session.client.startTurn(tctx, session.threadID, workspacePath, title, inputText, a.approvalPolicy, a.sandboxPolicy)
	if err != nil {
		return nil, err
	}

	result := &TurnResult{TurnID: turnID}
	if onEvent != nil {
		onEvent("turn/started", map[string]any{
			"turn_id":   turnID,
			"thread_id": session.threadID,
		})
	}
	for {
		select {
		case <-tctx.Done():
			_ = session.proc.kill()
			return nil, &Error{Category: ErrTurnTimeout.Error(), Err: tctx.Err()}
		case msg, ok := <-session.client.events:
			if !ok {
				return nil, &Error{Category: ErrPortExit.Error(), Err: errors.New("app-server exited")}
			}

			if msg.method != "" {
				result.LastEvent = msg.method
			}
			if onEvent != nil {
				onEvent(msg.method, msg.payload)
			}

			updateUsageAndRateLimits(result, msg.payload)

			switch msg.method {
			case "turn/completed":
				return result, nil
			case "turn/failed":
				return nil, &Error{Category: ErrTurnFailed.Error(), Err: errors.New("turn failed")}
			case "turn/cancelled":
				return nil, &Error{Category: ErrTurnCancelled.Error(), Err: errors.New("turn cancelled")}
			default:
				if isUserInputRequired(msg.method, msg.payload) {
					return nil, &Error{Category: ErrTurnInputRequired.Error(), Err: errors.New("user input required")}
				}
			}
		}
	}
}

type rpcMessage struct {
	idPresent bool
	id        int
	method    string
	payload   map[string]any
}

type rpcClient struct {
	proc        *process
	readTimeout time.Duration
	tools       toolExecutor

	mu        sync.Mutex
	nextID    int
	waiters   map[int]chan rpcEnvelope
	events    chan rpcMessage
	started   bool
	startOnce sync.Once
}

type rpcEnvelope struct {
	ID     json.RawMessage `json:"id,omitempty"`
	Method string          `json:"method,omitempty"`
	Params json.RawMessage `json:"params,omitempty"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  json.RawMessage `json:"error,omitempty"`
}

func newRPCClient(proc *process, readTimeout time.Duration, tools toolExecutor) *rpcClient {
	return &rpcClient{
		proc:        proc,
		readTimeout: readTimeout,
		tools:       tools,
		nextID:      1,
		waiters:     map[int]chan rpcEnvelope{},
		events:      make(chan rpcMessage, 256),
	}
}

func (c *rpcClient) start(ctx context.Context) error {
	c.startOnce.Do(func() {
		go c.readLoop()
	})
	if c.started {
		return nil
	}
	if _, err := c.request(ctx, "initialize", map[string]any{
		"clientInfo":   map[string]any{"name": "symphony", "version": "1.0"},
		"capabilities": map[string]any{},
	}); err != nil {
		return err
	}
	// initialized notification
	if err := c.notify("initialized", map[string]any{}); err != nil {
		return err
	}
	c.started = true
	return nil
}

func (c *rpcClient) startThread(ctx context.Context, cwd string, approvalPolicy any, sandbox any) (string, error) {
	params := map[string]any{
		"approvalPolicy": approvalPolicy,
		"sandbox":        sandbox,
		"cwd":            cwd,
	}
	if c.tools != nil {
		params["dynamicTools"] = c.tools.ToolSpecs()
	}
	resp, err := c.request(ctx, "thread/start", params)
	if err != nil {
		return "", err
	}
	var decoded struct {
		Thread struct {
			ID string `json:"id"`
		} `json:"thread"`
	}
	if err := json.Unmarshal(resp.Result, &decoded); err != nil {
		return "", &Error{Category: ErrResponseError.Error(), Err: err}
	}
	if decoded.Thread.ID == "" {
		return "", &Error{Category: ErrResponseError.Error(), Err: errors.New("missing thread id")}
	}
	return decoded.Thread.ID, nil
}

func (c *rpcClient) startTurn(ctx context.Context, threadID, cwd, title, inputText string, approvalPolicy any, sandboxPolicy any) (string, error) {
	resp, err := c.request(ctx, "turn/start", map[string]any{
		"threadId": threadID,
		"input": []map[string]any{
			{"type": "text", "text": inputText},
		},
		"cwd":            cwd,
		"title":          title,
		"approvalPolicy": approvalPolicy,
		"sandboxPolicy":  sandboxPolicy,
	})
	if err != nil {
		return "", err
	}
	var decoded struct {
		Turn struct {
			ID string `json:"id"`
		} `json:"turn"`
	}
	if err := json.Unmarshal(resp.Result, &decoded); err != nil {
		return "", &Error{Category: ErrResponseError.Error(), Err: err}
	}
	if decoded.Turn.ID == "" {
		return "", &Error{Category: ErrResponseError.Error(), Err: errors.New("missing turn id")}
	}
	return decoded.Turn.ID, nil
}

func (c *rpcClient) notify(method string, params any) error {
	msg := map[string]any{
		"method": method,
		"params": params,
	}
	return c.proc.writeJSON(msg)
}

func (c *rpcClient) request(ctx context.Context, method string, params any) (rpcEnvelope, error) {
	c.mu.Lock()
	id := c.nextID
	c.nextID++
	ch := make(chan rpcEnvelope, 1)
	c.waiters[id] = ch
	c.mu.Unlock()

	cleanupWaiter := func() {
		c.mu.Lock()
		delete(c.waiters, id)
		c.mu.Unlock()
	}

	msg := map[string]any{
		"id":     id,
		"method": method,
		"params": params,
	}
	if err := c.proc.writeJSON(msg); err != nil {
		cleanupWaiter()
		return rpcEnvelope{}, &Error{Category: ErrResponseError.Error(), Err: err}
	}

	timeout := c.readTimeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		cleanupWaiter()
		return rpcEnvelope{}, &Error{Category: ErrResponseTimeout.Error(), Err: ctx.Err()}
	case <-timer.C:
		cleanupWaiter()
		return rpcEnvelope{}, &Error{Category: ErrResponseTimeout.Error(), Err: errors.New("timeout")}
	case resp := <-ch:
		return resp, nil
	}
}

func (c *rpcClient) readLoop() {
	defer close(c.events)
	for {
		line, err := c.proc.readLine()
		if err != nil {
			return
		}
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}

		var env rpcEnvelope
		if err := json.Unmarshal(line, &env); err != nil {
			c.events <- rpcMessage{method: ErrMalformedProtocolMsg.Error(), payload: map[string]any{"error": err.Error()}}
			continue
		}

		id, ok := parseNumericID(env.ID)
		if ok && (len(env.Result) > 0 || len(env.Error) > 0) {
			c.mu.Lock()
			ch := c.waiters[id]
			delete(c.waiters, id)
			c.mu.Unlock()
			if ch != nil {
				ch <- env
				close(ch)
				continue
			}
		}

		payload := map[string]any{}
		if len(env.Params) > 0 {
			_ = json.Unmarshal(env.Params, &payload)
		}

		// Handle inbound requests that expect an immediate response (approvals / tools).
		if ok && len(env.Result) == 0 && len(env.Error) == 0 && env.Method != "" {
			switch env.Method {
			case "item/tool/call":
				toolName := toolCallName(payload)
				arguments := toolCallArguments(payload)
				var res map[string]any
				if c.tools != nil {
					res = c.tools.Execute(toolName, arguments)
				} else {
					res = dynamicToolResponse(false, map[string]any{
						"error": map[string]any{
							"message":        "Dynamic tools are not configured.",
							"supportedTools": []any{linearGraphQLToolName},
						},
					})
				}
				_ = c.proc.writeJSON(map[string]any{
					"id":     id,
					"result": res,
				})

				eventName := "tool_call_failed"
				if ok, _ := res["success"].(bool); ok {
					eventName = "tool_call_completed"
				} else if strings.TrimSpace(toolName) == "" {
					eventName = ErrUnsupportedToolCall.Error()
				}
				c.events <- rpcMessage{method: eventName, payload: map[string]any{"tool": toolName}}
				continue
			default:
				// Preserve legacy "auto approve everything" behavior when app-server emits approval requests.
				// Symphony runs unattended by default; if the operator wants interactive approvals, they should
				// choose a compatible Codex approval policy and extend this client accordingly.
				if strings.Contains(strings.ToLower(env.Method), "approval") {
					_ = c.proc.writeJSON(map[string]any{
						"id": id,
						"result": map[string]any{
							"approved": true,
						},
					})
					c.events <- rpcMessage{method: "approval_auto_approved", payload: map[string]any{"method": env.Method}}
					continue
				}
				if strings.Contains(env.Method, "tool") {
					_ = c.proc.writeJSON(map[string]any{
						"id":     id,
						"result": map[string]any{"success": false, "error": ErrUnsupportedToolCall.Error()},
					})
					c.events <- rpcMessage{method: ErrUnsupportedToolCall.Error(), payload: map[string]any{"method": env.Method}}
					continue
				}
			}
		}

		c.events <- rpcMessage{idPresent: ok, id: id, method: env.Method, payload: payload}
	}
}

func toolCallName(params map[string]any) string {
	if params == nil {
		return ""
	}
	for _, key := range []string{"tool", "name"} {
		if v, ok := params[key]; ok {
			if s, ok := v.(string); ok {
				return strings.TrimSpace(s)
			}
		}
	}
	return ""
}

func toolCallArguments(params map[string]any) any {
	if params == nil {
		return map[string]any{}
	}
	if v, ok := params["arguments"]; ok {
		return v
	}
	return map[string]any{}
}

func parseNumericID(raw json.RawMessage) (int, bool) {
	if len(raw) == 0 {
		return 0, false
	}
	var i int
	if err := json.Unmarshal(raw, &i); err == nil {
		return i, true
	}
	var f float64
	if err := json.Unmarshal(raw, &f); err == nil {
		return int(f), true
	}
	return 0, false
}

type process struct {
	cmd   *exec.Cmd
	stdin io.WriteCloser
	out   *bufio.Reader

	mu     sync.Mutex
	closed bool
}

func startProcess(ctx context.Context, command, cwd string) (*process, error) {
	cmd := exec.CommandContext(ctx, "bash", "-lc", command)
	cmd.Dir = cwd
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, &Error{Category: ErrCodexNotFound.Error(), Err: err}
	}

	p := &process{
		cmd:   cmd,
		stdin: stdin,
		out:   bufio.NewReaderSize(stdout, 128*1024),
	}

	go func() {
		s := bufio.NewScanner(stderr)
		for s.Scan() {
			log.Printf("symphony codex_stderr msg=%s", strings.TrimSpace(s.Text()))
		}
	}()

	return p, nil
}

func (p *process) readLine() ([]byte, error) {
	line, err := p.out.ReadBytes('\n')
	if err != nil {
		return nil, err
	}
	if len(line) > 10*1024*1024 {
		return nil, errors.New("protocol line too large")
	}
	return line, nil
}

func (p *process) writeJSON(v any) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return errors.New("process closed")
	}
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	b = append(b, '\n')
	_, err = p.stdin.Write(b)
	return err
}

func (p *process) kill() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed || p.cmd.Process == nil {
		return nil
	}
	return p.cmd.Process.Kill()
}

func (p *process) close() error {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil
	}
	p.closed = true
	_ = p.stdin.Close()
	p.mu.Unlock()

	_ = p.kill()
	_ = p.cmd.Wait()
	return nil
}

func isUserInputRequired(method string, payload map[string]any) bool {
	m := strings.ToLower(method)
	if strings.Contains(m, "requestuserinput") || strings.Contains(m, "input_required") {
		return true
	}
	if payload == nil {
		return false
	}
	if v, ok := payload["inputRequired"].(bool); ok && v {
		return true
	}
	return false
}

func updateUsageAndRateLimits(res *TurnResult, payload map[string]any) {
	if res == nil || payload == nil {
		return
	}

	asMap := func(v any) (map[string]any, bool) {
		m, ok := v.(map[string]any)
		return m, ok
	}

	getMap := func(m map[string]any, keys ...string) map[string]any {
		for _, k := range keys {
			if v, ok := m[k]; ok {
				if mm, ok := asMap(v); ok {
					return mm
				}
			}
		}
		return nil
	}

	// Best-effort extraction; tolerate schema drift across Codex protocol versions.
	//
	// Seen shapes include:
	// - { usage: { input_tokens, output_tokens, total_tokens, rate_limits } }
	// - { tokenUsage: { inputTokens, outputTokens, totalTokens } }
	// - { inputTokens, outputTokens, totalTokens }
	usage := getMap(payload, "usage", "tokenUsage", "token_usage")
	if usage == nil {
		usage = payload
	}

	res.InputTokens = asInt(usage["input_tokens"], res.InputTokens)
	res.InputTokens = asInt(usage["inputTokens"], res.InputTokens)
	res.InputTokens = asInt(usage["prompt_tokens"], res.InputTokens)
	res.InputTokens = asInt(usage["promptTokens"], res.InputTokens)

	res.OutputTokens = asInt(usage["output_tokens"], res.OutputTokens)
	res.OutputTokens = asInt(usage["outputTokens"], res.OutputTokens)
	res.OutputTokens = asInt(usage["completion_tokens"], res.OutputTokens)
	res.OutputTokens = asInt(usage["completionTokens"], res.OutputTokens)

	res.TotalTokens = asInt(usage["total_tokens"], res.TotalTokens)
	res.TotalTokens = asInt(usage["totalTokens"], res.TotalTokens)

	// Rate limits
	if rl := getMap(usage, "rate_limits", "rateLimits"); rl != nil {
		res.RateLimits = rl
	}
	if rl := getMap(payload, "rate_limits", "rateLimits"); rl != nil {
		res.RateLimits = rl
	}
}

func asInt(v any, fallback int) int {
	switch t := v.(type) {
	case int:
		return t
	case int64:
		return int(t)
	case float64:
		return int(t)
	case json.Number:
		if n, err := t.Int64(); err == nil {
			return int(n)
		}
	}
	return fallback
}
