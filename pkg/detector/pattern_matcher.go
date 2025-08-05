// Package detector provides pattern matching utilities for command detection
package detector

import (
	"strings"

	"github.com/krmcbride/claudecode-hooks/pkg/shellparse"
)

// PatternMatcher provides utilities for matching command patterns
type PatternMatcher struct{}

// NewPatternMatcher creates a new pattern matcher
func NewPatternMatcher() *PatternMatcher {
	return &PatternMatcher{}
}

// MatchesPattern checks if the command arguments match a blocked pattern
func (pm *PatternMatcher) MatchesPattern(args []string, fullArgs, pattern, command string) bool {
	lowerFullArgs := strings.ToLower(fullArgs)
	lowerPattern := strings.ToLower(pattern)

	// Direct substring match
	if strings.Contains(lowerFullArgs, lowerPattern) {
		return true
	}

	// Check individual args for exact matches
	for _, arg := range args {
		if strings.ToLower(arg) == lowerPattern {
			return true
		}
	}

	// For compound patterns like "git push", check proximity
	patternWords := strings.Fields(lowerPattern)
	if len(patternWords) > 1 {
		return pm.ContainsPatternWords(lowerFullArgs, patternWords)
	}

	return false
}

// ContainsPatternWords checks if all pattern words exist in reasonable proximity
func (pm *PatternMatcher) ContainsPatternWords(text string, words []string) bool {
	// Check if all words exist
	for _, word := range words {
		if !strings.Contains(text, word) {
			return false
		}
	}

	// Simple proximity check - if words are within reasonable distance
	if len(words) == 2 {
		firstIndex := strings.Index(text, words[0])
		secondIndex := strings.Index(text, words[1])
		if firstIndex >= 0 && secondIndex >= 0 {
			distance := secondIndex - firstIndex
			if distance > 0 && distance < 20 {
				return true
			}
		}
	}

	return false
}

// HasAllowException checks if the command matches any allow exceptions
func (pm *PatternMatcher) HasAllowException(fullArgs string, allowExceptions []string) bool {
	for _, exception := range allowExceptions {
		if exception != "" && strings.Contains(strings.ToLower(fullArgs), strings.ToLower(exception)) {
			return true
		}
	}
	return false
}

// HasBlockedPattern checks if args match any blocked patterns
func (pm *PatternMatcher) HasBlockedPattern(args []string, fullArgs string, rule CommandRule) bool {
	for _, pattern := range rule.BlockedPatterns {
		if pattern == "" {
			continue
		}

		// Check if pattern matches
		if pm.MatchesPattern(args, fullArgs, pattern, rule.Command) {
			return true
		}
	}
	return false
}

// IsMatchingCommand checks if a command matches the rule's command pattern
func (pm *PatternMatcher) IsMatchingCommand(cmd, ruleCmd string) bool {
	// Direct match
	if cmd == ruleCmd {
		return true
	}

	// Use shellparse to normalize the command path (handles Windows paths properly)
	normalizedCmd := shellparse.NormalizeCommandPath(cmd)
	if normalizedCmd == ruleCmd {
		return true
	}

	// Handle full paths like /usr/bin/git, /usr/local/bin/git, ./git
	if strings.HasSuffix(cmd, "/"+ruleCmd) {
		return true
	}

	// Handle Windows paths (cmd.exe)
	if strings.HasSuffix(cmd, ruleCmd+".exe") || strings.HasSuffix(cmd, "/"+ruleCmd+".exe") || strings.HasSuffix(cmd, "\\"+ruleCmd+".exe") {
		return true
	}

	return false
}

// ContainsAnyCommandPattern checks if text contains any configured command patterns
func (pm *PatternMatcher) ContainsAnyCommandPattern(text string, commandRules []CommandRule) bool {
	lowerText := strings.ToLower(text)

	for _, rule := range commandRules {
		lowerCommand := strings.ToLower(rule.Command)

		for _, pattern := range rule.BlockedPatterns {
			if pattern == "" {
				continue
			}

			lowerPattern := strings.ToLower(pattern)

			// Direct pattern match
			fullPattern := lowerCommand + " " + lowerPattern
			if strings.Contains(lowerText, fullPattern) {
				return true
			}

			// Check proximity if both command and pattern exist
			if strings.Contains(lowerText, lowerCommand) && strings.Contains(lowerText, lowerPattern) {
				cmdIndex := strings.Index(lowerText, lowerCommand)
				patternIndex := strings.Index(lowerText, lowerPattern)
				if cmdIndex >= 0 && patternIndex >= 0 {
					distance := patternIndex - cmdIndex
					if distance > 0 && distance < 20 {
						return true
					}
				}
			}
		}
	}

	return false
}
