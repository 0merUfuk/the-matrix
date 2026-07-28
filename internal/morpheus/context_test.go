package morpheus

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadContextProfile_Valid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".neo.json")

	content := `{
		"projectName": "my-service",
		"projectType": "microservice",
		"isInternal": true,
		"stacks": [
			{
				"name": "go-chi",
				"language": "go",
				"framework": "chi"
			}
		]
	}`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	profile, err := LoadContextProfile(path)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if profile.ProjectName != "my-service" {
		t.Errorf("expected projectName='my-service', got '%s'", profile.ProjectName)
	}
	if !profile.IsInternal {
		t.Error("expected isInternal=true")
	}
	if len(profile.Stacks) != 1 {
		t.Fatalf("expected 1 stack, got %d", len(profile.Stacks))
	}
	if profile.Stacks[0].Language != "go" {
		t.Errorf("expected language='go', got '%s'", profile.Stacks[0].Language)
	}
	if profile.Stacks[0].Name != "go-chi" {
		t.Errorf("expected name='go-chi', got '%s'", profile.Stacks[0].Name)
	}
	if profile.Stacks[0].Framework != "chi" {
		t.Errorf("expected framework='chi', got '%s'", profile.Stacks[0].Framework)
	}
}

func TestLoadContextProfile_MissingFile(t *testing.T) {
	_, err := LoadContextProfile("/nonexistent/path/.neo.json")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
	expected := "context file not found: /nonexistent/path/.neo.json"
	if err.Error() != expected {
		t.Errorf("expected error '%s', got '%s'", expected, err.Error())
	}
}

func TestLoadContextProfile_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".neo.json")

	if err := os.WriteFile(path, []byte("not json {{{"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadContextProfile(path)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if len(err.Error()) < 10 {
		t.Errorf("expected descriptive error, got: %v", err)
	}
}

func TestLoadContextProfile_MissingProjectName(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".neo.json")

	content := `{
		"projectType": "microservice",
		"isInternal": true,
		"stacks": [{"name": "go-chi", "language": "go", "framework": "chi"}]
	}`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadContextProfile(path)
	if err == nil {
		t.Fatal("expected error for missing projectName")
	}
	expected := "context file missing required field: projectName"
	if err.Error() != expected {
		t.Errorf("expected error '%s', got '%s'", expected, err.Error())
	}
}

func TestLoadContextProfile_EmptyStacks(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".neo.json")

	content := `{
		"projectName": "my-service",
		"projectType": "microservice",
		"isInternal": false,
		"stacks": []
	}`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadContextProfile(path)
	if err == nil {
		t.Fatal("expected error for empty stacks")
	}
	expected := "context file must include at least one stack"
	if err.Error() != expected {
		t.Errorf("expected error '%s', got '%s'", expected, err.Error())
	}
}

func TestMapContextToProject_GoStack(t *testing.T) {
	profile := &ContextProfile{
		ProjectName: "payment-service",
		ProjectType: "microservice",
		IsInternal:    true,
		Stacks: []ContextStack{
			{Name: "go-chi", Language: "go", Framework: "chi"},
		},
	}

	pc := MapContextToProject(profile)

	if pc.ServiceName != "payment-service" {
		t.Errorf("expected ServiceName='payment-service', got '%s'", pc.ServiceName)
	}
	if pc.ServiceNamePascal != "PaymentService" {
		t.Errorf("expected ServiceNamePascal='PaymentService', got '%s'", pc.ServiceNamePascal)
	}
	if !pc.IsInternal {
		t.Error("expected IsInternal=true")
	}
	if pc.ProjectType != "internal" {
		t.Errorf("expected ProjectType='internal', got '%s'", pc.ProjectType)
	}
	if pc.Tech != "go" {
		t.Errorf("expected Tech='go', got '%s'", pc.Tech)
	}
	if !pc.IsGo {
		t.Error("expected IsGo=true")
	}
	if pc.IsNode {
		t.Error("expected IsNode=false")
	}
	if pc.CreatedDate == "" {
		t.Error("expected CreatedDate to be set")
	}
}

