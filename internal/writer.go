package internal

import (
	"bytes"
	"fmt"
	"sort"

	"google.golang.org/protobuf/types/known/structpb"
	"gopkg.in/yaml.v3"
)

// FormatMarkdownFile serializes metadata and body into the on-disk markdown
// format with a YAML frontmatter header:
//
//	---
//	priority: P2
//	status: todo
//	---
//
//	# Markdown body here...
//
// If metadata is nil or has no fields, only the body is returned.
func FormatMarkdownFile(metadata *structpb.Struct, body []byte) ([]byte, error) {
	var buf bytes.Buffer

	if metadata != nil && len(metadata.Fields) > 0 {
		m := metadata.AsMap()

		// Sort keys for deterministic output.
		keys := make([]string, 0, len(m))
		for k := range m {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		sortedMap := &yamlOrderedMap{keys: keys, m: m}
		yamlBytes, err := yaml.Marshal(sortedMap)
		if err != nil {
			return nil, fmt.Errorf("marshal YAML frontmatter: %w", err)
		}

		buf.WriteString("---\n")
		buf.Write(yamlBytes)
		buf.WriteString("---\n")
		buf.WriteString("\n")
	}

	buf.Write(body)

	return buf.Bytes(), nil
}

// yamlOrderedMap implements yaml.Marshaler for deterministic key ordering.
type yamlOrderedMap struct {
	keys []string
	m    map[string]any
}

func (o *yamlOrderedMap) MarshalYAML() (any, error) {
	node := &yaml.Node{Kind: yaml.MappingNode}
	for _, k := range o.keys {
		keyNode := &yaml.Node{Kind: yaml.ScalarNode, Value: k}
		valNode := &yaml.Node{}
		if err := valNode.Encode(o.m[k]); err != nil {
			return nil, err
		}
		node.Content = append(node.Content, keyNode, valNode)
	}
	return node, nil
}
