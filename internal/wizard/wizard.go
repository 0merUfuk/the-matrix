// Package wizard provides shared interactive prompt helpers using charmbracelet/huh.
package wizard

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	huh "charm.land/huh/v2"
)

// Option represents a choice in select/checkbox prompts.
type Option struct {
	Value string
	Label string
}

// AskInput prompts for a single-line text input.
func AskInput(label, defaultVal string, validate func(string) error) (string, error) {
	var result string
	field := huh.NewInput().
		Title(label).
		Value(&result)

	if defaultVal != "" {
		field.Placeholder(defaultVal)
		result = defaultVal
	}
	if validate != nil {
		field.Validate(validate)
	}

	form := huh.NewForm(huh.NewGroup(field))
	if err := form.Run(); err != nil {
		return "", err
	}
	return strings.TrimSpace(result), nil
}

// AskSelect prompts the user to select one option from a list.
func AskSelect(label string, options []Option) (string, error) {
	var result string
	opts := make([]huh.Option[string], len(options))
	for i, o := range options {
		opts[i] = huh.NewOption(o.Label, o.Value)
	}

	field := huh.NewSelect[string]().
		Title(label).
		Options(opts...).
		Value(&result)

	form := huh.NewForm(huh.NewGroup(field))
	if err := form.Run(); err != nil {
		return "", err
	}
	return result, nil
}

// AskCheckbox prompts the user to select multiple options.
func AskCheckbox(label string, options []Option) ([]string, error) {
	var result []string
	opts := make([]huh.Option[string], len(options))
	for i, o := range options {
		opts[i] = huh.NewOption(o.Label, o.Value)
	}

	field := huh.NewMultiSelect[string]().
		Title(label).
		Options(opts...).
		Value(&result)

	form := huh.NewForm(huh.NewGroup(field))
	if err := form.Run(); err != nil {
		return nil, err
	}
	return result, nil
}

// AskConfirm prompts the user for a yes/no confirmation.
func AskConfirm(label string, defaultVal bool) (bool, error) {
	result := defaultVal

	field := huh.NewConfirm().
		Title(label).
		Value(&result)

	form := huh.NewForm(huh.NewGroup(field))
	if err := form.Run(); err != nil {
		return false, err
	}
	return result, nil
}

// --- Validators ---

var kebabCaseRe = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

// ValidateKebabCase validates that input is lowercase kebab-case.
func ValidateKebabCase(input string) error {
	if input == "" {
		return fmt.Errorf("required")
	}
	if !kebabCaseRe.MatchString(input) {
		return fmt.Errorf("must be kebab-case (lowercase, hyphens only, start with letter)")
	}
	return nil
}

// ValidatePort validates that input is a valid port number (1-65535).
func ValidatePort(input string) error {
	port, err := strconv.Atoi(strings.TrimSpace(input))
	if err != nil {
		return fmt.Errorf("must be a number")
	}
	if port < 1 || port > 65535 {
		return fmt.Errorf("must be between 1 and 65535")
	}
	return nil
}

// ValidateMinLength returns a validator that checks minimum string length.
func ValidateMinLength(min int) func(string) error {
	return func(input string) error {
		if len(strings.TrimSpace(input)) < min {
			return fmt.Errorf("must be at least %d characters", min)
		}
		return nil
	}
}

// --- Parsers ---

// ParseCommaSeparated splits a comma-separated string, trims whitespace, and filters empty entries.
func ParseCommaSeparated(input string) []string {
	parts := strings.Split(input, ",")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// ToPascalCase converts a kebab-case or snake_case string to PascalCase.
func ToPascalCase(s string) string {
	words := regexp.MustCompile(`[-_\s]+`).Split(s, -1)
	var result strings.Builder
	for _, w := range words {
		if w == "" {
			continue
		}
		runes := []rune(w)
		runes[0] = unicode.ToUpper(runes[0])
		result.WriteString(string(runes))
	}
	return result.String()
}

// TodayISO returns today's date in YYYY-MM-DD format.
func TodayISO() string {
	return time.Now().Format("2006-01-02")
}
