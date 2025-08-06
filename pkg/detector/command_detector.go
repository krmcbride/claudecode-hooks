// Package detector provides generic command detection logic with configurable rules
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
	commandRules        []CommandRule
	issues              []string
	maxDepth            int
	currentDepth        int
	patternMatcher      *PatternMatcher
	securityChecker     *SecurityChecker
	obfuscationDetector *ObfuscationDetector
}

// NewCommandDetector creates a new detector with maximum security
func NewCommandDetector(rules []CommandRule, maxDepth int) *CommandDetector {
	if maxDepth <= 0 {
		maxDepth = 10 // Default safe recursion limit
	}

	patternMatcher := NewPatternMatcher()
	securityChecker := NewSecurityChecker(patternMatcher)
	obfuscationDetector := NewObfuscationDetector(patternMatcher)

	return &CommandDetector{
		commandRules:        rules,
		issues:              make([]string, 0),
		maxDepth:            maxDepth,
		currentDepth:        0,
		patternMatcher:      patternMatcher,
		securityChecker:     securityChecker,
		obfuscationDetector: obfuscationDetector,
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
// This is the main entry point that resets state and delegates to recursive analysis.
func (d *CommandDetector) ShouldBlockCommand(command string) bool {
	// Reset recursion tracking for new analysis
	d.currentDepth = 0
	// Clear any issues from previous analysis (reuse slice for efficiency)
	d.issues = d.issues[:0]
	return d.shouldBlockCommandRecursive(command)
}

// shouldBlockCommandRecursive performs comprehensive analysis with recursion tracking.
// This handles nested commands like "sh -c 'git push'" or "eval $(echo git push)".
// Returns true if command should be BLOCKED.
func (d *CommandDetector) shouldBlockCommandRecursive(command string) bool {
	// Increment recursion depth to prevent DoS attacks via deeply nested commands
	d.currentDepth++
	if d.currentDepth > d.maxDepth {
		d.issues = append(d.issues, "Max recursion depth exceeded - potential DoS attempt")
		return true // BLOCK: Suspicious deep nesting
	}
	// Ensure depth counter is decremented when function exits
	defer func() { d.currentDepth-- }()

	// Parse command string into structured call expressions using mvdan.cc/sh
	// This handles complex shell syntax: pipes, redirects, subshells, etc.
	calls, err := shellparse.ParseCommand(command)
	if err != nil {
		// FAIL-SECURE: If we can't parse it, assume it's malicious
		// Better to block legitimate edge cases than allow attacks
		d.issues = append(d.issues, "Failed to parse command: "+err.Error())
		return true // BLOCK: Unparseable commands
	}

	// Analyze each parsed call expression (commands in pipes, subshells, etc.)
	// Returns true if ANY call should be blocked
	return slices.ContainsFunc(calls, d.analyzeCallExpr)
}

// analyzeCallExpr analyzes a single call expression for all security threats.
// This is the core detection logic that runs multiple security checks:
// 1. Direct command pattern matching (git push, aws terminate-instances)
// 2. Shell interpreter detection (sh -c, bash -c)
// 3. Eval command detection (eval, source)
// 4. Execution pattern detection (xargs, find -exec)
// 5. Obfuscation detection (base64, hex encoding)
// Returns true if call should be BLOCKED.
func (d *CommandDetector) analyzeCallExpr(call *syntax.CallExpr) bool {
	// Skip empty calls (shouldn't happen with valid shell syntax)
	if len(call.Args) == 0 {
		return false // ALLOW: Empty call
	}

	// Extract the command name from call.Args[0]
	// cmd = resolved command string, cmdIsStatic = true if not using variables/substitution
	cmd, cmdIsStatic := shellparse.ResolveStaticWord(call.Args[0])

	// SECURITY CHECK 1: Block dynamic commands (using variables/substitution)
	// Examples: $CMD push, $(echo git) push, `whoami` push
	if blocked, issues := d.securityChecker.CheckDynamicCommand(cmdIsStatic); blocked {
		d.issues = append(d.issues, issues...)
		return true // BLOCK: Dynamic command name
	}

	// SECURITY CHECK 2: Direct command pattern matching
	// This is where "git push", "aws terminate-instances", etc. get detected
	// Handles interspersed flags: "aws --region us-east-1 ec2 terminate-instances"
	if d.checkDirectCommand(call, cmd) {
		return true // BLOCK: Matched configured rule
	}

	// SECURITY CHECK 3: Shell interpreter detection
	// Detects: sh -c 'command', bash -c 'command', zsh -c 'command'
	// These are common ways to bypass detection by wrapping commands
	if blocked, issues := d.securityChecker.CheckShellInterpreter(call, d.commandRules); blocked {
		d.issues = append(d.issues, issues...)
		// RECURSIVE ANALYSIS: Extract and analyze the wrapped commands
		// Example: "sh -c 'git push'" -> recursively analyze "git push"
		shellCommands, _ := shellparse.ExtractShellCommands(call)
		for _, shellCmd := range shellCommands {
			if d.shouldBlockCommandRecursive(shellCmd) {
				d.issues = append(d.issues, "Blocked command detected in shell command: "+shellCmd)
				return true // BLOCK: Nested command matched
			}
		}
		return true // BLOCK: Shell interpreter detected (even if nested analysis didn't find issues)
	}

	// SECURITY CHECK 4: Eval command detection
	// Detects: eval 'command', source script.sh, . script.sh
	// These execute dynamic content and are common attack vectors
	if blocked, issues := d.securityChecker.CheckEvalCommand(call, cmd, d.commandRules); blocked {
		d.issues = append(d.issues, issues...)
		// RECURSIVE ANALYSIS: Extract and analyze eval content
		// Example: "eval 'git push'" -> recursively analyze "git push"
		evalContent := shellparse.AnalyzeEvalCommand(call)
		if slices.ContainsFunc(evalContent, d.shouldBlockCommandRecursive) {
			d.issues = append(d.issues, "Blocked command detected in eval command")
			return true // BLOCK: Nested command in eval matched
		}
		return true // BLOCK: Eval pattern detected (even if nested analysis didn't find issues)
	}

	// SECURITY CHECK 5: Other execution patterns
	// Detects: xargs, find -exec, parallel, etc.
	// These can execute commands and bypass simple pattern matching
	if blocked, issues := d.securityChecker.CheckExecutionPatterns(call, cmd, d.commandRules); blocked {
		d.issues = append(d.issues, issues...)
		return true // BLOCK: Execution pattern detected
	}

	// SECURITY CHECK 6: Obfuscation detection
	// Detects: base64 encoding, hex encoding, character escaping
	// Examples: echo cHVzaA== | base64 -d, echo -e "\x70\x75\x73\x68"
	if blocked, issues := d.obfuscationDetector.CheckObfuscationPatterns(call, d.commandRules); blocked {
		d.issues = append(d.issues, issues...)
		return true // BLOCK: Obfuscation detected
	}

	return false // ALLOW: No threats detected
}

// checkDirectCommand checks for direct command matches with configured rules.
// This is where the main pattern matching happens for commands like:
//
//	"git push" -> matches rule.Command="git", rule.BlockedPatterns=["push"]
//	"aws --region us-east-1 ec2 terminate-instances" -> extracts all args after "aws"
//
// IMPORTANT: This function handles interspersed flags by including ALL arguments
// in pattern matching, not just subcommands. The PatternMatcher uses proximity
// logic to find patterns within the full argument string.
//
// LIMITATION: 20-character proximity limit can cause issues with long flags!
func (d *CommandDetector) checkDirectCommand(call *syntax.CallExpr, cmd string) bool {
	// Check each configured rule (git, aws, kubectl, etc.)
	for _, rule := range d.commandRules {
		// Skip if command doesn't match this rule
		// Handles: git, /usr/bin/git, ./git, git.exe
		if !d.patternMatcher.IsMatchingCommand(cmd, rule.Command) {
			continue
		}

		// Skip if no arguments (need subcommand for pattern matching)
		// Example: just "git" with no args -> nothing to check
		if len(call.Args) < 2 {
			continue
		}

		// Extract ALL arguments after the command name
		// call.Args[0] = command ("git", "aws", etc.)
		// call.Args[1:] = ALL flags and subcommands mixed together
		// Example: ["--region", "us-east-1", "ec2", "terminate-instances", "--instance-ids", "i-123"]
		args := make([]string, 0, len(call.Args)-1)
		for _, arg := range call.Args[1:] {
			// Resolve argument (handle quoted strings, escape sequences, etc.)
			argVal, argIsStatic := shellparse.ResolveStaticWord(arg)

			// SECURITY: Block dynamic subcommands (using variables/substitution)
			// Examples: git $ACTION, aws $(echo delete-bucket)
			if blocked, issues := d.securityChecker.CheckDynamicSubcommand(argIsStatic, rule.Command); blocked {
				d.issues = append(d.issues, issues...)
				return true // BLOCK: Dynamic subcommand
			}

			// Only include static arguments in pattern matching
			// This filters out unresolved variables while keeping flags and subcommands
			if argIsStatic {
				args = append(args, argVal)
			}
		}

		// Create full argument string for pattern matching
		// Example: "--region us-east-1 ec2 terminate-instances --instance-ids i-123"
		fullArgs := strings.Join(args, " ")

		// STEP 1: Check allow exceptions FIRST (fail-fast for legitimate commands)
		// Example: "delete pod" exception allows "kubectl delete pod my-app"
		// WARNING: Subject to 20-char proximity limit! Long flags can break exceptions.
		if d.patternMatcher.HasAllowException(fullArgs, rule.AllowExceptions) {
			continue // ALLOW: Matches exception pattern
		}

		// STEP 2: Check blocked patterns (only if no exception matched)
		// Example: "terminate-instances" pattern blocks AWS terminate commands
		// This uses both individual args and fullArgs for flexible matching
		if d.patternMatcher.HasBlockedPattern(args, fullArgs, rule) {
			d.issues = append(d.issues, "Blocked "+rule.Command+" pattern detected")
			return true // BLOCK: Matches blocked pattern
		}
	}

	return false // ALLOW: No rules matched
}
