package detector

import (
	"testing"
)

func TestNewCommandDetector(t *testing.T) {
	rules := []CommandRule{
		{
			BlockedCommand:  "git",
			BlockedPatterns: []string{"push"},
			Description:     "Block git push",
		},
	}

	detector := NewCommandDetector(rules, 5)

	if len(detector.commandRules) != 1 {
		t.Errorf("Expected 1 rule, got %d", len(detector.commandRules))
	}

	if detector.maxDepth != 5 {
		t.Errorf("Expected maxDepth 5, got %d", detector.maxDepth)
	}
}

func TestCommandDetector_BasicGitPush(t *testing.T) {
	rules := []CommandRule{
		{
			BlockedCommand:  "git",
			BlockedPatterns: []string{"push"},
			Description:     "Block git push",
		},
	}

	tests := []struct {
		name      string
		command   string
		wantBlock bool
	}{
		{
			name:      "Direct git push",
			command:   "git push",
			wantBlock: true,
		},
		{
			name:      "Git push with arguments",
			command:   "git push origin main",
			wantBlock: true,
		},
		{
			name:      "Git pull (allowed)",
			command:   "git pull",
			wantBlock: false,
		},
		{
			name:      "Shell command with git push",
			command:   "sh -c 'git push'",
			wantBlock: true,
		},
		{
			name:      "Git push with variable",
			command:   "CMD=push; git $CMD",
			wantBlock: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detector := NewCommandDetector(rules, 10)
			gotBlock := detector.ShouldBlockCommand(tt.command)

			if gotBlock != tt.wantBlock {
				t.Errorf("ShouldBlockCommand() = %v, want %v. Issues: %v", gotBlock, tt.wantBlock, detector.GetIssues())
			}
		})
	}
}

func TestCommandDetector_MultipleRules(t *testing.T) {
	rules := []CommandRule{
		{
			BlockedCommand:  "git",
			BlockedPatterns: []string{"push"},
			Description:     "Block git push",
		},
		{
			BlockedCommand:  "aws",
			BlockedPatterns: []string{"delete-bucket", "terminate-instances"},
			Description:     "Block dangerous AWS operations",
		},
		{
			BlockedCommand:  "kubectl",
			BlockedPatterns: []string{"delete"},
			AllowedPatterns: []string{"delete pod"},
			Description:     "Block kubectl delete with exceptions",
		},
	}

	tests := []struct {
		name      string
		command   string
		wantBlock bool
	}{
		{
			name:      "Git push blocked",
			command:   "git push",
			wantBlock: true,
		},
		{
			name:      "AWS delete-bucket blocked",
			command:   "aws s3api delete-bucket --bucket my-bucket",
			wantBlock: true,
		},
		{
			name:      "AWS terminate-instances blocked",
			command:   "aws ec2 terminate-instances --instance-ids i-1234567890abcdef0",
			wantBlock: true,
		},
		{
			name:      "Kubectl delete blocked",
			command:   "kubectl delete namespace production",
			wantBlock: true,
		},
		{
			name:      "Kubectl delete pod allowed (exception)",
			command:   "kubectl delete pod my-pod",
			wantBlock: false,
		},
		{
			name:      "AWS list operations allowed",
			command:   "aws s3 ls",
			wantBlock: false,
		},
		{
			name:      "Git pull allowed",
			command:   "git pull",
			wantBlock: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detector := NewCommandDetector(rules, 10)
			gotBlock := detector.ShouldBlockCommand(tt.command)

			if gotBlock != tt.wantBlock {
				t.Errorf("ShouldBlockCommand() = %v, want %v. Issues: %v", gotBlock, tt.wantBlock, detector.GetIssues())
			}
		})
	}
}

