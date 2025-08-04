package tests

import (
	"testing"

	"github.com/krmcbride/claudecode-hooks/pkg/detector"
)

func TestDirectGitPushCommands(t *testing.T) {
	tests := []struct {
		command     string
		shouldBlock bool
		description string
	}{
		{"git push", true, "Basic git push"},
		{"git push origin main", true, "Git push with remote and branch"},
		{"git push --force", true, "Git push with force flag"},
		{"git push -u origin main", true, "Git push with upstream flag"},
		{"/usr/bin/git push", true, "Full path git push"},
		{"./git push", true, "Relative path git push"},
		{"git.exe push", true, "Windows git.exe push"},
		{"/usr/local/bin/git.exe push", true, "Windows full path git.exe push"},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			d := detector.NewGitPushDetector()
			blocked := d.AnalyzeCommand(tt.command)

			if blocked != tt.shouldBlock {
				t.Errorf("Command '%s' blocked=%v, want %v. Issues: %v",
					tt.command, blocked, tt.shouldBlock, d.GetIssues())
			}
		})
	}
}

func TestSafeGitCommands(t *testing.T) {
	tests := []struct {
		command     string
		shouldBlock bool
		description string
	}{
		{"git pull", false, "Git pull should be allowed"},
		{"git clone https://github.com/example/repo.git", false, "Git clone should be allowed"},
		{"git status", false, "Git status should be allowed"},
		{"git commit -m 'test'", false, "Git commit should be allowed"},
		{"git log --oneline", false, "Git log should be allowed"},
		{"git diff", false, "Git diff should be allowed"},
		{"git add .", false, "Git add should be allowed"},
		{"git checkout main", false, "Git checkout should be allowed"},
		{"git branch feature", false, "Git branch should be allowed"},
		{"git merge feature", false, "Git merge should be allowed"},
		{"git stash", false, "Git stash should be allowed"},
		{"git fetch", false, "Git fetch should be allowed"},
		{"git remote -v", false, "Git remote should be allowed"},
		{"git tag v1.0", false, "Git tag should be allowed"},
		{"git reset HEAD~1", false, "Git reset should be allowed"},
		{"git rebase main", false, "Git rebase should be allowed"},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			d := detector.NewGitPushDetector()
			blocked := d.AnalyzeCommand(tt.command)

			if blocked != tt.shouldBlock {
				t.Errorf("Command '%s' blocked=%v, want %v. Issues: %v",
					tt.command, blocked, tt.shouldBlock, d.GetIssues())
			}
		})
	}
}

func TestShellInterpreterPatterns(t *testing.T) {
	tests := []struct {
		command     string
		shouldBlock bool
		description string
	}{
		{`sh -c "git push"`, true, "sh -c with git push"},
		{`bash -c "git push origin main"`, true, "bash -c with git push"},
		{`zsh -c "git push --force"`, true, "zsh -c with git push"},
		{`dash -c "git push"`, true, "dash -c with git push"},
		{`ksh -c "git push"`, true, "ksh -c with git push"},
		{`sh -c "echo hello && git push"`, true, "sh -c with git push in chain"},
		{`bash -c "cd /tmp && git push"`, true, "bash -c with cd and git push"},
		{`sh -c "ls -la"`, false, "sh -c without git push"},
		{`bash -c "git status"`, false, "bash -c with safe git command"},
		// Command substitution bypass tests
		{`sh -c "$(echo 'git push')"`, true, "Command substitution bypass attempt"},
		{`bash -c "$COMMAND"`, true, "Variable as command (dynamic content)"},
		{`sh -c "${CMD}"`, true, "Variable expansion in command"},
		{`bash -c "$(cat file.txt)"`, true, "Command substitution with cat"},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			d := detector.NewGitPushDetector()
			blocked := d.AnalyzeCommand(tt.command)

			if blocked != tt.shouldBlock {
				t.Errorf("Command '%s' blocked=%v, want %v. Issues: %v",
					tt.command, blocked, tt.shouldBlock, d.GetIssues())
			}
		})
	}
}

