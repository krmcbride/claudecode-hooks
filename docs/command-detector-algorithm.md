# Command Detector Algorithm Documentation

## Overview

The `CommandDetector` in the `pkg/detector` package implements a multi-layered security analysis system for detecting and blocking potentially dangerous shell commands. It provides an **additional layer of security** on top of Claude Code's built-in deny permissions, using configurable rules and pattern matching to provide defense-in-depth command filtering for Claude Code hooks.

### Package Organization

The detector package is organized into focused, single-purpose files:

- **detector.go** - Core detector struct and main entry points
- **direct_check.go** - Direct command checking and rule matching
- **arguments_check.go** - Checking arguments for blocked commands
- **string_literals_check.go** - String literal analysis for embedded commands
- **obfuscation_check.go** - Obfuscation detection techniques
- **shellparse.go** - Shell parsing utilities
- **command_utils.go** - Command matching utilities
- **pattern_utils.go** - Shared pattern matching utilities

## Core Components

### 1. Simplified Universal Approach

The detector operates with a streamlined, universal approach:

- Pattern matching for direct commands
- Universal analysis of command arguments as potential blocked commands
- Context-aware string literal analysis for shell execution patterns
- Obfuscation detection (base64, hex, escaping)
- Recursive analysis of nested commands
- Dynamic content blocking (variables, substitutions)

### 2. Command Rules

Each rule defines:

- **BlockedCommand**: Primary command to monitor (e.g., "git", "aws", "kubectl")
- **BlockedPatterns**: Subcommand patterns to block (e.g., "push", "delete", "*" for all)

### 3. Detection Philosophy

**Key Insight**: Since these hooks only run on Bash tool calls (not Write/Edit tools), any string literal or command argument could potentially be executed. The detector leverages this to provide comprehensive coverage without needing special knowledge of specific command patterns.

## Algorithm Flow

### Main Entry Point: `ShouldBlockShellExpr()`

```
1. Reset state (clear issues, reset depth counter)
2. Call analyzeShellExprRecursive()
3. Return true if any issues detected (blocks command)
```

### Recursive Analysis: `analyzeShellExprRecursive()`

```
1. Recursion Depth Check
   ├─ Increment depth counter
   ├─ If depth > maxDepth → Block (DoS protection)
   └─ Defer decrement on return

2. Command Parsing
   ├─ Parse command using shell parser
   └─ If parse fails → Block (conservative approach)

3. Expression Analysis
   └─ For each call expression → shouldBlockCallExpr()
```

### Call Expression Analysis: `shouldBlockCallExpr()`

This is the core detection logic that evaluates whether each command in the parsed shell expression should be blocked:

```
1. Command Resolution
   ├─ Extract command from call arguments
   └─ Determine if command is static or dynamic

2. Dynamic Command Check
   ├─ If command contains variables/substitutions → BLOCK
   └─ Always blocks dynamic content

3. Direct Command Check
   ├─ Match against configured rules
   └─ Check blocked patterns (including wildcard support)

4. Arguments as Commands Check (NEW)
   ├─ Check if any argument is itself a blocked command
   ├─ Handles: xargs git push, find -exec git push, etc.
   └─ No special knowledge of xargs/find needed

5. String Literal Analysis (SIMPLIFIED)
   ├─ Context-aware: Only for shell interpreters, eval, echo/printf
   ├─ Recursively analyze string literals as shell expressions
   └─ Smart filtering to avoid false positives

6. Obfuscation Detection
   ├─ Base64/hex encoding detection
   ├─ Character escaping detection
   └─ Echo with escape sequences
```

### Direct Command Checking: `checkDirectCommand()`

```
1. Rule Iteration
   └─ For each configured rule:

2. Command Matching
   ├─ Check if command matches rule.BlockedCommand
   └─ Skip if no match

3. Argument Processing
   ├─ Extract all arguments after command
   ├─ Check for dynamic subcommands
   └─ Block if dynamic (maximum security)

4. Pattern Evaluation
   ├─ Join arguments into full string
   ├─ Check blocked patterns (block if match)
   └─ Special handling for wildcard (*) patterns
```

### Arguments as Commands Detection: `checkArgumentsForBlockedCommands()`

This universal approach catches commands used as arguments to other commands:

```
1. Iterate through all arguments (skip command itself)
2. For each argument:
   ├─ Resolve to static string
   ├─ Check if it matches any blocked command
   └─ If match, check remaining args for blocked patterns
3. Block if blocked command + pattern found
```

This single mechanism handles:
- `xargs git push` - git is an argument to xargs
- `find . -exec git push` - git follows -exec
- `parallel git push ::: args` - git is an argument
- Any future command execution pattern