func TestCommandDetector_CommandMatching(t *testing.T) {
	rules := []CommandRule{
		{
			BlockedCommand:  "git",
			BlockedPatterns: []string{"push"},
			Description:     "Block git push",
		},
	}

	tests := []struct {
		name      string
		command   string
		wantBlock bool
	}{
		{
			name:      "Direct git command",
			command:   "git push",
			wantBlock: true,
		},
		{
			name:      "Full path git command",
			command:   "/usr/bin/git push",
			wantBlock: true,
		},
		{
			name:      "Local git command",
			command:   "./git push",
			wantBlock: true,
		},
		{
			name:      "Windows git command",
			command:   "git.exe push",
			wantBlock: true,
		},
		{
			name:      "Windows full path git command",
			command:   "\"C:\\Program Files\\Git\\bin\\git.exe\" push",
			wantBlock: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detector := NewCommandDetector(rules, 10)
			gotBlock := detector.ShouldBlockCommand(tt.command)

			if gotBlock != tt.wantBlock {
				t.Errorf("ShouldBlockCommand() = %v, want %v. Issues: %v", gotBlock, tt.wantBlock, detector.GetIssues())
			}
		})
	}
}

func TestCommandDetector_AllowedPatterns(t *testing.T) {
	rules := []CommandRule{
		{
			BlockedCommand:  "kubectl",
			BlockedPatterns: []string{"delete"},
			AllowedPatterns: []string{"delete pod", "delete configmap"},
			Description:     "Block kubectl delete with exceptions",
		},
	}

	tests := []struct {
		name      string
		command   string
		wantBlock bool
	}{
		{
			name:      "Delete namespace blocked",
			command:   "kubectl delete namespace production",
			wantBlock: true,
		},
		{
			name:      "Delete pod allowed",
			command:   "kubectl delete pod my-pod",
			wantBlock: false,
		},
		{
			name:      "Delete configmap allowed",
			command:   "kubectl delete configmap my-config",
			wantBlock: false,
		},
		{
			name:      "Delete service blocked",
			command:   "kubectl delete service my-service",
			wantBlock: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detector := NewCommandDetector(rules, 10)
			gotBlock := detector.ShouldBlockCommand(tt.command)

			if gotBlock != tt.wantBlock {
				t.Errorf("ShouldBlockCommand() = %v, want %v. Issues: %v", gotBlock, tt.wantBlock, detector.GetIssues())
			}
		})
	}
}

func TestCommandDetector_MaxDepthValidation(t *testing.T) {
	// Test that invalid max depth defaults to 10
	detector := NewCommandDetector([]CommandRule{}, 0)
	if detector.maxDepth != 10 {
		t.Errorf("Expected default maxDepth 10, got %d", detector.maxDepth)
	}

	detector = NewCommandDetector([]CommandRule{}, -5)
	if detector.maxDepth != 10 {
		t.Errorf("Expected default maxDepth 10, got %d", detector.maxDepth)
	}
}

func TestCommandDetector_IssueReporting(t *testing.T) {
	rules := []CommandRule{
		{
			BlockedCommand:  "git",
			BlockedPatterns: []string{"push"},
			Description:     "Block git push",
		},
	}

	detector := NewCommandDetector(rules, 10)

	// Test that issues are cleared between analyses
	detector.ShouldBlockCommand("git push")
	firstIssues := len(detector.GetIssues())

	detector.ShouldBlockCommand("git pull") // Should not add issues
	secondIssues := len(detector.GetIssues())

	if secondIssues != 0 {
		t.Errorf("Expected 0 issues after analyzing allowed command, got %d", secondIssues)
	}

	if firstIssues == 0 {
		t.Errorf("Expected issues for blocked command, got %d", firstIssues)
	}
}

