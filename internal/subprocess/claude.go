// Package subprocess provides shared wrappers for agent runtimes and bash scripts.
package subprocess

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/0merUfuk/the-matrix/internal/runtime"
	_ "github.com/0merUfuk/the-matrix/internal/runtime/builtin"
)

// Options configures an agent runtime subprocess call.
type Options struct {
	Model        string
	ModelTier    runtime.ModelTier
	ModelTierSet bool
	MaxTurns     int
	TimeoutMs    int64
	Runtime      string
	WorkingDir   string
	OutputJSON   bool
}

// DefaultOptions returns sensible defaults for Claude subprocess calls.
func DefaultOptions() Options {
	return Options{
		Model:        "claude-sonnet-4-6",
		ModelTier:    runtime.ModelBalanced,
		ModelTierSet: true,
		MaxTurns:     5,
		TimeoutMs:    300000, // 5 minutes
	}
}

// CallClaude invokes claude -p with the given prompt and options.
// Returns the trimmed stdout on success, or an error.
func CallClaude(prompt string, opts Options) (string, error) {
	opts.Runtime = "claude"
	return Invoke(prompt, opts)
}

// Invoke runs a prompt through the configured agent runtime.
func Invoke(prompt string, opts Options) (string, error) {
	if opts.Runtime == "" {
		opts.Runtime = "claude"
	}
	rt := runtime.Registry.Get(opts.Runtime)
	if rt == nil {
		return "", fmt.Errorf("runtime %q not registered", opts.Runtime)
	}

	if opts.MaxTurns == 0 {
		opts.MaxTurns = DefaultOptions().MaxTurns
	}
	if opts.TimeoutMs == 0 {
		opts.TimeoutMs = DefaultOptions().TimeoutMs
	}
	modelTier := opts.ModelTier
	if !opts.ModelTierSet && modelTier == runtime.ModelFast {
		modelTier = runtime.ModelBalanced
	}

	cmdSpec, err := rt.BuildCommand(runtime.InvokeOpts{
		Prompt:     prompt,
		MaxTurns:   opts.MaxTurns,
		Model:      modelTier,
		OutputJSON: opts.OutputJSON,
		WorkingDir: opts.WorkingDir,
	})
	if err != nil {
		return "", fmt.Errorf("build command: %w", err)
	}

	timeout := time.Duration(opts.TimeoutMs) * time.Millisecond
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, cmdSpec.Binary, cmdSpec.Args...)
	cmd.Dir = opts.WorkingDir

	env := os.Environ()
	for _, key := range cmdSpec.Unset {
		filtered := make([]string, 0, len(env))
		for _, e := range env {
			if !strings.HasPrefix(e, key+"=") {
				filtered = append(filtered, e)
			}
		}
		env = filtered
	}
	for key, value := range cmdSpec.Env {
		env = append(env, key+"="+value)
	}
	cmd.Env = env

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	if ctx.Err() == context.DeadlineExceeded {
		return "", fmt.Errorf("%s subprocess timed out after %dms", rt.Name(), opts.TimeoutMs)
	}
	if err != nil {
		stderrSnippet := stderr.String()
		if len(stderrSnippet) > 500 {
			stderrSnippet = stderrSnippet[:500]
		}
		return "", fmt.Errorf("%s subprocess failed: %w. stderr: %s", rt.Name(), err, stderrSnippet)
	}

	result := strings.TrimSpace(stdout.String())
	if result == "" {
		return "", fmt.Errorf("%s subprocess returned empty output", rt.Name())
	}
	if strings.HasPrefix(result, "Error:") {
		firstLine := strings.SplitN(result, "\n", 2)[0]
		return "", fmt.Errorf("%s subprocess error: %s", rt.Name(), firstLine)
	}

	return result, nil
}

// RunBash executes a bash script file. If inherit is true, stdout/stderr go to the terminal.
func RunBash(script string, cwd string, inherit bool) error {
	cmd := exec.Command("bash", script)
	cmd.Dir = cwd
	cmd.Env = os.Environ()

	if inherit {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("bash script %s failed: %w", script, err)
	}
	return nil
}
