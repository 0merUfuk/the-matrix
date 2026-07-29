// Package validation contains shared document validation used by the tools.
package validation

import (
	"fmt"
	"strings"
)

// ValidateInjectDoc validates a knowledge doc for oracle inject.
// Stricter than trinity's ValidateDoc — also checks frontmatter fields.
// Returns a slice of error strings (empty = valid).
//
// Rules:
//  1. Minimum 50 lines (prevents truncated docs)
//  2. Required frontmatter fields: **Version**, **Created**, **Authors:**
//  3. No raw EJS tags (<%) (unrendered template)
func ValidateInjectDoc(filename, content string) []string {
	var errors []string
	lines := strings.Split(content, "\n")

	// Rule 1: minimum line count
	if len(lines) < 50 {
		errors = append(errors, fmt.Sprintf("Too short: %d lines (minimum 50) — doc may be truncated", len(lines)))
	}

	// Rule 2: required frontmatter fields
	requiredFields := []string{"**Version**", "**Created**", "**Authors:**"}
	for _, field := range requiredFields {
		if !strings.Contains(content, field) {
			errors = append(errors, fmt.Sprintf("Missing frontmatter field: %s", field))
		}
	}

	// Rule 3: no raw EJS tags (unrendered template)
	if strings.Contains(content, "<%") {
		var ejsLines []string
		for i, line := range lines {
			if strings.Contains(line, "<%") {
				ejsLines = append(ejsLines, fmt.Sprintf("line %d", i+1))
				if len(ejsLines) >= 3 {
					break
				}
			}
		}
		errors = append(errors, fmt.Sprintf("Contains raw EJS tags (<%%) — template was not rendered. Found at: %s", strings.Join(ejsLines, ", ")))
	}

	return errors
}
