// Package main implements a Claude Code hook to format files after editing.
package main

import (
	"context"
	"errors"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/krmcbride/claudecode-hooks/pkg/hook"
)

// FileFormatter handles file formatting operations
type FileFormatter struct {
	Command     string
	Extensions  []string
	BlockOnFail bool
}

// NewFileFormatter creates a new FileFormatter instance
func NewFileFormatter(command string, extensions []string, blockOnFail bool) *FileFormatter {
	return &FileFormatter{
		Command:     command,
		Extensions:  extensions,
		BlockOnFail: blockOnFail,
	}
}

// ProcessInput processes PostToolUse input and formats files
func (f *FileFormatter) ProcessInput(input *hook.PostToolUseInput) error {
	if !f.shouldProcessInput(input) {
		return nil
	}

	filesToFormat := f.getFilesToFormat(input)
	if len(filesToFormat) == 0 {
		return nil
	}

	formatFailed := f.formatFiles(filesToFormat)
	if formatFailed && f.BlockOnFail {
		return errors.New("file formatting failed")
	}

	return nil
}

// shouldProcessInput checks if we should process this input
func (f *FileFormatter) shouldProcessInput(input *hook.PostToolUseInput) bool {
	// PostToolUse hooks only run after successful operations, so we don't need to check success
	return input.ToolName == "Edit" || input.ToolName == "MultiEdit" || input.ToolName == "Write"
}

// getFilesToFormat checks if the file should be formatted
func (f *FileFormatter) getFilesToFormat(input *hook.PostToolUseInput) []string {
	// All three tools (Edit, Write, MultiEdit) have file_path at the root level
	if input.ToolInput.FilePath == "" {
		return nil
	}

	// Check if the file extension is allowed
	if !f.isAllowedExtension(input.ToolInput.FilePath) {
		return nil
	}

	return []string{input.ToolInput.FilePath}
}

// isAllowedExtension checks if the file extension is allowed
func (f *FileFormatter) isAllowedExtension(filePath string) bool {
	ext := filepath.Ext(filePath)
	return slices.Contains(f.Extensions, ext)
}

// formatFiles formats each file and returns whether any failed
func (f *FileFormatter) formatFiles(filesToFormat []string) bool {
	formatFailed := false
	for _, filePath := range filesToFormat {
		if err := f.formatFile(filePath); err != nil {
			formatFailed = true
		}
	}
	return formatFailed
}

// formatFile runs the format command on a single file
func (f *FileFormatter) formatFile(filePath string) error {
	// Replace {FILEPATH} placeholder with actual file path
	// This allows flexible command templates like:
	// - "gofmt -w {FILEPATH}"
	// - "make fmt-file FILE={FILEPATH}"
	// - "prettier --write {FILEPATH} --config .prettierrc"
	expandedCommand := strings.ReplaceAll(f.Command, "{FILEPATH}", filePath)

	// Parse the command (with placeholder replaced if it was present)
	parts := strings.Fields(expandedCommand)
	if len(parts) == 0 {
		return nil
	}

	baseCommand := parts[0]
	args := parts[1:]

	// If no placeholder was found and command hasn't changed, use legacy behavior
	// This maintains backwards compatibility for commands without placeholders
	if expandedCommand == f.Command {
		// If the last argument ends with =, concatenate the filepath without a space
		// This handles legacy cases like "make fmt-file FILE="
		if len(args) > 0 && strings.HasSuffix(args[len(args)-1], "=") {
			args[len(args)-1] = args[len(args)-1] + filePath
		} else {
			// Default: append filepath as last argument
			args = append(args, filePath)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, baseCommand, args...) // #nosec G204 - command is user-configured
	_, err := cmd.CombinedOutput()
	return err
}
