// Package main provides a hook logger for debugging Claude Code hook payloads.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

func main() {
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
		fmt.Fprintf(os.Stderr, "Error parsing JSON: %v\n", err)
		// Output raw input
		fmt.Printf("HOOK_PAYLOAD_RAW: %s\n", string(input))
		os.Exit(0) // Don't block the operation
	}

	// Pretty print the JSON
	prettyJSON, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error formatting JSON: %v\n", err)
		fmt.Printf("HOOK_PAYLOAD_RAW: %s\n", string(input))
		os.Exit(0)
	}

	// Output to stdout - this should appear in Claude's context
	fmt.Printf("=== HOOK PAYLOAD ===\n%s\n===================\n", string(prettyJSON))

	// Always exit 0 to not block operations
	os.Exit(0)
}
