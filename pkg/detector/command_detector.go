// Package detector provides command detection logic with configurable rules
package detector

import (
	"slices"
	"strings"

	"mvdan.cc/sh/v3/syntax"

	"github.com/krmcbride/claudecode-hooks/pkg/shellparse"
)

// CommandRule defines what commands and patterns to detect
type CommandRule struct {
	Command         string   // Primary command (git, aws, kubectl)
	BlockedPatterns []string // Subcommand patterns to block
	AllowExceptions []string // Patterns to allow despite blocks
	Description     string   // Human readable description
}

// CommandDetector provides comprehensive command detection with maximum security
type CommandDetector struct {
	commandRules []CommandRule
	issues       []string
	maxDepth     int
	currentDepth int
}

// NewCommandDetector creates a new detector with maximum security
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
func (d *CommandDetector) analyzeCommandRecursive(command string) bool {
	// Prevent DoS via deeply nested commands
	d.currentDepth++
	if d.currentDepth > d.maxDepth {
		d.issues = append(d.issues, "Max recursion depth exceeded - potential DoS attempt")
		return true // BLOCK
	}
	defer func() { d.currentDepth-- }()

	// Parse command into structured call expressions
	calls, err := shellparse.ParseCommand(command)
	if err != nil {
		// FAIL-SECURE: Can't parse = block
		d.issues = append(d.issues, "Failed to parse command: "+err.Error())
		return true // BLOCK
	}

	// Check if any call should be blocked
	return slices.ContainsFunc(calls, d.analyzeCallExpr)
}

// analyzeCallExpr analyzes a single call expression for threats
func (d *CommandDetector) analyzeCallExpr(call *syntax.CallExpr) bool {
	if len(call.Args) == 0 {
		return false // ALLOW: Empty call
	}

	// Extract command name
	cmd, cmdIsStatic := shellparse.ResolveStaticWord(call.Args[0])

	// Block dynamic commands (variables/substitution)
	if !cmdIsStatic {
		d.issues = append(d.issues, "Command uses dynamic substitution which could hide malicious intent")
		return true // BLOCK
	}

	// Check direct command patterns
	if d.checkDirectCommand(call, cmd) {
		return true // BLOCK
	}

	// Check for shell interpreters (sh -c, bash -c)
	if d.isShellInterpreter(cmd) {
		d.issues = append(d.issues, "Shell interpreter detected: "+cmd)
		// Recursively analyze wrapped commands
		if commands, _ := shellparse.ExtractShellCommands(call); len(commands) > 0 {
			for _, shellCmd := range commands {
				if d.analyzeCommandRecursive(shellCmd) {
					d.issues = append(d.issues, "Blocked command in shell: "+shellCmd)
					return true // BLOCK
				}
			}
		}
		// Block shell interpreters even without detected nested commands
		return true // BLOCK
	}

	// Check for eval/source commands
	if d.isEvalCommand(cmd) {
		d.issues = append(d.issues, "Eval/source command detected: "+cmd)
		// Recursively analyze eval content
		if evalContent := shellparse.AnalyzeEvalCommand(call); len(evalContent) > 0 {
			for _, content := range evalContent {
				if d.analyzeCommandRecursive(content) {
					d.issues = append(d.issues, "Blocked command in eval: "+content)
					return true // BLOCK
				}
			}
		}
		return true // BLOCK
	}

	// Check for execution patterns (xargs, find -exec)
	if d.checkExecutionPatterns(call, cmd) {
		return true // BLOCK
	}

	// Check for obfuscation
	if d.checkObfuscation(call) {
		return true // BLOCK
	}

	return false // ALLOW
}

// checkDirectCommand checks for direct command matches with configured rules
func (d *CommandDetector) checkDirectCommand(call *syntax.CallExpr, cmd string) bool {
	for _, rule := range d.commandRules {
		// Check if command matches this rule
		if !d.isMatchingCommand(cmd, rule.Command) {
			continue
		}

		// Need arguments for pattern matching
		if len(call.Args) < 2 {
			continue
		}

		// Extract all arguments
		args := make([]string, 0, len(call.Args)-1)
		for _, arg := range call.Args[1:] {
			argVal, argIsStatic := shellparse.ResolveStaticWord(arg)

			// Block dynamic subcommands
			if !argIsStatic {
				d.issues = append(d.issues, rule.Command+" uses dynamic subcommand")
				return true // BLOCK
			}

			args = append(args, argVal)
		}

		fullArgs := strings.Join(args, " ")

		// Check allow exceptions first
		if d.hasAllowException(fullArgs, rule.AllowExceptions) {
			continue // ALLOW
		}

		// Check blocked patterns
		if d.hasBlockedPattern(fullArgs, rule.BlockedPatterns) {
			d.issues = append(d.issues, "Blocked "+rule.Command+" pattern detected")
			return true // BLOCK
		}
	}

	return false // ALLOW
}

// isMatchingCommand checks if cmd matches the rule command
func (d *CommandDetector) isMatchingCommand(cmd, ruleCmd string) bool {
	// Direct match
	if cmd == ruleCmd {
		return true
	}

	// Check if cmd ends with the rule command (handles paths)
	// Examples: /usr/bin/git, ./git, git.exe
	if strings.HasSuffix(cmd, "/"+ruleCmd) || strings.HasSuffix(cmd, "\\"+ruleCmd) {
		return true
	}

	// Check for .exe on Windows
	if strings.HasSuffix(cmd, ".exe") {
		baseName := strings.TrimSuffix(cmd, ".exe")
		return d.isMatchingCommand(baseName, ruleCmd)
	}

	return false
}

