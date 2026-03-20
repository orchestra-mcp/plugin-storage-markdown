package internal

import (
	"context"
	"strings"
	"testing"

	pluginv1 "github.com/orchestra-mcp/gen-go/orchestra/plugin/v1"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestParseMarkdownFile(t *testing.T) {
	input := []byte("---\npriority: P1\nstatus: in-progress\n---\n\n# Feature Title\n\nBody text here.\n")

	metadata, body, err := ParseMarkdownFile(input)
	if err != nil {
		t.Fatalf("ParseMarkdownFile: %v", err)
	}

	if metadata == nil {
		t.Fatal("expected metadata, got nil")
	}

	status, ok := metadata.Fields["status"]
	if !ok {
		t.Fatal("expected 'status' in metadata")
	}
	if status.GetStringValue() != "in-progress" {
		t.Errorf("status: got %q, want %q", status.GetStringValue(), "in-progress")
	}

	priority, ok := metadata.Fields["priority"]
	if !ok {
		t.Fatal("expected 'priority' in metadata")
	}
	if priority.GetStringValue() != "P1" {
		t.Errorf("priority: got %q, want %q", priority.GetStringValue(), "P1")
	}

	expectedBody := "# Feature Title\n\nBody text here.\n"
	if string(body) != expectedBody {
		t.Errorf("body: got %q, want %q", string(body), expectedBody)
	}
}

func TestParseMarkdownFileNoMeta(t *testing.T) {
	input := []byte("# Just a title\n\nSome content.\n")

	metadata, body, err := ParseMarkdownFile(input)
	if err != nil {
		t.Fatalf("ParseMarkdownFile: %v", err)
	}

	if metadata != nil {
		t.Errorf("expected nil metadata, got %v", metadata)
	}

	if string(body) != string(input) {
		t.Errorf("body: got %q, want %q", string(body), string(input))
	}
}

func TestFormatMarkdownFile(t *testing.T) {
	metadata, err := structpb.NewStruct(map[string]any{
		"status":   "todo",
		"priority": "P2",
	})
	if err != nil {
		t.Fatalf("NewStruct: %v", err)
	}

	body := []byte("# My Feature\n\nDescription.\n")

	data, err := FormatMarkdownFile(metadata, body)
	if err != nil {
		t.Fatalf("FormatMarkdownFile: %v", err)
	}

	// Parse it back to verify roundtrip.
	parsedMeta, parsedBody, err := ParseMarkdownFile(data)
	if err != nil {
		t.Fatalf("ParseMarkdownFile roundtrip: %v", err)
	}

	if parsedMeta == nil {
		t.Fatal("roundtrip: expected metadata, got nil")
	}

	if parsedMeta.Fields["status"].GetStringValue() != "todo" {
		t.Errorf("roundtrip status: got %q, want %q", parsedMeta.Fields["status"].GetStringValue(), "todo")
	}
	if parsedMeta.Fields["priority"].GetStringValue() != "P2" {
		t.Errorf("roundtrip priority: got %q, want %q", parsedMeta.Fields["priority"].GetStringValue(), "P2")
	}

	if string(parsedBody) != string(body) {
		t.Errorf("roundtrip body: got %q, want %q", string(parsedBody), string(body))
	}
}

func TestFormatMarkdownFileNilMetadata(t *testing.T) {
	body := []byte("# No metadata\n\nJust body.\n")

	data, err := FormatMarkdownFile(nil, body)
	if err != nil {
		t.Fatalf("FormatMarkdownFile: %v", err)
	}

	if string(data) != string(body) {
		t.Errorf("got %q, want %q", string(data), string(body))
	}
}

func TestStorageReadWrite(t *testing.T) {
	workspace := t.TempDir()
	sp := NewStoragePlugin(workspace)
	ctx := context.Background()

	metadata, err := structpb.NewStruct(map[string]any{
		"status":   "backlog",
		"assignee": "go-architect",
	})
	if err != nil {
		t.Fatalf("NewStruct: %v", err)
	}

	// Write a new file (expected_version=0 means create).
	writeResp, err := sp.Write(ctx, &pluginv1.StorageWriteRequest{
		Path:            "projects/test-app/features/FEAT-001.md",
		Content:         []byte("# Feature 001\n\nDescription.\n"),
		Metadata:        metadata,
		ExpectedVersion: 0,
	})
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if !writeResp.Success {
		t.Fatalf("Write failed: %s", writeResp.Error)
	}
	if writeResp.NewVersion != 1 {
		t.Errorf("new_version: got %d, want 1", writeResp.NewVersion)
	}

	// Read it back.
	readResp, err := sp.Read(ctx, &pluginv1.StorageReadRequest{
		Path: "projects/test-app/features/FEAT-001.md",
	})
	if err != nil {
		t.Fatalf("Read: %v", err)
	}

	if readResp.Metadata == nil {
		t.Fatal("expected metadata in read response")
	}
	if readResp.Metadata.Fields["status"].GetStringValue() != "backlog" {
		t.Errorf("status: got %q, want %q", readResp.Metadata.Fields["status"].GetStringValue(), "backlog")
	}
	if readResp.Metadata.Fields["assignee"].GetStringValue() != "go-architect" {
		t.Errorf("assignee: got %q, want %q", readResp.Metadata.Fields["assignee"].GetStringValue(), "go-architect")
	}
	if string(readResp.Content) != "# Feature 001\n\nDescription.\n" {
		t.Errorf("content: got %q, want %q", string(readResp.Content), "# Feature 001\n\nDescription.\n")
	}
	if readResp.Version != 1 {
		t.Errorf("version: got %d, want 1", readResp.Version)
	}
}

func TestStorageDelete(t *testing.T) {
	workspace := t.TempDir()
	sp := NewStoragePlugin(workspace)
	ctx := context.Background()

	// Write a file first.
	writeResp, err := sp.Write(ctx, &pluginv1.StorageWriteRequest{
		Path:            "projects/test-app/to-delete.md",
		Content:         []byte("# Delete me\n"),
		ExpectedVersion: 0,
	})
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if !writeResp.Success {
		t.Fatalf("Write failed: %s", writeResp.Error)
	}

	// Verify the file is readable.
	_, err = sp.Read(ctx, &pluginv1.StorageReadRequest{
		Path: "projects/test-app/to-delete.md",
	})
	if err != nil {
		t.Fatalf("Read before delete: %v", err)
	}

	// Delete the file.
	delResp, err := sp.Delete(ctx, &pluginv1.StorageDeleteRequest{
		Path: "projects/test-app/to-delete.md",
	})
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if !delResp.Success {
		t.Error("expected delete success=true")
	}

	// Verify the file is gone.
	_, err = sp.Read(ctx, &pluginv1.StorageReadRequest{
		Path: "projects/test-app/to-delete.md",
	})
	if err == nil {
		t.Error("expected error reading deleted file")
	}
}

func TestStorageList(t *testing.T) {
	workspace := t.TempDir()
	sp := NewStoragePlugin(workspace)
	ctx := context.Background()

	// Write multiple files.
	files := []string{
		"projects/app/features/FEAT-001.md",
		"projects/app/features/FEAT-002.md",
		"projects/app/tasks/TASK-001.md",
	}
	for _, f := range files {
		writeResp, err := sp.Write(ctx, &pluginv1.StorageWriteRequest{
			Path:            f,
			Content:         []byte("# " + f + "\n"),
			ExpectedVersion: 0,
		})
		if err != nil {
			t.Fatalf("Write %s: %v", f, err)
		}
		if !writeResp.Success {
			t.Fatalf("Write %s failed: %s", f, writeResp.Error)
		}
	}

	// List features directory.
	listResp, err := sp.List(ctx, &pluginv1.StorageListRequest{
		Prefix:  "projects/app/features/",
		Pattern: "*.md",
	})
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	if len(listResp.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(listResp.Entries))
	}

	// Verify paths.
	paths := make(map[string]bool)
	for _, e := range listResp.Entries {
		paths[e.Path] = true
	}
	if !paths["projects/app/features/FEAT-001.md"] {
		t.Error("missing FEAT-001.md in list")
	}
	if !paths["projects/app/features/FEAT-002.md"] {
		t.Error("missing FEAT-002.md in list")
	}

	// List all with broader prefix.
	listResp2, err := sp.List(ctx, &pluginv1.StorageListRequest{
		Prefix:  "projects/app/",
		Pattern: "*.md",
	})
	if err != nil {
		t.Fatalf("List all: %v", err)
	}
	if len(listResp2.Entries) != 3 {
		t.Fatalf("expected 3 entries for broad list, got %d", len(listResp2.Entries))
	}
}

