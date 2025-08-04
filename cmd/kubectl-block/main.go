// Package main implements a Claude Code hook to block dangerous kubectl commands.
package main

import (
	"log"
	"strings"

	"github.com/krmcbride/claudecode-hooks/pkg/common"
)

// Dangerous kubectl commands that should be blocked
var dangerousCommands = map[string][]string{
	"kubectl": {
		"delete namespace",
		"delete --all",
		"delete deployment",
		"delete service",
		"delete pv",
		"delete pvc",
		"patch --type=merge",
	},
}

// Production-like namespaces that should be protected
var protectedNamespaces = []string{
	"kube-system",
	"kube-public",
	"kube-node-lease",
	"default",
	"production",
	"prod",
	"staging",
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
		// If we can't parse it, allow execution for kubectl commands
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

			// Check for dangerous command patterns
			for _, pattern := range patterns {
				if strings.Contains(fullCommand, pattern) {
					hasDangerousCommand = true
					issues = append(issues, "Detected dangerous kubectl command: "+pattern)
				}
			}

			// Check for operations on protected namespaces
			if strings.Contains(fullCommand, "delete") || strings.Contains(fullCommand, "patch") {
				for _, ns := range protectedNamespaces {
					if strings.Contains(fullCommand, "-n "+ns) || strings.Contains(fullCommand, "--namespace="+ns) {
						hasDangerousCommand = true
						issues = append(issues, "Detected operation on protected namespace: "+ns)
					}
				}
			}
		}
	}

	if hasDangerousCommand {
		common.BlockExecution("Dangerous kubectl command detected!", issues)
	}

	common.AllowExecution()
}
