// Package detector provides generic command detection logic with configurable rules
package detector

import (
	"slices"
	"strings"

	"mvdan.cc/sh/v3/syntax"

	"github.com/krmcbride/claudecode-hooks/pkg/shellparse"
)

// SecurityLevel defines the security analysis depth
type SecurityLevel string

const (
	// SecurityBasic enables pattern matching only
	SecurityBasic SecurityLevel = "basic"
	// SecurityAdvanced enables pattern matching + obfuscation detection
	SecurityAdvanced SecurityLevel = "advanced"
	// SecurityParanoid enables all dynamic content blocking
	SecurityParanoid SecurityLevel = "paranoid"
)

// CommandRule defines what commands and patterns to detect
type CommandRule struct {
	Command         string   // Primary command (git, aws, kubectl)
	BlockedPatterns []string // Subcommand patterns to block
	AllowExceptions []string // Patterns to allow despite blocks
	Description     string   // Human readable description
}

// CommandDetector provides comprehensive command detection with configurable rules
type CommandDetector struct {
	commandRules        []CommandRule
	securityLevel       SecurityLevel
	issues              []string
	maxDepth            int
	currentDepth        int
	patternMatcher      *PatternMatcher
	securityChecker     *SecurityChecker
	obfuscationDetector *ObfuscationDetector
}

// NewCommandDetector creates a new detector with specified rules and security level
func NewCommandDetector(rules []CommandRule, securityLevel SecurityLevel, maxDepth int) *CommandDetector {
	if maxDepth <= 0 {
		maxDepth = 10 // Default safe recursion limit
	}

	patternMatcher := NewPatternMatcher()
	securityChecker := NewSecurityChecker(securityLevel, patternMatcher)
	obfuscationDetector := NewObfuscationDetector(patternMatcher)

	return &CommandDetector{
		commandRules:        rules,
		securityLevel:       securityLevel,
		issues:              make([]string, 0),
		maxDepth:            maxDepth,
		currentDepth:        0,
		patternMatcher:      patternMatcher,
		securityChecker:     securityChecker,
		obfuscationDetector: obfuscationDetector,
	}
}

// GetIssues returns all detected issues
func (d *CommandDetector) GetIssues() []string {
	return d.issues
}

// AnalyzeCommand is the main entry point for command analysis
func (d *CommandDetector) AnalyzeCommand(command string) bool {
	d.currentDepth = 0
	d.issues = d.issues[:0] // Clear previous issues
	return d.analyzeCommandRecursive(command)
}

// analyzeCommandRecursive performs comprehensive analysis with recursion tracking
func (d *CommandDetector) analyzeCommandRecursive(command string) bool {
	// Check recursion depth
	d.currentDepth++
	if d.currentDepth > d.maxDepth {
		d.issues = append(d.issues, "Max recursion depth exceeded - potential DoS attempt")
		return true
	}
	defer func() { d.currentDepth-- }()

	// Parse the main command
	calls, err := shellparse.ParseCommand(command)
	if err != nil {
		// If we can't parse it, be conservative and block
		d.issues = append(d.issues, "Failed to parse command: "+err.Error())
		return true
	}

	// Analyze each call expression
	return slices.ContainsFunc(calls, d.analyzeCallExpr)
}

// analyzeCallExpr analyzes a single call expression for configured command patterns
func (d *CommandDetector) analyzeCallExpr(call *syntax.CallExpr) bool {
	if len(call.Args) == 0 {
		return false
	}

	// Get command with enhanced resolution
	cmd, cmdIsStatic := shellparse.ResolveStaticWord(call.Args[0])

	// Check for dynamic command
	if blocked, issues := d.securityChecker.CheckDynamicCommand(cmdIsStatic); blocked {
		d.issues = append(d.issues, issues...)
		return true
	}

	// Check for direct command matches
	if d.checkDirectCommand(call, cmd) {
		return true
	}

	// Advanced security checks (skip for basic level)
	if d.securityLevel != SecurityBasic {
		// Check for shell interpreter patterns (sh -c, bash -c)
		if blocked, issues := d.securityChecker.CheckShellInterpreter(call, d.commandRules); blocked {
			d.issues = append(d.issues, issues...)
			// Also recursively analyze shell commands
			shellCommands, _ := shellparse.ExtractShellCommands(call)
			for _, shellCmd := range shellCommands {
				if d.analyzeCommandRecursive(shellCmd) {
					d.issues = append(d.issues, "Blocked command detected in shell command: "+shellCmd)
					return true
				}
			}
			return true
		}

		// Check for eval patterns
		if blocked, issues := d.securityChecker.CheckEvalCommand(call, cmd, d.commandRules); blocked {
			d.issues = append(d.issues, issues...)
			// Also recursively analyze eval content
			evalContent := shellparse.AnalyzeEvalCommand(call)
			for _, content := range evalContent {
				if d.analyzeCommandRecursive(content) {
					d.issues = append(d.issues, "Blocked command detected in eval command")
					return true
				}
			}
			return true
		}

		// Check for other execution patterns
		if blocked, issues := d.securityChecker.CheckExecutionPatterns(call, cmd, d.commandRules); blocked {
			d.issues = append(d.issues, issues...)
			return true
		}

		// Check for obfuscation patterns
		if blocked, issues := d.obfuscationDetector.CheckObfuscationPatterns(call, d.commandRules); blocked {
			d.issues = append(d.issues, issues...)
			return true
		}
	}

	return false
}

// checkDirectCommand checks for direct command matches with configured rules
func (d *CommandDetector) checkDirectCommand(call *syntax.CallExpr, cmd string) bool {
	// Check each command rule
	for _, rule := range d.commandRules {
		if !d.patternMatcher.IsMatchingCommand(cmd, rule.Command) {
			continue
		}

		// Need at least command + subcommand for pattern matching
		if len(call.Args) < 2 {
			continue
		}

		// Get the subcommand or arguments
		args := make([]string, 0, len(call.Args)-1)
		for _, arg := range call.Args[1:] {
			argVal, argIsStatic := shellparse.ResolveStaticWord(arg)

			// Check for dynamic subcommands
			if blocked, issues := d.securityChecker.CheckDynamicSubcommand(argIsStatic, rule.Command); blocked {
				d.issues = append(d.issues, issues...)
				return true
			}

			if argIsStatic {
				args = append(args, argVal)
			}
		}

		// Check if command arguments match blocked patterns
		fullArgs := strings.Join(args, " ")

		// First check allow exceptions
		if d.patternMatcher.HasAllowException(fullArgs, rule.AllowExceptions) {
			continue
		}

		// Then check blocked patterns
		if d.patternMatcher.HasBlockedPattern(args, fullArgs, rule) {
			d.issues = append(d.issues, "Blocked "+rule.Command+" pattern detected")
			return true
		}
	}

	return false
}
