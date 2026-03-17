package template

import (
	"testing"

	"github.com/wibus-wee/synclax/pkg/symphony/domain"
)

func TestRenderer_DefaultPromptOnEmpty(t *testing.T) {
	r, err := Compile("")
	if err != nil {
		t.Fatalf("Compile error: %v", err)
	}
	out, err := r.RenderIssuePrompt(domain.Issue{ID: "1", Identifier: "A-1", Title: "t", State: "Todo"}, nil)
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}
	if out == "" {
		t.Fatal("expected default prompt")
	}
}

func TestRenderer_StrictUnknownVariable(t *testing.T) {
	r, err := Compile("{{ issue.nope }}")
	if err != nil {
		t.Fatalf("Compile error: %v", err)
	}
	_, err = r.RenderIssuePrompt(domain.Issue{ID: "1", Identifier: "A-1", Title: "t", State: "Todo"}, nil)
	if err == nil {
		t.Fatal("expected render error")
	}
	if err != ErrTemplateRenderError {
		t.Fatalf("expected %v, got %v", ErrTemplateRenderError, err)
	}
}

func TestRenderer_UnknownFilterFails(t *testing.T) {
	r, err := Compile("{{ issue.identifier | does_not_exist }}")
	if err != nil {
		t.Fatalf("Compile error: %v", err)
	}
	_, err = r.RenderIssuePrompt(domain.Issue{ID: "1", Identifier: "A-1", Title: "t", State: "Todo"}, nil)
	if err == nil {
		t.Fatal("expected render error")
	}
	if err != ErrTemplateRenderError {
		t.Fatalf("expected %v, got %v", ErrTemplateRenderError, err)
	}
}

func TestRenderer_RendersIssueIdentifier(t *testing.T) {
	r, err := Compile("Issue={{ issue.identifier }} Attempt={{ attempt }}")
	if err != nil {
		t.Fatalf("Compile error: %v", err)
	}
	a := 2
	out, err := r.RenderIssuePrompt(domain.Issue{ID: "1", Identifier: "A-1", Title: "t", State: "Todo"}, &a)
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}
	if out != "Issue=A-1 Attempt=2" {
		t.Fatalf("unexpected output: %q", out)
	}
}
