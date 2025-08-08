// Package detector provides command detection logic with configurable rules
package detector

import (
	"slices"
	"strings"

	"mvdan.cc/sh/v3/syntax"
)

// CommandRule defines what commands and patterns to detect
type CommandRule struct {
	BlockedCommand  string   // Primary command to block (git, aws, kubectl)
	BlockedPatterns []string // Subcommand patterns to block
}

// CommandDetector provides command detection for safety validation
type CommandDetector struct {
	commandRules []CommandRule
	issues       []string
	maxDepth     int
	currentDepth int
}

// NewCommandDetector creates a new detector with safety checks
func NewCommandDetector(rules []CommandRule, maxDepth int) *CommandDetector {
	if maxDepth <= 0 {
		maxDepth = 10 // Default safe recursion limit
	}

	return &CommandDetector{
		commandRules: rules,
		issues:       make([]string, 0),
		maxDepth:     maxDepth,
		currentDepth: 0,
	}
}

// GetIssues returns all detected issues (returns a copy to prevent aliasing)
func (d *CommandDetector) GetIssues() []string {
	if len(d.issues) == 0 {
		return nil
	}
	result := make([]string, len(d.issues))
	copy(result, d.issues)
	return result
}

// ShouldBlockShellExpr determines if a command should be blocked.
// Returns true if command should be BLOCKED, false if allowed.
func (d *CommandDetector) ShouldBlockShellExpr(shellExpr string) bool {
	// Reset state for new analysis
	d.currentDepth = 0
	d.issues = d.issues[:0]
	return d.analyzeShellExprRecursive(shellExpr)
}

// analyzeShellExprRecursive performs analysis with recursion tracking
func (d *CommandDetector) analyzeShellExprRecursive(shellExpr string) bool {
	// Prevent excessive nesting that could cause performance issues
	d.currentDepth++
	if d.currentDepth > d.maxDepth {
		d.addIssue("Maximum nesting depth exceeded - command too complex")
		return true // BLOCK
	}
	defer func() { d.currentDepth-- }()

	// Parse shell expression into an AST
	ast, err := parseShellExpression(shellExpr)
	if err != nil {
		// Safety principle: If we can't understand it, don't run it
		d.addIssue("Unable to parse shell expression: " + err.Error())
		return true // BLOCK
	}

	// Extract command calls from the AST
	calls := extractCallExprs(ast)

	// Check if any command call should be blocked
	return slices.ContainsFunc(calls, d.shouldBlockCallExpr)
}

// shouldBlockCallExpr evaluates whether a shell call expression should be blocked.
// This is the core detection logic that checks for:
// - Commands matching configured blocking rules
// - Dynamic command substitution attempts
// - Shell interpreters and eval commands
// - Command execution patterns (xargs, find -exec, etc.)
// - Obfuscation attempts (encoding, escaping, etc.)
// Returns true if the command should be blocked, false if allowed.
func (d *CommandDetector) shouldBlockCallExpr(call *syntax.CallExpr) bool {
	if len(call.Args) == 0 {
		return false // ALLOW: Empty call
	}

	// Extract command name
	cmd, cmdIsStatic := resolveStaticWord(call.Args[0])

	// Check dynamic commands
	if d.checkDynamicCommand(cmdIsStatic) {
		return true // BLOCK
	}

	// Check direct command patterns
	if d.checkDirectCommand(call, cmd) {
		return true // BLOCK
	}

	// Check if any arguments are themselves blocked commands
	// This handles cases like: xargs git push, find . -exec git push
	if d.checkArgumentsForBlockedCommands(call) {
		return true // BLOCK
	}

	// Analyze all string literals in the command for nested commands
	if d.analyzeStringLiterals(call) {
		return true // BLOCK
	}

	// Check obfuscation
	if d.checkObfuscation(call) {
		return true // BLOCK
	}

	return false // ALLOW
}

// checkDynamicCommand checks for dynamic command substitution
func (d *CommandDetector) checkDynamicCommand(cmdIsStatic bool) bool {
	if !cmdIsStatic {
		d.addIssue("Command uses dynamic substitution - unable to verify safety")
		return true
	}
	return false
}

// checkArgumentsForBlockedCommands checks if any arguments are themselves blocked commands
// This handles cases like: xargs git push, find . -exec git push
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

