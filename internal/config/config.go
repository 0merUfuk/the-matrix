// Package config provides shared file I/O and config parsing utilities.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// ConfigValues holds parsed values from a config.sh file.
type ConfigValues struct {
	MaxCycles         int
	DeveloperMaxTurns int
	TesterMaxTurns    int
	ReviewerMaxTurns  int
}

// ReadJSON reads and unmarshals a JSON file into the given type.
func ReadJSON[T any](path string) (T, error) {
	var result T
	data, err := os.ReadFile(path)
	if err != nil {
		return result, fmt.Errorf("reading %s: %w", path, err)
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return result, fmt.Errorf("parsing JSON %s: %w", path, err)
	}
	return result, nil
}

// WriteJSON marshals and writes a value to a JSON file with indentation.
func WriteJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling JSON: %w", err)
	}
	data = append(data, '\n')
	if err := EnsureDir(filepath.Dir(path)); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// ReadConfigSh parses KEY=VALUE lines from a config.sh file.
// Only matches lines starting with the key (skips commented lines).
func ReadConfigSh(path string) (ConfigValues, error) {
	var cfg ConfigValues
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, fmt.Errorf("reading %s: %w", path, err)
	}
	content := string(data)

	cfg.MaxCycles = matchConfigInt(content, "MAX_CYCLES")
	cfg.DeveloperMaxTurns = matchConfigInt(content, "DEVELOPER_MAX_TURNS")
	cfg.TesterMaxTurns = matchConfigInt(content, "TESTER_MAX_TURNS")
	cfg.ReviewerMaxTurns = matchConfigInt(content, "REVIEWER_MAX_TURNS")

	return cfg, nil
}

// matchConfigInt extracts an integer from a KEY=N line, anchored to line start.
func matchConfigInt(content, key string) int {
	re := regexp.MustCompile(`(?m)^` + regexp.QuoteMeta(key) + `=(\d+)`)
	m := re.FindStringSubmatch(content)
	if m == nil {
		return 0
	}
	v, _ := strconv.Atoi(m[1])
	return v
}

// FileExists returns true if the path exists (file or directory).
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// IsDir returns true if the path exists and is a directory.
func IsDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// EnsureDir creates the directory and all parents if they don't exist.
func EnsureDir(path string) error {
	return os.MkdirAll(path, 0755)
}

// ReadFileString reads a file and returns its content as a string.
func ReadFileString(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// WriteFileString writes a string to a file, creating parent directories.
func WriteFileString(path, content string) error {
	if err := EnsureDir(filepath.Dir(path)); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0644)
}

// AppendFileString appends a string to a file, creating it if needed.
func AppendFileString(path, content string) error {
	if err := EnsureDir(filepath.Dir(path)); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(content)
	return err
}

// ListMDFiles returns all .md files in a directory (excluding _index.md), sorted alphabetically.
func ListMDFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var files []string
	for _, e := range entries {
		name := e.Name()
		if !e.IsDir() && strings.HasSuffix(name, ".md") && name != "_index.md" {
			files = append(files, name)
		}
	}
	sort.Strings(files)
	return files, nil
}

// CopyFile copies a file from src to dst, creating parent directories. The
// destination is constrained to baseDst: after path cleaning and symlink
// resolution, dst must resolve to a location under baseDst. A `..` segment,
// an absolute dst escaping the base, or a symlinked parent directory pointing
// outside baseDst will be rejected with an error. See
// .claude/rules/security-baseline.md §1.
//
// baseDst must be a non-empty path. It does not need to already exist on disk
// — the containment check tolerates missing intermediate parents by walking up
// to the nearest existing ancestor for symlink resolution.
func CopyFile(src, dst, baseDst string) error {
	if baseDst == "" {
		return fmt.Errorf("CopyFile: baseDst must not be empty")
	}
	safeDst, err := containedPath(dst, baseDst)
	if err != nil {
		return err
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("reading %s: %w", src, err)
	}
	if err := EnsureDir(filepath.Dir(safeDst)); err != nil {
		return err
	}
	return os.WriteFile(safeDst, data, 0644) //nolint:gosec // G703 false positive: safeDst is the output of containedPath() which performs filepath.Abs + EvalSymlinks + HasPrefix base-containment validation per .claude/rules/security-baseline.md §1
}

// containedPath returns a cleaned, symlink-resolved absolute form of dst and
// verifies the result stays under base. Returns an error if dst escapes base
// lexically or via a symlink.
func containedPath(dst, base string) (string, error) {
	absBase, err := filepath.Abs(base)
	if err != nil {
		return "", fmt.Errorf("resolving base %q: %w", base, err)
	}
	// Resolve any symlinks in the base to the real on-disk location. We
	// need to compare realpaths, not lexical paths, because a symlinked
	// base would otherwise never match a realpath-resolved dst.
	if resolved, err := filepath.EvalSymlinks(absBase); err == nil {
		absBase = resolved
	}
	cleanBase := filepath.Clean(absBase)

	var absDst string
	if filepath.IsAbs(dst) {
		absDst = filepath.Clean(dst)
	} else {
		absDst = filepath.Clean(filepath.Join(cleanBase, dst))
	}

	// Resolve symlinks on the nearest existing ancestor of absDst. The dst
	// file itself may not exist yet (this is a copy target), but any
	// existing parent component could be a symlink pointing outside base.
	resolvedDst := resolveAncestorSymlinks(absDst)

	prefix := cleanBase + string(os.PathSeparator)
	if resolvedDst != cleanBase && !strings.HasPrefix(resolvedDst, prefix) {
		return "", fmt.Errorf("path escapes base directory: dst=%q base=%q", dst, base)
	}
	return resolvedDst, nil
}

// resolveAncestorSymlinks walks up path until it finds an existing ancestor,
// resolves symlinks on that ancestor, and rejoins the remaining unresolved
// suffix. This lets us validate a dst whose final component has not been
// created yet while still catching a symlinked parent that escapes the base.
func resolveAncestorSymlinks(path string) string {
	path = filepath.Clean(path)
	var missing []string
	cur := path
	for {
		if _, err := os.Lstat(cur); err == nil {
			resolved, err := filepath.EvalSymlinks(cur)
			if err != nil {
				// Stat said it exists; EvalSymlinks should succeed. If it
				// doesn't (race, broken symlink), fall back to lexical.
				return path
			}
			if len(missing) == 0 {
				return resolved
			}
			// Rebuild by joining resolved ancestor with the missing tail
			// (reversed because we prepended as we walked up).
			tail := ""
			for i := len(missing) - 1; i >= 0; i-- {
				tail = filepath.Join(tail, missing[i])
			}
			return filepath.Join(resolved, tail)
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			// Reached filesystem root without finding an existing ancestor.
			return path
		}
		missing = append(missing, filepath.Base(cur))
		cur = parent
	}
}

// MakeExecutable sets the executable bit on a file.
func MakeExecutable(path string) error {
	return os.Chmod(path, 0755)
}
