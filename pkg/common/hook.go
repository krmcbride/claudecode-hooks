// Package common provides shared functionality for Claude Code hooks.
package common //nolint:revive // Package name 'common' is appropriate for shared utilities

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

// PostToolUseInput represents the JSON input from Claude Code PostToolUse hooks
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
