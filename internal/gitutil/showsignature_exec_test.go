package gitutil

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestReadOnlyLogDoesNotRunARepoSigningProgram is iss-2610090821531394. With
// log.showSignature=true in a repository's own config, `git log` and `git show`
// verify the signature of every commit carrying a gpgsig header, and the
// verifier is gpg.program — a program the repository names. Run, RunLimited and
// RunCapped present those reads as probes that run nothing, so they must pin
// signature display off; the subject they return is unchanged.
func TestReadOnlyLogDoesNotRunARepoSigningProgram(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX sentinel script")
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("GIT_TERMINAL_PROMPT", "0")

	repo := t.TempDir()
	git := func(stdin string, args ...string) string {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
		cmd.Env = gitEnv()
		if stdin != "" {
			cmd.Stdin = strings.NewReader(stdin)
		}
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	git("", "init", "-q")
	tree := git("", "hash-object", "-t", "tree", "-w", "--stdin")
	// A commit object carrying a gpgsig header: git only starts the verifier
	// for a commit that has one.
	body := "tree " + tree + "\n" +
		"author A <a@example.invalid> 1700000000 +0000\n" +
		"committer A <a@example.invalid> 1700000000 +0000\n" +
		"gpgsig -----BEGIN PGP SIGNATURE-----\n \n iQEzBAABCAAdFiEE\n -----END PGP SIGNATURE-----\n" +
		"\nsigned subject\n"
	sha := git(body, "hash-object", "-t", "commit", "-w", "--stdin")
	git("", "update-ref", "HEAD", sha)

	sentinel := filepath.Join(t.TempDir(), "gpg-ran")
	script := filepath.Join(repo, "evil-gpg.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\ntouch "+sentinel+"\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	git("", "config", "gpg.program", script)
	git("", "config", "log.showSignature", "true")

	// The fixture is live: a plain log under the same environment runs it.
	git("", "log", "-1", "--format=%s")
	if _, err := os.Stat(sentinel); err != nil {
		t.Fatalf("fixture: a plain git log did not start gpg.program, so this test proves nothing: %v", err)
	}
	if err := os.Remove(sentinel); err != nil {
		t.Fatal(err)
	}

	reads := map[string]func() (string, error){
		"Run log": func() (string, error) { return Run(repo, "log", "-1", "--format=%s") },
		"RunLimited log": func() (string, error) {
			return RunLimited(repo, 4096, "log", "--format=%s")
		},
		"RunCapped show": func() (string, error) {
			return RunCapped(repo, 4096, "show", "--no-patch", "--format=%s", "HEAD")
		},
	}
	for name, read := range reads {
		got, err := read()
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if got != "signed subject" {
			t.Errorf("%s returned %q, want the subject unchanged", name, got)
		}
		if _, err := os.Stat(sentinel); err == nil {
			t.Fatalf("%s started the repository's gpg.program: log.showSignature is not pinned off", name)
		}
	}
}
