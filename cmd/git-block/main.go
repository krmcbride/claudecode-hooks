// Package main implements a Claude Code hook to block git push commands.
package main

import (
	"log"
	"strings"

	"github.com/krmcbride/claudecode-hooks/pkg/common"
)

func main() {
	// Read hook input
	input, err := common.ReadHookInput()
	if err != nil {
		log.Printf("Failed to decode JSON: %v", err)
		common.AllowExecution() // Allow execution if we can't parse input
	}

	// Only process Bash commands
	if input.ToolName != "Bash" {
		common.AllowExecution()
	}

	command := input.ToolInput.Command
	if command == "" {
		common.AllowExecution()
	}

	// Parse the command
	calls, err := common.ParseCommand(command)
	if err != nil {
		// If we can't parse it, be cautious and block
		common.BlockExecution("Failed to parse command", []string{err.Error()})
	}

	var issues []string
	hasGitPush := false

	// Check each command call
	for _, call := range calls {
		cmd := common.GetCommandName(call)
		args := common.GetCommandArgs(call)

		// Check for git push patterns
		if cmd == "git" && len(args) > 0 && args[0] == "push" {
			hasGitPush = true
			issues = append(issues, "Detected 'git push' command")
		}

		// Check for command substitution containing git push
		if cmd == "bash" || cmd == "sh" {
			for _, arg := range args {
				if strings.Contains(arg, "git") && strings.Contains(arg, "push") {
					hasGitPush = true
					issues = append(issues, "Detected 'git push' in subshell")
				}
			}
		}

		// Check other dangerous patterns
		if cmd == "eval" || cmd == "exec" {
			for _, arg := range args {
				if strings.Contains(arg, "git") && strings.Contains(arg, "push") {
					hasGitPush = true
					issues = append(issues, "Detected 'git push' in "+cmd)
				}
			}
		}
	}

	if hasGitPush {
		common.BlockExecution("Detected git push command!", issues)
	}

	common.AllowExecution()
}
