// Package subprocess provides a shared wrapper for calling claude -p and bash scripts.
package subprocess

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// Options configures a Claude subprocess call.
type Options struct {
	Model     string
	MaxTurns  int
	TimeoutMs int64
}

// DefaultOptions returns sensible defaults for Claude subprocess calls.
func DefaultOptions() Options {
	return Options{
		Model:     "claude-sonnet-4-6",
		MaxTurns:  5,
		TimeoutMs: 300000, // 5 minutes
	}
}

// CallClaude invokes claude -p with the given prompt and options.
// Returns the trimmed stdout on success, or an error.
func CallClaude(prompt string, opts Options) (string, error) {
	if opts.Model == "" {
		opts.Model = DefaultOptions().Model
	}
	if opts.MaxTurns == 0 {
		opts.MaxTurns = DefaultOptions().MaxTurns
	}
	if opts.TimeoutMs == 0 {
		opts.TimeoutMs = DefaultOptions().TimeoutMs
	}

	timeout := time.Duration(opts.TimeoutMs) * time.Millisecond
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	args := []string{
		"-p", prompt,
		"--output-format", "text",
		"--max-turns", strconv.Itoa(opts.MaxTurns),
		"--model", opts.Model,
		"--no-session-persistence",
	}

	cmd := exec.CommandContext(ctx, "claude", args...)

	// Unset CLAUDECODE to prevent nested session errors
	env := os.Environ()
	filtered := make([]string, 0, len(env))
	for _, e := range env {
		if !strings.HasPrefix(e, "CLAUDECODE=") {
			filtered = append(filtered, e)
		}
	}
	cmd.Env = filtered

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if ctx.Err() == context.DeadlineExceeded {
		return "", fmt.Errorf("claude subprocess timed out after %dms", opts.TimeoutMs)
	}
	if err != nil {
		stderrSnippet := stderr.String()
		if len(stderrSnippet) > 500 {
			stderrSnippet = stderrSnippet[:500]
		}
		return "", fmt.Errorf("claude subprocess failed: %w. stderr: %s", err, stderrSnippet)
	}

	result := strings.TrimSpace(stdout.String())
	if result == "" {
		return "", fmt.Errorf("claude subprocess returned empty output")
	}
	if strings.HasPrefix(result, "Error:") {
		firstLine := strings.SplitN(result, "\n", 2)[0]
		return "", fmt.Errorf("claude subprocess error: %s", firstLine)
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