// checkPatternInArgs checks if remaining arguments match a blocked pattern
func (d *CommandDetector) checkPatternInArgs(args []*syntax.Word, rule CommandRule) bool {
	// If no patterns specified, allow the command
	if len(rule.BlockedPatterns) == 0 {
		return false
	}

	// Check for wildcard pattern
	for _, pattern := range rule.BlockedPatterns {
		if pattern == "*" {
			return true // Wildcard blocks all
		}
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

// analyzeStringLiterals analyzes all string literals in the command as potential shell expressions
func (d *CommandDetector) analyzeStringLiterals(call *syntax.CallExpr) bool {
	// First check what command we're dealing with
	cmd, _ := resolveStaticWord(call.Args[0])
	normalizedCmd := normalizeCommand(cmd)

	// For certain commands, we should analyze their string arguments
	// These are commands that typically execute strings as shell code
	shouldAnalyzeStrings := false
	switch normalizedCmd {
	case "sh", "bash", "zsh", "ksh", "dash", "fish":
		// Shell interpreters with -c flag
		shouldAnalyzeStrings = true
	case "eval", "source", ".":
		// Commands that evaluate strings as shell code
		shouldAnalyzeStrings = true
	case "echo", "printf":
		// These commands output text - only analyze if piped to shell
		// We'll analyze their strings but with more conservative filtering
		shouldAnalyzeStrings = true
	}

	if !shouldAnalyzeStrings {
		return false
	}

	// Skip the first argument (the command itself) and analyze the rest
	for i := 1; i < len(call.Args); i++ {
		arg := call.Args[i]

		// Extract all string literals from this argument
		if strings := extractStringLiterals(arg); len(strings) > 0 {
			for _, str := range strings {
				// For echo/printf, be more conservative - only check if it really looks like a command
				if normalizedCmd == "echo" || normalizedCmd == "printf" {
					if !d.definitelyLooksLikeCommand(str) {
						continue
					}
				} else {
					// For shell interpreters and eval, check more broadly
					if !d.looksLikeCommand(str) {
						continue
					}
				}

				// Try to parse and analyze each string as a shell expression
				if d.analyzeShellExprRecursive(str) {
					d.addIssue("Blocked command found in string: " + str)
					return true // BLOCK
				}
			}
		}
	}
	return false
}

// looksLikeCommand checks if a string looks like it might contain a command
// rather than just being a flag or simple argument
func (d *CommandDetector) looksLikeCommand(str string) bool {
	// Skip if it's just a flag (starts with -)
	if strings.HasPrefix(str, "-") {
		return false
	}

	// Skip if it's a single word without spaces (likely just an argument)
	if !strings.Contains(str, " ") && !strings.Contains(str, ";") &&
		!strings.Contains(str, "|") && !strings.Contains(str, "&") {
		return false
	}

	// Additional check: if the string starts with a known blocked command, it's likely a command
	// This helps with cases like "git push origin" in various contexts
	for _, rule := range d.commandRules {
		if strings.HasPrefix(str, rule.BlockedCommand+" ") {
			return true
		}
	}

	return true
}

// definitelyLooksLikeCommand is a stricter version used for echo/printf strings
// It only returns true if the string definitely looks like executable commands
func (d *CommandDetector) definitelyLooksLikeCommand(str string) bool {
	// Must start with a known blocked command to be considered
	startsWithCommand := false
	for _, rule := range d.commandRules {
		if strings.HasPrefix(str, rule.BlockedCommand+" ") {
			startsWithCommand = true
			break
		}
	}

	if !startsWithCommand {
		// Check for shell metacharacters that indicate command execution
		if strings.Contains(str, ";") || strings.Contains(str, "&&") ||
			strings.Contains(str, "||") || strings.Contains(str, "|") {
			// Could be a command chain
			return true
		}
		return false
	}

	return true
}

// checkDirectCommand checks for direct command matches with configured rules
func (d *CommandDetector) checkDirectCommand(call *syntax.CallExpr, cmd string) bool {
	for _, rule := range d.commandRules {
		if blocked := d.checkRuleMatch(call, cmd, rule); blocked {
			return true
		}
	}
	return false
}

// checkRuleMatch checks if a command matches a specific rule
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
		for _, pattern := range rule.BlockedPatterns {
			if pattern == "*" {
				d.addIssue("Blocked " + rule.BlockedCommand + " command")
				return true // BLOCK
			}
		}
	}

	return false
}

// extractArguments extracts static arguments from call args
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

// checkObfuscation checks for obfuscated commands
func (d *CommandDetector) checkObfuscation(call *syntax.CallExpr) bool {
	// Collect all static string content
	content := d.collectStaticContent(call)

	// Check for base64/hex encoding
	if obfuscated, obfIssues := detectObfuscation(content); obfuscated {
		d.issues = append(d.issues, obfIssues...)
		return true // BLOCK
	}

	// Check echo with escape sequences
	return d.checkEchoEscapes(call)
}

// collectStaticContent collects all static string content from call
func (d *CommandDetector) collectStaticContent(call *syntax.CallExpr) string {
	var allContent strings.Builder
	for _, arg := range call.Args {
		val, isStatic := resolveStaticWord(arg)
		if isStatic && val != "" {
			allContent.WriteString(val)
			allContent.WriteString(" ")
		}
	}
	return allContent.String()
}

// checkEchoEscapes checks for echo with escape sequences
func (d *CommandDetector) checkEchoEscapes(call *syntax.CallExpr) bool {
	cmd, _ := resolveStaticWord(call.Args[0])
	if !isEchoCommand(cmd) {
		return false
	}

	for _, arg := range call.Args[1:] {
		argStr, _ := resolveStaticWord(arg)

		// Check for hex escapes
		if strings.Contains(argStr, "\\x") || strings.Contains(argStr, "\\0") {
			d.addIssue("echo with escape sequences detected (possible obfuscation)")
			return true // BLOCK
		}

		// Check for -e flag
		if argStr == "-e" {
			d.addIssue("echo -e detected which enables escape sequences")
			return true // BLOCK
		}
	}
	return false
}

// addIssue is a helper to add issues to the detector
func (d *CommandDetector) addIssue(issue string) {
	d.issues = append(d.issues, issue)
}
