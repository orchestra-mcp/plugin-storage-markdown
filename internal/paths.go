package internal

import (
	"fmt"
	"path/filepath"
	"strings"
)

// projectsDir is the subdirectory under the workspace where all storage files
// are kept.
const projectsDir = ".projects"

// resolvePath joins the workspace root with the .projects directory and the
// given storage path. It validates that the resolved path does not escape the
// workspace using path traversal.
func resolvePath(workspace, storagePath string) (string, error) {
	if storagePath == "" {
		return "", fmt.Errorf("storage path must not be empty")
	}

	// Reject paths that contain ".." components to prevent directory traversal.
	cleaned := filepath.Clean(storagePath)
	if strings.Contains(cleaned, "..") {
		return "", fmt.Errorf("storage path must not contain '..': %q", storagePath)
	}

	base := filepath.Join(workspace, projectsDir)
	full := filepath.Join(base, cleaned)

	// Double-check that the resolved path is still under the base directory.
	rel, err := filepath.Rel(base, full)
	if err != nil {
		return "", fmt.Errorf("resolve path: %w", err)
	}
	if strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("storage path escapes workspace: %q", storagePath)
	}

	return full, nil
}

// versionPath returns the sidecar version file path for a given storage file
// path. The version file is stored alongside the main file with a ".version"
// extension appended.
func versionPath(filePath string) string {
	return filePath + ".version"
}
