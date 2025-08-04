// Package common provides shared functionality for Claude Code hooks.
package common

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
	os.Stderr.WriteString("🚫 BLOCKED: " + message + "\n")
	for _, issue := range issues {
		os.Stderr.WriteString("Issue: " + issue + "\n")
	}
	os.Exit(2) // Block execution
}

// AllowExecution allows the command to proceed
func AllowExecution() {
	os.Exit(0)
}