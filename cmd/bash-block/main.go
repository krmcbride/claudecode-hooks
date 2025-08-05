// Package main provides a generic bash command blocker for Claude Code hooks
package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/krmcbride/claudecode-hooks/pkg/detector"
	"github.com/krmcbride/claudecode-hooks/pkg/hook"
)

const (
	defaultSecurityLevel = "advanced"
	defaultMaxRecursion  = 10
)

func main() {
	// Parse command-line flags
	var (
		command     = flag.String("cmd", "", "Primary command to monitor (required)")
		patterns    = flag.String("patterns", "", "Comma-separated blocked patterns (required)")
		security    = flag.String("security", defaultSecurityLevel, "Security level: basic|advanced|paranoid")
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

	// Parse security level
	securityLevel := detector.SecurityLevel(*security)
	switch securityLevel {
	case detector.SecurityBasic, detector.SecurityAdvanced, detector.SecurityParanoid:
		// Valid
	default:
		fmt.Fprintf(os.Stderr, "Error: invalid security level '%s'. Must be: basic, advanced, or paranoid\n", *security)
		os.Exit(1)
	}

	// Parse max recursion
	maxRecursion, err := strconv.Atoi(*maxRecur)
	if err != nil || maxRecursion <= 0 {
		fmt.Fprintf(os.Stderr, "Error: invalid max-recursion '%s'. Must be a positive integer\n", *maxRecur)
		os.Exit(1)
	}

	// Parse patterns and allow list
	blockedPatterns := parseCommaSeparated(*patterns)
	allowExceptions := parseCommaSeparated(*allowList)

	// Create command rule
	rule := detector.CommandRule{
		Command:         *command,
		BlockedPatterns: blockedPatterns,
		AllowExceptions: allowExceptions,
		Description:     *description,
	}

	// Read hook input
	input, err := hook.ReadHookInput()
	if err != nil {
		// If we can't parse input, allow execution to avoid blocking legitimate commands
		hook.AllowExecution()
		return
	}

	// Create detector with configuration
	commandDetector := detector.NewCommandDetector([]detector.CommandRule{rule}, securityLevel, maxRecursion)

	// Analyze the command
	if commandDetector.AnalyzeCommand(input.ToolInput.Command) {
		issues := commandDetector.GetIssues()
		hook.BlockExecution(fmt.Sprintf("Blocked %s command detected!", *command), issues)
		return
	}

	// Allow execution if no issues found
	hook.AllowExecution()
}

// parseCommaSeparated splits a comma-separated string and trims whitespace
func parseCommaSeparated(input string) []string {
	if input == "" {
		return []string{}
	}

	parts := strings.Split(input, ",")
	result := make([]string, 0, len(parts))

	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}

	return result
}

// showUsage displays usage information
func showUsage() {
	fmt.Fprintf(os.Stderr, `bash-block: Generic bash command blocker for Claude Code hooks

USAGE:
    bash-block -cmd=COMMAND -patterns=PATTERNS [OPTIONS]

REQUIRED FLAGS:
    -cmd string
            Primary command to monitor (e.g., "git", "aws", "kubectl")
    
    -patterns string
            Comma-separated list of blocked patterns (e.g., "push", "delete-bucket,terminate-instances")

OPTIONAL FLAGS:
    -security string
            Security level (default: %s)
            • basic:    Pattern matching only (fastest)
            • advanced: + obfuscation detection (recommended)
            • paranoid: + blocks all dynamic content (most secure)
    
    -desc string
            Human-readable description for logging
    
    -allow string
            Comma-separated list of exception patterns to allow despite blocks
    
    -max-recursion string
            Maximum recursion depth for command analysis (default: %d)
    
    -help
            Show this help message

EXAMPLES:
    # Block git push commands with advanced security
    bash-block -cmd=git -patterns=push -desc="Block git push"
    
    # Block dangerous AWS operations with basic security (faster)
    bash-block -cmd=aws -patterns="delete-bucket,terminate-instances" -security=basic
    
    # Block kubectl delete with exceptions for specific namespaces
    bash-block -cmd=kubectl -patterns="delete" -allow="delete pod" -desc="Block dangerous kubectl deletes"
    
    # Paranoid security for sensitive commands
    bash-block -cmd=rm -patterns="-rf" -security=paranoid -desc="Block rm -rf"

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
        "args": ["-cmd=aws", "-patterns=delete-bucket,terminate-instances", "-security=basic"]
      }
    ]
  }
}

`, defaultSecurityLevel, defaultMaxRecursion)
}