func TestEvalPatterns(t *testing.T) {
	tests := []struct {
		command     string
		shouldBlock bool
		description string
	}{
		{`eval "git push"`, true, "eval with git push"},
		{`eval "git push origin main"`, true, "eval with git push and args"},
		{`eval 'git push --force'`, true, "eval with single quotes"},
		{`eval git push`, true, "eval without quotes"},
		{`eval "cd /tmp && git push"`, true, "eval with command chain"},
		{`eval "echo test"`, false, "eval without git push"},
		{`eval "git status"`, false, "eval with safe git command"},
		{`eval "$COMMAND"`, false, "eval with variable (no git push detected in static analysis)"},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			d := detector.NewGitPushDetector()
			blocked := d.AnalyzeCommand(tt.command)

			if blocked != tt.shouldBlock {
				t.Errorf("Command '%s' blocked=%v, want %v. Issues: %v",
					tt.command, blocked, tt.shouldBlock, d.GetIssues())
			}
		})
	}
}

func TestQuotingVariations(t *testing.T) {
	tests := []struct {
		command     string
		shouldBlock bool
		description string
	}{
		{`"git" "push"`, true, "Quoted git and push"},
		{`'git' 'push'`, true, "Single quoted git and push"},
		{`"git" push`, true, "Mixed quoting 1"},
		{`git "push"`, true, "Mixed quoting 2"},
		{`gi"t" pu"sh"`, true, "Partial quoting"},
		{`gi't' pu'sh'`, true, "Partial single quoting"},
		{`/usr/bin/"git" "push"`, true, "Quoted with path"},
		{`"gi""t" "pu""sh"`, true, "Empty quote insertion"},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			d := detector.NewGitPushDetector()
			blocked := d.AnalyzeCommand(tt.command)

			if blocked != tt.shouldBlock {
				t.Errorf("Command '%s' blocked=%v, want %v. Issues: %v",
					tt.command, blocked, tt.shouldBlock, d.GetIssues())
			}
		})
	}
}

func TestNonGitCommands(t *testing.T) {
	tests := []struct {
		command     string
		shouldBlock bool
		description string
	}{
		{"ls -la", false, "Basic ls command"},
		{"echo 'Hello World'", false, "Echo command"},
		{"cat file.txt", false, "Cat command"},
		{"grep pattern file.txt", false, "Grep command"},
		{"cd /tmp", false, "Change directory"},
		{"pwd", false, "Print working directory"},
		{"mkdir test", false, "Make directory"},
		{"rm file.txt", false, "Remove file"},
		{"cp source dest", false, "Copy file"},
		{"mv source dest", false, "Move file"},
		{"find . -name '*.txt'", false, "Find command"},
		{"ps aux", false, "Process list"},
		{"docker ps", false, "Docker command"},
		{"npm install", false, "NPM command"},
		{"python script.py", false, "Python command"},
		{"make build", false, "Make command"},
		{"curl https://example.com", false, "Curl command"},
		{"wget https://example.com", false, "Wget command"},
		{"ssh user@host", false, "SSH command"},
		{"scp file user@host:", false, "SCP command"},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			d := detector.NewGitPushDetector()
			blocked := d.AnalyzeCommand(tt.command)

			if blocked != tt.shouldBlock {
				t.Errorf("Command '%s' blocked=%v, want %v. Issues: %v",
					tt.command, blocked, tt.shouldBlock, d.GetIssues())
			}
		})
	}
}

func TestDynamicCommandDetection(t *testing.T) {
	tests := []struct {
		command     string
		shouldBlock bool
		description string
	}{
		{`$CMD push`, true, "Variable as command"},
		{`${CMD} push`, true, "Variable expansion as command"},
		{`"$CMD" push`, true, "Quoted variable as command"},
		{`$(echo git) push`, true, "Command substitution as command"},
		{`git $SUBCMD`, true, "Variable as git subcommand"},
		{`git ${SUBCMD}`, true, "Variable expansion as git subcommand"},
		{`git "$SUBCMD"`, true, "Quoted variable as git subcommand"},
		{`git $(echo push)`, true, "Command substitution as git subcommand"},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			d := detector.NewGitPushDetector()
			blocked := d.AnalyzeCommand(tt.command)

			if blocked != tt.shouldBlock {
				t.Errorf("Command '%s' blocked=%v, want %v. Issues: %v",
					tt.command, blocked, tt.shouldBlock, d.GetIssues())
			}
		})
	}
}

