package runtime

import (
	"fmt"
	"strings"
)

// Select resolves an explicit runtime name or detects one from a project.
// Built-in adapters must be imported by the calling binary before selection.
func Select(requested, projectDir string) (Runtime, error) {
	name := strings.TrimSpace(requested)
	if name == "" {
		name = DetectRuntime(projectDir)
	}
	rt := Registry.Get(name)
	if rt == nil {
		return nil, fmt.Errorf("unknown runtime %q", name)
	}
	return rt, nil
}
