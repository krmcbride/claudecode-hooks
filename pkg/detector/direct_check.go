// Package detector - direct command checking strategies
package detector

import (
	"slices"
	"strings"

	"mvdan.cc/sh/v3/syntax"
)

// checkDirectCommand checks if the command directly matches any blocking rules.
// This handles straightforward cases like "git push" or "aws delete-bucket"
// where the command is explicitly stated without obfuscation.
func (d *CommandDetector) checkDirectCommand(call *syntax.CallExpr, cmd string) bool {
	for _, rule := range d.commandRules {
		if blocked := d.checkRuleMatch(call, cmd, rule); blocked {
			return true
		}
	}
	return false
}

// checkRuleMatch evaluates a single command against a specific blocking rule.
// It checks both the command name and its arguments/subcommands against
// the rule's patterns. Handles wildcards and glob patterns.
func (d *CommandDetector) checkRuleMatch(call *syntax.CallExpr, cmd string, rule CommandRule) bool {
	// Check if command matches this rule
	if !isMatchingCommand(cmd, rule.BlockedCommand) {
		return false
	}

	// Extract arguments if any exist
	var fullArgs string
	if len(call.Args) > 1 {
		// Extract and validate arguments
		args, hasDynamic := d.extractArguments(call.Args[1:], rule.BlockedCommand)
		if hasDynamic {
			return true // BLOCK: Dynamic subcommand
		}
		fullArgs = strings.Join(args, " ")
	}

	// Check blocked patterns
	// If no arguments and pattern is "*", block the command
	// If no arguments and specific patterns, don't block (command alone is OK)
	// If arguments exist, check against patterns
	if len(rule.BlockedPatterns) > 0 {
		if hasBlockedPattern(fullArgs, rule.BlockedPatterns) {
			d.addIssue("Blocked " + rule.BlockedCommand + " pattern detected")
			return true // BLOCK
		}
		// Special case: wildcard pattern blocks even commands with no args
		if slices.Contains(rule.BlockedPatterns, "*") {
			d.addIssue("Blocked " + rule.BlockedCommand + " command")
			return true // BLOCK
		}
	}

	return false
}

// extractArguments converts AST argument nodes to string values.
// Returns the extracted strings and a flag indicating if any argument
// contains dynamic content (variables, command substitution) that can't
// be statically analyzed.
func (d *CommandDetector) extractArguments(args []*syntax.Word, command string) ([]string, bool) {
	result := make([]string, 0, len(args))
	for _, arg := range args {
		argVal, argIsStatic := resolveStaticWord(arg)

		// Check for dynamic subcommands
		if !argIsStatic {
			d.addIssue(command + " uses dynamic subcommand")
			return nil, true // Has dynamic content
		}

		result = append(result, argVal)
	}
	return result, false
}
