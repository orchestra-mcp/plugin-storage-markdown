# Orchestra Storage Markdown Plugin

File-based storage plugin that persists structured metadata and Markdown content to disk.

## Install

```bash
go get github.com/orchestra-mcp/plugin-storage-markdown
```

## Usage

```bash
# Build
go build -o bin/storage-markdown ./cmd/

# Run (started automatically by the orchestrator)
bin/storage-markdown --workspace /path/to/project --orchestrator-addr localhost:9100
```

## Storage Format

Each file uses standard YAML frontmatter for metadata with a Markdown body:

```markdown
---
id: FEAT-001
priority: high
status: in-progress
---

# Feature Title

Description and content goes here.
```

The YAML frontmatter block (`---` delimiters) stores structured fields. Everything after the closing `---` and blank line is the Markdown body.

## Supported Operations

| Operation | Description |
|-----------|-------------|
| **StorageRead** | Read metadata + body from a file path |
| **StorageWrite** | Write metadata + body with optimistic concurrency (CAS) |
| **StorageDelete** | Delete a file |
| **StorageList** | List files by prefix and glob pattern |

## Related Packages

| Package | Description |
|---------|-------------|
| [sdk-go](https://github.com/orchestra-mcp/sdk-go) | Plugin SDK this plugin is built on |
| [orchestrator](https://github.com/orchestra-mcp/orchestrator) | Central hub that loads this plugin |
| [plugin-tools-features](https://github.com/orchestra-mcp/plugin-tools-features) | Feature tools that use this storage |

## License

[MIT](LICENSE)
