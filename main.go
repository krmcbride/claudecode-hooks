package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"mvdan.cc/sh/v3/syntax"
)

type HookInput struct {
	ToolName  string `json:"tool_name"`
	ToolInput struct {
		Command string `json:"command"`
	} `json:"tool_input"`
}

// Analyze command using proper shell parsing
func analyzeCommand(command string) (bool, []string) {
	var issues []string

	// Parse the shell command into an AST
	parser := syntax.NewParser()
	node, err := parser.Parse(strings.NewReader(command), "")
	if err != nil {
		// If we can't parse it, be cautious
		issues = append(issues, fmt.Sprintf("Failed to parse command: %v", err))
		return true, issues // Block unparseable commands
	}

	// Walk the AST looking for dangerous patterns
	hasGitPush := false
	syntax.Walk(node, func(node syntax.Node) bool {
		switch n := node.(type) {
		case *syntax.CallExpr:
			// Check if this is a command call
			if len(n.Args) > 0 {
				if word, ok := n.Args[0].(*syntax.Word); ok {
					cmd := wordToString(word)

					// Check for git push patterns
					if cmd == "git" && len(n.Args) > 1 {
						if word2, ok := n.Args[1].(*syntax.Word); ok {
							arg := wordToString(word2)
							if arg == "push" {
								hasGitPush = true
								issues = append(issues, "Detected 'git push' command")
							}
						}
					}

					// Check for command substitution containing git push
					if cmd == "bash" || cmd == "sh" {
						for _, arg := range n.Args[1:] {
							if word, ok := arg.(*syntax.Word); ok {
								argStr := wordToString(word)
								if strings.Contains(argStr, "git") && strings.Contains(argStr, "push") {
									hasGitPush = true
									issues = append(issues, "Detected 'git push' in subshell")
								}
							}
						}
					}

					// Check other dangerous patterns
					if cmd == "eval" || cmd == "exec" {
						for _, arg := range n.Args[1:] {
							if word, ok := arg.(*syntax.Word); ok {
								argStr := wordToString(word)
								if strings.Contains(argStr, "git") && strings.Contains(argStr, "push") {
									hasGitPush = true
									issues = append(issues, fmt.Sprintf("Detected 'git push' in %s", cmd))
								}
							}
						}
					}
				}
			}
		case *syntax.CmdSubst:
			// Check command substitutions $(...)
			if n.Stmts != nil {
				for _, stmt := range n.Stmts {
					// Recursively analyze command substitutions
					// (This is a simplified check - could be more thorough)
					stmtStr := stmtToString(stmt)
					if strings.Contains(stmtStr, "git") && strings.Contains(stmtStr, "push") {
						hasGitPush = true
						issues = append(issues, "Detected 'git push' in command substitution")
					}
				}
			}
		}
		return true // Continue walking
	})

	return hasGitPush, issues
}

// Convert Word node to string (simplified)
func wordToString(word *syntax.Word) string {
	var result strings.Builder
	for _, part := range word.Parts {
		switch p := part.(type) {
		case *syntax.Lit:
			result.WriteString(p.Value)
		case *syntax.SglQuoted:
			result.WriteString(p.Value)
		case *syntax.DblQuoted:
			for _, dqPart := range p.Parts {
				if lit, ok := dqPart.(*syntax.Lit); ok {
					result.WriteString(lit.Value)
				}
			}
		}
	}
	return result.String()
}

// Convert statement to string (simplified)
func stmtToString(stmt *syntax.Stmt) string {
	// This is a simplified conversion - in practice you'd want more thorough handling
	return fmt.Sprintf("%v", stmt)
}

func main() {
	// Read JSON input from stdin
	var input HookInput
	decoder := json.NewDecoder(os.Stdin)
	if err := decoder.Decode(&input); err != nil {
		log.Printf("Failed to decode JSON: %v", err)
		os.Exit(0) // Allow execution if we can't parse input
	}

	// Only process Bash commands
	if input.ToolName != "Bash" {
		os.Exit(0)
	}

	command := input.ToolInput.Command
	if command == "" {
		os.Exit(0)
	}

	// Analyze the command using proper shell parsing
	shouldBlock, issues := analyzeCommand(command)

	if shouldBlock {
		fmt.Fprintf(os.Stderr, "🚫 BLOCKED: Detected git push command!\n")
		fmt.Fprintf(os.Stderr, "Command: %s\n", command)
		for _, issue := range issues {
			fmt.Fprintf(os.Stderr, "Issue: %s\n", issue)
		}
		os.Exit(2) // Block execution
	}

	// Allow execution
	os.Exit(0)
}
