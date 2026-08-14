package runtime_test

import (
	"testing"

	agentruntime "github.com/0merUfuk/the-matrix/internal/runtime"
	_ "github.com/0merUfuk/the-matrix/internal/runtime/builtin"
)

func TestSelectExplicitRuntime(t *testing.T) {
	rt, err := agentruntime.Select("codex", t.TempDir())
	if err != nil {
		t.Fatalf("Select() error: %v", err)
	}
	if rt.Name() != "codex" {
		t.Fatalf("Select() = %q, want codex", rt.Name())
	}
}

func TestSelectDetectedDefault(t *testing.T) {
	rt, err := agentruntime.Select("", t.TempDir())
	if err != nil {
		t.Fatalf("Select() error: %v", err)
	}
	if rt.Name() != "claude" {
		t.Fatalf("Select() = %q, want claude", rt.Name())
	}
}

func TestSelectUnknownRuntime(t *testing.T) {
	if _, err := agentruntime.Select("not-a-runtime", t.TempDir()); err == nil {
		t.Fatal("Select() should reject an unknown runtime")
	}
}
