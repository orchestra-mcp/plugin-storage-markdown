# Storage Format

## Overview

The `storage.markdown` plugin stores data as files on disk under `{workspace}/.projects/`. Each file has an optional YAML frontmatter header followed by a Markdown body. A companion `.version` sidecar file tracks the optimistic concurrency version.

## File Format

```markdown
---
key: value
priority: P1
status: backlog
---

# Feature Title

The markdown body goes here.
```

### Rules

1. The frontmatter block must be at the very start of the file.
2. It begins with `---` on its own line and ends with `---` on its own line.
3. The content between the delimiters must be valid YAML.
4. A blank line separates the closing `---` from the body.
5. If the file has no metadata, the frontmatter block is omitted entirely and the file is pure Markdown.
6. Keys are sorted alphabetically for deterministic output.

### Parsing

The reader (`ParseMarkdownFile`) handles three cases:

| Case | Behavior |
|---|---|
| File starts with `---` | Extract YAML, return metadata + body |
| File does not start with `---` | Return `nil` metadata, entire content as body |
| Malformed frontmatter (missing closing `---`) | Return error |

### Writing

The writer (`FormatMarkdownFile`) serializes metadata + body:

- If metadata is non-nil and has fields, write the YAML frontmatter block followed by a blank line, then the body.
- If metadata is nil or empty, write only the body.

## Version Sidecar

Each file has a companion version file at `{path}.version` containing a single integer:

```
.projects/my-app/features/FEAT-ABC.md           # The file
.projects/my-app/features/FEAT-ABC.md.version    # Contains: 3
```

- Version starts at 0 (file does not exist yet).
- Each successful write increments the version by 1.
- The `.version` file is deleted when the main file is deleted.

## Optimistic Concurrency (CAS)

The `StorageWriteRequest.expected_version` field controls concurrency:

| `expected_version` | Behavior |
|---|---|
| `0` | **Create**: Fails if the file already exists |
| `> 0` | **Update**: Fails if the current version does not match |

This prevents lost updates when multiple agents write to the same file.

## Directory Structure

```
{workspace}/
  .projects/
    {project-slug}/
      project.json                    # Project metadata
      features/
        FEAT-ABC.md                   # Feature file
        FEAT-ABC.md.version           # Version sidecar
        FEAT-DEF.md
        FEAT-DEF.md.version
      wip.json                        # WIP limits config (optional)
      wip.json.version
```

## Path Resolution

Storage paths are relative to `{workspace}/.projects/`. The plugin resolves them as:

```
input path:    "my-app/features/FEAT-ABC.md"
resolved:      {workspace}/.projects/my-app/features/FEAT-ABC.md
```

Path traversal (`..`) is rejected to prevent escaping the workspace.

## Example: Feature File

```markdown
---
assignee: go-architect
created_at: "2025-01-15T10:00:00Z"
id: FEAT-ABC
labels:
  - auth
  - frontend
priority: P1
project_id: my-app
status: in-progress
title: Add login page
updated_at: "2025-01-16T14:30:00Z"
version: 3
---

# Add login page

Implement a login page with email/password authentication.

## Note (2025-01-16T14:30:00Z)

Started implementation. Using JWT for session management.

---
**todo -> in-progress**: Beginning implementation
```

## List Operation

The `StorageList` operation walks the directory tree under the resolved prefix path, matching files against the glob pattern (default: `*.md`). It skips `.version` sidecar files.

Each returned `StorageEntry` contains:
- `path` -- Relative to `{workspace}/.projects/`
- `size` -- File size in bytes
- `version` -- From the sidecar file (0 if no sidecar)
- `modified_at` -- File modification timestamp
