// Package main implements a Claude Code hook to block git push commands.
package main

import (
	"log"

	"github.com/krmcbride/claudecode-hooks/pkg/detector"
	"github.com/krmcbride/claudecode-hooks/pkg/hook"
)

func main() {
	// Read hook input
	input, err := hook.ReadHookInput()
	if err != nil {
		log.Printf("Failed to decode JSON: %v", err)
		hook.AllowExecution() // Allow execution if we can't parse input
	}

	// Only process Bash commands
	if input.ToolName != "Bash" {
		hook.AllowExecution()
	}

	command := input.ToolInput.Command
	if command == "" {
		hook.AllowExecution()
	}

	// Analyze the command for git push patterns
	gitDetector := detector.NewGitPushDetector()

	if gitDetector.AnalyzeCommand(command) {
		hook.BlockExecution("Detected git push command!", gitDetector.GetIssues())
	}

	hook.AllowExecution()
}
