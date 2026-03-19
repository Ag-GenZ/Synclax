package ssh

import (
	"os"
	"reflect"
	"testing"
)

func TestParseTarget(t *testing.T) {
	cases := []struct {
		in   string
		want Target
	}{
		{"host", Target{Host: "host", Port: "22"}},
		{"host:2222", Target{Host: "host", Port: "2222"}},
		{"user@host", Target{User: "user", Host: "host", Port: "22"}},
		{"user@host:2222", Target{User: "user", Host: "host", Port: "2222"}},
		{"[::1]:2222", Target{Host: "::1", Port: "2222"}},
		{"user@[::1]:2222", Target{User: "user", Host: "::1", Port: "2222"}},
		{"::1", Target{Host: "::1", Port: "22"}},
		{"user@::1", Target{User: "user", Host: "::1", Port: "22"}},
	}
	for _, tc := range cases {
		got := ParseTarget(tc.in)
		if got.Port == "" {
			t.Fatalf("ParseTarget(%q) produced empty port", tc.in)
		}
		if !reflect.DeepEqual(got, tc.want) {
			t.Fatalf("ParseTarget(%q)=%#v, want %#v", tc.in, got, tc.want)
		}
	}
}

func TestRemoteShellCommand_EscapesSingleQuotes(t *testing.T) {
	got := remoteShellCommand("echo 'hi'")
	want := "bash -lc 'echo '\\''hi'\\'''"
	if got != want {
		t.Fatalf("remoteShellCommand=%q, want %q", got, want)
	}
}

func TestBuildSSHArgs(t *testing.T) {
	t.Setenv("SYMPHONY_SSH_CONFIG", "/tmp/ssh_config")
	defer os.Unsetenv("SYMPHONY_SSH_CONFIG")

	args := buildSSHArgs(Target{Host: "example.com", Port: "2222"})
	// Required flags.
	for _, required := range [][]string{
		{"-T"},
		{"-o", "BatchMode=yes"},
		{"-o", "StrictHostKeyChecking=accept-new"},
		{"-p", "2222"},
		{"-F", "/tmp/ssh_config"},
	} {
		if !containsSubslice(args, required) {
			t.Fatalf("expected args to contain %v, got %v", required, args)
		}
	}
}

func containsSubslice(hay []string, needle []string) bool {
	if len(needle) == 0 {
		return true
	}
	for i := 0; i+len(needle) <= len(hay); i++ {
		if reflect.DeepEqual(hay[i:i+len(needle)], needle) {
			return true
		}
	}
	return false
}

