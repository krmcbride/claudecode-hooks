package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/krmcbride/claudecode-hooks/pkg/hook"
)

func TestFileFormatter_ProcessInput_Integration(t *testing.T) {
	// Create temporary test files
	tempDir := t.TempDir()
	testGoFile := filepath.Join(tempDir, "test.go")
	testJsFile := filepath.Join(tempDir, "test.js")
	testTxtFile := filepath.Join(tempDir, "test.txt")

	// Create the files
	if err := os.WriteFile(testGoFile, []byte("package main\n\nfunc main() {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(testJsFile, []byte("console.log('test');\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(testTxtFile, []byte("test content\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name          string
		formatter     *FileFormatter
		input         *hook.PostToolUseInput
		expectError   bool
		expectedFiles []string
	}{
		{
			name:      "Edit processing with Go file",
			formatter: NewFileFormatter("echo formatted", []string{".go"}, false),
			input: &hook.PostToolUseInput{
				ToolName: "Edit",
				ToolInput: struct {
					FilePath string `json:"file_path"`
				}{
					FilePath: testGoFile,
				},
			},
			expectError:   false,
			expectedFiles: []string{testGoFile},
		},
		{
			name:      "MultiEdit processing with JS file",
			formatter: NewFileFormatter("echo formatted", []string{".js"}, false),
			input: &hook.PostToolUseInput{
				ToolName: "MultiEdit",
				ToolInput: struct {
					FilePath string `json:"file_path"`
				}{
					FilePath: testJsFile,
				},
			},
			expectError:   false,
			expectedFiles: []string{testJsFile},
		},
		{
			name:      "Write processing with Go file",
			formatter: NewFileFormatter("echo formatted", []string{".go"}, false),
			input: &hook.PostToolUseInput{
				ToolName: "Write",
				ToolInput: struct {
					FilePath string `json:"file_path"`
				}{
					FilePath: testGoFile,
				},
			},
			expectError:   false,
			expectedFiles: []string{testGoFile},
		},
		{
			name:      "Skip file with wrong extension",
			formatter: NewFileFormatter("echo formatted", []string{".go"}, false),
			input: &hook.PostToolUseInput{
				ToolName: "Edit",
				ToolInput: struct {
					FilePath string `json:"file_path"`
				}{
					FilePath: testTxtFile,
				},
			},
			expectError:   false,
			expectedFiles: nil, // File should be skipped
		},
		{
			name:      "Skip unsupported tool",
			formatter: NewFileFormatter("echo formatted", []string{".go"}, false),
			input: &hook.PostToolUseInput{
				ToolName: "Read",
				ToolInput: struct {
					FilePath string `json:"file_path"`
				}{
					FilePath: testGoFile,
				},
			},
			expectError:   false,
			expectedFiles: nil, // Tool should be skipped
		},
		{
			name:      "Block on format failure",
			formatter: NewFileFormatter("nonexistent-command-12345", []string{".go"}, true),
			input: &hook.PostToolUseInput{
				ToolName: "Edit",
				ToolInput: struct {
					FilePath string `json:"file_path"`
				}{
					FilePath: testGoFile,
				},
			},
			expectError:   true,
			expectedFiles: []string{testGoFile},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.formatter.ProcessInput(tt.input)

			if tt.expectError && err == nil {
				t.Errorf("ProcessInput() expected error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("ProcessInput() expected no error, got %v", err)
			}
		})
	}
}

func TestFileFormatter_ProcessInput_EmptyFilePath(t *testing.T) {
	formatter := NewFileFormatter("echo test", []string{".go"}, false)

	input := &hook.PostToolUseInput{
		ToolName: "Edit",
		ToolInput: struct {
			FilePath string `json:"file_path"`
		}{
			FilePath: "",
		},
	}

	// Should not error, just skip processing
	err := formatter.ProcessInput(input)
	if err != nil {
		t.Errorf("ProcessInput() with empty filepath should not error, got %v", err)
	}
}

func TestFileFormatter_ProcessInput_PlaceholderExpansion(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.go")
	if err := os.WriteFile(testFile, []byte("package main"), 0o600); err != nil {
		t.Fatal(err)
	}

	// Test with {FILEPATH} placeholder
	formatter := NewFileFormatter("echo {FILEPATH}", []string{".go"}, false)
	input := &hook.PostToolUseInput{
		ToolName: "Edit",
		ToolInput: struct {
			FilePath string `json:"file_path"`
		}{
			FilePath: testFile,
		},
	}

	err := formatter.ProcessInput(input)
	if err != nil {
		t.Errorf("ProcessInput() with placeholder should not error, got %v", err)
	}
}
