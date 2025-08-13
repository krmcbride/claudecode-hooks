// Package detector provides command detection logic with configurable rules
package detector

import (
	"slices"

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
