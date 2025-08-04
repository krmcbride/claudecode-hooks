// Package common provides shared shell parsing utilities.
package common

import (
	"fmt"
	"strings"

	"mvdan.cc/sh/v3/syntax"
)

// ParseCommand parses a shell command and extracts command calls
func ParseCommand(command string) ([]*syntax.CallExpr, error) {
	parser := syntax.NewParser()
	node, err := parser.Parse(strings.NewReader(command), "")
	if err != nil {
		return nil, fmt.Errorf("failed to parse command: %w", err)
	}

	var calls []*syntax.CallExpr
	syntax.Walk(node, func(node syntax.Node) bool {
		if call, ok := node.(*syntax.CallExpr); ok {
			calls = append(calls, call)
		}
		return true
	})

	return calls, nil
}

// WordToString converts a syntax.Word to string
func WordToString(word *syntax.Word) string {
	if word == nil {
		return ""
	}
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

// GetCommandName extracts the command name from a CallExpr
func GetCommandName(call *syntax.CallExpr) string {
	if len(call.Args) == 0 {
		return ""
	}
	return WordToString(call.Args[0])
}

// GetCommandArgs extracts command arguments from a CallExpr
func GetCommandArgs(call *syntax.CallExpr) []string {
	if len(call.Args) <= 1 {
		return nil
	}
	
	args := make([]string, 0, len(call.Args)-1)
	for _, arg := range call.Args[1:] {
		args = append(args, WordToString(arg))
	}
	return args
}