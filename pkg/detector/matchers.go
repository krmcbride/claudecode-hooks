// Package detector provides command matching and classification utilities
package detector

import (
	"path"
	"strings"
)

// normalizeCommand extracts the base command name from a path
func normalizeCommand(cmd string) string {
	// Extract just the base name from the path
	// This handles any path like /usr/bin/git, ./git, ~/.nix-profile/bin/aws, etc.
	base := path.Base(cmd)

	// Remove .exe suffix for Windows
	base = strings.TrimSuffix(base, ".exe")

	return base
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

// isEchoCommand checks if the command is echo
func isEchoCommand(cmd string) bool {
	return normalizeCommand(cmd) == "echo"
}
