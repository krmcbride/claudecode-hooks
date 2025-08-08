package detector

import (
	"testing"
)

func TestStringLiteralAnalysis(t *testing.T) {
	rules := []CommandRule{
		{BlockedCommand: "git", BlockedPatterns: []string{"push"}},
		{BlockedCommand: "aws", BlockedPatterns: []string{"delete-*", "terminate-*"}},
	}

	tests := []struct {
		name        string
		command     string
		shouldBlock bool
		description string
	}{
		// Shell interpreters with blocked commands
		{
			name:        "sh -c with git push",
			command:     `sh -c "git push origin main"`,
			shouldBlock: true,
			description: "Should block git push inside sh -c",
		},
		{
			name:        "bash -c with git push",
			command:     `bash -c "git push --force"`,
			shouldBlock: true,
			description: "Should block git push inside bash -c",
		},
		{
			name:        "zsh -c with aws delete",
			command:     `zsh -c "aws s3 delete-bucket my-bucket"`,
			shouldBlock: true,
			description: "Should block aws delete inside zsh -c",
		},

		// Eval commands
		{
			name:        "eval with git push",
			command:     `eval "git push"`,
			shouldBlock: true,
			description: "Should block git push inside eval",
		},
		{
			name:        "source with blocked command",
			command:     `source <(echo "git push")`,
			shouldBlock: true, // Dynamic content - block for safety
			description: "Dynamic source - should block for safety",
		},

		// Echo piped to shell
		{
			name:        "echo git push piped to bash",
			command:     `echo "git push origin main" | bash`,
			shouldBlock: true,
			description: "Should block git push in echo string",
		},
		{
			name:        "echo aws delete piped to sh",
			command:     `echo "aws ec2 terminate-instances --instance-ids i-1234" | sh`,
			shouldBlock: true,
			description: "Should block aws terminate in echo string",
		},

		// xargs patterns
		{
			name:        "xargs with git push",
			command:     `echo origin | xargs git push`,
			shouldBlock: true,
			description: "Should block direct git push with xargs",
		},
		{
			name:        "xargs -I with git push",
			command:     `echo main | xargs -I {} git push origin {}`,
			shouldBlock: true,
			description: "Should block git push with xargs substitution",
		},

		// find -exec patterns
		{
			name:        "find with -exec git push",
			command:     `find . -name "*.txt" -exec git push {} \;`,
			shouldBlock: true,
			description: "Should block git push in find -exec",
		},
		{
			name:        "find with -exec aws delete",
			command:     `find . -name "*.json" -exec aws s3 delete-object --bucket test --key {} \;`,
			shouldBlock: true,
			description: "Should block aws delete in find -exec",
		},

		// Parallel command execution
		{
			name:        "parallel with git push",
			command:     `parallel git push ::: origin upstream`,
			shouldBlock: true,
			description: "Should block git push with parallel",
		},

		// Complex nested cases
		{
			name:        "nested shells with git push",
			command:     `sh -c "bash -c 'git push'"`,
			shouldBlock: true,
			description: "Should block git push in nested shells",
		},
		{
			name:        "command substitution with git push",
			command:     `echo $(git push 2>&1)`,
			shouldBlock: true, // Dynamic substitution - block for safety
			description: "Dynamic substitution - should block for safety",
		},

		// Safe commands that should NOT be blocked
		{
			name:        "simple echo with git in string",
			command:     `echo "Remember to git push later"`,
			shouldBlock: false,
			description: "Should not block simple echo with git in message",
		},
		{
			name:        "git pull allowed",
			command:     `sh -c "git pull origin main"`,
			shouldBlock: false,
			description: "Should allow git pull even in sh -c",
		},
		{
			name:        "aws list allowed",
			command:     `bash -c "aws s3 list-buckets"`,
			shouldBlock: false,
			description: "Should allow aws list operations",
		},
		{
			name:        "simple string argument",
			command:     `grep "git push" file.txt`,
			shouldBlock: false,
			description: "Should not block grep searching for git push",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detector := NewCommandDetector(rules, 10)
			blocked := detector.ShouldBlockShellExpr(tt.command)
			if blocked != tt.shouldBlock {
				t.Errorf("%s: got blocked=%v, want %v", tt.description, blocked, tt.shouldBlock)
			}

			if blocked && tt.shouldBlock {
				// Verify we have meaningful issues reported
				issues := detector.GetIssues()
				if len(issues) == 0 {
					t.Error("Should have issues when blocking")
				}
			}
		})
	}
}

func TestLooksLikeCommand(t *testing.T) {
	// Create a detector with sample rules for testing
	rules := []CommandRule{
		{BlockedCommand: "git", BlockedPatterns: []string{"push"}},
		{BlockedCommand: "aws", BlockedPatterns: []string{"delete-*"}},
	}
	detector := NewCommandDetector(rules, 10)

	tests := []struct {
		input    string
		expected bool
		reason   string
	}{
		// Should look like commands
		{"git push", true, "Space-separated words"},
		{"aws s3 delete-bucket", true, "Multiple words"},
		{"echo foo; git push", true, "Contains semicolon"},
		{"git pull && npm test", true, "Contains &&"},
		{"test | grep result", true, "Contains pipe"},

		// Should NOT look like commands
		{"-f", false, "Just a flag"},
		{"--verbose", false, "Just a long flag"},
		{"origin", false, "Single word"},
		{"main", false, "Single word"},
		{"file.txt", false, "Filename"},
		{"/path/to/file", false, "Path"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := detector.looksLikeCommand(tt.input)
			if result != tt.expected {
				t.Errorf("%s: got %v, want %v", tt.reason, result, tt.expected)
			}
		})
	}
}
