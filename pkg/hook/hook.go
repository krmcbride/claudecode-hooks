// Package hook provides types and functions for Claude Code hooks.
package hook

import (
	"encoding/json"
	"os"
)

// PreToolUseInput represents the JSON input from Claude Code PreToolUse hooks.
// This is specifically for Bash tool hooks that need to inspect commands.
type PreToolUseInput struct {
	ToolName  string `json:"tool_name"`
	ToolInput struct {
		Command string `json:"command"`
	} `json:"tool_input"`
}

// PostToolUseInput represents the JSON input from Claude Code PostToolUse hooks.
//
// NOTE: This is a minimal struct containing only the fields we actually use.
// The actual JSON payloads contain many more fields that vary by tool type.
//
// To discover the full structure for each tool, use the hook-logger:
//  1. Build: make build
//  2. Configure in .claude/settings.json with matcher ".*"
//  3. Redirect output: command: "/path/to/hook-logger >> /path/to/log.txt"
//  4. Use various Claude Code tools and inspect the captured payloads
//
// Full payload structure (not all fields are decoded):
// - session_id, transcript_path, cwd, hook_event_name
// - tool_input varies by tool:
//   - Edit/MultiEdit/Write: file_path (we only use this)
//   - Edit: old_string, new_string
//   - MultiEdit: edits array with old_string, new_string
//   - Write: content
//   - Bash: command
//
// - tool_response varies by tool and contains the results
//
// See docs/tool-hook-inputs.md for documented examples.
type PostToolUseInput struct {
	ToolName  string `json:"tool_name"`
	ToolInput struct {
		FilePath string `json:"file_path"`
	} `json:"tool_input"`
	ToolResponse map[string]any `json:"tool_response"`
}

// PostToolUseResponse represents the JSON response for PostToolUse hooks.
// Used to block further actions after a tool has been executed.
type PostToolUseResponse struct {
	Decision string `json:"decision,omitempty"` // "block" or omit for allow
	Reason   string `json:"reason,omitempty"`   // Optional explanation when blocking
}

// ReadPreToolUseInput reads and parses PreToolUse hook input from stdin.
// This is typically used by hooks that need to inspect Bash commands.
func ReadPreToolUseInput() (*PreToolUseInput, error) {
	var input PreToolUseInput
	decoder := json.NewDecoder(os.Stdin)
	if err := decoder.Decode(&input); err != nil {
		return nil, err
	}
	return &input, nil
}

// ReadPostToolUseInput reads and parses PostToolUse hook input from stdin
func ReadPostToolUseInput() (*PostToolUseInput, error) {
	var input PostToolUseInput
	decoder := json.NewDecoder(os.Stdin)
	if err := decoder.Decode(&input); err != nil {
		return nil, err
	}
	return &input, nil
}

// BlockPreToolUse blocks the tool execution with an error message (PreToolUse hooks).
// Exit code 2 tells Claude Code to block the tool and show stderr output.
func BlockPreToolUse(message string, issues []string) {
	_, _ = os.Stderr.WriteString("🚫 BLOCKED: " + message + "\n") //nolint:errcheck // Error writing to stderr is not actionable in blocking function
	for _, issue := range issues {
		_, _ = os.Stderr.WriteString("Issue: " + issue + "\n") //nolint:errcheck // Error writing to stderr is not actionable in blocking function
	}
	os.Exit(2) // Block execution
}

// AllowPreToolUse allows the tool to proceed (PreToolUse hooks).
func AllowPreToolUse() {
	os.Exit(0)
}

// BlockPostToolUse blocks further actions with a JSON response
func BlockPostToolUse(reason string) {
	response := PostToolUseResponse{
		Decision: "block",
		Reason:   reason,
	}
	encoder := json.NewEncoder(os.Stdout)
	if err := encoder.Encode(response); err != nil {
		_, _ = os.Stderr.WriteString("Error encoding block response: " + err.Error() + "\n") //nolint:errcheck
	}
	os.Exit(0)
}

// AllowPostToolUse allows the action to proceed (PostToolUse)
func AllowPostToolUse() {
	os.Exit(0)
}
