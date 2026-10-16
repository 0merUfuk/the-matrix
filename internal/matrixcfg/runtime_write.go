package matrixcfg

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// WriteRuntime records the selected runtime in the project's Matrix config.
// Empty runtime names are ignored so callers can preserve automatic detection.
func WriteRuntime(root, runtimeName string) error {
	runtimeName = strings.TrimSpace(runtimeName)
	if runtimeName == "" {
		return nil
	}

	path := findProjectConfigFile(root)
	if path == "" {
		path = filepath.Join(root, ".matrix.yaml")
	}

	var document yaml.Node
	if data, err := os.ReadFile(path); err == nil { // #nosec G304 -- reads .matrix.yaml/.matrix.yml at the project root
		if err := yaml.Unmarshal(data, &document); err != nil {
			return fmt.Errorf("parsing config %s: %w", path, err)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("reading config %s: %w", path, err)
	}

	mapping, err := topLevelMapping(&document)
	if err != nil {
		return fmt.Errorf("updating config %s: %w", path, err)
	}
	setMappingString(mapping, "runtime", runtimeName)

	var data bytes.Buffer
	encoder := yaml.NewEncoder(&data)
	encoder.SetIndent(2)
	if err := encoder.Encode(&document); err != nil {
		return fmt.Errorf("encoding config %s: %w", path, err)
	}
	if err := encoder.Close(); err != nil {
		return fmt.Errorf("encoding config %s: %w", path, err)
	}

	if err := writeFileAtomic(path, data.Bytes(), 0644); err != nil {
		return fmt.Errorf("writing config %s: %w", path, err)
	}
	return nil
}

func topLevelMapping(document *yaml.Node) (*yaml.Node, error) {
	if len(document.Content) == 0 {
		document.Kind = yaml.DocumentNode
		mapping := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		document.Content = []*yaml.Node{mapping}
		return mapping, nil
	}

	if document.Kind != yaml.DocumentNode || len(document.Content) != 1 {
		return nil, fmt.Errorf("config must contain one YAML document")
	}
	mapping := document.Content[0]
	if mapping.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("config root must be a mapping")
	}
	return mapping, nil
}

func setMappingString(mapping *yaml.Node, key, value string) {
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value != key {
			continue
		}

		node := mapping.Content[i+1]
		node.Kind = yaml.ScalarNode
		node.Tag = "!!str"
		node.Value = value
		node.Style = 0
		node.Content = nil
		node.Anchor = ""
		node.Alias = nil
		return
	}

	mapping.Content = append(mapping.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key},
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: value},
	)
}

func writeFileAtomic(path string, data []byte, mode os.FileMode) error {
	if info, err := os.Stat(path); err == nil {
		mode = info.Mode().Perm()
	} else if !os.IsNotExist(err) {
		return err
	}

	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".matrix-config-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if err := tmp.Chmod(mode); err != nil {
		return errors.Join(err, tmp.Close())
	}
	if _, err := tmp.Write(data); err != nil {
		return errors.Join(err, tmp.Close())
	}
	if err := tmp.Sync(); err != nil {
		return errors.Join(err, tmp.Close())
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}