func TestVersioning(t *testing.T) {
	workspace := t.TempDir()
	sp := NewStoragePlugin(workspace)
	ctx := context.Background()

	path := "projects/test-app/versioned.md"

	// Create (expected_version=0).
	writeResp, err := sp.Write(ctx, &pluginv1.StorageWriteRequest{
		Path:            path,
		Content:         []byte("# Version 1\n"),
		ExpectedVersion: 0,
	})
	if err != nil {
		t.Fatalf("Write v1: %v", err)
	}
	if !writeResp.Success {
		t.Fatalf("Write v1 failed: %s", writeResp.Error)
	}
	if writeResp.NewVersion != 1 {
		t.Errorf("v1: got version %d, want 1", writeResp.NewVersion)
	}

	// Try to create again (expected_version=0) — should fail.
	writeResp2, err := sp.Write(ctx, &pluginv1.StorageWriteRequest{
		Path:            path,
		Content:         []byte("# Duplicate\n"),
		ExpectedVersion: 0,
	})
	if err != nil {
		t.Fatalf("Write duplicate: %v", err)
	}
	if writeResp2.Success {
		t.Error("expected duplicate create to fail")
	}

	// Update with correct version (expected_version=1).
	writeResp3, err := sp.Write(ctx, &pluginv1.StorageWriteRequest{
		Path:            path,
		Content:         []byte("# Version 2\n"),
		ExpectedVersion: 1,
	})
	if err != nil {
		t.Fatalf("Write v2: %v", err)
	}
	if !writeResp3.Success {
		t.Fatalf("Write v2 failed: %s", writeResp3.Error)
	}
	if writeResp3.NewVersion != 2 {
		t.Errorf("v2: got version %d, want 2", writeResp3.NewVersion)
	}

	// Update with wrong version (expected_version=1, current=2) — should fail.
	writeResp4, err := sp.Write(ctx, &pluginv1.StorageWriteRequest{
		Path:            path,
		Content:         []byte("# Version 3 fail\n"),
		ExpectedVersion: 1,
	})
	if err != nil {
		t.Fatalf("Write v3 stale: %v", err)
	}
	if writeResp4.Success {
		t.Error("expected stale version update to fail")
	}

	// Update with correct version (expected_version=2).
	writeResp5, err := sp.Write(ctx, &pluginv1.StorageWriteRequest{
		Path:            path,
		Content:         []byte("# Version 3\n"),
		ExpectedVersion: 2,
	})
	if err != nil {
		t.Fatalf("Write v3: %v", err)
	}
	if !writeResp5.Success {
		t.Fatalf("Write v3 failed: %s", writeResp5.Error)
	}
	if writeResp5.NewVersion != 3 {
		t.Errorf("v3: got version %d, want 3", writeResp5.NewVersion)
	}

	// Read and verify version.
	readResp, err := sp.Read(ctx, &pluginv1.StorageReadRequest{Path: path})
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if readResp.Version != 3 {
		t.Errorf("read version: got %d, want 3", readResp.Version)
	}
	if string(readResp.Content) != "# Version 3\n" {
		t.Errorf("content: got %q, want %q", string(readResp.Content), "# Version 3\n")
	}
}

