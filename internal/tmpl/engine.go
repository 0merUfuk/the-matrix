// Package tmpl provides shared Go template rendering with go:embed support.
package tmpl

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"
)

// FuncMap returns the shared custom template functions available in all templates.
func FuncMap() template.FuncMap {
	return template.FuncMap{
		"join":      strings.Join,
		"lower":     strings.ToLower,
		"upper":     strings.ToUpper,
		"contains":  strings.Contains,
		"hasPrefix": strings.HasPrefix,
		"hasSuffix": strings.HasSuffix,
		"trimSpace": strings.TrimSpace,
		"repeat":    strings.Repeat,
		"toKebab":   toKebab,
		"toPascal":  toPascal,
		"todayISO":  todayISO,
		"add":       func(a, b int) int { return a + b },
		"sub":       func(a, b int) int { return a - b },
		"seq":       seq,
		"default": func(def, val string) string {
			if val == "" {
				return def
			}
			return val
		},
		"padStart": func(n, width int) string {
			s := fmt.Sprintf("%d", n)
			for len(s) < width {
				s = "0" + s
			}
			return s
		},
		"abbreviate": func(s string) string {
			if len(s) >= 2 {
				return strings.ToLower(s[:2])
			}
			return strings.ToLower(s)
		},
	}
}

// Render renders a template string with the given data and custom functions.
func Render(name, tmplStr string, data any) (string, error) {
	t, err := template.New(name).Funcs(FuncMap()).Parse(tmplStr)
	if err != nil {
		return "", fmt.Errorf("parsing template %s: %w", name, err)
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("rendering template %s: %w", name, err)
	}
	return buf.String(), nil
}

// RenderToFile renders a template string and writes the result to a file.
func RenderToFile(name, tmplStr string, data any, outPath string) error {
	content, err := Render(name, tmplStr, data)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
		return fmt.Errorf("creating directory for %s: %w", outPath, err)
	}
	return os.WriteFile(outPath, []byte(content), 0644)
}

// --- Helper functions for templates ---

func toKebab(s string) string {
	var result strings.Builder
	for i, r := range s {
		if r >= 'A' && r <= 'Z' {
			if i > 0 {
				result.WriteByte('-')
			}
			result.WriteRune(r + 32) // lowercase
		} else {
			result.WriteRune(r)
		}
	}
	return result.String()
}

func toPascal(s string) string {
	words := strings.FieldsFunc(s, func(r rune) bool {
		return r == '-' || r == '_' || r == ' '
	})
	var result strings.Builder
	for _, w := range words {
		if w == "" {
			continue
		}
		runes := []rune(w)
		runes[0] = rune(strings.ToUpper(string(runes[0]))[0])
		result.WriteString(string(runes))
	}
	return result.String()
}

func todayISO() string {
	return time.Now().Format("2006-01-02")
}

// seq generates a sequence of integers [start, end) for range loops in templates.
func seq(start, end int) []int {
	if end <= start {
		return nil
	}
	s := make([]int, end-start)
	for i := range s {
		s[i] = start + i
	}
	return s
}
