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
	if err := os.WriteFile(testGoFile, []byte("package main\n\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(testJsFile, []byte("console.log('test');\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(testTxtFile, []byte("test content\n"), 0o644); err != nil {
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
			name:      "Edit processing with echo command",
			formatter: NewFileFormatter("echo formatted", []string{".go"}, false),
			input: &hook.PostToolUseInput{
				ToolName: "Edit",
				ToolInput: struct {
					FilePath string `json:"file_path"`
					Content  string `json:"content,omitempty"`
					Edits    []struct {
						FilePath string `json:"file_path"`
					} `json:"edits,omitempty"`
				}{
					FilePath: testGoFile,
				},
				ToolResponse: struct {
					FilePath string `json:"filePath,omitempty"`
					Success  bool   `json:"success"`
				}{
					Success: true,
				},
			},
			expectError:   false, // echo command should succeed
			expectedFiles: []string{testGoFile},
		},
		{
			name:      "Skip failed operation",
			formatter: NewFileFormatter("echo formatted", []string{".go"}, false),
			input: &hook.PostToolUseInput{
				ToolName: "Edit",
				ToolInput: struct {
					FilePath string `json:"file_path"`
					Content  string `json:"content,omitempty"`
					Edits    []struct {
						FilePath string `json:"file_path"`
					} `json:"edits,omitempty"`
				}{
					FilePath: testGoFile,
				},
				ToolResponse: struct {
					FilePath string `json:"filePath,omitempty"`
					Success  bool   `json:"success"`
				}{
					Success: false,
				},
			},
			expectError:   false,
			expectedFiles: nil, // No files because operation failed
		},
		{
			name:      "Skip wrong tool",
			formatter: NewFileFormatter("echo formatted", []string{".go"}, false),
			input: &hook.PostToolUseInput{
				ToolName: "Read",
				ToolInput: struct {
					FilePath string `json:"file_path"`
					Content  string `json:"content,omitempty"`
					Edits    []struct {
						FilePath string `json:"file_path"`
					} `json:"edits,omitempty"`
				}{
					FilePath: testGoFile,
				},
				ToolResponse: struct {
					FilePath string `json:"filePath,omitempty"`
					Success  bool   `json:"success"`
				}{
					Success: true,
				},
			},
			expectError:   false,
			expectedFiles: nil, // No files because wrong tool
		},
		{
			name:      "Filter by extension",
			formatter: NewFileFormatter("echo formatted", []string{".go"}, false),
			input: &hook.PostToolUseInput{
				ToolName: "MultiEdit",
				ToolInput: struct {
					FilePath string `json:"file_path"`
					Content  string `json:"content,omitempty"`
					Edits    []struct {
						FilePath string `json:"file_path"`
					} `json:"edits,omitempty"`
				}{
					Edits: []struct {
						FilePath string `json:"file_path"`
					}{
						{FilePath: testGoFile},
						{FilePath: testJsFile},
						{FilePath: testTxtFile},
					},
				},
				ToolResponse: struct {
					FilePath string `json:"filePath,omitempty"`
					Success  bool   `json:"success"`
				}{
					Success: true,
				},
			},
			expectError:   false, // echo command should succeed
			expectedFiles: []string{testGoFile},
		},
		{
			name:      "No files to format",
			formatter: NewFileFormatter("echo formatted", []string{".rs"}, false),
			input: &hook.PostToolUseInput{
				ToolName: "Edit",
				ToolInput: struct {
					FilePath string `json:"file_path"`
					Content  string `json:"content,omitempty"`
					Edits    []struct {
						FilePath string `json:"file_path"`
					} `json:"edits,omitempty"`
				}{
					FilePath: testGoFile,
				},
				ToolResponse: struct {
					FilePath string `json:"filePath,omitempty"`
					Success  bool   `json:"success"`
				}{
					Success: true,
				},
			},
			expectError:   false,
			expectedFiles: nil, // No matching extensions
		},
		{
			name:      "Block on failure enabled",
			formatter: NewFileFormatter("nonexistent-command-12345", []string{".go"}, true),
			input: &hook.PostToolUseInput{
				ToolName: "Edit",
				ToolInput: struct {
					FilePath string `json:"file_path"`
					Content  string `json:"content,omitempty"`
					Edits    []struct {
						FilePath string `json:"file_path"`
					} `json:"edits,omitempty"`
				}{
					FilePath: testGoFile,
				},
				ToolResponse: struct {
					FilePath string `json:"filePath,omitempty"`
					Success  bool   `json:"success"`
				}{
					Success: true,
				},
			},
			expectError:   true, // Should error because command doesn't exist and BlockOnFail=true
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

			// Verify that the correct files would be processed (only if we should process input)
			if tt.formatter.shouldProcessInput(tt.input) {
				filesToFormat := tt.formatter.getFilesToFormat(tt.input)
				if len(filesToFormat) != len(tt.expectedFiles) {
					t.Errorf("Expected %d files to format, got %d", len(tt.expectedFiles), len(filesToFormat))
				}

				// Check that expected files are in the list
				fileMap := make(map[string]bool)
				for _, file := range filesToFormat {
					fileMap[file] = true
				}
				for _, expectedFile := range tt.expectedFiles {
					if !fileMap[expectedFile] {
						t.Errorf("Expected file %s not found in files to format", expectedFile)
					}
				}
			} else {
				// If we shouldn't process input, there should be no expected files
				if len(tt.expectedFiles) != 0 {
					t.Errorf("Expected no files for skipped input, but test expects %d files", len(tt.expectedFiles))
				}
			}
		})
	}
}

