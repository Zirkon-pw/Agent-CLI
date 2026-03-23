package global

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/docup/agentctl/internal/config/loader"
)

// GetValue retrieves a value from GlobalConfig by dot-notation path.
// Example: "execution.default_agent" returns "claude".
func GetValue(cfg *loader.GlobalConfig, path string) (string, error) {
	node, err := marshalToNode(cfg)
	if err != nil {
		return "", err
	}

	parts := strings.Split(path, ".")
	leaf, err := walkNode(node, parts)
	if err != nil {
		return "", err
	}

	if leaf.Kind == yaml.SequenceNode {
		values := make([]string, 0, len(leaf.Content))
		for _, item := range leaf.Content {
			values = append(values, item.Value)
		}
		return strings.Join(values, ","), nil
	}

	return leaf.Value, nil
}

// SetValue sets a value in GlobalConfig by dot-notation path and returns the updated config.
// For sequence (list) fields, use comma-separated values: "a,b,c".
func SetValue(cfg *loader.GlobalConfig, path, value string) (*loader.GlobalConfig, error) {
	node, err := marshalToNode(cfg)
	if err != nil {
		return nil, err
	}

	parts := strings.Split(path, ".")
	leaf, err := walkNode(node, parts)
	if err != nil {
		return nil, err
	}

	if leaf.Kind == yaml.SequenceNode {
		leaf.Content = nil
		for _, v := range strings.Split(value, ",") {
			v = strings.TrimSpace(v)
			if v == "" {
				continue
			}
			leaf.Content = append(leaf.Content, &yaml.Node{
				Kind:  yaml.ScalarNode,
				Tag:   "!!str",
				Value: v,
			})
		}
	} else {
		leaf.Value = value
		leaf.Tag = "" // let yaml.v3 infer the tag
	}

	// Unmarshal back into GlobalConfig.
	var updated loader.GlobalConfig
	if err := node.Decode(&updated); err != nil {
		return nil, fmt.Errorf("decoding updated config: %w", err)
	}
	return &updated, nil
}

// ListPaths returns all available dot-notation paths in GlobalConfig.
func ListPaths(cfg *loader.GlobalConfig) ([]string, error) {
	node, err := marshalToNode(cfg)
	if err != nil {
		return nil, err
	}

	var paths []string
	collectPaths(node, "", &paths)
	return paths, nil
}

func marshalToNode(cfg *loader.GlobalConfig) (*yaml.Node, error) {
	var doc yaml.Node
	if err := doc.Encode(cfg); err != nil {
		return nil, fmt.Errorf("encoding config to node: %w", err)
	}
	// doc is a document node; the actual mapping is its first child.
	if doc.Kind == yaml.DocumentNode && len(doc.Content) > 0 {
		return doc.Content[0], nil
	}
	return &doc, nil
}

func walkNode(node *yaml.Node, parts []string) (*yaml.Node, error) {
	current := node
	for _, part := range parts {
		if current.Kind != yaml.MappingNode {
			return nil, fmt.Errorf("path segment %q: expected mapping, got %v", part, current.Kind)
		}
		found := false
		for i := 0; i < len(current.Content)-1; i += 2 {
			if current.Content[i].Value == part {
				current = current.Content[i+1]
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("path %q not found", part)
		}
	}
	return current, nil
}

func collectPaths(node *yaml.Node, prefix string, paths *[]string) {
	if node.Kind != yaml.MappingNode {
		return
	}
	for i := 0; i < len(node.Content)-1; i += 2 {
		key := node.Content[i].Value
		val := node.Content[i+1]
		fullPath := key
		if prefix != "" {
			fullPath = prefix + "." + key
		}
		if val.Kind == yaml.MappingNode {
			collectPaths(val, fullPath, paths)
		} else {
			*paths = append(*paths, fullPath)
		}
	}
}
