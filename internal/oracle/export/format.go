// Package export transforms oracle knowledge docs into formats consumed by
// other AI coding tools (Cursor, Copilot, AGENTS.md, Windsurf).
package export

import "sort"

// Format is the interface every export adapter must implement.
type Format interface {
	Name() string
	Description() string
	Export(docs []KnowledgeDoc, targetDir string, opts ExportOpts) error
}

// KnowledgeDoc represents a single oracle knowledge document.
type KnowledgeDoc struct {
	Filename string // e.g. "04-error-handling.md"
	Topic    string // e.g. "04-error-handling" (filename without .md)
	Title    string // e.g. "Error Handling" (extracted from # heading or topic name)
	Content  string // full file content
}

// ExportOpts contains metadata about the source knowledge.
type ExportOpts struct {
	StackName string // e.g. "go-internal"
	Language  string // e.g. "go", "typescript", "python"
	Framework string // e.g. "chi", "express", "fastapi" (best-effort from dir name)
}

// Registry maps format names to their implementations.
var Registry = map[string]Format{}

// Register adds a format to the registry. Call from init() in each adapter.
func Register(f Format) {
	Registry[f.Name()] = f
}

// FormatNames returns all registered format names sorted.
func FormatNames() []string {
	names := make([]string, 0, len(Registry))
	for name := range Registry {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