func TestFileFormatter_ProcessInput_WithAllowedCommand(t *testing.T) {
	// Create temporary test file
	tempDir := t.TempDir()
	testGoFile := filepath.Join(tempDir, "test.go")

	if err := os.WriteFile(testGoFile, []byte("package main\n\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Use gofmt which should be available and allowed
	formatter := NewFileFormatter("gofmt", []string{".go"}, false)

	input := &hook.PostToolUseInput{
		ToolName: "Edit",
		ToolInput: struct {
			FilePath string `json:"file_path"`
			Content  string `json:"content,omitempty"`
			Edits    []struct {
				FilePath string `json:"file_path"`
			} `json:"edits,omitempty"`
		}{
			FilePath: testGoFile,
		},
		ToolResponse: struct {
			FilePath string `json:"filePath,omitempty"`
			Success  bool   `json:"success"`
		}{
			Success: true,
		},
	}

	err := formatter.ProcessInput(input)
	if err != nil {
		t.Errorf("ProcessInput() with gofmt should not error, got %v", err)
	}
}

func TestFileFormatter_ProcessInput_NonExistentFile(t *testing.T) {
	formatter := NewFileFormatter("gofmt", []string{".go"}, false)

	input := &hook.PostToolUseInput{
		ToolName: "Edit",
		ToolInput: struct {
			FilePath string `json:"file_path"`
			Content  string `json:"content,omitempty"`
			Edits    []struct {
				FilePath string `json:"file_path"`
			} `json:"edits,omitempty"`
		}{
			FilePath: "nonexistent.go",
		},
		ToolResponse: struct {
			FilePath string `json:"filePath,omitempty"`
			Success  bool   `json:"success"`
		}{
			Success: true,
		},
	}

	// Should attempt to format the file - let gofmt handle the error
	err := formatter.ProcessInput(input)
	if err != nil {
		t.Errorf("ProcessInput() with non-existent file should not error (BlockOnFail=false), got %v", err)
	}

	// Verify file is included for formatting
	filesToFormat := formatter.getFilesToFormat(input)
	if len(filesToFormat) != 1 || filesToFormat[0] != "nonexistent.go" {
		t.Errorf("Expected 1 file to format [nonexistent.go], got %v", filesToFormat)
	}
}
