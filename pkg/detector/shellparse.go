// Package detector - internal shell parsing utilities
package detector

import (
	"fmt"
	"strings"

	"mvdan.cc/sh/v3/syntax"
)

// parseShellExpression parses a shell expression into an Abstract Syntax Tree.
// The input shellExpr can be a simple command ("ls -la") or a complex expression
// with pipes, conditionals, loops, and subshells ("cd /tmp && git pull || echo failed").
// Returns the AST root node which can be traversed to extract various elements
// like command calls, redirections, variables, etc.
func parseShellExpression(shellExpr string) (syntax.Node, error) {
	parser := syntax.NewParser()
	node, err := parser.Parse(strings.NewReader(shellExpr), "")
	if err != nil {
		return nil, fmt.Errorf("failed to parse shell expression: %w", err)
	}
	return node, nil
}

// extractCallExprs walks the AST and collects all command call expressions.
// These represent actual command invocations (e.g., "git push", "echo hello").
// The traversal is depth-first, capturing commands in nested structures like
// subshells, conditionals, and loops.
func extractCallExprs(node syntax.Node) []*syntax.CallExpr {
	var calls []*syntax.CallExpr
	syntax.Walk(node, func(n syntax.Node) bool {
		if call, ok := n.(*syntax.CallExpr); ok {
			calls = append(calls, call)
		}
		return true // Continue traversing into child nodes
	})
	return calls
}

// resolveStaticWord attempts to resolve a word into a static string.
// It returns the resolved string and a boolean indicating if the resolution is complete
// (i.e., the word contained no dynamic parts like variables or command substitutions).
func resolveStaticWord(word *syntax.Word) (val string, isStatic bool) {
	if word == nil {
		return "", true
	}

	var sb strings.Builder
	isStatic = true

	for _, part := range word.Parts {
		switch p := part.(type) {
		case *syntax.Lit:
			sb.WriteString(p.Value)
		case *syntax.SglQuoted:
			sb.WriteString(p.Value)
		case *syntax.DblQuoted:
			// Handle parts inside double quotes
			for _, subPart := range p.Parts {
				switch sp := subPart.(type) {
				case *syntax.Lit:
					sb.WriteString(sp.Value)
				case *syntax.ParamExp:
					// Variable expansion makes it dynamic
					isStatic = false
					// For partial resolution, we could try to handle simple cases
					// but for safety, we'll mark it as dynamic
				case *syntax.CmdSubst:
					// Command substitution makes it dynamic
					isStatic = false
				case *syntax.ArithmExp:
					// Arithmetic expansion makes it dynamic
					isStatic = false
				default:
					// Any other dynamic element
					isStatic = false
				}
			}
		case *syntax.ParamExp:
			// Variable expansion outside quotes
			isStatic = false
		case *syntax.CmdSubst:
			// Command substitution outside quotes
			isStatic = false
		case *syntax.ArithmExp:
			// Arithmetic expansion outside quotes
			isStatic = false
		case *syntax.ProcSubst:
			// Process substitution
			isStatic = false
		default:
			// Any other dynamic element
			isStatic = false
		}
	}

	return sb.String(), isStatic
}
