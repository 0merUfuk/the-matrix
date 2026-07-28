package oracle

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// maxGoldStandardFiles is the maximum number of gold standard files to load.
const maxGoldStandardFiles = 5

// ResolveGoldStandardsDir determines the gold standards directory using a cascade:
// 1. Explicit flagDir (from --gold-standards flag) — if non-empty and contains .md files
// 2. Convention directory at ~/.the-matrix/gold-standards/<language>/ — auto-detected from stackName
// 3. Empty string (fall through to placeholder behavior)
func ResolveGoldStandardsDir(stackName string, flagDir string) string {
	// Step 1: explicit flag
	if flagDir != "" {
		if hasMDFiles(flagDir) {
			return flagDir
		}
		// Dir exists but no .md files — fall through to placeholder
		return ""
	}

	// Step 2: convention directory
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	lang := LanguageToConventionDir(stackName)
	if lang == "" {
		return ""
	}
	conventionDir := filepath.Join(home, ".the-matrix", "gold-standards", lang)
	if hasMDFiles(conventionDir) {
		return conventionDir
	}
	return ""
}

// LanguageToConventionDir maps a stack name or language identifier to a convention
// directory name under ~/.the-matrix/gold-standards/.
func LanguageToConventionDir(stackName string) string {
	lower := strings.ToLower(stackName)
	// Extract the first segment for compound names like "go-internal", "nodejs-express"
	parts := strings.SplitN(lower, "-", 2)
	first := parts[0]

	switch {
	case first == "go" || first == "golang":
		return "go"
	case first == "nodejs" || first == "node" || first == "javascript" || first == "typescript":
		return "nodejs"
	case first == "python":
		return "python"
	case first == "dart" || first == "flutter":
		return "dart"
	case first == "rust":
		return "rust"
	case first == "ruby":
		return "ruby"
	default:
		return ""
	}
}

// LoadGoldStandardsFromDir reads up to maxGoldStandardFiles .md files from dir,
// sorted alphabetically. Returns a map of basename (without .md) to file content.
// Returns an empty map if the directory doesn't exist or has no .md files.
func LoadGoldStandardsFromDir(dir string) map[string]string {
	standards := make(map[string]string)
	if dir == "" {
		return standards
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return standards
	}

	// Collect .md filenames, sorted
	var mdFiles []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
			mdFiles = append(mdFiles, e.Name())
		}
	}
	sort.Strings(mdFiles)

	// Read up to maxGoldStandardFiles
	count := len(mdFiles)
	if count > maxGoldStandardFiles {
		count = maxGoldStandardFiles
	}

	for i := 0; i < count; i++ {
		filename := mdFiles[i]
		filePath := filepath.Join(dir, filename)
		data, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}
		// Key is the basename without .md extension
		key := strings.TrimSuffix(filename, ".md")
		standards[key] = string(data)
	}

	return standards
}

// GoldStandardRegistrationFiles selects up to 5 representative files from a directory
// using a priority order suited for knowledge calibration.
func GoldStandardRegistrationFiles(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	var allMD []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") && e.Name() != "_index.md" {
			allMD = append(allMD, e.Name())
		}
	}
	sort.Strings(allMD)

	// Priority patterns — earlier patterns have higher priority
	priorities := [][]string{
		{"error", "04"},
		{"testing", "05"},
		{"architecture", "layered", "02"},
		{"core", "principles", "00"},
		{"project-structure", "01"},
	}

	var selected []string
	used := make(map[string]bool)

	// First pass: pick files matching priority patterns
	for _, patterns := range priorities {
		if len(selected) >= maxGoldStandardFiles {
			break
		}
		for _, f := range allMD {
			if used[f] {
				continue
			}
			lower := strings.ToLower(f)
			for _, p := range patterns {
				if strings.Contains(lower, p) {
					selected = append(selected, f)
					used[f] = true
					break
				}
			}
			if len(selected) >= maxGoldStandardFiles {
				break
			}
		}
	}

	// Second pass: fill remaining slots with any unselected files
	for _, f := range allMD {
		if len(selected) >= maxGoldStandardFiles {
			break
		}
		if !used[f] {
			selected = append(selected, f)
		}
	}

	return selected
}

// hasMDFiles returns true if dir exists and contains at least one .md file.
func hasMDFiles(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
			return true
		}
	}
	return false
}
