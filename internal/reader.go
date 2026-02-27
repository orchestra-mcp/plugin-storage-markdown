package internal

import (
	"bytes"
	"fmt"

	"google.golang.org/protobuf/types/known/structpb"
	"gopkg.in/yaml.v3"
)

// frontmatterDelim is the YAML frontmatter delimiter line.
var frontmatterDelim = []byte("---")

// ParseMarkdownFile parses a markdown file that begins with an optional YAML
// frontmatter block. The format is:
//
//	---
//	status: in-progress
//	priority: high
//	---
//
//	# Markdown body here...
//
// If the file does not start with a frontmatter block, the entire content is
// treated as the body with nil metadata.
func ParseMarkdownFile(data []byte) (metadata *structpb.Struct, body []byte, err error) {
	// Check if file starts with "---\n" or "---\r\n".
	if !bytes.HasPrefix(data, frontmatterDelim) {
		return nil, data, nil
	}

	// The first line must be exactly "---" (possibly with \r\n).
	firstNewline := bytes.IndexByte(data, '\n')
	if firstNewline == -1 {
		// Single line "---" with no body — not valid frontmatter.
		return nil, data, nil
	}

	firstLine := bytes.TrimRight(data[:firstNewline], "\r")
	if !bytes.Equal(firstLine, frontmatterDelim) {
		// Line starts with "---" but has trailing content — not frontmatter.
		return nil, data, nil
	}

	// Find the closing "---" delimiter.
	rest := data[firstNewline+1:]
	closingIdx := -1
	offset := 0
	for offset < len(rest) {
		lineEnd := bytes.IndexByte(rest[offset:], '\n')
		var line []byte
		if lineEnd == -1 {
			line = rest[offset:]
		} else {
			line = rest[offset : offset+lineEnd]
		}
		trimmedLine := bytes.TrimRight(line, "\r")
		if bytes.Equal(trimmedLine, frontmatterDelim) {
			closingIdx = offset
			break
		}
		if lineEnd == -1 {
			break
		}
		offset += lineEnd + 1
	}

	if closingIdx == -1 {
		return nil, nil, fmt.Errorf("malformed YAML frontmatter: missing closing ---")
	}

	yamlData := rest[:closingIdx]

	// Parse the YAML into a map.
	var m map[string]any
	if err := yaml.Unmarshal(yamlData, &m); err != nil {
		return nil, nil, fmt.Errorf("parse YAML frontmatter: %w", err)
	}

	metadata, err = structpb.NewStruct(m)
	if err != nil {
		return nil, nil, fmt.Errorf("convert frontmatter to structpb: %w", err)
	}

	// Body starts after the closing "---\n".
	bodyStart := closingIdx + len(frontmatterDelim)
	if bodyStart < len(rest) && rest[bodyStart] == '\r' {
		bodyStart++
	}
	if bodyStart < len(rest) && rest[bodyStart] == '\n' {
		bodyStart++
	}
	// Skip one additional blank line if present (separator between
	// frontmatter and body).
	if bodyStart < len(rest) && rest[bodyStart] == '\n' {
		bodyStart++
	} else if bodyStart+1 < len(rest) && rest[bodyStart] == '\r' && rest[bodyStart+1] == '\n' {
		bodyStart += 2
	}

	return metadata, rest[bodyStart:], nil
}
