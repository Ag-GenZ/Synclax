package tracker

import "testing"

func TestSupported_Linear(t *testing.T) {
	if !Supported("linear") {
		t.Fatal("expected linear to be supported")
	}
}

func TestSupported_Unknown(t *testing.T) {
	if Supported("jira") {
		t.Fatal("expected jira to not be supported")
	}
	if Supported("") {
		t.Fatal("expected empty string to not be supported")
	}
}
