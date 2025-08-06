# Claude Code Tool Hook Input Reference

This document describes the JSON input structure that each Claude Code tool sends to hooks. Since this is not comprehensively documented in the official Claude Code documentation, these structures are derived from observed behavior and testing.

## Common Structure

All hook inputs share this base structure:

```json
{
  "session_id": "unique-session-id",
  "transcript_path": "/path/to/transcript",
  "cwd": "/current/working/directory",
  "hook_event_name": "PreToolUse|PostToolUse",
  "tool_name": "ToolName",
  "tool_input": {
    /* tool-specific fields */
  },
  "tool_response": {
    /* only in PostToolUse */
  }
}
```

## Tool-Specific Inputs

### Edit Tool

Used for modifying existing files.

```json
{
  "tool_name": "Edit",
  "tool_input": {
    "file_path": "/path/to/file.go",
    "old_string": "original content",
    "new_string": "replacement content"
  },
  "tool_response": {
    "filePath": "/path/to/file.go",
    "success": true
  }
}
```

### MultiEdit Tool

Used for multiple edits to one or more files.

```json
{
  "tool_name": "MultiEdit",
  "tool_input": {
    "file_path": "/path/to/primary/file.go",
    "edits": [
      {
        "file_path": "/path/to/file1.go",
        "old_string": "original1",
        "new_string": "replacement1"
      },
      {
        "file_path": "/path/to/file2.go",
        "old_string": "original2",
        "new_string": "replacement2"
      }
    ]
  },
  "tool_response": {
    "success": true
  }
}
```

### Write Tool

Used for creating new files or overwriting existing ones.

```json
{
  "tool_name": "Write",
  "tool_input": {
    "file_path": "/path/to/new/file.go",
    "content": "package main\n\nfunc main() {\n\t// New file content\n}"
  },
  "tool_response": {
    "filePath": "/path/to/new/file.go",
    "success": true
  }
}
```

### Bash Tool

Used for executing shell commands.

```json
{
  "tool_name": "Bash",
  "tool_input": {
    "command": "go test ./..."
  },
  "tool_response": {
    "output": "ok  \tgithub.com/example/project\t0.123s",
    "exit_code": 0,
    "success": true
  }
}
```

### Read Tool

Used for reading file contents.

```json
{
  "tool_name": "Read",
  "tool_input": {
    "file_path": "/path/to/file.go",
    "offset": 0,
    "limit": 100
  },
  "tool_response": {
    "content": "file contents here...",
    "success": true
  }
}
```

### Grep Tool

Used for searching file contents.

```json
{
  "tool_name": "Grep",
  "tool_input": {
    "pattern": "func.*Test",
    "path": "/path/to/search",
    "glob": "*.go"
  },
  "tool_response": {
    "matches": ["file1.go:10:func TestExample(t *testing.T) {"],
    "success": true
  }
}
```

### Glob Tool

Used for finding files by pattern.

```json
{
  "tool_name": "Glob",
  "tool_input": {
    "pattern": "**/*.go",
    "path": "/path/to/search"
  },
  "tool_response": {
    "files": ["/path/to/file1.go", "/path/to/file2.go"],
    "success": true
  }
}
```

### LS Tool

Used for listing directory contents.

```json
{
  "tool_name": "LS",
  "tool_input": {
    "path": "/path/to/directory",
    "ignore": [".git", "node_modules"]
  },
  "tool_response": {
    "entries": ["file1.go", "file2.go", "subdir/"],
    "success": true
  }
}
```

## Notes

1. **Field Availability**: Not all fields may be present in every hook call. Use defensive programming when accessing fields.

2. **PostToolUse vs PreToolUse**:

   - PreToolUse hooks receive proposed `tool_input` before execution
   - PostToolUse hooks receive both `tool_input` and `tool_response` after execution

3. **Success Field**: The `success` field in `tool_response` indicates whether the tool executed successfully.

4. **File Paths**: File paths are typically absolute paths.

5. **Optional Fields**: Fields marked with `omitempty` in Go structs may not appear in the JSON if they're empty.

## Working with Hook Inputs

When writing hooks, it's recommended to:

1. Parse the JSON input carefully
2. Check for required fields before accessing them
3. Handle missing or malformed data gracefully
4. Log unexpected input structures for debugging

Example bash hook script:

```bash
#!/bin/bash
# Read JSON from stdin
input=$(cat)

# Extract tool name
tool_name=$(echo "$input" | jq -r '.tool_name')

# Extract file path (handling different tools)
if [[ "$tool_name" == "Edit" ]] || [[ "$tool_name" == "Write" ]]; then
    file_path=$(echo "$input" | jq -r '.tool_input.file_path')
elif [[ "$tool_name" == "MultiEdit" ]]; then
    # Get all file paths from edits
    file_paths=$(echo "$input" | jq -r '.tool_input.edits[].file_path')
fi
```

## Disclaimer

This documentation is based on observed behavior and reverse engineering. The actual implementation may vary, and Anthropic may change these structures in future versions. Always test your hooks thoroughly and be prepared to handle variations in input structure.

For the most up-to-date and authoritative information, refer to:

- [Official Claude Code Hooks Documentation](https://docs.anthropic.com/en/docs/claude-code/hooks)
- [Claude Code Settings Reference](https://docs.anthropic.com/en/docs/claude-code/settings)

