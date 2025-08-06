# Claude Code Hooks

Security and automation hooks for [Claude Code](https://claude.ai/code) that prevent dangerous command execution and automatically format code after edits.

## Features

### 🛡️ bash-block: Generic Command Blocker

- **Configurable Security**: Block any command with flexible pattern matching
- **Advanced Detection**: Detects obfuscated commands, shell escaping, and complex execution patterns
- **Multiple Security Levels**: Choose between speed and thoroughness
- **Flexible Rules**: Support for blocked patterns and allow exceptions

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
            "command": "/path/to/krmcbride-bash-block -cmd=git -patterns=push -desc=\"Block git push\""
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
bash-block -cmd=COMMAND -patterns=PATTERNS [OPTIONS]
```

**Required Flags:**

- `-cmd` - Primary command to monitor (e.g., "git", "aws", "kubectl")
- `-patterns` - Comma-separated blocked patterns (e.g., "push", "delete-bucket")

**Optional Flags:**

- `-security` - Security level: `basic`, `advanced` (default), `paranoid`
- `-allow` - Comma-separated exception patterns
- `-desc` - Human-readable description for logging
- `-max-recursion` - Maximum analysis depth (default: 10)

**Security Levels:**

- **basic**: Fast pattern matching only
- **advanced**: + obfuscation detection (recommended)
- **paranoid**: + blocks all dynamic content (most secure)

**Examples:**

```bash
# Block git push with advanced security
bash-block -cmd=git -patterns=push

# Block dangerous AWS operations with basic security (faster)
bash-block -cmd=aws -patterns="delete-bucket,terminate-instances" -security=basic

# Block kubectl delete with exceptions
bash-block -cmd=kubectl -patterns=delete -allow="delete pod"
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
            "command": "/path/to/bash-block -cmd=git -patterns=push -desc=\"Block git push\""
          },
          {
            "type": "command",
            "command": "/path/to/bash-block -cmd=git -patterns=\"reset --hard\" -desc=\"Block git reset --hard\""
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

For maximum security, use `paranoid` mode, but note it may block legitimate dynamic commands.

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
├── detector/       # Command detection engine
├── hook/          # Claude Code hook utilities
└── shellparse/    # Shell command parsing
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