func TestMapContextToProject_NodeStack(t *testing.T) {
	profile := &ContextProfile{
		ProjectName: "api-gateway",
		ProjectType: "service",
		IsInternal:    false,
		Stacks: []ContextStack{
			{Name: "express-api", Language: "nodejs", Framework: "express"},
		},
	}

	pc := MapContextToProject(profile)

	if pc.ServiceName != "api-gateway" {
		t.Errorf("expected ServiceName='api-gateway', got '%s'", pc.ServiceName)
	}
	if pc.ServiceNamePascal != "ApiGateway" {
		t.Errorf("expected ServiceNamePascal='ApiGateway', got '%s'", pc.ServiceNamePascal)
	}
	if pc.IsInternal {
		t.Error("expected IsInternal=false")
	}
	if pc.ProjectType != "general" {
		t.Errorf("expected ProjectType='general', got '%s'", pc.ProjectType)
	}
	if pc.Tech != "node" {
		t.Errorf("expected Tech='node', got '%s'", pc.Tech)
	}
	if pc.IsGo {
		t.Error("expected IsGo=false")
	}
	if !pc.IsNode {
		t.Error("expected IsNode=true")
	}
	if pc.NodeArchitecture != "4-layer" {
		t.Errorf("expected NodeArchitecture='4-layer', got '%s'", pc.NodeArchitecture)
	}
}

func TestMapContextToProject_UnknownLanguage(t *testing.T) {
	profile := &ContextProfile{
		ProjectName: "ml-pipeline",
		ProjectType: "service",
		IsInternal:    false,
		Stacks: []ContextStack{
			{Name: "python-fastapi", Language: "python", Framework: "fastapi"},
		},
	}

	// Capture stdout to verify the warning is emitted.
	oldStdout := os.Stdout
	defer func() { os.Stdout = oldStdout }()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w

	pc := MapContextToProject(profile)

	w.Close()
	capturedBytes, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = oldStdout
	captured := string(capturedBytes)

	if pc.Tech != "" {
		t.Errorf("expected Tech='' for unknown language, got '%s'", pc.Tech)
	}
	if pc.IsGo {
		t.Error("expected IsGo=false for unknown language")
	}
	if pc.IsNode {
		t.Error("expected IsNode=false for unknown language")
	}
	if pc.ProjectType != "general" {
		t.Errorf("expected ProjectType='general', got '%s'", pc.ProjectType)
	}

	// Verify the warning message is printed for the unknown language.
	if !strings.Contains(captured, "Unknown language") || !strings.Contains(captured, "python") {
		t.Errorf("expected warning about unknown language 'python', got output: %q", captured)
	}
}

func TestLoadContextProfile_UnreadableFile(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("running as root — permission checks are bypassed")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, ".neo.json")

	content := `{"projectName":"ok","stacks":[{"name":"go","language":"go","framework":"chi"}]}`
	if err := os.WriteFile(path, []byte(content), 0000); err != nil {
		t.Fatal(err)
	}

	_, err := LoadContextProfile(path)
	if err == nil {
		t.Fatal("expected error for unreadable file")
	}
	expected := "failed to read context file"
	if len(err.Error()) < len(expected) || err.Error()[:len(expected)] != expected {
		t.Errorf("expected error to start with '%s', got '%s'", expected, err.Error())
	}
}

