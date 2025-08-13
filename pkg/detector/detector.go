// Package detector provides command detection logic with configurable rules
package detector

import (
	"path"
	"slices"
	"strings"

	"mvdan.cc/sh/v3/syntax"
)

// CommandRule defines what commands and patterns to detect
type CommandRule struct {
	BlockedCommand  string   // Primary command to block (git, aws, kubectl)
	BlockedPatterns []string // Subcommand patterns to block
}

// CommandDetector provides command detection for safety validation.
// It analyzes shell commands to identify potentially dangerous operations
// based on configured rules, detecting both direct and obfuscated attempts
// to execute blocked commands.
type CommandDetector struct {
	commandRules []CommandRule
	issues       []string
	maxDepth     int
	currentDepth int
}

// NewCommandDetector creates a new detector with safety checks.
// Parameters:
//   - rules: List of commands and patterns to block
//   - maxDepth: Maximum recursion depth for analyzing nested commands (default: 10)
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

// GetIssues returns all detected security/safety issues found during analysis.
// Returns a copy of the issues slice to prevent external modification.
// Each issue describes why a command was blocked or flagged as suspicious.
func (d *CommandDetector) GetIssues() []string {
	if len(d.issues) == 0 {
		return nil
	}
	result := make([]string, len(d.issues))
	copy(result, d.issues)
	return result
}

// ShouldBlockShellExpr is the main entry point for command analysis.
// It parses and analyzes a shell expression to determine if it contains
// any blocked commands or patterns.
// Returns true if the command should be BLOCKED, false if allowed.
// Side effect: Updates the internal issues list with detailed reasons for blocking.
func (d *CommandDetector) ShouldBlockShellExpr(shellExpr string) bool {
	// Reset state for new analysis
	d.currentDepth = 0
	d.issues = d.issues[:0]
	return d.analyzeShellExprRecursive(shellExpr)
}

// addIssue records a security/safety issue found during analysis.
// These issues are returned to the user to explain why a command was blocked.
func (d *CommandDetector) addIssue(issue string) {
	d.issues = append(d.issues, issue)
}

// analyzeShellExprRecursive performs recursive analysis of shell expressions.
// It parses the expression into an AST and checks each command call.
// Tracks recursion depth to prevent stack overflow from deeply nested commands
// or maliciously crafted input.
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

// checkDynamicCommand detects attempts to use variable substitution or
// command substitution to dynamically construct command names.
// Example: $CMD push (where CMD="git") would be blocked.
func (d *CommandDetector) checkDynamicCommand(cmdIsStatic bool) bool {
	if !cmdIsStatic {
		d.addIssue("Command uses dynamic substitution - unable to verify safety")
		return true
	}
	return false
}

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

// analyzeStringLiterals performs deep inspection of string literals that might
// contain embedded commands. This is crucial for detecting commands hidden in:
//   - Shell interpreter arguments (bash -c "git push")
//   - Eval statements (eval "aws delete")
//   - Echo piped to shell (echo "git push" | sh)
//
// Only analyzes strings for commands known to execute their string arguments.
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

// definitelyLooksLikeCommand applies strict heuristics for echo/printf content.
// Since echo/printf are often used legitimately, this function reduces false
// positives by only flagging strings that:
//   - Start with a known blocked command
//   - Contain shell metacharacters indicating command chaining (;, &&, ||, |)
//
// This avoids blocking legitimate output like: echo "Use git for version control"
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

// looksLikeCommand heuristically determines if a string contains executable commands.
// Used to filter out simple arguments before expensive parsing.
// Returns false for:
//   - Flags (strings starting with -)
//   - Single words without shell metacharacters
//
// Returns true if the string appears to contain commands or command chains.
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

// checkObfuscation detects various obfuscation techniques used to hide commands:
//   - Base64 encoding
//   - Hex encoding
//   - Echo with escape sequences (\x codes)
//   - Character substitution patterns
//
// These techniques are commonly used to bypass simple string matching.
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

// collectStaticContent aggregates all statically resolvable string content
// from a command call. Used for obfuscation detection where we need to
// analyze the command as a whole rather than individual arguments.
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

// checkEchoEscapes detects echo commands using escape sequences to construct
// hidden commands. Examples:
//   - echo -e "\x67\x69\x74" (hex for "git")
//   - echo $'\147\151\164' (octal for "git")
//
// These can be piped to shell interpreters to execute obfuscated commands.
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

// ============================================================================
// Pattern Matching Utilities
// ============================================================================

// hasBlockedPattern checks if text matches any blocked patterns.
// Supports:
//   - Wildcard "*" to block all subcommands
//   - Glob patterns like "delete-*" or "terminate-*"
//   - Exact string matching for specific subcommands
//
// Case-insensitive matching for better coverage.
func hasBlockedPattern(text string, patterns []string) bool {
	if len(patterns) == 0 {
		return false
	}

	textLower := strings.ToLower(text)
	for _, pattern := range patterns {
		// Handle wildcard pattern - blocks everything
		if pattern == "*" {
			return true
		}

		// Handle glob patterns (e.g., "delete-*", "terminate-*")
		if strings.Contains(pattern, "*") {
			prefix := strings.TrimSuffix(strings.ToLower(pattern), "*")
			if strings.HasPrefix(textLower, prefix) {
				return true
			}
			// Also check if it appears as a word (for "aws delete-bucket")
			if strings.Contains(textLower, " "+prefix) {
				return true
			}
		} else {
			// Simple substring matching for exact patterns
			if strings.Contains(textLower, strings.ToLower(pattern)) {
				return true
			}
		}
	}
	return false
}

// ============================================================================
// Command Matching Utilities
// ============================================================================

// normalizeCommand extracts the base command name from a full path.
// Handles various path formats:
//   - Full paths: /usr/bin/git -> git
//   - Relative paths: ./git -> git
//   - User paths: ~/.nix-profile/bin/aws -> aws
//   - Windows paths with .exe: git.exe -> git
func normalizeCommand(cmd string) string {
	// Extract just the base name from the path
	// This handles any path like /usr/bin/git, ./git, ~/.nix-profile/bin/aws, etc.
	base := path.Base(cmd)

	// Remove .exe suffix for Windows
	base = strings.TrimSuffix(base, ".exe")

	return base
}

// isMatchingCommand determines if a command string matches a rule's blocked command.
// Handles multiple formats:
//   - Direct match: "git" == "git"
//   - Path match: "/usr/bin/git" matches "git"
//   - Windows: "git.exe" matches "git"
//
// This ensures commands are caught regardless of how they're invoked.
func isMatchingCommand(cmd, ruleCmd string) bool {
	// Direct match
	if cmd == ruleCmd {
		return true
	}

	// Check if cmd ends with the rule command (handles paths)
	// Examples: /usr/bin/git, ./git, git.exe
	if strings.HasSuffix(cmd, "/"+ruleCmd) || strings.HasSuffix(cmd, "\\"+ruleCmd) {
		return true
	}

	// Check for .exe on Windows (recursive check for path + .exe)
	if strings.HasSuffix(cmd, ".exe") {
		baseName := strings.TrimSuffix(cmd, ".exe")
		return isMatchingCommand(baseName, ruleCmd)
	}

	// Check normalized version
	return normalizeCommand(cmd) == ruleCmd
}

// isEchoCommand determines if a command is echo (including path variations).
// Used for special handling of echo commands that might output executable strings.
func isEchoCommand(cmd string) bool {
	return normalizeCommand(cmd) == "echo"
}
