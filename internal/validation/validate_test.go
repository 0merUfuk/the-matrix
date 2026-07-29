package validation

import (
	"strings"
	"testing"
)

func TestValidateInjectDoc_Valid(t *testing.T) {
	content := "**Version**: 1.0\n**Created**: 2026-03-15\n**Authors:** Oracle CLI\n\n---\n\n# Test\n\n"
	content += strings.Repeat("Content line.\n", 50)

	errors := ValidateInjectDoc("valid.md", content)
	if len(errors) != 0 {
		t.Errorf("expected no errors, got: %v", errors)
	}
}

func TestValidateInjectDoc_TooShort(t *testing.T) {
	content := "**Version**: 1.0\n**Created**: 2026-03-15\n**Authors:** Oracle CLI\nShort.\n"

	errors := ValidateInjectDoc("short.md", content)
	found := false
	for _, e := range errors {
		if strings.Contains(e, "Too short") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'Too short' error, got: %v", errors)
	}
}

func TestValidateInjectDoc_MissingVersion(t *testing.T) {
	content := "**Created**: 2026-03-15\n**Authors:** Oracle CLI\n"
	content += strings.Repeat("line\n", 55)

	errors := ValidateInjectDoc("noversion.md", content)
	found := false
	for _, e := range errors {
		if strings.Contains(e, "**Version**") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected missing **Version** error, got: %v", errors)
	}
}

func TestValidateInjectDoc_MissingCreated(t *testing.T) {
	content := "**Version**: 1.0\n**Authors:** Oracle CLI\n"
	content += strings.Repeat("line\n", 55)

	errors := ValidateInjectDoc("nocreated.md", content)
	found := false
	for _, e := range errors {
		if strings.Contains(e, "**Created**") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected missing **Created** error, got: %v", errors)
	}
}

func TestValidateInjectDoc_MissingAuthors(t *testing.T) {
	content := "**Version**: 1.0\n**Created**: 2026-03-15\n"
	content += strings.Repeat("line\n", 55)

	errors := ValidateInjectDoc("noauthors.md", content)
	found := false
	for _, e := range errors {
		if strings.Contains(e, "**Authors:**") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected missing **Authors:** error, got: %v", errors)
	}
}

func TestValidateInjectDoc_RawEJS(t *testing.T) {
	content := "**Version**: 1.0\n**Created**: 2026-03-15\n**Authors:** Oracle CLI\n"
	content += strings.Repeat("line\n", 50)
	content += "some <%- field %> here\n"

	errors := ValidateInjectDoc("ejs.md", content)
	found := false
	for _, e := range errors {
		if strings.Contains(e, "raw EJS tags") || strings.Contains(e, "Raw EJS tags") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected EJS error, got: %v", errors)
	}
}

func TestValidateInjectDoc_MultipleErrors(t *testing.T) {
	content := "short and missing everything\n<% bad %>\n"

	errors := ValidateInjectDoc("bad.md", content)
	// Should have: too short + missing Version + missing Created + missing Authors + raw EJS = 5 errors
	if len(errors) != 5 {
		t.Errorf("expected 5 errors, got %d: %v", len(errors), errors)
	}
}
