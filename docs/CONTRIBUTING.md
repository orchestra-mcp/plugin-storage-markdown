# Contributing to plugin-storage-markdown

## Prerequisites

- Go 1.23+
- `gofmt`, `go vet`

## Development Setup

```bash
git clone https://github.com/orchestra-mcp/plugin-storage-markdown.git
cd plugin-storage-markdown
go mod download
go build ./cmd/...
```

## Running Locally

```bash
go build -o storage-markdown ./cmd/
./storage-markdown --workspace=. --listen-addr=localhost:0 --certs-dir=~/.orchestra/certs
```

The plugin prints `READY <addr>` to stderr when it is ready to accept QUIC connections.

## Running Tests

```bash
go test ./...
```

Tests use temporary directories and do not require a running orchestrator.

## Code Style

- Run `gofmt` on all files.
- Run `go vet ./...` before committing.
- All exported functions and types must have doc comments.
- Error handling: wrap errors with context via `fmt.Errorf`.

## Key Implementation Details

- **Thread safety**: A `sync.Mutex` guards concurrent file reads and writes to prevent race conditions during version checking.
- **Path safety**: All incoming paths are validated to prevent directory traversal attacks.
- **Version sidecar**: Stored as a plain text file alongside each managed file.
- **No in-memory cache**: Every read goes to disk. This keeps the plugin stateless and avoids consistency issues.

## Pull Request Process

1. Fork the repository and create a feature branch from `main`.
2. Write or update tests for your changes. Include edge cases (empty files, missing directories, CAS conflicts).
3. Run `go test ./...` and `go vet ./...`.
4. Update `docs/STORAGE_FORMAT.md` if changing the file format.

## Related Repositories

- [orchestra-mcp/proto](https://github.com/orchestra-mcp/proto) -- Protobuf schema
- [orchestra-mcp/sdk-go](https://github.com/orchestra-mcp/sdk-go) -- Go Plugin SDK
- [orchestra-mcp/orchestrator](https://github.com/orchestra-mcp/orchestrator) -- Central hub
- [orchestra-mcp](https://github.com/orchestra-mcp) -- Organization home
