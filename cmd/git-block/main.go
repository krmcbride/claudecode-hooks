// Package main implements a Claude Code hook to block git push commands.
package main

import (
	"log"

	"github.com/krmcbride/claudecode-hooks/pkg/common"
	"github.com/krmcbride/claudecode-hooks/pkg/detector"
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

	// Analyze the command for git push patterns
	gitDetector := detector.NewGitPushDetector()

	if gitDetector.AnalyzeCommand(command) {
		common.BlockExecution("Detected git push command!", gitDetector.GetIssues())
	}

	common.AllowExecution()
}
