// Package main provides a hook logger for debugging Claude Code hook payloads.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func main() {
	// Parse command-line flags
	silent := flag.Bool("silent", false, "Suppress stdout output (for logging only)")
	logFile := flag.String("log", "", "Log file path (if not specified, outputs to stdout)")
	flag.Parse()

	// Read JSON input from stdin
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
		os.Exit(1)
	}

	// Parse JSON to pretty print it
	var data any
	err = json.Unmarshal(input, &data)
	if err != nil {
		if !*silent {
			fmt.Fprintf(os.Stderr, "Error parsing JSON: %v\n", err)
			// Output raw input
			fmt.Printf("HOOK_PAYLOAD_RAW: %s\n", string(input))
		}
		os.Exit(0) // Don't block the operation
	}

	// Pretty print the JSON
	prettyJSON, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		if !*silent {
			fmt.Fprintf(os.Stderr, "Error formatting JSON: %v\n", err)
			fmt.Printf("HOOK_PAYLOAD_RAW: %s\n", string(input))
		}
		os.Exit(0)
	}

	// Format output
	output := fmt.Sprintf("=== HOOK PAYLOAD ===\n%s\n===================\n", string(prettyJSON))

	// Output to log file or stdout
	if *logFile != "" {
		// Ensure directory exists
		dir := filepath.Dir(*logFile)
		if err := os.MkdirAll(dir, 0o750); err != nil {
			if !*silent {
				fmt.Fprintf(os.Stderr, "Error creating log directory: %v\n", err)
			}
			os.Exit(0)
		}

		// Append to log file
		f, err := os.OpenFile(*logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
		if err != nil {
			if !*silent {
				fmt.Fprintf(os.Stderr, "Error opening log file: %v\n", err)
			}
			os.Exit(0)
		}
		defer func() {
			if err := f.Close(); err != nil && !*silent {
				fmt.Fprintf(os.Stderr, "Error closing log file: %v\n", err)
			}
		}()

		if _, err := f.WriteString(output); err != nil {
			if !*silent {
				fmt.Fprintf(os.Stderr, "Error writing to log file: %v\n", err)
			}
			os.Exit(0)
		}
	} else if !*silent {
		// Output to stdout only if not silent
		fmt.Print(output)
	}

	// Always exit 0 to not block operations
	os.Exit(0)
}