func TestPathTraversal(t *testing.T) {
	workspace := t.TempDir()
	sp := NewStoragePlugin(workspace)
	ctx := context.Background()

	// Try to write with path traversal.
	writeResp, err := sp.Write(ctx, &pluginv1.StorageWriteRequest{
		Path:            "../../../etc/passwd",
		Content:         []byte("pwned\n"),
		ExpectedVersion: 0,
	})
	if err != nil {
		// Error is acceptable.
		return
	}
	if writeResp.Success {
		t.Error("expected path traversal write to fail")
	}

	// Try to read with path traversal.
	_, err = sp.Read(ctx, &pluginv1.StorageReadRequest{
		Path: "../../etc/passwd",
	})
	if err == nil {
		t.Error("expected path traversal read to fail")
	}
}

func TestDeleteNonexistent(t *testing.T) {
	workspace := t.TempDir()
	sp := NewStoragePlugin(workspace)
	ctx := context.Background()

	_, err := sp.Delete(ctx, &pluginv1.StorageDeleteRequest{
		Path: "projects/nonexistent.md",
	})
	if err == nil {
		t.Error("expected error deleting nonexistent file")
	}
}

func TestDeleteCleansUpVersionSidecar(t *testing.T) {
	workspace := t.TempDir()
	sp := NewStoragePlugin(workspace)
	ctx := context.Background()

	path := "projects/test-app/with-version.md"

	// Write a file (creates version sidecar).
	writeResp, err := sp.Write(ctx, &pluginv1.StorageWriteRequest{
		Path:            path,
		Content:         []byte("# Has version\n"),
		ExpectedVersion: 0,
	})
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if !writeResp.Success {
		t.Fatalf("Write failed: %s", writeResp.Error)
	}
	if writeResp.NewVersion != 1 {
		t.Errorf("expected version 1, got %d", writeResp.NewVersion)
	}

	// Verify the version sidecar exists by reading version.
	readResp, err := sp.Read(ctx, &pluginv1.StorageReadRequest{Path: path})
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if readResp.Version != 1 {
		t.Errorf("expected version 1, got %d", readResp.Version)
	}

	// Delete the file — should also clean up version sidecar.
	delResp, err := sp.Delete(ctx, &pluginv1.StorageDeleteRequest{Path: path})
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if !delResp.Success {
		t.Error("expected delete success=true")
	}

	// Verify file is gone.
	_, err = sp.Read(ctx, &pluginv1.StorageReadRequest{Path: path})
	if err == nil {
		t.Error("expected error reading deleted file")
	}
}

