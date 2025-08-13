// Package detector - checking arguments for blocked commands
package detector

import (
	"slices"
	"strings"

	"mvdan.cc/sh/v3/syntax"
)

// checkArgumentsForBlockedCommands scans command arguments for blocked commands.
// This catches indirect execution attempts where blocked commands appear as
// arguments to other commands.
// Examples:
//   - xargs git push
//   - find . -exec aws delete-bucket {}
//   - parallel git push ::: branch1 branch2
func (d *CommandDetector) checkArgumentsForBlockedCommands(call *syntax.CallExpr) bool {
	// Skip the first argument (the command itself) and check the rest
	for i := 1; i < len(call.Args); i++ {
		arg := call.Args[i]

		// Resolve the argument to a static string
		argStr, isStatic := resolveStaticWord(arg)
		if !isStatic || argStr == "" {
			continue
		}

		// Check if this argument matches any blocked command
		for _, rule := range d.commandRules {
			if isMatchingCommand(argStr, rule.BlockedCommand) {
				// Found a blocked command as an argument
				// Now check if the next arguments match any blocked patterns
				remainingArgs := call.Args[i+1:]
				if d.checkPatternInArgs(remainingArgs, rule) {
					d.addIssue("Blocked command '" + rule.BlockedCommand + "' found as argument")
					return true // BLOCK
				}
			}
		}
	}
	return false
}

// checkPatternInArgs validates arguments against a rule's blocked patterns.
// Used when a blocked command is found as an argument to check if its
// subcommands/arguments also match the blocking criteria.
// Returns true if the pattern matches and should be blocked.
func (d *CommandDetector) checkPatternInArgs(args []*syntax.Word, rule CommandRule) bool {
	// If no patterns specified, allow the command
	if len(rule.BlockedPatterns) == 0 {
		return false
	}

	// Check for wildcard pattern
	if slices.Contains(rule.BlockedPatterns, "*") {
		return true // Wildcard blocks all
	}

	// Collect arguments as strings
	var argStrings []string
	for _, arg := range args {
		argStr, isStatic := resolveStaticWord(arg)
		if isStatic && argStr != "" && !strings.HasPrefix(argStr, "-") {
			argStrings = append(argStrings, argStr)
		}
	}

	fullArgs := strings.Join(argStrings, " ")
	return hasBlockedPattern(fullArgs, rule.BlockedPatterns)
}
