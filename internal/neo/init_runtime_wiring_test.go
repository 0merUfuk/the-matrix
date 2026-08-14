package neo

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"testing"

	agentruntime "github.com/0merUfuk/the-matrix/internal/runtime"
)

func TestRunInitPassesResolvedRuntimeToMorpheus(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".matrix.yaml"), []byte("runtime: codex\n"), 0644); err != nil {
		t.Fatalf("write runtime config: %v", err)
	}
	rt, err := agentruntime.Select("", dir)
	if err != nil {
		t.Fatalf("Select auto-detected runtime: %v", err)
	}
	if got := rt.Name(); got != "codex" {
		t.Fatalf("selected runtime = %q, want codex", got)
	}

	file, err := parser.ParseFile(token.NewFileSet(), "init.go", nil, 0)
	if err != nil {
		t.Fatalf("parse init.go: %v", err)
	}
	found := false
	ast.Inspect(file, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		ident, ok := call.Fun.(*ast.Ident)
		if !ok || ident.Name != "RunMorpheusIntegration" || len(call.Args) != 3 {
			return true
		}
		arg, ok := call.Args[2].(*ast.Ident)
		if !ok || arg.Name != "runtimeName" {
			t.Errorf("RunMorpheusIntegration runtime argument = %s, want runtimeName", formatExpr(call.Args[2]))
		}
		found = true
		return false
	})
	if !found {
		t.Fatal("RunInit does not call RunMorpheusIntegration with three arguments")
	}
}

func formatExpr(expr ast.Expr) string {
	if ident, ok := expr.(*ast.Ident); ok {
		return ident.Name
	}
	return "non-identifier expression"
}
