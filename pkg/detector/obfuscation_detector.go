// Package detector provides obfuscation detection capabilities for command analysis
package detector

import (
	"strings"

	"mvdan.cc/sh/v3/syntax"

	"github.com/krmcbride/claudecode-hooks/pkg/shellparse"
)

// ObfuscationDetector handles detection and analysis of command obfuscation
type ObfuscationDetector struct {
	patternMatcher *PatternMatcher
}

// NewObfuscationDetector creates a new obfuscation detector
func NewObfuscationDetector(patternMatcher *PatternMatcher) *ObfuscationDetector {
	return &ObfuscationDetector{
		patternMatcher: patternMatcher,
	}
}

// CheckObfuscationPatterns checks for common obfuscation patterns (advanced security only)
func (od *ObfuscationDetector) CheckObfuscationPatterns(call *syntax.CallExpr, commandRules []CommandRule) (bool, []string) {
	var issues []string

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

	// Use common obfuscation detection
	if obfuscated, obfIssues := shellparse.DetectObfuscation(content); obfuscated {
		issues = append(issues, obfIssues...)
		// If obfuscated AND contains blocked command terms, block it
		if od.patternMatcher.ContainsAnyCommandPattern(content, commandRules) {
			return true, issues
		}
	}

	// Check for specific obfuscated patterns for each command rule
	for _, rule := range commandRules {
		if od.containsObfuscatedPatterns(content, rule) {
			issues = append(issues, "Obfuscated "+rule.Command+" pattern detected")
			return true, issues
		}
	}

	return false, issues
}

// containsObfuscatedPatterns checks for obfuscated versions of command patterns
func (od *ObfuscationDetector) containsObfuscatedPatterns(content string, rule CommandRule) bool {
	for _, pattern := range rule.BlockedPatterns {
		if pattern == "" {
			continue
		}

		// Generate common obfuscation variants
		obfuscatedPatterns := od.generateObfuscatedPatterns(rule.Command, pattern)

		for _, obfPattern := range obfuscatedPatterns {
			if strings.Contains(content, obfPattern) {
				return true
			}
		}
	}
	return false
}

// generateObfuscatedPatterns creates common obfuscation variants
func (od *ObfuscationDetector) generateObfuscatedPatterns(command, pattern string) []string {
	patterns := []string{}

	// For simple patterns, create quote-based obfuscation
	if !strings.Contains(pattern, " ") {
		// Single word pattern obfuscation
		word := pattern
		patterns = append(patterns,
			"\""+word+"\"", "'"+word+"'",
			string(word[0])+"\""+word[1:]+"\"",
			string(word[0])+"'"+word[1:]+"'",
		)
	} else {
		// Multi-word pattern obfuscation
		fullPattern := command + " " + pattern
		patterns = append(patterns,
			strings.ReplaceAll(fullPattern, " ", "\" \""),
			strings.ReplaceAll(fullPattern, " ", "' '"),
			strings.ReplaceAll(fullPattern, " ", "\\ "),
		)
	}

	return patterns
}
