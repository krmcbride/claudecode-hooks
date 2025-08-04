package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/krmcbride/claudecode-hooks/pkg/common"
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
		input    *common.PostToolUseInput
		expected bool
	}{
		{
			name: "Successful Edit",
			input: &common.PostToolUseInput{
				ToolName: "Edit",
				ToolResponse: struct {
					FilePath string `json:"filePath,omitempty"`
					Success  bool   `json:"success"`
				}{
					Success: true,
				},
			},
			expected: true,
		},
		{
			name: "Successful MultiEdit",
			input: &common.PostToolUseInput{
				ToolName: "MultiEdit",
				ToolResponse: struct {
					FilePath string `json:"filePath,omitempty"`
					Success  bool   `json:"success"`
				}{
					Success: true,
				},
			},
			expected: true,
		},
		{
			name: "Failed Edit",
			input: &common.PostToolUseInput{
				ToolName: "Edit",
				ToolResponse: struct {
					FilePath string `json:"filePath,omitempty"`
					Success  bool   `json:"success"`
				}{
					Success: false,
				},
			},
			expected: false,
		},
		{
			name: "Wrong tool",
			input: &common.PostToolUseInput{
				ToolName: "Write",
				ToolResponse: struct {
					FilePath string `json:"filePath,omitempty"`
					Success  bool   `json:"success"`
				}{
					Success: true,
				},
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

func TestFileFormatter_collectFilePaths(t *testing.T) {
	formatter := NewFileFormatter("echo test", []string{".go"}, false)

	tests := []struct {
		name     string
		input    *common.PostToolUseInput
		expected []string
	}{
		{
			name: "Edit with single file",
			input: &common.PostToolUseInput{
				ToolName: "Edit",
				ToolInput: struct {
					FilePath string `json:"file_path"`
					Content  string `json:"content,omitempty"`
					Edits    []struct {
						FilePath string `json:"file_path"`
					} `json:"edits,omitempty"`
				}{
					FilePath: "main.go",
				},
			},
			expected: []string{"main.go"},
		},
		{
			name: "MultiEdit with multiple files",
			input: &common.PostToolUseInput{
				ToolName: "MultiEdit",
				ToolInput: struct {
					FilePath string `json:"file_path"`
					Content  string `json:"content,omitempty"`
					Edits    []struct {
						FilePath string `json:"file_path"`
					} `json:"edits,omitempty"`
				}{
					FilePath: "main.go",
					Edits: []struct {
						FilePath string `json:"file_path"`
					}{
						{FilePath: "utils.go"},
						{FilePath: "config.go"},
					},
				},
			},
			expected: []string{"utils.go", "config.go", "main.go"},
		},
		{
			name: "MultiEdit with duplicate files",
			input: &common.PostToolUseInput{
				ToolName: "MultiEdit",
				ToolInput: struct {
					FilePath string `json:"file_path"`
					Content  string `json:"content,omitempty"`
					Edits    []struct {
						FilePath string `json:"file_path"`
					} `json:"edits,omitempty"`
				}{
					FilePath: "main.go",
					Edits: []struct {
						FilePath string `json:"file_path"`
					}{
						{FilePath: "main.go"},
						{FilePath: "utils.go"},
					},
				},
			},
			expected: []string{"main.go", "utils.go"},
		},
		{
			name: "Edit with empty file path",
			input: &common.PostToolUseInput{
				ToolName: "Edit",
				ToolInput: struct {
					FilePath string `json:"file_path"`
					Content  string `json:"content,omitempty"`
					Edits    []struct {
						FilePath string `json:"file_path"`
					} `json:"edits,omitempty"`
				}{
					FilePath: "",
				},
			},
			expected: nil, // Empty slice is returned as nil
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatter.collectFilePaths(tt.input)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("collectFilePaths() = %v, want %v", result, tt.expected)
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

func TestFileFormatter_filterAndValidateFiles(t *testing.T) {
	formatter := NewFileFormatter("echo test", []string{".go", ".js"}, false)

	tests := []struct {
		name      string
		filePaths []string
		expected  []string
	}{
		{
			name:      "Filter by extension",
			filePaths: []string{"test.go", "test.js", "test.txt"},
			expected:  []string{"test.go", "test.js"},
		},
		{
			name:      "Include any path with correct extension",
			filePaths: []string{"test.go", "../other/file.go", "/absolute/path/file.js"},
			expected:  []string{"test.go", "../other/file.go", "/absolute/path/file.js"},
		},
		{
			name:      "Empty input",
			filePaths: []string{},
			expected:  nil, // Empty slice is returned as nil
		},
		{
			name:      "No matching extensions",
			filePaths: []string{"test.txt", "config.yaml"},
			expected:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatter.filterAndValidateFiles(tt.filePaths)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("filterAndValidateFiles(%v) = %v, want %v", tt.filePaths, result, tt.expected)
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
	if err := os.WriteFile(tempFile, []byte("package main"), 0o644); err != nil {
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
