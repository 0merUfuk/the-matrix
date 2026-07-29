package safewrite

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type OverwriteOpts struct {
	Path  string
	Force bool
	IsTTY bool
}

func ResolveOutputPath(dir string) (string, error) {
	if dir == "" {
		return "", fmt.Errorf("output path cannot be empty")
	}
	absPath, err := filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("resolve output path: %w", err)
	}
	if IsProtectedPath(absPath) {
		return "", fmt.Errorf("refusing to use protected output path: %s", absPath)
	}
	return absPath, nil
}

func IsProtectedPath(p string) bool {
	if p == "" {
		return true
	}
	absPath, err := filepath.Abs(p)
	if err != nil {
		return true
	}
	if filepath.Clean(absPath) == filepath.VolumeName(absPath)+string(filepath.Separator) {
		return true
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	homePath, err := filepath.Abs(home)
	if err != nil {
		return false
	}
	return filepath.Clean(absPath) == filepath.Clean(homePath)
}

func ConfirmOverwrite(opts OverwriteOpts) (bool, error) {
	path, err := ResolveOutputPath(opts.Path)
	if err != nil {
		return false, err
	}
	_, err = os.Stat(path)
	if os.IsNotExist(err) {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	if opts.Force {
		return true, nil
	}
	if !opts.IsTTY {
		return false, nil
	}
	fmt.Fprintf(os.Stdout, "Directory %s already exists. Overwrite? [y/N]:", path)
	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return false, err
		}
		return false, nil
	}
	answer := strings.ToLower(strings.TrimSpace(scanner.Text()))
	return answer == "y" || answer == "yes", nil
}
