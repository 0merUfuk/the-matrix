package tmpl

import (
	"strings"
	"testing"
)

func TestRender(t *testing.T) {
	data := struct {
		Name    string
		Items   []string
		HasFlag bool
	}{
		Name:    "test-service",
		Items:   []string{"User", "Order"},
		HasFlag: true,
	}

	tmplStr := `Service: {{ .Name }}
Items: {{ join .Items ", " }}
{{ if .HasFlag }}Flag is set{{ end }}`

	result, err := Render("test", tmplStr, data)
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}
	if !strings.Contains(result, "Service: test-service") {
		t.Error("Render() missing service name")
	}
	if !strings.Contains(result, "Items: User, Order") {
		t.Error("Render() missing joined items")
	}
	if !strings.Contains(result, "Flag is set") {
		t.Error("Render() missing conditional flag")
	}
}

func TestRender_CustomFunctions(t *testing.T) {
	tests := []struct {
		name     string
		tmplStr  string
		data     any
		contains string
	}{
		{
			name:     "toPascal",
			tmplStr:  `{{ toPascal "my-service" }}`,
			data:     nil,
			contains: "MyService",
		},
		{
			name:     "toKebab",
			tmplStr:  `{{ toKebab "MyService" }}`,
			data:     nil,
			contains: "my-service",
		},
		{
			name:     "lower",
			tmplStr:  `{{ lower "HELLO" }}`,
			data:     nil,
			contains: "hello",
		},
		{
			name:     "add",
			tmplStr:  `{{ add 3 4 }}`,
			data:     nil,
			contains: "7",
		},
		{
			name:     "seq",
			tmplStr:  `{{ range seq 0 3 }}{{ . }} {{ end }}`,
			data:     nil,
			contains: "0 1 2",
		},
		{
			name:     "default with empty",
			tmplStr:  `{{ default "fallback" "" }}`,
			data:     nil,
			contains: "fallback",
		},
		{
			name:     "default with value",
			tmplStr:  `{{ default "fallback" "actual" }}`,
			data:     nil,
			contains: "actual",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Render(tt.name, tt.tmplStr, tt.data)
			if err != nil {
				t.Fatalf("Render() error: %v", err)
			}
			if !strings.Contains(result, tt.contains) {
				t.Errorf("Render() = %q, want to contain %q", result, tt.contains)
			}
		})
	}
}

func TestRenderWithPartials(t *testing.T) {
	result, err := RenderWithPartials(
		"main",
		`Before {{ template "shared/greeting.md" . }} after`,
		map[string]string{"shared/greeting.md": `Hello, {{ .Name }}!`},
		struct{ Name string }{Name: "Neo"},
	)
	if err != nil {
		t.Fatalf("RenderWithPartials() error: %v", err)
	}
	if result != "Before Hello, Neo! after" {
		t.Fatalf("RenderWithPartials() = %q, want %q", result, "Before Hello, Neo! after")
	}
}

func TestRenderWithPartials_FunctionsInPartial(t *testing.T) {
	result, err := RenderWithPartials(
		"main",
		`{{ template "shared/name.md" . }}`,
		map[string]string{"shared/name.md": `{{ upper .Name }}`},
		struct{ Name string }{Name: "trinity"},
	)
	if err != nil {
		t.Fatalf("RenderWithPartials() error: %v", err)
	}
	if result != "TRINITY" {
		t.Fatalf("RenderWithPartials() = %q, want %q", result, "TRINITY")
	}
}

func TestRenderWithPartials_MissingPartial(t *testing.T) {
	_, err := RenderWithPartials("main", `{{ template "missing" . }}`, nil, nil)
	if err == nil {
		t.Fatal("RenderWithPartials() should error when a partial is missing")
	}
	if !strings.HasPrefix(err.Error(), "rendering template main:") {
		t.Fatalf("RenderWithPartials() error = %q, want rendering error prefix", err)
	}
}

func TestRenderWithPartials_BadPartial(t *testing.T) {
	_, err := RenderWithPartials(
		"main",
		`{{ template "bad" . }}`,
		map[string]string{"bad": `{{ if }}`},
		nil,
	)
	if err == nil {
		t.Fatal("RenderWithPartials() should error when a partial cannot be parsed")
	}
	if !strings.HasPrefix(err.Error(), "parsing template main:") {
		t.Fatalf("RenderWithPartials() error = %q, want parsing error prefix", err)
	}
}

func TestRenderToFile(t *testing.T) {
	dir := t.TempDir()
	outPath := dir + "/sub/output.md"

	data := struct{ Title string }{Title: "Test"}
	err := RenderToFile("test", "# {{ .Title }}", data, outPath)
	if err != nil {
		t.Fatalf("RenderToFile() error: %v", err)
	}
}

func TestRender_InvalidTemplate(t *testing.T) {
	_, err := Render("bad", "{{ .Missing }", nil)
	if err == nil {
		t.Error("Render() should error on invalid template syntax")
	}
}