func TestObfuscationPatterns(t *testing.T) {
	tests := []struct {
		command     string
		shouldBlock bool
		description string
	}{
		{`gi\t pu\sh`, true, "Backslash obfuscation"},
		{`g'i't p'u's'h'`, true, "Excessive single quotes"},
		{`g"i"t p"u"s"h"`, true, "Excessive double quotes"},
		{`gi${x}t pu${x}sh`, true, "Variable insertion obfuscation"},
		{`gi*t pu*sh`, true, "Glob pattern obfuscation"},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			d := detector.NewGitPushDetector()
			blocked := d.AnalyzeCommand(tt.command)

			if blocked != tt.shouldBlock {
				t.Errorf("Command '%s' blocked=%v, want %v. Issues: %v",
					tt.command, blocked, tt.shouldBlock, d.GetIssues())
			}
		})
	}
}

func TestExecutionPatterns(t *testing.T) {
	tests := []struct {
		command     string
		shouldBlock bool
		description string
	}{
		{`xargs git push`, true, "xargs with git push"},
		{`find . -exec git push {} \;`, true, "find -exec with git push"},
		{`parallel git push ::: origin`, true, "parallel with git push"},
		{`env GIT_TERMINAL_PROMPT=0 git push`, true, "env with git push"},
		{`nohup git push &`, true, "nohup with git push"},
		{`timeout 60 git push`, true, "timeout with git push"},
		{`time git push`, true, "time with git push"},
		{`watch git push`, true, "watch with git push"},
		// Safe execution patterns
		{`xargs ls`, false, "xargs without git"},
		{`find . -exec echo {} \;`, false, "find -exec without git"},
		{`parallel echo ::: hello`, false, "parallel without git"},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			d := detector.NewGitPushDetector()
			blocked := d.AnalyzeCommand(tt.command)

			if blocked != tt.shouldBlock {
				t.Errorf("Command '%s' blocked=%v, want %v. Issues: %v",
					tt.command, blocked, tt.shouldBlock, d.GetIssues())
			}
		})
	}
}

// Test that all tests pass
func TestAllTestsPass(t *testing.T) {
	// Run a sample of each test type to ensure basic functionality
	criticalTests := []struct {
		command     string
		shouldBlock bool
		description string
	}{
		// Direct git push
		{"git push", true, "Direct git push"},
		// Command substitution bypass
		{`sh -c "$(echo 'git push')"`, true, "Command substitution bypass"},
		// Variable expansion
		{`git $SUBCMD`, true, "Variable as git subcommand"},
		// Safe command
		{"git pull", false, "Safe git command"},
		// Non-git command
		{"ls -la", false, "Non-git command"},
	}

	passCount := 0
	for _, tt := range criticalTests {
		d := detector.NewGitPushDetector()
		blocked := d.AnalyzeCommand(tt.command)

		if blocked == tt.shouldBlock {
			passCount++
		} else {
			t.Logf("Failed: '%s' - blocked=%v, want %v", tt.command, blocked, tt.shouldBlock)
		}
	}

	t.Logf("Passed %d/%d critical tests", passCount, len(criticalTests))
	if passCount != len(criticalTests) {
		t.Errorf("Not all critical tests passed")
	}
}

// Comprehensive bypass tests as recommended in the review
func TestComprehensiveBypassScenarios(t *testing.T) {
	tests := []struct {
		command     string
		shouldBlock bool
		description string
	}{
		// Command substitution bypasses
		{`sh -c "$(echo git push)"`, true, "Command substitution without quotes"},
		{`bash -c "$(printf 'git push')"`, true, "Command substitution with printf"},
		{`sh -c "$(cat <<EOF
git push
EOF
)"`, true, "Command substitution with heredoc"},

		// Variable expansion bypasses
		{`CMD='git push'; bash -c "$CMD"`, true, "Variable assignment and use"},
		{`export CMD='git push'; sh -c "$CMD"`, true, "Exported variable"},

		// Pipe bypasses
		{`echo "git push" | bash`, true, "Pipe to bash"},
		{`echo "git push" | sh`, true, "Pipe to sh"},
		{`printf "git push" | bash`, true, "Printf pipe"},

		// Here-string bypasses
		{`bash <<< "git push"`, true, "Here-string to bash"},
		{`sh <<< "git push"`, true, "Here-string to sh"},

		// Character escaping bypasses
		{`g\it pu\sh`, true, "Backslash escaping"},
		{`g\i\t p\u\s\h`, true, "Full backslash escaping"},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			d := detector.NewGitPushDetector()
			blocked := d.AnalyzeCommand(tt.command)

			if blocked != tt.shouldBlock {
				t.Errorf("Command '%s' blocked=%v, want %v. Issues: %v",
					tt.command, blocked, tt.shouldBlock, d.GetIssues())
			}
		})
	}
}
