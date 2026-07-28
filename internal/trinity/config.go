package trinity

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/0merUfuk/the-matrix/internal/config"
)

// TrinityConfig represents the .trinity.json configuration file.
type TrinityConfig struct {
	Stacks []SyncPair `json:"stacks"`
}

// SyncPair represents a single source→target sync mapping.
type SyncPair struct {
	Name string `json:"name"`
	From string `json:"from"`
	To   string `json:"to"`
}

// LoadConfig reads and validates .trinity.json from the project root.
// Returns resolved sync pairs. Prints errors and calls os.Exit(1) on failure.
func LoadConfig(root string) []SyncPair {
	configPath := filepath.Join(root, ".trinity.json")

	if !config.FileExists(configPath) {
		fmt.Fprintf(os.Stderr, "  Error: No --from/--to flags and no .trinity.json found at %s\n", root)
		fmt.Fprintf(os.Stderr, "  Usage: trinity sync --from <source> --to <target>\n")
		fmt.Fprintf(os.Stderr, "  Or create .trinity.json with stack mappings\n\n")
		os.Exit(1)
	}

	cfg, err := config.ReadJSON[TrinityConfig](configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "  Error: Invalid JSON in .trinity.json\n\n")
		os.Exit(1)
	}

	if len(cfg.Stacks) == 0 {
		fmt.Fprintf(os.Stderr, "  Error: .trinity.json has no stacks defined\n\n")
		os.Exit(1)
	}

	// Resolve paths relative to project root
	pairs := make([]SyncPair, len(cfg.Stacks))
	for i, s := range cfg.Stacks {
		name := s.Name
		if name == "" {
			name = filepath.Base(s.From)
		}
		from := s.From
		if !filepath.IsAbs(from) {
			from = filepath.Join(root, from)
		}
		to := s.To
		if !filepath.IsAbs(to) {
			to = filepath.Join(root, to)
		}
		pairs[i] = SyncPair{Name: name, From: from, To: to}
	}

	return pairs
}

// LoadTrinityConfig reads .trinity.json if it exists, returning nil if not found or invalid.
func LoadTrinityConfig(root string) *TrinityConfig {
	configPath := filepath.Join(root, ".trinity.json")
	if !config.FileExists(configPath) {
		return nil
	}
	cfg, err := config.ReadJSON[TrinityConfig](configPath)
	if err != nil {
		return nil
	}
	return &cfg
}
