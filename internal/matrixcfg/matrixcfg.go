// Package matrixcfg provides unified configuration loading from .matrix.yaml files.
//
// Configuration is discovered in priority order:
//  1. .matrix.yaml in the project root
//  2. .matrix.yml in the project root
//  3. ~/.config/the-matrix/config.yaml (global fallback)
//
// Project-level config is used exclusively when found (no merging with global).
// If no config file exists, a zero-value Config is returned without error.
package matrixcfg

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config is the top-level .matrix.yaml schema.
type Config struct {
	Models   ModelsConfig   `yaml:"models"`
	Neo      NeoConfig      `yaml:"neo"`
	Morpheus MorpheusConfig `yaml:"morpheus"`
	Oracle   OracleConfig   `yaml:"oracle"`
	Trinity  TrinityConfig  `yaml:"trinity"`
}

// ModelsConfig controls which AI models are used for each agent role.
type ModelsConfig struct {
	Default   *string `yaml:"default,omitempty"`
	Manager   *string `yaml:"manager,omitempty"`
	Architect *string `yaml:"architect,omitempty"`
	Developer *string `yaml:"developer,omitempty"`
	Tester    *string `yaml:"tester,omitempty"`
	Reviewer  *string `yaml:"reviewer,omitempty"`
}

// NeoConfig holds neo-specific configuration.
type NeoConfig struct {
	AutoOracle  *bool   `yaml:"auto-oracle,omitempty"`
	AutoTrinity *bool   `yaml:"auto-trinity,omitempty"`
	TeamSize    *string `yaml:"team-size,omitempty"`
}

// MorpheusConfig holds morpheus-specific configuration.
type MorpheusConfig struct {
	MaxCycles      *int `yaml:"max-cycles,omitempty"`
	DeveloperTurns *int `yaml:"developer-turns,omitempty"`
	TesterTurns    *int `yaml:"tester-turns,omitempty"`
	ReviewerTurns  *int `yaml:"reviewer-turns,omitempty"`
}

// OracleConfig holds oracle-specific configuration.
type OracleConfig struct {
	ResearcherTurns  *int    `yaml:"researcher-turns,omitempty"`
	ReviewerTurns    *int    `yaml:"reviewer-turns,omitempty"`
	SynthesizerTurns *int    `yaml:"synthesizer-turns,omitempty"`
	OutputDir        *string `yaml:"output-dir,omitempty"`
}

// TrinityConfig holds trinity-specific configuration.
type TrinityConfig struct {
	MaxAgeMonths *float64 `yaml:"max-age-months,omitempty"`
}

// Load discovers and parses a .matrix.yaml configuration file.
// Discovery order: .matrix.yaml in root, .matrix.yml in root, ~/.config/the-matrix/config.yaml.
// Returns a zero-value Config (no error) if no config file is found.
func Load(root string) (Config, error) {
	var cfg Config

	path := findConfigFile(root)
	if path == "" {
		return cfg, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, fmt.Errorf("reading config %s: %w", path, err)
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("parsing config %s: %w", path, err)
	}

	if err := validate(cfg); err != nil {
		return cfg, fmt.Errorf("validating config %s: %w", path, err)
	}

	return cfg, nil
}

// findConfigFile checks for config files in priority order and returns
// the first one found, or empty string if none exist.
func findConfigFile(root string) string {
	// Project-level configs
	candidates := []string{
		filepath.Join(root, ".matrix.yaml"),
		filepath.Join(root, ".matrix.yml"),
	}

	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}

	// Global fallback
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	global := filepath.Join(home, ".config", "the-matrix", "config.yaml")
	if _, err := os.Stat(global); err == nil {
		return global
	}

	return ""
}

// validate checks that set fields consumed by a tool have valid values.
// Only validates fields that are currently consumed by a tool. Fields
// reserved for future use (models, neo, morpheus, oracle turn budgets)
// are parsed but not validated until consumption is wired.
// Returns an error listing all validation failures joined with "; ".
func validate(cfg Config) error {
	var errs []string

	checkNonEmptyString := func(field string, v *string) {
		if v != nil && *v == "" {
			errs = append(errs, fmt.Sprintf("%s must be non-empty", field))
		}
	}

	// Float fields: must be > 0 if set
	if cfg.Trinity.MaxAgeMonths != nil && *cfg.Trinity.MaxAgeMonths <= 0 {
		errs = append(errs, fmt.Sprintf("trinity.max-age-months must be > 0 (got %g)", *cfg.Trinity.MaxAgeMonths))
	}

	// Oracle output-dir: non-empty if set
	checkNonEmptyString("oracle.output-dir", cfg.Oracle.OutputDir)

	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}
	return nil
}
