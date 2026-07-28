package registry

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

const (
	// DefaultRegistryURL is the raw GitHub URL for the central knowledge registry index.
	DefaultRegistryURL = "https://raw.githubusercontent.com/0merUfuk/knowledge-registry/main/registry.json"
)

// BasePackURL is the base path for pack detail and file downloads.
// It is a variable (not a constant) so that tests can inject a mock server URL.
var BasePackURL = "https://raw.githubusercontent.com/0merUfuk/knowledge-registry/main/packs"

// Load fetches the registry index, using the local cache when fresh.
// On cache miss or stale cache, it fetches from GitHub and updates the cache.
// Returns an error if both cache and network are unavailable.
func Load() (*RegistryIndex, error) {
	// Try cache first.
	cached, fresh := ReadCachedIndex()
	if fresh && cached != nil {
		return cached, nil
	}

	// Fetch from network.
	index, err := FetchRegistryIndex(DefaultRegistryURL)
	if err != nil {
		// If we have a stale cache, return it with no error — better stale than nothing.
		if cached != nil {
			return cached, nil
		}
		return nil, fmt.Errorf("registry unreachable: %w", err)
	}

	// Update cache (best-effort — don't fail if cache write fails).
	_ = WriteCachedIndex(index)

	return index, nil
}

// Search filters packs by a case-insensitive query against name, language,
// framework, and description fields.
func Search(index *RegistryIndex, query string) []PackMeta {
	if index == nil || query == "" {
		return nil
	}

	q := strings.ToLower(query)
	var results []PackMeta

	for _, p := range index.Packs {
		if strings.Contains(strings.ToLower(p.Name), q) ||
			strings.Contains(strings.ToLower(p.Language), q) ||
			strings.Contains(strings.ToLower(p.Framework), q) ||
			strings.Contains(strings.ToLower(p.Description), q) {
			results = append(results, p)
		}
	}
	return results
}

// FindMatch finds an exact match for a language+framework combination in the registry.
// Language comparison is case-insensitive. Framework comparison is case-insensitive
// and uses a contains check to handle version suffixes (e.g., "chi v5" matches "chi").
// Returns nil if no match is found.
func FindMatch(index *RegistryIndex, language, framework string) *PackMeta {
	if index == nil {
		return nil
	}

	lang := strings.ToLower(language)
	fw := strings.ToLower(framework)

	// Don't match on empty framework — strings.Contains(anything, "") is always
	// true, which would false-positive on the first pack of the matching language.
	if fw == "" {
		return nil
	}

	for i := range index.Packs {
		p := &index.Packs[i]
		if strings.ToLower(p.Language) != lang {
			continue
		}

		packFw := strings.ToLower(p.Framework)
		// Exact match or contains (e.g., wizard says "chi v5", pack says "chi")
		if packFw == fw || strings.Contains(fw, packFw) || strings.Contains(packFw, fw) {
			return p
		}
	}
	return nil
}

// Info fetches full pack details for the named pack.
// Uses cache if fresh; otherwise fetches from GitHub.
func Info(packName string) (*PackDetail, error) {
	if err := ValidatePackName(packName); err != nil {
		return nil, err
	}

	// Try cache.
	data, fresh := ReadCachedPack(packName)
	if fresh && data != nil {
		var detail PackDetail
		if err := json.Unmarshal(data, &detail); err == nil {
			return &detail, nil
		}
		// Corrupt cache — fall through to fetch.
	}

	// Fetch from network.
	packURL := fmt.Sprintf("%s/%s/pack.json", BasePackURL, packName)
	detail, err := FetchPackDetail(packURL)
	if err != nil {
		// Try stale cache.
		if data != nil {
			var d PackDetail
			if jsonErr := json.Unmarshal(data, &d); jsonErr == nil {
				return &d, nil
			}
		}
		return nil, fmt.Errorf("fetching pack %q: %w", packName, err)
	}

	// Update cache (best-effort).
	if cacheData, err := json.MarshalIndent(detail, "", "  "); err == nil {
		_ = WriteCachedPack(packName, cacheData)
	}

	return detail, nil
}

// Pull downloads all knowledge doc files from a pack to the target directory.
// Each file is validated using the provided validateFn before writing.
// Returns an error if any file fails validation or download.
func Pull(packName string, targetDir string, validateFn func(filename, content string) error) error {
	if err := ValidatePackName(packName); err != nil {
		return err
	}

	detail, err := Info(packName)
	if err != nil {
		return fmt.Errorf("loading pack details: %w", err)
	}

	if len(detail.Topics) == 0 {
		return fmt.Errorf("pack %q has no topics", packName)
	}

	// Download and validate each file.
	type docFile struct {
		filename string
		content  string
	}
	var files []docFile

	for _, topic := range detail.Topics {
		// Sanitize filename — topic.File is untrusted input from a remote pack.json.
		// Strip directory components to prevent path traversal (e.g., "../../../.ssh/authorized_keys").
		safeName := filepath.Base(topic.File)
		if safeName == "." || safeName == ".." || safeName == "" {
			return fmt.Errorf("pull: invalid topic filename: %q", topic.File)
		}

		fileURL := fmt.Sprintf("%s/%s/%s", BasePackURL, packName, url.PathEscape(safeName))
		content, err := FetchPackFile(fileURL)
		if err != nil {
			return fmt.Errorf("downloading %s: %w", topic.File, err)
		}

		// Validate before accepting.
		if validateFn != nil {
			if err := validateFn(safeName, content); err != nil {
				return fmt.Errorf("validation failed for %s: %w", safeName, err)
			}
		}

		files = append(files, docFile{filename: safeName, content: content})
	}

	// All files downloaded and validated — write them.
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("creating target directory: %w", err)
	}

	for _, f := range files {
		path := filepath.Join(targetDir, f.filename)
		if err := os.WriteFile(path, []byte(f.content), 0644); err != nil {
			return fmt.Errorf("writing %s: %w", f.filename, err)
		}
	}

	return nil
}