func TestLoadContextProfile_InvalidJSON_ErrorMessage(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".neo.json")

	if err := os.WriteFile(path, []byte("{bad json"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadContextProfile(path)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	expected := "failed to parse context file"
	if len(err.Error()) < len(expected) || err.Error()[:len(expected)] != expected {
		t.Errorf("expected error to start with '%s', got '%s'", expected, err.Error())
	}
}

func TestMapContextToProject_GoStack_NonInternal(t *testing.T) {
	// Verifies that IsInternal=false with a Go stack produces ProjectType="general"
	// and still sets IsGo=true. Exercises the IsInternal branch independently of language.
	profile := &ContextProfile{
		ProjectName: "my-go-oss",
		ProjectType: "service",
		IsInternal:    false,
		Stacks: []ContextStack{
			{Name: "go-chi", Language: "go", Framework: "chi"},
		},
	}

	pc := MapContextToProject(profile)

	if pc.ProjectType != "general" {
		t.Errorf("expected ProjectType='general', got '%s'", pc.ProjectType)
	}
	if pc.Tech != "go" {
		t.Errorf("expected Tech='go', got '%s'", pc.Tech)
	}
	if !pc.IsGo {
		t.Error("expected IsGo=true")
	}
	if pc.IsNode {
		t.Error("expected IsNode=false")
	}
	if pc.IsInternal {
		t.Error("expected IsInternal=false")
	}
}

func TestMapContextToProject_WizardFieldsAreZero(t *testing.T) {
	// Verifies that fields the wizard must fill in (ports, entities, auth, etc.)
	// are left at zero values when MapContextToProject returns — the wizard delta
	// questions are responsible for populating these.
	profile := &ContextProfile{
		ProjectName: "my-service",
		ProjectType: "microservice",
		IsInternal:    true,
		Stacks: []ContextStack{
			{Name: "go-chi", Language: "go", Framework: "chi"},
		},
	}

	pc := MapContextToProject(profile)

	if pc.ApiPort != 0 {
		t.Errorf("expected ApiPort=0 (wizard fills this), got %d", pc.ApiPort)
	}
	if pc.WorkerPort != 0 {
		t.Errorf("expected WorkerPort=0 (wizard fills this), got %d", pc.WorkerPort)
	}
	if pc.HasWorker {
		t.Error("expected HasWorker=false (wizard fills this)")
	}
	if len(pc.Entities) != 0 {
		t.Errorf("expected Entities=[] (wizard fills this), got %v", pc.Entities)
	}
	if pc.Description != "" {
		t.Errorf("expected Description='' (wizard fills this), got '%s'", pc.Description)
	}
	if pc.HasJWT {
		t.Error("expected HasJWT=false (wizard fills this)")
	}
	if pc.HasInternalKey {
		t.Error("expected HasInternalKey=false (wizard fills this)")
	}
	if pc.ModulePath != "" {
		t.Errorf("expected ModulePath='' (wizard fills this), got '%s'", pc.ModulePath)
	}
}

func TestMapContextToProject_CreatedDateFormat(t *testing.T) {
	// Verifies the date format is YYYY-MM-DD as required by templates.
	profile := &ContextProfile{
		ProjectName: "svc",
		IsInternal:    false,
		Stacks:      []ContextStack{{Name: "go", Language: "go", Framework: "chi"}},
	}

	pc := MapContextToProject(profile)

	if len(pc.CreatedDate) != 10 {
		t.Errorf("expected CreatedDate length 10 (YYYY-MM-DD), got %d: '%s'", len(pc.CreatedDate), pc.CreatedDate)
	}
	if pc.CreatedDate[4] != '-' || pc.CreatedDate[7] != '-' {
		t.Errorf("expected CreatedDate format YYYY-MM-DD, got '%s'", pc.CreatedDate)
	}
}

func TestLoadContextProfile_MultipleStacks(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".neo.json")

	content := `{
		"projectName": "fullstack-app",
		"projectType": "fullstack",
		"isInternal": false,
		"stacks": [
			{"name": "go-api", "language": "go", "framework": "chi"},
			{"name": "react-ui", "language": "nodejs", "framework": "react"}
		]
	}`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	profile, err := LoadContextProfile(path)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if len(profile.Stacks) != 2 {
		t.Fatalf("expected 2 stacks, got %d", len(profile.Stacks))
	}

	// MapContextToProject should use first stack
	pc := MapContextToProject(profile)
	if pc.Tech != "go" {
		t.Errorf("expected Tech='go' from first stack, got '%s'", pc.Tech)
	}
	if !pc.IsGo {
		t.Error("expected IsGo=true from first stack")
	}
}
