package gitutil_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// fakeSSH puts an ssh on PATH that records its arguments and fails, so a
// fetch over an ssh URL shows the command git's transport ran without any
// network.
func fakeSSH(t *testing.T) (record string) {
	t.Helper()
	dir := t.TempDir()
	record = filepath.Join(dir, "args")
	script := "#!/bin/sh\nprintf '%s\\n' \"$@\" > '" + record + "'\nexit 255\n"
	if err := os.WriteFile(filepath.Join(dir, "ssh"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return record
}

// TestNetsSSHNeverPrompts: GIT_TERMINAL_PROMPT reaches git's own prompts, not
// ssh's, which reads the terminal itself, so a first-contact host-key question
// would hang a session until NetTimeout. The ssh command Net composes runs in
// batch mode; a command the developer configured is left exactly as it is
// (review finding R6 on spc-2610031753359849 steps 1-3).
func TestNetsSSHNeverPrompts(t *testing.T) {
	gittest.Env(t)
	// Unset, not empty: git takes an empty GIT_SSH_COMMAND as the command.
	for _, k := range []string{"GIT_SSH_COMMAND", "GIT_SSH"} {
		t.Setenv(k, "")
		if err := os.Unsetenv(k); err != nil {
			t.Fatal(err)
		}
	}
	r := gittest.NewRepo(t)
	record := fakeSSH(t)
	_, _ = gitutil.Net(r.Root(), "fetch", "ssh://git@example.invalid/repo.git", "main")
	args, err := os.ReadFile(record)
	if err != nil {
		t.Fatalf("git never ran ssh: %v", err)
	}
	if !strings.Contains(string(args), "BatchMode=yes") {
		t.Fatalf("the composed ssh command does not run in batch mode:\n%s", args)
	}

	// The developer's own ssh command, through either spelling, is untouched.
	for _, how := range []string{"env", "config"} {
		t.Run(how, func(t *testing.T) {
			record := fakeSSH(t)
			own := "ssh -o MineOnly=yes"
			if how == "env" {
				t.Setenv("GIT_SSH_COMMAND", own)
			} else {
				r.Git("config", "core.sshCommand", own)
				t.Cleanup(func() { r.Git("config", "--unset", "core.sshCommand") })
			}
			_, _ = gitutil.Net(r.Root(), "fetch", "ssh://git@example.invalid/repo.git", "main")
			args, err := os.ReadFile(record)
			if err != nil {
				t.Fatalf("git never ran ssh: %v", err)
			}
			if !strings.Contains(string(args), "MineOnly=yes") || strings.Contains(string(args), "BatchMode") {
				t.Fatalf("the developer's ssh command was changed:\n%s", args)
			}
		})
	}
}
