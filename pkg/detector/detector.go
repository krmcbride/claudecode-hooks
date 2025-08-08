// Package detector provides command detection logic with configurable rules
package detector

import (
	"slices"
	"strings"

	"mvdan.cc/sh/v3/syntax"
)

// CommandRule defines what commands and patterns to detect
type CommandRule struct {
	Command         string   // Primary command (git, aws, kubectl)
	BlockedPatterns []string // Subcommand patterns to block
	AllowExceptions []string // Patterns to allow despite blocks
	Description     string   // Human readable description
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

// ShouldBlockCommand determines if a command should be blocked.
// Returns true if command should be BLOCKED, false if allowed.
func (d *CommandDetector) ShouldBlockCommand(command string) bool {
	// Reset state for new analysis
	d.currentDepth = 0
	d.issues = d.issues[:0]
	return d.analyzeCommandRecursive(command)
}

// analyzeCommandRecursive performs analysis with recursion tracking
func (d *CommandDetector) analyzeCommandRecursive(shellExpr string) bool {
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
	return slices.ContainsFunc(calls, d.analyzeCallExpr)
}

// analyzeCallExpr analyzes a single call expression for threats
func (d *CommandDetector) analyzeCallExpr(call *syntax.CallExpr) bool {
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

	// Check shell interpreters
	if d.checkShellInterpreter(call, cmd) {
		return true // BLOCK
	}

	// Check eval commands
	if d.checkEvalCommand(call, cmd) {
		return true // BLOCK
	}

	// Check execution patterns
	if d.checkExecutionPatterns(call, cmd) {
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

// checkShellInterpreter checks for shell interpreter commands
func (d *CommandDetector) checkShellInterpreter(call *syntax.CallExpr, cmd string) bool {
	if !isShellInterpreter(cmd) {
		return false
	}

	d.addIssue("Shell interpreter detected: " + cmd)

	// Recursively analyze wrapped commands
	if commands, _ := extractShellCommands(call); len(commands) > 0 {
		for _, shellCmd := range commands {
			if d.analyzeCommandRecursive(shellCmd) {
				d.addIssue("Blocked command in shell: " + shellCmd)
				return true // BLOCK
			}
		}
	}
	return true // BLOCK shell interpreters
}

// checkEvalCommand checks for eval/source commands
func (d *CommandDetector) checkEvalCommand(call *syntax.CallExpr, cmd string) bool {
	if !isEvalCommand(cmd) {
		return false
	}

	d.addIssue("Eval/source command detected: " + cmd)

	// Recursively analyze eval content
	if evalContent := analyzeEvalCommand(call); len(evalContent) > 0 {
		for _, content := range evalContent {
			if d.analyzeCommandRecursive(content) {
				d.addIssue("Blocked command in eval: " + content)
				return true // BLOCK
			}
		}
	}
	return true // BLOCK eval commands
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
	if !isMatchingCommand(cmd, rule.Command) {
		return false
	}

	// Need arguments for pattern matching
	if len(call.Args) < 2 {
		return false
	}

	// Extract and validate arguments
	args, hasDynamic := d.extractArguments(call.Args[1:], rule.Command)
	if hasDynamic {
		return true // BLOCK: Dynamic subcommand
	}

	fullArgs := strings.Join(args, " ")

	// Check allow exceptions first
	if hasAllowException(fullArgs, rule.AllowExceptions) {
		return false // ALLOW
	}

	// Check blocked patterns
	if hasBlockedPattern(fullArgs, rule.BlockedPatterns) {
		d.addIssue("Blocked " + rule.Command + " pattern detected")
		return true // BLOCK
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

// checkExecutionPatterns checks for command execution patterns
func (d *CommandDetector) checkExecutionPatterns(call *syntax.CallExpr, cmd string) bool {
	// Check xargs
	if d.checkXargsCommand(call, cmd) {
		return true
	}

	// Check find -exec
	if d.checkFindExecCommand(call, cmd) {
		return true
	}

	// Check GNU parallel
	if d.checkParallelCommand(cmd) {
		return true
	}

	return false
}

// checkXargsCommand checks for xargs usage
func (d *CommandDetector) checkXargsCommand(call *syntax.CallExpr, cmd string) bool {
	if !isXargsCommand(cmd) {
		return false
	}

	d.addIssue("xargs command detected which can execute piped commands")

	// Check for blocked commands in xargs arguments
	for _, arg := range call.Args[1:] {
		argStr, _ := resolveStaticWord(arg)
		for _, rule := range d.commandRules {
			if isMatchingCommand(argStr, rule.Command) {
				d.addIssue("Blocked command passed to xargs: " + argStr)
				return true // BLOCK
			}
		}
	}
	return true // BLOCK xargs itself
}

// checkFindExecCommand checks for find with -exec
func (d *CommandDetector) checkFindExecCommand(call *syntax.CallExpr, cmd string) bool {
	if !isFindCommand(cmd) {
		return false
	}

	// Look for -exec flags
	execIndex := -1
	for i, arg := range call.Args[1:] {
		argStr, _ := resolveStaticWord(arg)
		if argStr == "-exec" || argStr == "-execdir" || argStr == "-ok" {
			execIndex = i
			d.addIssue("find with -exec detected which can execute commands")
			break
		}
	}

	if execIndex < 0 {
		return false // No -exec flag
	}

	// Check for blocked commands after -exec
	if execIndex+2 < len(call.Args) {
		nextArg, _ := resolveStaticWord(call.Args[execIndex+2])
		for _, rule := range d.commandRules {
			if isMatchingCommand(nextArg, rule.Command) {
				d.addIssue("Blocked command in find -exec: " + nextArg)
				return true // BLOCK
			}
		}
	}

	return true // BLOCK find -exec
}

// checkParallelCommand checks for GNU parallel
func (d *CommandDetector) checkParallelCommand(cmd string) bool {
	if isParallelCommand(cmd) {
		d.addIssue("GNU parallel detected which can execute multiple commands")
		return true
	}
	return false
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
