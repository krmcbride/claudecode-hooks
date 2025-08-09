# Claude Code Hooks

Security and automation hooks for [Claude Code](https://claude.ai/code) that prevent dangerous command execution and automatically format code after edits.

## Features

### 🛡️ bash-block: Generic Command Blocker

- **Maximum Security**: Provides additional defense-in-depth on top of Claude Code's built-in permissions
- **Advanced Detection**: Detects obfuscated commands, shell escaping, and complex execution patterns
- **Always Paranoid**: Uses maximum security checks to prevent any bypass attempts
- **Flexible Rules**: Support for multiple commands with pattern matching and wildcards

### 🎨 file-format: Automatic Code Formatting

- **Post-Edit Formatting**: Automatically format files after Claude edits or creates them
- **Extension Filtering**: Only format files with specified extensions
- **Configurable Commands**: Use any formatter (goimports, prettier, black, etc.)
- **Failure Handling**: Optional blocking on format failures

## Quick Start

### Installation

1. **Build the hooks:**

   ```bash
   make build
   ```

2. **Install hooks:**

   ```bash
   # Install to project directory (recommended for project-specific settings)
   make install

   # OR install to user directory (global settings)
   make install-user
   ```

### Configuration

Add hooks to your Claude Code settings.json:

```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash",
        "hooks": [
          {
            "type": "command",
            "command": "/path/to/krmcbride-bash-block -cmd='git push'"
          }
        ]
      }
    ],
    "PostToolUse": [
      {
        "matcher": "Edit|MultiEdit|Write",
        "hooks": [
          {
            "type": "command",
            "command": "/path/to/krmcbride-file-format -cmd=\"goimports -w {FILEPATH}\" -ext=.go"
          }
        ]
      }
    ]
  }
}
```

## Hook Reference

### bash-block

Block dangerous shell commands with sophisticated detection capabilities.

**Usage:**

```bash
bash-block -cmd COMMAND_SPEC [-cmd COMMAND_SPEC ...] [OPTIONS]
```

**Required Flags:**

- `-cmd` - Command and optional patterns to block (can be specified multiple times)
  - Format: `"command [pattern1] [pattern2] ..."`
  - Just the command blocks all subcommands
  - With patterns, blocks only matching subcommands
  - Supports wildcards: `*` blocks all, `delete-*` blocks prefixes

**Optional Flags:**

- `-max-recursion` - Maximum analysis depth (default: 10)
- `-help` - Show help message

**Security Features:**

- **Maximum Security**: Always uses the most comprehensive detection available
- **Obfuscation Detection**: Detects base64, hex, and character escaping attempts
- **Dynamic Content Blocking**: Blocks all variable substitutions and command substitutions
- **Recursive Analysis**: Analyzes nested commands (sh -c, eval, source)
- **Defense-in-Depth**: Provides additional security layer beyond Claude Code's built-in permissions

**Examples:**

```bash
# Block all git commands
bash-block -cmd git

# Block only git push
bash-block -cmd "git push"

# Block multiple git subcommands
bash-block -cmd "git push pull force-push"

# Block dangerous AWS operations with wildcards
bash-block -cmd "aws delete-* terminate-*"

# Multiple command rules
bash-block -cmd "git push" -cmd "aws delete-*" -cmd kubectl
```

### file-format

Automatically format files after Claude edits them.

**Usage:**

```bash
file-format -cmd=FORMAT_COMMAND -ext=EXTENSIONS [OPTIONS]
```

**Required Flags:**

- `-cmd` - Format command to execute with optional `{FILEPATH}` placeholder
  - Use `{FILEPATH}` to specify where the file path should be inserted
  - If no placeholder is used, the file path is appended to the command
- `-ext` - Comma-separated file extensions to process (e.g., ".go", ".js,.ts,.jsx,.tsx")

**Optional Flags:**

- `-block` - Block execution if formatting fails
- `-help` - Show help message

**Examples:**

```bash
# Format Go files with goimports (using placeholder)
file-format -cmd="goimports -w {FILEPATH}" -ext=.go

# Format with make command (using placeholder)
file-format -cmd="make fmt-file FILE={FILEPATH}" -ext=.go

# Format TypeScript files with prettier, blocking on failure
file-format -cmd="prettier --write {FILEPATH}" -ext=".ts,.tsx,.js,.jsx" -block

# Format Python files with black (legacy syntax - appends filepath)
file-format -cmd="black --quiet" -ext=.py

# Complex command with multiple flags
file-format -cmd="rustfmt --edition 2021 --config-path .rustfmt.toml {FILEPATH}" -ext=.rs
```

## Advanced Usage

### Multiple Instances

Configure multiple instances of the same hook with different settings:

```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash",
        "hooks": [
          {
            "type": "command",
            "command": "/path/to/bash-block -cmd='git push'"
          },
          {
            "type": "command",
            "command": "/path/to/bash-block -cmd='git reset' -cmd='git force-push'"
          }
        ]
      }
    ]
  }
}
```

### Security Considerations

The `bash-block` hook detects sophisticated bypass attempts including:

- Obfuscated commands: `gi"t pu"sh`, `g'i't p'u's'h`
- Shell escaping: `gi\t pu\sh`
- Command substitution: `$(echo git) push`
- Shell interpreters: `sh -c 'git push'`
- Execution wrappers: `xargs git push`

The tool always operates at maximum security to provide robust defense-in-depth protection.

## Development

### Prerequisites

- Go 1.24.3+
- Make

### Development Commands

```bash
# Full development workflow
make dev

# Run tests
make test

# Format code
make fmt

# Run linter
make lint

# Build specific hook
make build-bash-block
```

### Project Structure

```
cmd/
├── bash-block/     # Generic command blocker
└── file-format/    # File formatter

pkg/
├── detector/       # Command detection engine with shell parsing
├── hook/          # Claude Code hook utilities
└── utils/         # Shared utility functions
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run `make check` to ensure tests pass
5. Submit a pull request

## License

This project is open source. See the license file for details.

## Support

For issues and questions:

- Review the built-in help: `bash-block -help` or `file-format -help`
- Check the [Claude Code documentation](https://docs.anthropic.com/en/docs/claude-code)
- Open an issue in this repository
