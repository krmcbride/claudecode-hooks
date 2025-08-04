// Package common provides shared functionality for Claude Code hooks.
package common //nolint:revive // Package name 'common' is appropriate for shared utilities

import (
	"encoding/json"
	"os"
)

// HookInput represents the JSON input from Claude Code hooks
type HookInput struct {
	ToolName  string `json:"tool_name"`
	ToolInput struct {
		Command string `json:"command"`
	} `json:"tool_input"`
}

// ReadHookInput reads and parses the hook input from stdin
func ReadHookInput() (*HookInput, error) {
	var input HookInput
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
