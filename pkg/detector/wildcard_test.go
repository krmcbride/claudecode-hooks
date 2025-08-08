package detector

import (
	"testing"
)

func TestCommandDetector_WildcardPatterns(t *testing.T) {
	tests := []struct {
		name      string
		rules     []CommandRule
		command   string
		wantBlock bool
	}{
		{
			name: "Wildcard blocks all git commands",
			rules: []CommandRule{
				{
					BlockedCommand:  "git",
					BlockedPatterns: []string{"*"},
				},
			},
			command:   "git",
			wantBlock: true,
		},
		{
			name: "Wildcard blocks git with any subcommand",
			rules: []CommandRule{
				{
					BlockedCommand:  "git",
					BlockedPatterns: []string{"*"},
				},
			},
			command:   "git status",
			wantBlock: true,
		},
		{
			name: "Wildcard blocks git push",
			rules: []CommandRule{
				{
					BlockedCommand:  "git",
					BlockedPatterns: []string{"*"},
				},
			},
			command:   "git push origin main",
			wantBlock: true,
		},
		{
			name: "Specific pattern only blocks matching subcommand",
			rules: []CommandRule{
				{
					BlockedCommand:  "git",
					BlockedPatterns: []string{"push"},
				},
			},
			command:   "git pull",
			wantBlock: false,
		},
		{
			name: "Glob pattern blocks delete-* commands",
			rules: []CommandRule{
				{
					BlockedCommand:  "aws",
					BlockedPatterns: []string{"delete-*"},
				},
			},
			command:   "aws s3api delete-bucket --bucket my-bucket",
			wantBlock: true,
		},
		{
			name: "Glob pattern blocks delete-object",
			rules: []CommandRule{
				{
					BlockedCommand:  "aws",
					BlockedPatterns: []string{"delete-*"},
				},
			},
			command:   "aws s3api delete-object --bucket my-bucket --key file.txt",
			wantBlock: true,
		},
		{
			name: "Glob pattern doesn't block non-matching",
			rules: []CommandRule{
				{
					BlockedCommand:  "aws",
					BlockedPatterns: []string{"delete-*"},
				},
			},
			command:   "aws s3 ls",
			wantBlock: false,
		},
		{
			name: "Multiple patterns with wildcard",
			rules: []CommandRule{
				{
					BlockedCommand:  "aws",
					BlockedPatterns: []string{"delete-*", "terminate-*"},
				},
			},
			command:   "aws ec2 terminate-instances --instance-ids i-1234567890abcdef0",
			wantBlock: true,
		},
		{
			name: "Multiple rules with different patterns",
			rules: []CommandRule{
				{
					BlockedCommand:  "git",
					BlockedPatterns: []string{"push", "force-push"},
				},
				{
					BlockedCommand:  "aws",
					BlockedPatterns: []string{"*"},
				},
			},
			command:   "aws s3 ls",
			wantBlock: true,
		},
		{
			name: "Multiple rules - git push blocked",
			rules: []CommandRule{
				{
					BlockedCommand:  "git",
					BlockedPatterns: []string{"push"},
				},
				{
					BlockedCommand:  "aws",
					BlockedPatterns: []string{"delete-*"},
				},
			},
			command:   "git push",
			wantBlock: true,
		},
		{
			name: "Multiple rules - git pull allowed",
			rules: []CommandRule{
				{
					BlockedCommand:  "git",
					BlockedPatterns: []string{"push"},
				},
				{
					BlockedCommand:  "aws",
					BlockedPatterns: []string{"delete-*"},
				},
			},
			command:   "git pull",
			wantBlock: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detector := NewCommandDetector(tt.rules, 10)
			gotBlock := detector.ShouldBlockShellExpr(tt.command)

			if gotBlock != tt.wantBlock {
				t.Errorf("ShouldBlockShellExpr() = %v, want %v. Command: %s, Issues: %v",
					gotBlock, tt.wantBlock, tt.command, detector.GetIssues())
			}
		})
	}
}
