// Package main implements a Claude Code hook to block dangerous kubectl commands.
package main

import (
	"log"
	"strings"

	"github.com/krmcbride/claudecode-hooks/pkg/hook"
	"github.com/krmcbride/claudecode-hooks/pkg/shellparse"
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
	input, err := hook.ReadHookInput()
	if err != nil {
		log.Printf("Failed to decode JSON: %v", err)
		hook.AllowExecution()
	}

	// Only process Bash commands
	if input.ToolName != "Bash" {
		hook.AllowExecution()
	}

	command := input.ToolInput.Command
	if command == "" {
		hook.AllowExecution()
	}

	// Parse the command
	calls, err := shellparse.ParseCommand(command)
	if err != nil {
		// If we can't parse it, allow execution for kubectl commands
		hook.AllowExecution()
	}

	var issues []string
	hasDangerousCommand := false

	// Check each command call
	for _, call := range calls {
		cmd := shellparse.GetCommandName(call)
		args := shellparse.GetCommandArgs(call)

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
		hook.BlockExecution("Dangerous kubectl command detected!", issues)
	}

	hook.AllowExecution()
}
