package internal

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	pluginv1 "github.com/orchestra-mcp/gen-go/orchestra/plugin/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// StoragePlugin implements the plugin.StorageHandler interface for the
// storage.markdown plugin. It persists Protobuf metadata + Markdown body files
// to disk under workspace/.projects/.
type StoragePlugin struct {
	workspace string

	// mu guards concurrent access to the filesystem to prevent race conditions
	// during version checking and file writes.
	mu sync.Mutex
}

// NewStoragePlugin creates a new storage plugin rooted at the given workspace
// directory.
func NewStoragePlugin(workspace string) *StoragePlugin {
	return &StoragePlugin{
		workspace: workspace,
	}
}

// Read loads a markdown file from disk, parsing its META header and body.
func (s *StoragePlugin) Read(_ context.Context, req *pluginv1.StorageReadRequest) (*pluginv1.StorageReadResponse, error) {
	filePath, err := resolvePath(s.workspace, req.Path)
	if err != nil {
		return nil, fmt.Errorf("resolve path: %w", err)
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	metadata, body, err := ParseMarkdownFile(data)
	if err != nil {
		return nil, fmt.Errorf("parse markdown: %w", err)
	}

	version, _ := readVersion(filePath)

	return &pluginv1.StorageReadResponse{
		Content:  body,
		Metadata: metadata,
		Version:  version,
	}, nil
}

// Write persists metadata and body to a markdown file on disk with CAS
// (compare-and-swap) versioning.
func (s *StoragePlugin) Write(_ context.Context, req *pluginv1.StorageWriteRequest) (*pluginv1.StorageWriteResponse, error) {
	filePath, err := resolvePath(s.workspace, req.Path)
	if err != nil {
		return &pluginv1.StorageWriteResponse{
			Success: false,
			Error:   fmt.Sprintf("resolve path: %v", err),
		}, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	currentVersion, _ := readVersion(filePath)

	// CAS versioning logic.
	if req.ExpectedVersion == 0 {
		// Create new: fail if the file already exists.
		if _, err := os.Stat(filePath); err == nil {
			return &pluginv1.StorageWriteResponse{
				Success: false,
				Error:   "file already exists (expected_version=0 means create new)",
			}, nil
		}
	} else {
		// Update: expected version must match current version.
		if currentVersion != req.ExpectedVersion {
			return &pluginv1.StorageWriteResponse{
				Success: false,
				Error:   fmt.Sprintf("version conflict: expected %d, current %d", req.ExpectedVersion, currentVersion),
			}, nil
		}
	}

	// Format the file content.
	data, err := FormatMarkdownFile(req.Metadata, req.Content)
	if err != nil {
		return &pluginv1.StorageWriteResponse{
			Success: false,
			Error:   fmt.Sprintf("format markdown: %v", err),
		}, nil
	}

	// Ensure parent directory exists.
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return &pluginv1.StorageWriteResponse{
			Success: false,
			Error:   fmt.Sprintf("create directory: %v", err),
		}, nil
	}

	// Write the file.
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return &pluginv1.StorageWriteResponse{
			Success: false,
			Error:   fmt.Sprintf("write file: %v", err),
		}, nil
	}

	// Increment and persist version.
	newVersion := currentVersion + 1
	if err := writeVersion(filePath, newVersion); err != nil {
		return &pluginv1.StorageWriteResponse{
			Success: false,
			Error:   fmt.Sprintf("write version: %v", err),
		}, nil
	}

	return &pluginv1.StorageWriteResponse{
		Success:    true,
		NewVersion: newVersion,
	}, nil
}

// Delete removes a markdown file and its version sidecar from disk.
func (s *StoragePlugin) Delete(_ context.Context, req *pluginv1.StorageDeleteRequest) (*pluginv1.StorageDeleteResponse, error) {
	filePath, err := resolvePath(s.workspace, req.Path)
	if err != nil {
		return &pluginv1.StorageDeleteResponse{Success: false}, fmt.Errorf("resolve path: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Remove the main file.
	if err := os.Remove(filePath); err != nil {
		if os.IsNotExist(err) {
			return &pluginv1.StorageDeleteResponse{Success: false}, fmt.Errorf("file not found: %s", req.Path)
		}
		return &pluginv1.StorageDeleteResponse{Success: false}, fmt.Errorf("delete file: %w", err)
	}

	// Remove the version sidecar (ignore errors if it does not exist).
	_ = os.Remove(versionPath(filePath))

	return &pluginv1.StorageDeleteResponse{Success: true}, nil
}

// List enumerates markdown files under the given prefix directory.
func (s *StoragePlugin) List(_ context.Context, req *pluginv1.StorageListRequest) (*pluginv1.StorageListResponse, error) {
	prefix := req.Prefix
	if prefix == "" {
		prefix = "."
	}

	basePath, err := resolvePath(s.workspace, prefix)
	if err != nil {
		return nil, fmt.Errorf("resolve prefix: %w", err)
	}

	// Determine the glob pattern for matching.
	pattern := req.Pattern
	if pattern == "" {
		pattern = "*.md"
	}

	var entries []*pluginv1.StorageEntry

	err = filepath.Walk(basePath, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return nil // skip inaccessible entries
		}
		if info.IsDir() {
			return nil
		}

		// Skip version sidecar files.
		if strings.HasSuffix(path, ".version") {
			return nil
		}

		// Match against the pattern.
		matched, matchErr := filepath.Match(pattern, filepath.Base(path))
		if matchErr != nil {
			return nil
		}
		if !matched {
			return nil
		}

		// Compute the storage-relative path.
		projectsBase := filepath.Join(s.workspace, projectsDir)
		relPath, relErr := filepath.Rel(projectsBase, path)
		if relErr != nil {
			return nil
		}

		version, _ := readVersion(path)

		entries = append(entries, &pluginv1.StorageEntry{
			Path:       relPath,
			Size:       info.Size(),
			Version:    version,
			ModifiedAt: timestamppb.New(info.ModTime()),
		})

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk directory: %w", err)
	}

	return &pluginv1.StorageListResponse{
		Entries: entries,
	}, nil
}

// readVersion reads the version number from the sidecar file. Returns 0 if
// the sidecar does not exist or cannot be parsed.
func readVersion(filePath string) (int64, error) {
	data, err := os.ReadFile(versionPath(filePath))
	if err != nil {
		return 0, err
	}
	v, err := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
	if err != nil {
		return 0, err
	}
	return v, nil
}

// writeVersion writes the version number to the sidecar file.
func writeVersion(filePath string, version int64) error {
	return os.WriteFile(versionPath(filePath), []byte(strconv.FormatInt(version, 10)), 0644)
}
