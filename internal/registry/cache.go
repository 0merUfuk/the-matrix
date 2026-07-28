package registry

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"
)

const (
	// indexTTL is how long the cached registry.json is considered fresh.
	indexTTL = 1 * time.Hour

	// packTTL is how long a cached pack is considered fresh.
	packTTL = 7 * 24 * time.Hour

	// cacheDir is the directory name under ~/.the-matrix/ for registry cache.
	cacheDirName = "registry"

	// indexFileName is the cached registry index filename.
	indexFileName = "registry.json"

	// packsDirName is the subdirectory for cached pack data.
	packsDirName = "packs"
)

// cacheBasePath returns the absolute path to ~/.the-matrix/registry/.
func cacheBasePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("determining home directory: %w", err)
	}
	return filepath.Join(home, ".the-matrix", cacheDirName), nil
}

// ensureCacheDir creates the cache directory if it doesn't exist.
func ensureCacheDir(path string) error {
	return os.MkdirAll(path, 0755)
}

// isFresh checks whether a file's modification time is within the given TTL.
func isFresh(path string, ttl time.Duration) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return time.Since(info.ModTime()) < ttl
}

// ReadCachedIndex reads the cached registry index. The second return value
// indicates whether the cached data is fresh (within indexTTL). If the cache
// file does not exist or is corrupt, (nil, false) is returned.
func ReadCachedIndex() (*RegistryIndex, bool) {
	base, err := cacheBasePath()
	if err != nil {
		return nil, false
	}
	path := filepath.Join(base, indexFileName)

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}

	var index RegistryIndex
	if err := json.Unmarshal(data, &index); err != nil {
		// Corrupt cache — delete it so next call fetches fresh data.
		_ = os.Remove(path)
		return nil, false
	}

	fresh := isFresh(path, indexTTL)
	return &index, fresh
}

// WriteCachedIndex writes the registry index to the local cache.
func WriteCachedIndex(index *RegistryIndex) error {
	base, err := cacheBasePath()
	if err != nil {
		return err
	}
	if err := ensureCacheDir(base); err != nil {
		return fmt.Errorf("creating cache directory: %w", err)
	}

	data, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling index: %w", err)
	}
	data = append(data, '\n')

	return os.WriteFile(filepath.Join(base, indexFileName), data, 0644)
}

// safePackNameRe matches pack names containing only safe characters:
// alphanumeric, hyphens, underscores, and dots. No path separators or
// traversal sequences (../) are permitted.
var safePackNameRe = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_\-\.]*$`)

// ValidatePackName checks that a pack name is safe to use as a filesystem
// directory component. It rejects names containing path separators, traversal
// sequences (../), or any characters outside [a-zA-Z0-9_\-.]. Names must also
// start with an alphanumeric character.
func ValidatePackName(name string) error {
	if name == "" {
		return fmt.Errorf("invalid pack name: must not be empty")
	}
	if !safePackNameRe.MatchString(name) {
		return fmt.Errorf("invalid pack name %q: must contain only alphanumeric characters, hyphens, underscores, and dots", name)
	}
	return nil
}

// ReadCachedPack reads a cached pack's data (pack.json bytes). The second
// return value indicates whether the cached data is fresh (within packTTL).
func ReadCachedPack(packName string) ([]byte, bool) {
	if err := ValidatePackName(packName); err != nil {
		return nil, false
	}
	base, err := cacheBasePath()
	if err != nil {
		return nil, false
	}
	path := filepath.Join(base, packsDirName, packName, "pack.json")

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}

	if !isFresh(path, packTTL) {
		return data, false
	}
	return data, true
}

// WriteCachedPack writes pack data to the local cache under packs/{packName}/.
func WriteCachedPack(packName string, data []byte) error {
	if err := ValidatePackName(packName); err != nil {
		return err
	}
	base, err := cacheBasePath()
	if err != nil {
		return err
	}
	packDir := filepath.Join(base, packsDirName, packName)
	if err := ensureCacheDir(packDir); err != nil {
		return fmt.Errorf("creating pack cache directory: %w", err)
	}

	return os.WriteFile(filepath.Join(packDir, "pack.json"), data, 0644)
}