### String Literal Analysis: `analyzeStringLiterals()`

Context-aware analysis based on the command type:

```
1. Identify command context:
   ├─ Shell interpreters (sh, bash, zsh): Analyze -c arguments
   ├─ Eval commands (eval, source): Analyze evaluated strings
   └─ Echo/printf: Only analyze if likely executable

2. For relevant contexts:
   ├─ Extract string literals from arguments
   ├─ Filter based on context (strict for echo, broad for eval)
   └─ Recursively analyze as shell expressions

3. Smart filtering:
   ├─ looksLikeCommand(): Basic command detection
   └─ definitelyLooksLikeCommand(): Strict for echo/printf
```

## Detection Examples

### Example 1: Direct Command
```
Input: "git push origin main"
Detection: Direct command match → BLOCKED
```

### Example 2: Command as Argument (xargs)
```
Input: "xargs git push"
Detection: "git" detected as argument, "push" matches pattern → BLOCKED
No special xargs knowledge needed!
```

### Example 3: Shell Interpreter
```
Input: sh -c "git push"
Detection: Shell interpreter context, string literal analyzed → BLOCKED
```

### Example 4: Find with -exec
```
Input: "find . -exec git push {} \;"
Detection: "git" detected as argument, "push" matches pattern → BLOCKED
No special find knowledge needed!
```

### Example 5: Echo to Shell
```
Input: echo "git push origin" | bash
Detection: Echo context with executable string → BLOCKED
```

### Example 6: Safe Echo
```
Input: echo "Remember to git push later"
Detection: Echo context, doesn't look executable → ALLOWED
```

## Key Improvements in Simplified Approach

### 1. Universal Argument Analysis
- **Before**: Special handlers for xargs, find, parallel, etc.
- **After**: Single mechanism checks all arguments as potential commands
- **Benefit**: Catches any command execution pattern, even unknown ones

### 2. Context-Aware String Analysis
- **Before**: Complex special cases for different command types
- **After**: Smart context detection with appropriate filtering
- **Benefit**: Reduces false positives while maintaining security

### 3. Reduced Complexity
- **Before**: ~200 lines of special-case handlers
- **After**: Two simple, universal mechanisms
- **Benefit**: Easier to maintain, understand, and verify

## Security Approach

The detector maintains maximum security while reducing complexity:

- Universal argument checking catches all command-as-argument patterns
- Context-aware string analysis prevents bypass through shell evaluation
- Recursive analysis catches nested commands at any depth
- Dynamic content always blocked for safety
- Obfuscation detection prevents encoding bypasses

## Recursion Protection

The detector implements DoS protection through recursion limits:

- Default max depth: 10
- Configurable via constructor
- Prevents infinite loops from malicious input
- Returns error if depth exceeded

## Issue Reporting

The detector collects detailed issue information:

- Specific pattern matched
- Type of threat detected
- Location in command structure
- All issues accessible via `GetIssues()`

## Performance Considerations

1. **Simplified Logic**: Fewer code paths mean faster execution
2. **Smart Filtering**: Context-aware analysis reduces unnecessary parsing
3. **Early Returns**: Fail-fast approach for common cases
4. **Recursion Limits**: Prevents DoS from deeply nested commands

## Integration Points

The CommandDetector integrates with:

- **Shell parsing**: Using mvdan.cc/sh/v3/syntax for accurate AST analysis
- **Pattern matching**: Flexible wildcard and glob pattern support
- **Obfuscation detection**: Base64, hex, and escape sequence detection

## Best Practices

1. **Configure Specific Rules**:
   - Be specific with blocked patterns or use wildcards (*) for broad blocking
   - Use multiple -cmd flags for different commands
   - Test rules thoroughly

2. **Monitor Issues**:
   - Check `GetIssues()` for detailed blocking reasons
   - Log issues for security auditing
   - Adjust rules based on false positives

## Limitations

1. **Parser limitations**: Extremely complex shell constructs may not parse correctly
2. **Sophisticated obfuscation**: Advanced obfuscation techniques may evade detection
3. **Dynamic evaluation**: Cannot predict runtime variable values

## Conclusion

The simplified CommandDetector provides robust security through universal principles rather than special cases. By treating all arguments as potential commands and using context-aware string analysis, it achieves comprehensive coverage with less code. This approach is more maintainable, easier to verify, and actually catches more cases than the previous special-case handlers. The detector serves as an effective additional security layer beyond Claude Code's built-in permissions, providing defense-in-depth protection for command execution scenarios.