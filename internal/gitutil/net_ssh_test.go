package gitutil_test

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

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
	// The fake ssh below is the transport under test, so the guard that
	// refuses ssh under a test binary is off here.
	gitutil.AllowNetTransportsForTest(t)
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

// TestTheSSHCommandsReadEndsWithItsContext: NetAskContext gives git the ssh
// command Net composes, reading core.sshCommand first; that read runs under
// the ask's own context, so a git that hangs on it holds the ask no longer
// than its deadline. Watched failing before: the read ran unbounded, and the
// ask returned only once the stand-in's ten seconds were up.
func TestTheSSHCommandsReadEndsWithItsContext(t *testing.T) {
	gittest.Env(t)
	for _, k := range []string{"GIT_SSH_COMMAND", "GIT_SSH"} {
		t.Setenv(k, "")
		if err := os.Unsetenv(k); err != nil {
			t.Fatal(err)
		}
	}
	r := gittest.NewRepo(t)
	git, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	script := "#!/bin/sh\ncase \"$*\" in *core.sshCommand*) exec /bin/sleep 10 ;; esac\nexec '" + git + "' \"$@\"\n"
	if err := os.WriteFile(filepath.Join(dir, "git"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	began := time.Now()
	_, err = gitutil.NetAskContext(ctx, r.Root(), 4096, "var", "GIT_ATTR_GLOBAL")
	if took := time.Since(began); err == nil || took > 5*time.Second {
		t.Fatalf("the ask ends at its deadline, its read of core.sshCommand with it (%s): %v", took.Round(time.Millisecond), err)
	}
}

// TestTheSSHCommandIsComposedOnlyOnGitsAnswerOfNone: the networked command is
// given the composed ssh command only when git answered that core.sshCommand
// is not set (exit 1 with nothing written, or exit 0 with nothing but white
// space). A setting git answered
// while a program it left holds the output for a few hundred milliseconds is
// still the person's, and the command is given no GIT_SSH_COMMAND over it; a
// read that fails with any other exit starts no networked command. Watched
// failing before: the read's output was copied by exec under a 50 ms wait
// delay, so the held answer and the failed read were both taken as none, and
// the command ran with the default.
func TestTheSSHCommandIsComposedOnlyOnGitsAnswerOfNone(t *testing.T) {
	gittest.Env(t)
	for _, k := range []string{"GIT_SSH_COMMAND", "GIT_SSH"} {
		t.Setenv(k, "")
		if err := os.Unsetenv(k); err != nil {
			t.Fatal(err)
		}
	}
	r := gittest.NewRepo(t)
	git, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, read string
		runs       bool
		want       string // the GIT_SSH_COMMAND the command was given
	}{
		{"set, its output held", "printf 'ssh -o MineOnly=yes\\n'; /bin/sleep 0.4 & exit 0", true, "unset"},
		{"none, exit 1", "exit 1", true, "ssh -o BatchMode=yes"},
		{"none, exit 0 with nothing written", "exit 0", true, "ssh -o BatchMode=yes"},
		{"failed, exit 3", "echo 'fatal: bad config line 1' >&2; exit 3", false, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			record := filepath.Join(dir, "ran")
			script := "#!/bin/sh\ncase \"$*\" in *core.sshCommand*) " + tc.read + " ;; esac\n" +
				"printf '%s\\n' \"${GIT_SSH_COMMAND-unset}\" > '" + record + "'\nexec '" + git + "' \"$@\"\n"
			if err := os.WriteFile(filepath.Join(dir, "git"), []byte(script), 0o755); err != nil {
				t.Fatal(err)
			}
			t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			_, err := gitutil.NetAskContext(ctx, r.Root(), 4096, "var", "GIT_ATTR_GLOBAL")
			got, readErr := os.ReadFile(record)
			if !tc.runs {
				var nr *gitutil.NotRunError
				if readErr == nil || !errors.As(err, &nr) || !strings.Contains(err.Error(), "core.sshCommand") {
					t.Fatalf("a failed read of core.sshCommand starts no networked git and names the read: ran=%v, err %v", readErr == nil, err)
				}
				// Its refusal says what to do (iss-2611140432160343).
				// Watched failing before: it named no remedy.
				if !strings.Contains(err.Error(), "remedy: ") || !strings.Contains(err.Error(), "git config --show-origin --get core.sshCommand") {
					t.Fatalf("the refusal of a failed read of core.sshCommand names a remedy: %v", err)
				}
				return
			}
			if err != nil || readErr != nil {
				t.Fatalf("the networked git ran: %v, %v", err, readErr)
			}
			if !strings.HasPrefix(strings.TrimSpace(string(got)), tc.want) {
				t.Fatalf("the networked git was given GIT_SSH_COMMAND %q, want %q", strings.TrimSpace(string(got)), tc.want)
			}
		})
	}
}