func TestListSkipsVersionSidecars(t *testing.T) {
	workspace := t.TempDir()
	sp := NewStoragePlugin(workspace)
	ctx := context.Background()

	// Write a file with versioning (creates .version sidecar).
	writeResp, err := sp.Write(ctx, &pluginv1.StorageWriteRequest{
		Path:            "projects/test/file.md",
		Content:         []byte("# File\n"),
		ExpectedVersion: 0,
	})
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if !writeResp.Success {
		t.Fatalf("Write failed: %s", writeResp.Error)
	}

	// List should NOT include .version files.
	listResp, err := sp.List(ctx, &pluginv1.StorageListRequest{
		Prefix:  "projects/test/",
		Pattern: "*",
	})
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	for _, entry := range listResp.Entries {
		if strings.HasSuffix(entry.Path, ".version") {
			t.Errorf("list should not include version sidecar: %s", entry.Path)
		}
	}
	if len(listResp.Entries) != 1 {
		t.Errorf("expected 1 entry, got %d", len(listResp.Entries))
	}
}

func TestReadRespectsContextCancellation(t *testing.T) {
	workspace := t.TempDir()
	sp := NewStoragePlugin(workspace)

	// Write a file so there's something to read.
	ctx := context.Background()
	sp.Write(ctx, &pluginv1.StorageWriteRequest{
		Path:            "projects/test/ctx.md",
		Content:         []byte("# Test\n"),
		ExpectedVersion: 0,
	})

	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := sp.Read(cancelledCtx, &pluginv1.StorageReadRequest{Path: "projects/test/ctx.md"})
	if err == nil {
		t.Error("expected error from cancelled context")
	}
	if !strings.Contains(err.Error(), "context cancel") {
		t.Errorf("expected context cancelled error, got: %v", err)
	}
}

