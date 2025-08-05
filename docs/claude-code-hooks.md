# Claude Code Hooks Documentation

## Overview

Claude Code hooks are a powerful configuration mechanism that allows users to execute custom shell commands in response to specific events during Claude Code's operation. They provide a way to extend and customize Claude's behavior with validation, automation, and integration capabilities.

For complete official documentation, see: [Claude Code Hooks Documentation](https://docs.anthropic.com/en/docs/claude-code/hooks)

## Configuration

Hooks are configured in settings files like `~/.claude/settings.json` or `<project>/.claude/settings.json` using the following structure:

```json
{
  "hooks": {
    "EventType": [
      {
        "matcher": "ToolName|Pattern",
        "hooks": [
          {
            "type": "command",
            "command": "/path/to/script.sh",
            "timeout": 30000
          }
        ]
      }
    ]
  }
}
```

**Configuration Tips:**

- Use absolute paths for hook commands
- Project-specific settings override user settings
- Environment variable `$CLAUDE_PROJECT_DIR` is available in hooks
- Timeout is optional (defaults to reasonable limits)

## Hook Events

### PreToolUse

- **When**: Runs before a tool is executed
- **Use case**: Validation, permission checks, command blocking
- **Input**: Tool name, parameters, session context
- **Control**: Can block tool execution (exit code 2)

### PostToolUse

- **When**: Runs after a tool completes successfully
- **Use case**: Cleanup, notifications, logging, formatting
- **Input**: Tool results, file paths, success status
- **Control**: Can inject additional context

### UserPromptSubmit

- **When**: Runs when a user submits a prompt
- **Use case**: Input filtering, logging, preprocessing
- **Input**: User prompt, session context
- **Control**: Can modify or block prompts

### Notification

- **When**: Runs for system notifications (tool permissions, idle states)
- **Use case**: Custom notification handling, alerting
- **Input**: Notification type and details

### Stop/SubagentStop

- **When**: Runs when Claude (or subagent) finishes responding
- **Use case**: Session cleanup, final validations, reporting
- **Input**: Response completion context

### SessionStart

- **When**: Runs when starting or resuming a session
- **Use case**: Environment setup, initialization, project validation
- **Input**: Session metadata

### PreCompact

- **When**: Runs before context compaction
- **Use case**: Context preservation, logging
- **Input**: Context size and compaction details

## Hook Input/Output

### Input Format

Hooks receive JSON input via stdin containing:

```json
{
  "session_id": "unique-session-id",
  "transcript_path": "/path/to/transcript",
  "cwd": "/current/working/directory",
  "hook_event_name": "PreToolUse",
  "tool_name": "Edit",
  "tool_input": {
    "file_path": "example.go",
    "content": "package main..."
  },
  "tool_response": {
    "success": true
  }
}
```

### Output Control

Hooks can control Claude's behavior through:

**Exit Codes:**

- `0`: Success, continue normally
- `2`: Block the action
- Other non-zero: Error (may block depending on context)

**JSON Response (advanced):**

```json
{
  "decision": "block",
  "reason": "Command not allowed in production",
  "additional_context": "Consider using staging environment",
  "inject_context": "Additional context for Claude"
}
```

## Tool Matchers

Matchers use regex patterns to target specific tools:

- `Bash`: Match Bash tool usage
- `Edit|MultiEdit|Write`: Match file modification tools
- `Read|Glob|Grep`: Match file reading tools
- `.*`: Match all tools (use carefully)

**Common Patterns:**

```json
{
  "matcher": "Bash",
  "hooks": [{ "type": "command", "command": "/path/to/bash-validator" }]
}
```

## Security Considerations

⚠️ **Critical Security Notes:**

1. **Execution Context**: Hooks execute with the same permissions as Claude Code
2. **Input Validation**: Always validate hook inputs to prevent injection attacks
3. **Path Security**: Use absolute paths and validate file operations
4. **Command Injection**: Be careful with dynamic command construction
5. **Sensitive Data**: Hooks may access file contents and user inputs

**Best Practices:**

- Use dedicated hook user accounts when possible
- Validate all inputs in hook scripts
- Log hook activities for audit trails
- Test hooks thoroughly before production use
- Implement proper error handling

## Advanced Features

### Environment Variables

- `$CLAUDE_PROJECT_DIR`: Absolute path to project directory
- Standard shell environment available

### Context Injection

Hooks can provide additional context to Claude:

```json
{
  "inject_context": "Build failed with error: connection timeout"
}
```

### Conditional Execution

Use matchers and script logic for conditional hook execution:

```bash
#!/bin/bash
if [[ "$TOOL_NAME" == "Bash" ]]; then
  # Only validate bash commands
  ./validate-command.sh
fi
```

## Example Use Cases

### 1. Command Blocking (Security)

```json
{
  "PreToolUse": [
    {
      "matcher": "Bash",
      "hooks": [
        {
          "type": "command",
          "command": "/usr/local/bin/bash-block -cmd=git -patterns=push"
        }
      ]
    }
  ]
}
```

### 2. Automatic Code Formatting

```json
{
  "PostToolUse": [
    {
      "matcher": "Edit|MultiEdit|Write",
      "hooks": [
        {
          "type": "command",
          "command": "/usr/local/bin/file-format -cmd=goimports -ext=.go"
        }
      ]
    }
  ]
}
```

### 3. Project Validation

```json
{
  "SessionStart": [
    {
      "matcher": ".*",
      "hooks": [
        {
          "type": "command",
          "command": "/usr/local/bin/validate-project"
        }
      ]
    }
  ]
}
```

### 4. Build Integration

```json
{
  "PostToolUse": [
    {
      "matcher": "Edit|MultiEdit",
      "hooks": [
        {
          "type": "command",
          "command": "/usr/local/bin/trigger-build",
          "timeout": 60000
        }
      ]
    }
  ]
}
```

### 5. Audit Logging

```json
{
  "UserPromptSubmit": [
    {
      "matcher": ".*",
      "hooks": [
        {
          "type": "command",
          "command": "/usr/local/bin/audit-log"
        }
      ]
    }
  ]
}
```

## Troubleshooting

### Common Issues

1. **Permission Errors**: Ensure hook scripts are executable
2. **Path Issues**: Use absolute paths for commands
3. **Timeout Errors**: Increase timeout for long-running hooks
4. **JSON Parsing**: Validate hook output format

### Debugging

- Check Claude Code logs for hook execution details
- Test hooks independently with sample JSON input
- Use `set -x` in bash scripts for detailed tracing
- Validate hook output with JSON parsers

### Performance Considerations

- Keep hooks lightweight for responsive experience
- Use appropriate timeouts
- Consider async operations for expensive hooks
- Cache results when possible

## Related Documentation

- [Official Claude Code Hooks Documentation](https://docs.anthropic.com/en/docs/claude-code/hooks)
- [Claude Code Settings Configuration](https://docs.anthropic.com/en/docs/claude-code/settings)
- [Claude Code Tool Reference](https://docs.anthropic.com/en/docs/claude-code/cli-reference)
- [Claude Code IDE Integrations](https://docs.anthropic.com/en/docs/claude-code/ide-integrations)

---

_This project provides practical examples of Claude Code hooks for security and automation. See the [main README](../README.md) for hook implementations._

