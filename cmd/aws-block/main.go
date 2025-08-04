// Package main implements a Claude Code hook to block dangerous AWS commands.
package main

import (
	"log"
	"strings"

	"github.com/krmcbride/claudecode-hooks/pkg/common"
)

// Dangerous AWS commands that should be blocked
var dangerousCommands = map[string][]string{
	"aws": {
		"s3api delete-bucket",
		"s3 rm --recursive",
		"iam delete-user",
		"iam delete-role",
		"ec2 terminate-instances",
		"rds delete-db-instance",
		"cloudformation delete-stack",
	},
}

func main() {
	// Read hook input
	input, err := common.ReadHookInput()
	if err != nil {
		log.Printf("Failed to decode JSON: %v", err)
		common.AllowExecution()
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
		// If we can't parse it, allow execution for AWS commands
		common.AllowExecution()
	}

	var issues []string
	hasDangerousCommand := false

	// Check each command call
	for _, call := range calls {
		cmd := common.GetCommandName(call)
		args := common.GetCommandArgs(call)

		if patterns, exists := dangerousCommands[cmd]; exists {
			fullCommand := cmd + " " + strings.Join(args, " ")

			for _, pattern := range patterns {
				if strings.Contains(fullCommand, pattern) {
					hasDangerousCommand = true
					issues = append(issues, "Detected dangerous AWS command: "+pattern)
				}
			}
		}
	}

	if hasDangerousCommand {
		common.BlockExecution("Dangerous AWS command detected!", issues)
	}

	common.AllowExecution()
}