func TestWriteRespectsContextCancellation(t *testing.T) {
	workspace := t.TempDir()
	sp := NewStoragePlugin(workspace)

	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := sp.Write(cancelledCtx, &pluginv1.StorageWriteRequest{
		Path:            "projects/test/ctx.md",
		Content:         []byte("# Test\n"),
		ExpectedVersion: 0,
	})
	if err == nil {
		t.Error("expected error from cancelled context")
	}
}

func TestDeleteRespectsContextCancellation(t *testing.T) {
	workspace := t.TempDir()
	sp := NewStoragePlugin(workspace)

	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := sp.Delete(cancelledCtx, &pluginv1.StorageDeleteRequest{Path: "projects/test/ctx.md"})
	if err == nil {
		t.Error("expected error from cancelled context")
	}
}

func TestListRespectsContextCancellation(t *testing.T) {
	workspace := t.TempDir()
	sp := NewStoragePlugin(workspace)

	// Write some files so Walk has something to iterate.
	ctx := context.Background()
	sp.Write(ctx, &pluginv1.StorageWriteRequest{
		Path:            "projects/test/a.md",
		Content:         []byte("# A\n"),
		ExpectedVersion: 0,
	})

	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := sp.List(cancelledCtx, &pluginv1.StorageListRequest{Prefix: "projects/test/"})
	if err == nil {
		t.Error("expected error from cancelled context")
	}
}

func TestEmptyPath(t *testing.T) {
	workspace := t.TempDir()
	sp := NewStoragePlugin(workspace)
	ctx := context.Background()

	_, err := sp.Read(ctx, &pluginv1.StorageReadRequest{Path: ""})
	if err == nil {
		t.Error("expected error for empty path")
	}
}

func TestPathTooLong(t *testing.T) {
	workspace := t.TempDir()
	sp := NewStoragePlugin(workspace)
	ctx := context.Background()

	longPath := strings.Repeat("a/", 2100) // > 4096 chars
	_, err := sp.Read(ctx, &pluginv1.StorageReadRequest{Path: longPath})
	if err == nil {
		t.Error("expected error for path exceeding max length")
	}
	if !strings.Contains(err.Error(), "too long") {
		t.Errorf("expected 'too long' error, got: %v", err)
	}
}

func TestGlobPatternTooLong(t *testing.T) {
	workspace := t.TempDir()
	sp := NewStoragePlugin(workspace)
	ctx := context.Background()

	longPattern := strings.Repeat("*", 300) // > 256 chars
	_, err := sp.List(ctx, &pluginv1.StorageListRequest{
		Prefix:  "projects/test/",
		Pattern: longPattern,
	})
	if err == nil {
		t.Error("expected error for glob pattern exceeding max length")
	}
	if !strings.Contains(err.Error(), "too long") {
		t.Errorf("expected 'too long' error, got: %v", err)
	}
}

func TestInvalidGlobPattern(t *testing.T) {
	workspace := t.TempDir()
	sp := NewStoragePlugin(workspace)
	ctx := context.Background()

	_, err := sp.List(ctx, &pluginv1.StorageListRequest{
		Prefix:  "projects/test/",
		Pattern: "[invalid",
	})
	if err == nil {
		t.Error("expected error for invalid glob pattern")
	}
	if !strings.Contains(err.Error(), "invalid glob pattern") {
		t.Errorf("expected 'invalid glob pattern' error, got: %v", err)
	}
}
