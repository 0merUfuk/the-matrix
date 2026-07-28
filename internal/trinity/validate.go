// Package trinity implements the trinity CLI commands: health, sync, and refresh.
package trinity

import (
	"fmt"
	"strings"
)

// ValidateDoc checks a knowledge doc's content for critical issues.
// Returns a slice of error strings (empty = valid).
//
// Critical checks:
//   - Minimum 50 lines (prevents truncated docs)
//   - No raw EJS tags (<%) (unrendered template)
func ValidateDoc(filename, content string) []string {
	var errors []string
	lines := strings.Split(content, "\n")

	// Rule 1: minimum line count
	if len(lines) < 50 {
		errors = append(errors, fmt.Sprintf("Too short: %d lines (minimum 50)", len(lines)))
	}

	// Rule 2: no raw EJS tags (unrendered template)
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
		errors = append(errors, fmt.Sprintf("Raw EJS tags (<%%) at: %s", strings.Join(ejsLines, ", ")))
	}

	return errors
}
