// Package detector provides command matching and classification utilities
package detector

import (
	"slices"
	"strings"
)

// Shell interpreter names
var shellInterpreters = []string{"sh", "bash", "zsh", "ksh", "fish", "dash", "ash", "csh", "tcsh"}

// Eval-like commands that execute code
var evalCommands = []string{"eval", "source", "."}

// normalizeCommand removes common prefixes and suffixes from command paths
func normalizeCommand(cmd string) string {
	// Remove common path prefixes
	normalized := strings.TrimPrefix(cmd, "/usr/bin/")
	normalized = strings.TrimPrefix(normalized, "/bin/")
	normalized = strings.TrimPrefix(normalized, "/usr/local/bin/")

	// Remove .exe suffix for Windows
	normalized = strings.TrimSuffix(normalized, ".exe")

	return normalized
}

// isMatchingCommand checks if cmd matches the rule command
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

// isShellInterpreter checks if the command is a shell interpreter
func isShellInterpreter(cmd string) bool {
	return slices.Contains(shellInterpreters, normalizeCommand(cmd))
}

// isEvalCommand checks if the command evaluates/sources code
func isEvalCommand(cmd string) bool {
	return slices.Contains(evalCommands, cmd)
}

// isXargsCommand checks if the command is xargs or similar
func isXargsCommand(cmd string) bool {
	normalized := normalizeCommand(cmd)
	return normalized == "xargs" || strings.HasSuffix(normalized, "/xargs")
}

// isFindCommand checks if the command is find
func isFindCommand(cmd string) bool {
	normalized := normalizeCommand(cmd)
	return normalized == "find" || strings.HasSuffix(normalized, "/find")
}

// isParallelCommand checks if the command is GNU parallel
func isParallelCommand(cmd string) bool {
	return strings.Contains(normalizeCommand(cmd), "parallel")
}

// isEchoCommand checks if the command is echo
func isEchoCommand(cmd string) bool {
	normalized := normalizeCommand(cmd)
	return normalized == "echo" || strings.HasSuffix(normalized, "/echo")
}
