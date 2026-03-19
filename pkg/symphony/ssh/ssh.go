package ssh

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"
)

type Target struct {
	User string // optional
	Host string
	Port string // optional, default "22"
}

func ParseTarget(hostString string) Target {
	s := strings.TrimSpace(hostString)
	out := Target{Port: "22"}
	if s == "" {
		return out
	}

	if at := strings.Index(s, "@"); at >= 0 {
		out.User = strings.TrimSpace(s[:at])
		s = strings.TrimSpace(s[at+1:])
	}

	host, port := parseHostPortLoose(s)
	out.Host = host
	if strings.TrimSpace(port) != "" {
		out.Port = strings.TrimSpace(port)
	}
	if strings.TrimSpace(out.Port) == "" {
		out.Port = "22"
	}
	return out
}

func parseHostPortLoose(s string) (host string, port string) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", ""
	}
	// Bracketed IPv6: [::1]:2222
	if strings.HasPrefix(s, "[") {
		end := strings.Index(s, "]")
		if end < 0 {
			return strings.TrimPrefix(s, "["), ""
		}
		host = s[1:end]
		rest := strings.TrimSpace(s[end+1:])
		if strings.HasPrefix(rest, ":") {
			port = strings.TrimSpace(strings.TrimPrefix(rest, ":"))
		}
		return host, port
	}

	// If there's exactly one colon, treat as host:port. Otherwise assume it's an IPv6-ish host.
	if strings.Count(s, ":") == 1 {
		parts := strings.SplitN(s, ":", 2)
		return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
	}
	return s, ""
}

func Run(ctx context.Context, target Target, command string) (output string, exitCode int, err error) {
	args := buildSSHArgs(target)
	args = append(args, targetArg(target), remoteShellCommand(command))

	cmd := exec.CommandContext(ctx, "ssh", args...)
	out, runErr := cmd.CombinedOutput()

	code := 0
	if runErr != nil {
		var ee *exec.ExitError
		if errors.As(runErr, &ee) && ee.ProcessState != nil {
			code = ee.ProcessState.ExitCode()
		} else {
			code = -1
		}
	}
	return string(out), code, runErr
}

type Process struct {
	cmd   *exec.Cmd
	stdin io.WriteCloser
	out   *bufio.Reader

	mu     sync.Mutex
	closed bool
}

func StartProcess(ctx context.Context, target Target, command string) (*Process, error) {
	args := buildSSHArgs(target)
	args = append(args, targetArg(target), remoteShellCommand(command))

	cmd := exec.CommandContext(ctx, "ssh", args...)
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
		return nil, err
	}

	p := &Process{
		cmd:   cmd,
		stdin: stdin,
		out:   bufio.NewReaderSize(stdout, 128*1024),
	}

	go func() {
		s := bufio.NewScanner(stderr)
		for s.Scan() {
			msg := strings.TrimSpace(s.Text())
			if msg == "" {
				continue
			}
			log.Printf("symphony ssh_stderr host=%s msg=%s", target.Host, msg)
		}
	}()

	return p, nil
}

func (p *Process) PID() *int {
	if p == nil || p.cmd == nil || p.cmd.Process == nil {
		return nil
	}
	pid := p.cmd.Process.Pid
	return &pid
}

func (p *Process) ReadLine() ([]byte, error) {
	line, err := p.out.ReadBytes('\n')
	if err != nil {
		return nil, err
	}
	line = bytes.TrimSpace(line)
	if len(line) > 10*1024*1024 {
		return nil, errors.New("protocol line too large")
	}
	// Keep the trailing newline behavior consistent with JSON-RPC reader expectations.
	return append(line, '\n'), nil
}

func (p *Process) WriteJSON(v any) error {
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

func (p *Process) Kill() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed || p.cmd == nil || p.cmd.Process == nil {
		return nil
	}
	return p.cmd.Process.Kill()
}

func (p *Process) Close() error {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil
	}
	p.closed = true
	_ = p.stdin.Close()
	p.mu.Unlock()

	_ = p.Kill()
	if p.cmd != nil {
		_ = p.cmd.Wait()
	}
	return nil
}

func remoteShellCommand(cmd string) string {
	escaped := escapeSingleQuotes(strings.TrimSpace(cmd))
	return "bash -lc '" + escaped + "'"
}

func escapeSingleQuotes(s string) string {
	return strings.ReplaceAll(s, "'", "'\\''")
}

func buildSSHArgs(target Target) []string {
	args := []string{
		"-T",
		"-o", "BatchMode=yes",
		"-o", "StrictHostKeyChecking=accept-new",
	}

	port := strings.TrimSpace(target.Port)
	if port != "" && port != "22" {
		args = append(args, "-p", port)
	}

	if cfg := strings.TrimSpace(os.Getenv("SYMPHONY_SSH_CONFIG")); cfg != "" {
		args = append(args, "-F", cfg)
	}

	return args
}

func targetArg(t Target) string {
	host := strings.TrimSpace(t.Host)
	user := strings.TrimSpace(t.User)
	if user == "" {
		return host
	}
	return user + "@" + host
}