func TestCommandDetector_InterspersedFlags(t *testing.T) {
	rules := []CommandRule{
		{
			BlockedCommand:  "aws",
			BlockedPatterns: []string{"terminate-instances"},
			Description:     "Block AWS terminate instances",
		},
		{
			BlockedCommand:  "kubectl",
			BlockedPatterns: []string{"delete"},
			AllowedPatterns: []string{"delete pod"},
			Description:     "Block kubectl delete with exceptions",
		},
	}

	tests := []struct {
		name      string
		command   string
		wantBlock bool
	}{
		{
			name:      "AWS terminate-instances with region flag before subcommand",
			command:   "aws --region us-east-1 ec2 terminate-instances --instance-ids i-1234567890abcdef0",
			wantBlock: true,
		},
		{
			name:      "AWS terminate-instances with multiple flags before subcommand",
			command:   "aws --region us-west-2 --profile prod ec2 terminate-instances --instance-ids i-1234567890abcdef0",
			wantBlock: true,
		},
		{
			name:      "AWS terminate-instances with output flag after subcommand",
			command:   "aws ec2 terminate-instances --instance-ids i-1234567890abcdef0 --output json",
			wantBlock: true,
		},
		{
			name:      "AWS terminate-instances with flags before and after subcommand",
			command:   "aws --region us-east-1 ec2 terminate-instances --instance-ids i-1234567890abcdef0 --output table",
			wantBlock: true,
		},
		{
			name:      "Kubectl delete namespace with context flag",
			command:   "kubectl --context prod delete --force namespace production",
			wantBlock: true,
		},
		{
			name:      "Kubectl delete namespace with multiple flags",
			command:   "kubectl --kubeconfig ~/.kube/config --context staging delete namespace test-env --grace-period=0",
			wantBlock: true,
		},
		{
			name:      "Kubectl delete pod with context flag (should be allowed)",
			command:   "kubectl --context prod delete pod my-pod",
			wantBlock: false,
		},
		{
			name:      "Kubectl delete pod with multiple flags (should be allowed)",
			command:   "kubectl --kubeconfig ~/.kube/config --context prod delete pod my-pod --grace-period=30",
			wantBlock: false,
		},
		{
			name:      "AWS list operations with flags (should be allowed)",
			command:   "aws --region us-east-1 ec2 describe-instances --output json",
			wantBlock: false,
		},
		{
			name:      "Complex AWS terminate-instances with many flags",
			command:   "aws --cli-read-timeout 60 --region us-east-1 --profile production ec2 terminate-instances --instance-ids i-1234567890abcdef0 i-abcdef1234567890 --output json --dry-run",
			wantBlock: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detector := NewCommandDetector(rules, 10)
			gotBlock := detector.ShouldBlockCommand(tt.command)

			if gotBlock != tt.wantBlock {
				t.Errorf("ShouldBlockCommand() = %v, want %v. Command: %s, Issues: %v", gotBlock, tt.wantBlock, tt.command, detector.GetIssues())
			}
		})
	}
}

func TestCommandDetector_ProximityLimits(t *testing.T) {
	rules := []CommandRule{
		{
			BlockedCommand:  "kubectl",
			BlockedPatterns: []string{"delete"},
			AllowedPatterns: []string{"delete pod"},
			Description:     "Block kubectl delete with pod exception",
		},
	}

	tests := []struct {
		name      string
		command   string
		wantBlock bool
		note      string
	}{
		{
			name:      "Normal delete pod (should be allowed)",
			command:   "kubectl delete pod my-app",
			wantBlock: false,
			note:      "Exception should work normally",
		},
		{
			name:      "Delete with long flags before pod (proximity test)",
			command:   "kubectl delete --some-very-long-flag-name-that-makes-distance-over-twenty-chars pod my-app",
			wantBlock: false, // This should still be allowed if exception works
			note:      "Exception should still work despite long flags",
		},
		{
			name:      "Delete namespace with long flags (should still block)",
			command:   "kubectl delete --some-very-long-flag-name-that-makes-distance-over-twenty-chars namespace prod",
			wantBlock: true,
			note:      "Block should still work despite long flags",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detector := NewCommandDetector(rules, 10)
			gotBlock := detector.ShouldBlockCommand(tt.command)

			if gotBlock != tt.wantBlock {
				t.Errorf("ShouldBlockCommand() = %v, want %v. %s. Command: %s, Issues: %v",
					gotBlock, tt.wantBlock, tt.note, tt.command, detector.GetIssues())
			}
		})
	}
}
