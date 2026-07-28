package trinity

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadTrinityConfig_NotFound(t *testing.T) {
	cfg := LoadTrinityConfig("/nonexistent/path")
	if cfg != nil {
		t.Error("expected nil for missing config")
	}
}

func TestLoadTrinityConfig_ValidConfig(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".trinity.json")
	content := `{
  "stacks": [
    { "name": "go-rules", "from": "output/go/", "to": ".claude/knowledge/go-rules/" },
    { "from": "output/node/" , "to": ".claude/knowledge/node-rules/" }
  ]
}`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := LoadTrinityConfig(dir)
	if cfg == nil {
		t.Fatal("expected config, got nil")
	}
	if len(cfg.Stacks) != 2 {
		t.Fatalf("expected 2 stacks, got %d", len(cfg.Stacks))
	}
	if cfg.Stacks[0].Name != "go-rules" {
		t.Errorf("expected name 'go-rules', got '%s'", cfg.Stacks[0].Name)
	}
	if cfg.Stacks[1].Name != "" {
		t.Errorf("expected empty name for second stack (default applied at LoadConfig), got '%s'", cfg.Stacks[1].Name)
	}
}

func TestLoadTrinityConfig_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".trinity.json")
	if err := os.WriteFile(configPath, []byte("{bad json}"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := LoadTrinityConfig(dir)
	if cfg != nil {
		t.Error("expected nil for invalid JSON")
	}
}
