package workflow

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_NoFrontMatter(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "WORKFLOW.md")
	if err := os.WriteFile(path, []byte("hello\nworld\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	def, err := Load(path)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if def == nil {
		t.Fatal("expected definition")
	}
	if len(def.Config) != 0 {
		t.Fatalf("expected empty config, got %#v", def.Config)
	}
	if def.PromptTemplate != "hello\nworld" {
		t.Fatalf("unexpected prompt: %q", def.PromptTemplate)
	}
}

func TestLoad_WithFrontMatter(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "WORKFLOW.md")
	content := `---
tracker:
  kind: linear
  project_slug: myproj
---
Do the thing`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	def, err := Load(path)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if def.PromptTemplate != "Do the thing" {
		t.Fatalf("unexpected prompt: %q", def.PromptTemplate)
	}
	trackerCfg, ok := def.Config["tracker"].(map[string]any)
	if !ok {
		t.Fatalf("expected tracker map, got %#v", def.Config["tracker"])
	}
	if trackerCfg["kind"] != "linear" {
		t.Fatalf("expected kind=linear, got %#v", trackerCfg["kind"])
	}
}

func TestLoad_NonMapFrontMatter(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "WORKFLOW.md")
	content := `---
- a
- b
---
body`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error")
	}
	if err != ErrWorkflowFrontMatterNotAMap {
		t.Fatalf("expected %v, got %v", ErrWorkflowFrontMatterNotAMap, err)
	}
}

func TestLoad_MissingFile(t *testing.T) {
	_, err := Load("/this/file/does/not/exist")
	if err == nil {
		t.Fatal("expected error")
	}
	if err != ErrMissingWorkflowFile {
		t.Fatalf("expected %v, got %v", ErrMissingWorkflowFile, err)
	}
}
