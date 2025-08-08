package main

import (
	"reflect"
	"testing"

	"github.com/krmcbride/claudecode-hooks/pkg/detector"
)

func TestParseCommandRules(t *testing.T) {
	tests := []struct {
		name     string
		commands []string
		want     []detector.CommandRule
	}{
		{
			name:     "Single command blocks all",
			commands: []string{"git"},
			want: []detector.CommandRule{
				{
					BlockedCommand:  "git",
					BlockedPatterns: []string{"*"},
				},
			},
		},
		{
			name:     "Command with single pattern",
			commands: []string{"git push"},
			want: []detector.CommandRule{
				{
					BlockedCommand:  "git",
					BlockedPatterns: []string{"push"},
				},
			},
		},
		{
			name:     "Command with multiple patterns",
			commands: []string{"git push pull force-push"},
			want: []detector.CommandRule{
				{
					BlockedCommand:  "git",
					BlockedPatterns: []string{"push", "pull", "force-push"},
				},
			},
		},
		{
			name:     "Command with wildcard patterns",
			commands: []string{"aws delete-* terminate-*"},
			want: []detector.CommandRule{
				{
					BlockedCommand:  "aws",
					BlockedPatterns: []string{"delete-*", "terminate-*"},
				},
			},
		},
		{
			name: "Multiple commands",
			commands: []string{
				"git push",
				"aws delete-*",
				"kubectl",
			},
			want: []detector.CommandRule{
				{
					BlockedCommand:  "git",
					BlockedPatterns: []string{"push"},
				},
				{
					BlockedCommand:  "aws",
					BlockedPatterns: []string{"delete-*"},
				},
				{
					BlockedCommand:  "kubectl",
					BlockedPatterns: []string{"*"},
				},
			},
		},
		{
			name:     "Empty command ignored",
			commands: []string{"", "git push", ""},
			want: []detector.CommandRule{
				{
					BlockedCommand:  "git",
					BlockedPatterns: []string{"push"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseCommandRules(tt.commands)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseCommandRules() = %+v, want %+v", got, tt.want)
			}
		})
	}
}