// hasAllowException checks if text matches any allow exception patterns
func (d *CommandDetector) hasAllowException(text string, exceptions []string) bool {
	if len(exceptions) == 0 {
		return false
	}

	textLower := strings.ToLower(text)
	for _, exception := range exceptions {
		// Check if all words in the exception pattern exist in the text
		// No proximity limit - just check existence
		words := strings.Fields(strings.ToLower(exception))
		allFound := true
		for _, word := range words {
			if !strings.Contains(textLower, word) {
				allFound = false
				break
			}
		}
		if allFound {
			return true
		}
	}
	return false
}

// hasBlockedPattern checks if text matches any blocked patterns
func (d *CommandDetector) hasBlockedPattern(text string, patterns []string) bool {
	if len(patterns) == 0 {
		return false
	}

	textLower := strings.ToLower(text)
	for _, pattern := range patterns {
		// Simple substring matching
		if strings.Contains(textLower, strings.ToLower(pattern)) {
			return true
		}
	}
	return false
}

// isShellInterpreter checks if the command is a shell interpreter
func (d *CommandDetector) isShellInterpreter(cmd string) bool {
	shells := []string{"sh", "bash", "zsh", "ksh", "fish", "dash", "ash", "csh", "tcsh"}
	cmdBase := strings.TrimPrefix(cmd, "/usr/bin/")
	cmdBase = strings.TrimPrefix(cmdBase, "/bin/")
	cmdBase = strings.TrimSuffix(cmdBase, ".exe")

	return slices.Contains(shells, cmdBase)
}

// isEvalCommand checks if the command evaluates/sources code
func (d *CommandDetector) isEvalCommand(cmd string) bool {
	evalCmds := []string{"eval", "source", "."}
	return slices.Contains(evalCmds, cmd)
}

// checkExecutionPatterns checks for other execution patterns
func (d *CommandDetector) checkExecutionPatterns(call *syntax.CallExpr, cmd string) bool {
	// Check xargs
	if strings.Contains(cmd, "xargs") {
		d.issues = append(d.issues, "xargs command detected which can execute piped commands")
		// Check for patterns in xargs arguments
		for _, arg := range call.Args[1:] {
			argStr, _ := shellparse.ResolveStaticWord(arg)
			for _, rule := range d.commandRules {
				if d.isMatchingCommand(argStr, rule.Command) {
					d.issues = append(d.issues, "Blocked command passed to xargs: "+argStr)
					return true // BLOCK
				}
			}
		}
		return true // BLOCK xargs itself
	}

	// Check find -exec
	if cmd == "find" || strings.HasSuffix(cmd, "/find") {
		hasExec := false
		for _, arg := range call.Args[1:] {
			argStr, _ := shellparse.ResolveStaticWord(arg)
			if argStr == "-exec" || argStr == "-execdir" || argStr == "-ok" {
				hasExec = true
				d.issues = append(d.issues, "find with -exec detected which can execute commands")
				break
			}
		}
		if hasExec {
			// Check for blocked commands in exec arguments
			for i, arg := range call.Args[1:] {
				argStr, _ := shellparse.ResolveStaticWord(arg)
				if argStr == "-exec" || argStr == "-execdir" || argStr == "-ok" {
					if i+1 < len(call.Args)-1 {
						nextArg, _ := shellparse.ResolveStaticWord(call.Args[i+2])
						for _, rule := range d.commandRules {
							if d.isMatchingCommand(nextArg, rule.Command) {
								d.issues = append(d.issues, "Blocked command in find -exec: "+nextArg)
								return true // BLOCK
							}
						}
					}
				}
			}
			return true // BLOCK find -exec
		}
	}

	// Check GNU parallel
	if strings.Contains(cmd, "parallel") {
		d.issues = append(d.issues, "GNU parallel detected which can execute multiple commands")
		return true // BLOCK
	}

	return false
}

// checkObfuscation checks for obfuscated commands
func (d *CommandDetector) checkObfuscation(call *syntax.CallExpr) bool {
	// Collect all static string content for analysis
	var allContent strings.Builder
	for _, arg := range call.Args {
		val, isStatic := shellparse.ResolveStaticWord(arg)
		if isStatic && val != "" {
			allContent.WriteString(val)
			allContent.WriteString(" ")
		}
	}

	content := allContent.String()

	// Check for base64/hex encoding indicators
	if obfuscated, obfIssues := shellparse.DetectObfuscation(content); obfuscated {
		d.issues = append(d.issues, obfIssues...)
		return true // BLOCK
	}

	// Check for echo with escape sequences
	cmd, _ := shellparse.ResolveStaticWord(call.Args[0])
	if cmd == "echo" || strings.HasSuffix(cmd, "/echo") {
		for _, arg := range call.Args[1:] {
			argStr, _ := shellparse.ResolveStaticWord(arg)
			// Check for hex escapes
			if strings.Contains(argStr, "\\x") || strings.Contains(argStr, "\\0") {
				d.issues = append(d.issues, "echo with escape sequences detected (possible obfuscation)")
				return true // BLOCK
			}
			// Check for -e flag with escapes
			if argStr == "-e" {
				d.issues = append(d.issues, "echo -e detected which enables escape sequences")
				return true // BLOCK
			}
		}
	}

	return false
}
