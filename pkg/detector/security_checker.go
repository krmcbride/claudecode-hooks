// Package detector provides security checking capabilities for command detection
package detector

import (
	"slices"

	"mvdan.cc/sh/v3/syntax"

	"github.com/krmcbride/claudecode-hooks/pkg/shellparse"
)

// SecurityChecker provides advanced security validation for commands
type SecurityChecker struct {
	securityLevel  SecurityLevel
	patternMatcher *PatternMatcher
}

// NewSecurityChecker creates a new security checker
func NewSecurityChecker(securityLevel SecurityLevel, patternMatcher *PatternMatcher) *SecurityChecker {
	return &SecurityChecker{
		securityLevel:  securityLevel,
		patternMatcher: patternMatcher,
	}
}

// CheckShellInterpreter checks for shell interpreter patterns (advanced security only)
func (sc *SecurityChecker) CheckShellInterpreter(call *syntax.CallExpr, commandRules []CommandRule) (bool, []string) {
	var issues []string

	// Extract shell commands using enhanced parsing
	shellCommands, hasDynamicContent := shellparse.ExtractShellCommands(call)

	// SECURITY: Block if dynamic content detected (prevents command substitution bypass)
	if hasDynamicContent && sc.securityLevel == SecurityParanoid {
		issues = append(issues, "Dynamic shell command content detected - potential command substitution")
		return true, issues
	}

	for _, shellCmd := range shellCommands {
		// Check for simple string patterns as fallback
		if sc.patternMatcher.ContainsAnyCommandPattern(shellCmd, commandRules) {
			issues = append(issues, "Blocked pattern detected in shell command")
			return true, issues
		}
	}

	return false, issues
}

// CheckEvalCommand checks for eval command patterns (advanced security only)
func (sc *SecurityChecker) CheckEvalCommand(call *syntax.CallExpr, cmd string, commandRules []CommandRule) (bool, []string) {
	var issues []string

	if cmd != "eval" {
		return false, issues
	}

	evalContent := shellparse.AnalyzeEvalCommand(call)

	for _, content := range evalContent {
		// Check for string patterns
		if sc.patternMatcher.ContainsAnyCommandPattern(content, commandRules) {
			issues = append(issues, "Blocked pattern detected in eval")
			return true, issues
		}
	}

	return false, issues
}

// CheckExecutionPatterns checks for other command execution patterns (advanced security only)
func (sc *SecurityChecker) CheckExecutionPatterns(call *syntax.CallExpr, cmd string, commandRules []CommandRule) (bool, []string) {
	var issues []string

	// List of commands that can execute other commands
	execCommands := []string{
		"xargs", "find", "parallel", "env", "nohup",
		"timeout", "time", "watch", "script",
	}

	if !slices.Contains(execCommands, cmd) {
		return false, issues
	}

	// Look for blocked patterns in arguments
	for i := 1; i < len(call.Args); i++ {
		arg, argIsStatic := shellparse.ResolveStaticWord(call.Args[i])
		if !argIsStatic && sc.securityLevel == SecurityParanoid {
			// Dynamic content in execution command
			issues = append(issues, "Dynamic content in "+cmd+" command")
			return true, issues
		}

		if argIsStatic && sc.patternMatcher.ContainsAnyCommandPattern(arg, commandRules) {
			issues = append(issues, "Blocked pattern detected in "+cmd+" arguments")
			return true, issues
		}
	}

	return false, issues
}

// CheckDynamicCommand checks if a dynamic command should be blocked based on security level
func (sc *SecurityChecker) CheckDynamicCommand(cmdIsStatic bool) (bool, []string) {
	var issues []string

	// If command is dynamic, check security level
	if !cmdIsStatic {
		if sc.securityLevel == SecurityAdvanced || sc.securityLevel == SecurityParanoid {
			issues = append(issues, "Dynamic command detected (potential obfuscation)")
			return true, issues
		}
	}

	return false, issues
}

// CheckDynamicSubcommand checks if dynamic subcommands should be blocked for paranoid level
func (sc *SecurityChecker) CheckDynamicSubcommand(argIsStatic bool, command string) (bool, []string) {
	var issues []string

	// For paranoid level, block any dynamic subcommands
	if !argIsStatic && sc.securityLevel == SecurityParanoid {
		issues = append(issues, "Dynamic "+command+" subcommand detected")
		return true, issues
	}

	return false, issues
}
