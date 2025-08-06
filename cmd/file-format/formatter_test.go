package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/krmcbride/claudecode-hooks/pkg/hook"
)

func TestParseExtensions(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "Single extension",
			input:    ".go",
			expected: []string{".go"},
		},
		{
			name:     "Multiple extensions",
			input:    ".go,.js,.py",
			expected: []string{".go", ".js", ".py"},
		},
		{
			name:     "Extensions with spaces",
			input:    ".go, .js , .py",
			expected: []string{".go", ".js", ".py"},
		},
		{
			name:     "Empty string",
			input:    "",
			expected: []string{""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseExtensions(tt.input)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("ParseExtensions(%s) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestNewFileFormatter(t *testing.T) {
	command := "./bin/golangci-lint fmt"
	extensions := []string{".go", ".js"}
	blockOnFail := true

	formatter := NewFileFormatter(command, extensions, blockOnFail)

	if formatter.Command != command {
		t.Errorf("Command = %s, want %s", formatter.Command, command)
	}
	if !reflect.DeepEqual(formatter.Extensions, extensions) {
		t.Errorf("Extensions = %v, want %v", formatter.Extensions, extensions)
	}
	if formatter.BlockOnFail != blockOnFail {
		t.Errorf("BlockOnFail = %v, want %v", formatter.BlockOnFail, blockOnFail)
	}
}

func TestFileFormatter_shouldProcessInput(t *testing.T) {
	formatter := NewFileFormatter("echo test", []string{".go"}, false)

	tests := []struct {
		name     string
		input    *hook.PostToolUseInput
		expected bool
	}{
		{
			name: "Edit tool",
			input: &hook.PostToolUseInput{
				ToolName: "Edit",
			},
			expected: true,
		},
		{
			name: "MultiEdit tool",
			input: &hook.PostToolUseInput{
				ToolName: "MultiEdit",
			},
			expected: true,
		},
		{
			name: "Write tool",
			input: &hook.PostToolUseInput{
				ToolName: "Write",
			},
			expected: true,
		},
		{
			name: "Wrong tool - Read",
			input: &hook.PostToolUseInput{
				ToolName: "Read",
			},
			expected: false,
		},
		{
			name: "Wrong tool - Bash",
			input: &hook.PostToolUseInput{
				ToolName: "Bash",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatter.shouldProcessInput(tt.input)
			if result != tt.expected {
				t.Errorf("shouldProcessInput() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestFileFormatter_getFilesToFormat(t *testing.T) {
	formatter := NewFileFormatter("echo test", []string{".go", ".js"}, false)

	tests := []struct {
		name     string
		input    *hook.PostToolUseInput
		expected []string
	}{
		{
			name: "Edit with Go file",
			input: &hook.PostToolUseInput{
				ToolName: "Edit",
				ToolInput: struct {
					FilePath string `json:"file_path"`
				}{
					FilePath: "main.go",
				},
			},
			expected: []string{"main.go"},
		},
		{
			name: "MultiEdit with Go file",
			input: &hook.PostToolUseInput{
				ToolName: "MultiEdit",
				ToolInput: struct {
					FilePath string `json:"file_path"`
				}{
					FilePath: "utils.go",
				},
			},
			expected: []string{"utils.go"},
		},
		{
			name: "Write with JS file",
			input: &hook.PostToolUseInput{
				ToolName: "Write",
				ToolInput: struct {
					FilePath string `json:"file_path"`
				}{
					FilePath: "app.js",
				},
			},
			expected: []string{"app.js"},
		},
		{
			name: "Edit with wrong extension",
			input: &hook.PostToolUseInput{
				ToolName: "Edit",
				ToolInput: struct {
					FilePath string `json:"file_path"`
				}{
					FilePath: "README.md",
				},
			},
			expected: nil, // Filtered out due to extension
		},
		{
			name: "Edit with empty file path",
			input: &hook.PostToolUseInput{
				ToolName: "Edit",
				ToolInput: struct {
					FilePath string `json:"file_path"`
				}{
					FilePath: "",
				},
			},
			expected: nil, // Empty file path
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatter.getFilesToFormat(tt.input)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("getFilesToFormat() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestFileFormatter_isAllowedExtension(t *testing.T) {
	formatter := NewFileFormatter("echo test", []string{".go", ".js", ".py"}, false)

	tests := []struct {
		name     string
		filePath string
		expected bool
	}{
		{
			name:     "Go file allowed",
			filePath: "main.go",
			expected: true,
		},
		{
			name:     "JavaScript file allowed",
			filePath: "script.js",
			expected: true,
		},
		{
			name:     "Python file allowed",
			filePath: "script.py",
			expected: true,
		},
		{
			name:     "Text file not allowed",
			filePath: "README.txt",
			expected: false,
		},
		{
			name:     "No extension",
			filePath: "Dockerfile",
			expected: false,
		},
		{
			name:     "Case sensitive extension",
			filePath: "Main.GO",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatter.isAllowedExtension(tt.filePath)
			if result != tt.expected {
				t.Errorf("isAllowedExtension(%s) = %v, want %v", tt.filePath, result, tt.expected)
			}
		})
	}
}

func TestFileFormatter_formatFile_placeholder(t *testing.T) {
	tests := []struct {
		name        string
		command     string
		filePath    string
		expectError bool
	}{
		{
			name:        "Command with {FILEPATH} placeholder",
			command:     "echo Formatting: {FILEPATH}",
			filePath:    "test.go",
			expectError: false,
		},
		{
			name:        "Command without placeholder appends filepath",
			command:     "echo",
			filePath:    "test.go",
			expectError: false,
		},
		{
			name:        "Command ending with = concatenates filepath",
			command:     "echo FILE=",
			filePath:    "test.go",
			expectError: false,
		},
		{
			name:        "Empty command",
			command:     "",
			filePath:    "test.go",
			expectError: false, // Empty command returns nil
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter := NewFileFormatter(tt.command, []string{".go"}, false)
			err := formatter.formatFile(tt.filePath)

			if tt.expectError && err == nil {
				t.Errorf("formatFile() expected error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("formatFile() expected no error, got %v", err)
			}
		})
	}
}

func TestFileFormatter_formatFile(t *testing.T) {
	formatter := NewFileFormatter("echo test", []string{".go"}, false)

	tests := []struct {
		name        string
		command     string
		expectError bool
	}{
		{
			name:        "Empty command",
			command:     "",
			expectError: false,
		},
		{
			name:        "Valid command",
			command:     "echo",
			expectError: false, // echo should work now
		},
		{
			name:        "Command that will fail",
			command:     "nonexistent-command-12345",
			expectError: true, // Will fail because command doesn't exist
		},
	}

	// Create a temporary file for testing
	tempFile := filepath.Join(t.TempDir(), "test.go")
	if err := os.WriteFile(tempFile, []byte("package main"), 0o600); err != nil {
		t.Fatal(err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Update formatter command for this test
			formatter.Command = tt.command
			err := formatter.formatFile(tempFile)

			if tt.expectError && err == nil {
				t.Errorf("formatFile() expected error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("formatFile() expected no error, got %v", err)
			}
		})
	}
}
