// Package main provides a generic bash command blocker for Claude Code hooks
package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"

	"github.com/krmcbride/claudecode-hooks/pkg/detector"
	"github.com/krmcbride/claudecode-hooks/pkg/hook"
	"github.com/krmcbride/claudecode-hooks/pkg/utils"
)

const defaultMaxRecursion = 10

func main() {
	// Parse command-line flags
	var (
		command     = flag.String("cmd", "", "Primary command to monitor (required)")
		patterns    = flag.String("patterns", "", "Comma-separated blocked patterns (required)")
		description = flag.String("desc", "", "Description for logging")
		allowList   = flag.String("allow", "", "Comma-separated exception patterns")
		maxRecur    = flag.String("max-recursion", strconv.Itoa(defaultMaxRecursion), "Max recursion depth")
		showHelp    = flag.Bool("help", false, "Show help message")
	)
	flag.Parse()

	// Show help if requested
	if *showHelp {
		showUsage()
		os.Exit(0)
	}

	// Validate required flags
	if *command == "" {
		fmt.Fprintf(os.Stderr, "Error: -cmd flag is required\n")
		showUsage()
		os.Exit(1)
	}

	if *patterns == "" {
		fmt.Fprintf(os.Stderr, "Error: -patterns flag is required\n")
		showUsage()
		os.Exit(1)
	}

	// Parse max recursion
	maxRecursion, err := strconv.Atoi(*maxRecur)
	if err != nil || maxRecursion <= 0 {
		fmt.Fprintf(os.Stderr, "Error: invalid max-recursion '%s'. Must be a positive integer\n", *maxRecur)
		os.Exit(1)
	}

	// Parse patterns and allow list
	blockedPatterns := utils.ParseCommaSeparated(*patterns)
	allowExceptions := utils.ParseCommaSeparated(*allowList)

	// Create command rule
	rule := detector.CommandRule{
		Command:         *command,
		BlockedPatterns: blockedPatterns,
		AllowExceptions: allowExceptions,
		Description:     *description,
	}

	// Read PreToolUse hook input
	input, err := hook.ReadPreToolUseInput()
	if err != nil {
		// Security tool must fail secure - block on parse errors
		hook.BlockPreToolUse("Failed to parse hook input", []string{err.Error()})
		return
	}

	// Create detector with configuration (always uses maximum security)
	commandDetector := detector.NewCommandDetector([]detector.CommandRule{rule}, maxRecursion)

	// Analyze the command
	if commandDetector.AnalyzeCommand(input.ToolInput.Command) {
		issues := commandDetector.GetIssues()
		hook.BlockPreToolUse(fmt.Sprintf("Blocked %s command detected!", *command), issues)
		return
	}

	// Allow execution if no issues found
	hook.AllowPreToolUse()
}

// showUsage displays usage information
func showUsage() {
	fmt.Fprintf(os.Stderr, `bash-block: Maximum security bash command blocker for Claude Code hooks

Provides an additional layer of security on top of Claude Code's built-in deny permissions.
Attempts to block commands in ALL forms including: variables, subshells, eval, obfuscation, etc.

USAGE:
    bash-block -cmd=COMMAND -patterns=PATTERNS [OPTIONS]

REQUIRED FLAGS:
    -cmd string
            Primary command to monitor (e.g., "git", "aws", "kubectl")
    
    -patterns string
            Comma-separated list of blocked patterns (e.g., "push", "delete-bucket,terminate-instances")

OPTIONAL FLAGS:
    -desc string
            Human-readable description for logging
    
    -allow string
            Comma-separated list of exception patterns to allow despite blocks
    
    -max-recursion string
            Maximum recursion depth for command analysis (default: %d)
    
    -help
            Show this help message

SECURITY FEATURES:
    • Attempts to block ALL forms of the command (variables, escaping, encoding)
    • Detects common obfuscation (base64, hex, character escaping)
    • Recursively analyzes nested commands (sh -c, eval, source)
    • Blocks dynamic content (variable substitution, command substitution)

EXAMPLES:
    # Block git push commands
    bash-block -cmd=git -patterns=push -desc="Block git push"
    
    # Block dangerous AWS operations
    bash-block -cmd=aws -patterns="delete-bucket,terminate-instances"
    
    # Block kubectl delete with exceptions for pods
    bash-block -cmd=kubectl -patterns="delete" -allow="delete pod"

CLAUDE CODE CONFIGURATION:
Add to your Claude Code settings.json:

{
  "hooks": {
    "preToolUse": [
      {
        "command": "/path/to/bash-block",
        "args": ["-cmd=git", "-patterns=push", "-desc=Block git push"]
      },
      {
        "command": "/path/to/bash-block",
        "args": ["-cmd=aws", "-patterns=delete-bucket,terminate-instances"]
      }
    ]
  }
}

`, defaultMaxRecursion)
}
