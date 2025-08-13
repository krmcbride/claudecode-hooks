// Package detector - command matching utilities
package detector

import (
	"path"
	"strings"
)

// normalizeCommand extracts the base command name from a full path.
// Handles various path formats:
//   - Full paths: /usr/bin/git -> git
//   - Relative paths: ./git -> git
//   - User paths: ~/.nix-profile/bin/aws -> aws
//   - Windows paths with .exe: git.exe -> git
func normalizeCommand(cmd string) string {
	// Extract just the base name from the path
	// This handles any path like /usr/bin/git, ./git, ~/.nix-profile/bin/aws, etc.
	base := path.Base(cmd)

	// Remove .exe suffix for Windows
	base = strings.TrimSuffix(base, ".exe")

	return base
}

// isMatchingCommand determines if a command string matches a rule's blocked command.
// Handles multiple formats:
//   - Direct match: "git" == "git"
//   - Path match: "/usr/bin/git" matches "git"
//   - Windows: "git.exe" matches "git"
//
// This ensures commands are caught regardless of how they're invoked.
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

// isEchoCommand determines if a command is echo (including path variations).
// Used for special handling of echo commands that might output executable strings.
func isEchoCommand(cmd string) bool {
	return normalizeCommand(cmd) == "echo"
}
