// Package main provides a bash command safety validator for Claude Code hooks
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

const defaultMaxRecursion = 10

// cmdFlag allows multiple -cmd flags to be specified
type cmdFlag []string

func (c *cmdFlag) String() string {
	return strings.Join(*c, ", ")
}

func (c *cmdFlag) Set(value string) error {
	*c = append(*c, value)
	return nil
}

func main() {
	// Parse command-line flags
	var commands cmdFlag
	flag.Var(&commands, "cmd", "Command and optional patterns to block (can be specified multiple times)")

	maxRecur := flag.String("max-recursion", strconv.Itoa(defaultMaxRecursion), "Max recursion depth")
	showHelp := flag.Bool("help", false, "Show help message")

	flag.Parse()

	// Show help if requested
	if *showHelp || len(commands) == 0 {
		showUsage()
		if *showHelp {
			os.Exit(0)
		}
		os.Exit(1)
	}

	// Parse max recursion
	maxRecursion, err := strconv.Atoi(*maxRecur)
	if err != nil || maxRecursion <= 0 {
		fmt.Fprintf(os.Stderr, "Error: invalid max-recursion '%s'. Must be a positive integer\n", *maxRecur)
		os.Exit(1)
	}

	// Parse command rules from -cmd flags
	rules := parseCommandRules(commands)
	if len(rules) == 0 {
		fmt.Fprintf(os.Stderr, "Error: no valid command rules specified\n")
		os.Exit(1)
	}

	// Read PreToolUse hook input
	input, err := hook.ReadPreToolUseInput()
	if err != nil {
		// Security tool must fail secure - block on parse errors
		hook.BlockPreToolUse("Failed to parse hook input", []string{err.Error()})
		return
	}

	// Create detector with configuration
	commandDetector := detector.NewCommandDetector(rules, maxRecursion)

	// Check if expression should be blocked
	if commandDetector.ShouldBlockShellExpr(input.ToolInput.Command) {
		issues := commandDetector.GetIssues()
		hook.BlockPreToolUse("Blocked command detected!", issues)
		return
	}

	// Allow execution if no issues found
	hook.AllowPreToolUse()
}

// parseCommandRules parses -cmd flag values into CommandRule structs
func parseCommandRules(commands []string) []detector.CommandRule {
	var rules []detector.CommandRule

	for _, cmd := range commands {
		parts := strings.Fields(cmd)
		if len(parts) == 0 {
			continue
		}

		// First part is the command to block
		blockedCommand := parts[0]

		// Remaining parts are patterns to block (if any)
		// If no patterns specified, block ALL uses of the command
		var blockedPatterns []string
		if len(parts) > 1 {
			blockedPatterns = parts[1:]
		} else {
			// Block all subcommands by using a wildcard pattern
			// Empty patterns means "check command only, not subcommands"
			// So we use "*" to indicate "block any subcommand"
			blockedPatterns = []string{"*"}
		}

		rule := detector.CommandRule{
			BlockedCommand:  blockedCommand,
			BlockedPatterns: blockedPatterns,
		}

		rules = append(rules, rule)
	}

	return rules
}

// showUsage displays usage information
func showUsage() {
	fmt.Fprintf(os.Stderr, `bash-block: Bash command blocker for Claude Code hooks

Provides an additional layer of safety on top of Claude Code's built-in deny permissions.
Blocks commands including through variables, subshells, eval, obfuscation, etc.

USAGE:
    bash-block -cmd COMMAND_SPEC [-cmd COMMAND_SPEC ...] [OPTIONS]

REQUIRED:
    -cmd string
            Command and optional patterns to block (can be specified multiple times)
            Format: "command [pattern1] [pattern2] ..."
            
            Examples:
              -cmd git                    Block all git commands
              -cmd "git push"             Block only git push
              -cmd "git push pull"        Block git push and git pull
              -cmd "aws delete-*"         Block aws delete-* commands
              -cmd kubectl                Block all kubectl commands

OPTIONAL:
    -max-recursion int
            Maximum recursion depth for command analysis (default: %d)
    
    -help
            Show this help message

EXAMPLES:
    # Block all git commands
    bash-block -cmd git
    
    # Block only git push
    bash-block -cmd "git push"
    
    # Block multiple specific commands
    bash-block -cmd "git push" -cmd "aws delete-bucket terminate-instances"
    
    # Block all aws and kubectl commands
    bash-block -cmd aws -cmd kubectl
    
    # Complex example with multiple rules
    bash-block -cmd "git push force-push" \
               -cmd "aws delete-* terminate-*" \
               -cmd "kubectl delete"

CLAUDE CODE CONFIGURATION:
Add to your Claude Code settings.json:

{
  "hooks": {
    "preToolUse": [
      {
        "command": "/path/to/bash-block",
        "args": ["-cmd", "git push", "-cmd", "aws delete-*"]
      }
    ]
  }
}

`, defaultMaxRecursion)
}
