// Package hook provides types and functions for Claude Code hooks.
package hook

import (
	"encoding/json"
	"os"
)

// HookInput represents the JSON input from Claude Code PreToolUse hooks (legacy)
type HookInput struct {
	ToolName  string `json:"tool_name"`
	ToolInput struct {
		Command string `json:"command"`
	} `json:"tool_input"`
}

// PostToolUseInput represents the JSON input from Claude Code PostToolUse hooks.
//
// NOTE: This is a simplified struct that covers common fields across different tools.
// The actual JSON structure varies significantly between tools (Edit, Write, Bash, etc).
// Each tool sends different fields in tool_input and tool_response.
//
// To discover the exact structure for each tool, use the hook-logger:
//  1. Build: make build
//  2. Configure in .claude/settings.json with matcher ".*"
//  3. Redirect output: command: "/path/to/hook-logger >> /path/to/log.txt"
//  4. Use various Claude Code tools and inspect the captured payloads
//
// Common variations:
// - Edit/MultiEdit: Contains old_string, new_string, replace_all in tool_input
// - Write: Contains content in tool_input
// - Bash: Contains command in tool_input
// - Read: Returns file content in tool_response
//
// See docs/tool-hook-inputs.md for documented examples.
type PostToolUseInput struct {
	SessionID      string `json:"session_id"`
	TranscriptPath string `json:"transcript_path"`
	CWD            string `json:"cwd"`
	HookEventName  string `json:"hook_event_name"`
	ToolName       string `json:"tool_name"`
	ToolInput      struct {
		FilePath string `json:"file_path"`
		Content  string `json:"content,omitempty"`
		Edits    []struct {
			FilePath string `json:"file_path"`
		} `json:"edits,omitempty"`
	} `json:"tool_input"`
	ToolResponse struct {
		FilePath string `json:"filePath,omitempty"`
		Success  bool   `json:"success"`
	} `json:"tool_response"`
}

// HookResponse represents the response that can be returned to Claude Code
type HookResponse struct {
	Decision string `json:"decision,omitempty"` // "block" or omit for allow
	Reason   string `json:"reason,omitempty"`   // Optional explanation when blocking
}

// ReadHookInput reads and parses the hook input from stdin (PreToolUse)
func ReadHookInput() (*HookInput, error) {
	var input HookInput
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

// BlockExecution blocks the command execution with an error message
func BlockExecution(message string, issues []string) {
	_, _ = os.Stderr.WriteString("🚫 BLOCKED: " + message + "\n") //nolint:errcheck // Error writing to stderr is not actionable in blocking function
	for _, issue := range issues {
		_, _ = os.Stderr.WriteString("Issue: " + issue + "\n") //nolint:errcheck // Error writing to stderr is not actionable in blocking function
	}
	os.Exit(2) // Block execution
}

// AllowExecution allows the command to proceed
func AllowExecution() {
	os.Exit(0)
}

// BlockPostToolUse blocks further actions with a JSON response
func BlockPostToolUse(reason string) {
	response := HookResponse{
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
