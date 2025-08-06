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

// getFilesToFormat collects and filters files to format
func (f *FileFormatter) getFilesToFormat(input *hook.PostToolUseInput) []string {
	filePaths := f.collectFilePaths(input)
	return f.filterAndValidateFiles(filePaths)
}

// collectFilePaths extracts file paths from the input
func (f *FileFormatter) collectFilePaths(input *hook.PostToolUseInput) []string {
	var filePaths []string

	switch input.ToolName {
	case "Edit", "Write":
		if input.ToolInput.FilePath != "" {
			filePaths = append(filePaths, input.ToolInput.FilePath)
		}
	case "MultiEdit":
		seen := make(map[string]bool)
		for _, edit := range input.ToolInput.Edits {
			if edit.FilePath != "" && !seen[edit.FilePath] {
				filePaths = append(filePaths, edit.FilePath)
				seen[edit.FilePath] = true
			}
		}
		if input.ToolInput.FilePath != "" && !seen[input.ToolInput.FilePath] {
			filePaths = append(filePaths, input.ToolInput.FilePath)
		}
	}

	return filePaths
}

// filterAndValidateFiles filters files by extension
func (f *FileFormatter) filterAndValidateFiles(filePaths []string) []string {
	var filesToFormat []string
	for _, filePath := range filePaths {
		if f.isAllowedExtension(filePath) {
			filesToFormat = append(filesToFormat, filePath)
		}
	}
	return filesToFormat
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
	parts := strings.Fields(f.Command)
	if len(parts) == 0 {
		return nil
	}

	baseCommand := parts[0]
	args := append(parts[1:], filePath)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, baseCommand, args...) // #nosec G204 - command is user-configured
	_, err := cmd.CombinedOutput()
	return err
}

// ParseExtensions parses a comma-separated extension string
func ParseExtensions(extensionsFlag string) []string {
	extensions := strings.Split(extensionsFlag, ",")
	for i, ext := range extensions {
		extensions[i] = strings.TrimSpace(ext)
	}
	return extensions
}
