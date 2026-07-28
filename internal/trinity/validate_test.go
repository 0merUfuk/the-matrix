package trinity

import (
	"strings"
	"testing"
)

func TestValidateDoc_TooShort(t *testing.T) {
	content := strings.Repeat("line\n", 10) // 10 lines
	errors := ValidateDoc("test.md", content)

	if len(errors) == 0 {
		t.Fatal("expected error for short doc")
	}
	if !strings.Contains(errors[0], "Too short") {
		t.Errorf("expected 'Too short' error, got: %s", errors[0])
	}
	if !strings.Contains(errors[0], "11 lines") {
		t.Errorf("expected line count in error, got: %s", errors[0])
	}
}

func TestValidateDoc_RawEJS(t *testing.T) {
	lines := make([]string, 60)
	for i := range lines {
		lines[i] = "normal line"
	}
	lines[5] = "some <%- field %> here"
	lines[10] = "another <%= var %> tag"
	lines[20] = "third <% if (x) { %>"
	lines[30] = "fourth <%- y %> won't be reported" // 4th should be excluded (limit 3)
	content := strings.Join(lines, "\n")

	errors := ValidateDoc("test.md", content)

	if len(errors) == 0 {
		t.Fatal("expected EJS error")
	}
	ejsErr := errors[0]
	if !strings.Contains(ejsErr, "Raw EJS tags") {
		t.Errorf("expected 'Raw EJS tags' error, got: %s", ejsErr)
	}
	// Should report exactly 3 line numbers
	if !strings.Contains(ejsErr, "line 6") {
		t.Errorf("expected 'line 6', got: %s", ejsErr)
	}
	if !strings.Contains(ejsErr, "line 11") {
		t.Errorf("expected 'line 11', got: %s", ejsErr)
	}
	if !strings.Contains(ejsErr, "line 21") {
		t.Errorf("expected 'line 21', got: %s", ejsErr)
	}
	// Should NOT contain the 4th
	if strings.Contains(ejsErr, "line 31") {
		t.Errorf("should not report more than 3 lines, got: %s", ejsErr)
	}
}

func TestValidateDoc_Valid(t *testing.T) {
	content := strings.Repeat("This is a valid line of content.\n", 60)
	errors := ValidateDoc("valid.md", content)

	if len(errors) != 0 {
		t.Errorf("expected no errors, got: %v", errors)
	}
}

func TestValidateDoc_BothErrors(t *testing.T) {
	// Short doc with EJS tags — should report both errors
	content := "line 1\n<% bad %>\nline 3\n"
	errors := ValidateDoc("bad.md", content)

	if len(errors) != 2 {
		t.Fatalf("expected 2 errors, got %d: %v", len(errors), errors)
	}
}
