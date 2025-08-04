# Claude Code Hooks Documentation

## Overview

Claude Code hooks are a configuration mechanism that allows users to execute custom shell commands in response to specific events during Claude Code's operation. They provide a way to extend and customize Claude's behavior.

## Configuration

Hooks are configured in settings files like `~/.claude/settings.json` using the following structure:

```json
{
  "hooks": {
    "EventType": [
      {
        "matcher": "ToolName|Pattern",
        "hooks": [
          {
            "type": "command",
            "command": "/path/to/script.sh"
          }
        ]
      }
    ]
  }
}
```

## Hook Events

### PreToolUse
- **When**: Runs before a tool is executed
- **Use case**: Validation, permission checks, command blocking
- **Input**: Tool name, parameters, session context

### PostToolUse
- **When**: Runs after a tool completes successfully
- **Use case**: Cleanup, notifications, logging, validation

### UserPromptSubmit
- **When**: Runs when a user submits a prompt
- **Use case**: Input filtering, logging, preprocessing

### Notification
- **When**: Runs for system notifications
- **Use case**: Custom notification handling

### Stop/SubagentStop
- **When**: Runs when Claude finishes responding
- **Use case**: Session cleanup, final validations

### SessionStart
- **When**: Runs when starting a new session
- **Use case**: Environment setup, initialization

## Hook Input/Output

### Input
Hooks receive JSON input via stdin containing:
- Session ID and context
- Event-specific data (tool name, parameters, etc.)
- User information
- Timestamp

### Output
Hooks can control Claude's behavior by:
- Returning specific JSON responses
- Using exit codes (0 = success, non-zero = block/error)
- Writing to stdout/stderr for logging

## Security Considerations

⚠️ **Important**: Hooks execute shell commands with the same permissions as Claude Code. Configure them carefully to prevent security risks.

## Example Use Cases

1. **Command Blocking**: Prevent execution of dangerous commands
2. **Validation**: Check code quality before file operations
3. **Logging**: Track tool usage and changes
4. **Integration**: Connect with external systems
5. **Notifications**: Alert on specific events

## Tool Matchers

Matchers use regex patterns to target specific tools:
- `Bash`: Match Bash tool usage
- `Write|Edit`: Match file modification tools
- `.*`: Match all tools