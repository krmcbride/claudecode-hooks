// Package main implements a Claude Code hook to format files after editing.
package main

import (
	"flag"
	"log"
	"os"

	"github.com/krmcbride/claudecode-hooks/pkg/hook"
	"github.com/krmcbride/claudecode-hooks/pkg/utils"
)

func main() {
	// Parse command-line flags
	var (
		formatCommand  = flag.String("cmd", "", "Format command to run (required)")
		extensionsFlag = flag.String("ext", "", "Comma-separated file extensions to process (required)")
		blockOnFailure = flag.Bool("block", false, "Block on formatting failures")
		showHelp       = flag.Bool("help", false, "Show help message")
	)
	flag.Parse()

	// Show help if requested
	if *showHelp {
		flag.Usage()
		os.Exit(0)
	}

	// Validate required flags
	if *formatCommand == "" {
		log.Fatal("Error: -cmd flag is required")
	}
	if *extensionsFlag == "" {
		log.Fatal("Error: -ext flag is required")
	}

	// Read input
	input, err := hook.ReadPostToolUseInput()
	if err != nil {
		log.Printf("Failed to decode JSON: %v", err)
		hook.AllowPostToolUse()
	}

	// Create formatter and process input
	extensions := utils.ParseCommaSeparated(*extensionsFlag)
	formatter := NewFileFormatter(*formatCommand, extensions, *blockOnFailure)

	if err := formatter.ProcessInput(input); err != nil {
		hook.BlockPostToolUse("File formatting failed")
	}

	hook.AllowPostToolUse()
}
