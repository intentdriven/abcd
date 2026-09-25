package banlist

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The pre-commit guard refreshes the sources corpus's generated block before it
// checks anything (itd-76 AC3, AC6). Both copies of the guard carry the step — the
// one this repository runs and the template `abcd ahoy` scaffolds into a managed
// repository — so every case runs against both.
func guardCopies(t *testing.T) map[string]string {
	t.Helper()
	hook := locateHook(t)
	top := filepath.Dir(filepath.Dir(hook))
	return map[string]string{
		"repo":     hook,
		"template": filepath.Join(top, "internal", "core", "ahoy", "defaults", "pre-commit"),
	}
}

// newGuardRepo is newHookRepo with a chosen copy of the guard installed.
func newGuardRepo(t *testing.T, hookPath, body string) *hookRepo {
	t.Helper()
	r := newHookRepo(t, body)
	src, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(r.dir, ".git", "hooks", "pre-commit"), src, 0o755); err != nil {
		t.Fatal(err)
	}
	return r
}

// fakeAbcd puts an `abcd` on the guard's PATH (through the repo-local guardPath
// extension) that records its arguments and then runs body, and returns the file
// the arguments land in.
func fakeAbcd(t *testing.T, r *hookRepo, body string) string {
	t.Helper()
	bin := t.TempDir()
	args := filepath.Join(t.TempDir(), "args")
	script := "#!/bin/sh\nprintf '%s\\n' \"$*\" > '" + args + "'\n" + body + "\n"
	if err := os.WriteFile(filepath.Join(bin, "abcd"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	r.git("config", "--local", "abcd.guardPath", bin)
	return args
}

func makeCorpusDir(t *testing.T) {
	t.Helper()
	home := os.Getenv("HOME")
	if home == "" {
		t.Skip("no HOME")
	}
	if err := os.MkdirAll(filepath.Join(home, ".abcd", "sources"), 0o700); err != nil {
		t.Fatal(err)
	}
}

// TestPreCommitHook_NoCorpusSaysSoAndProceeds is AC6 at the guard: no corpus is one
// line naming the skip, and the commit proceeds.
func TestPreCommitHook_NoCorpusSaysSoAndProceeds(t *testing.T) {
	for name, hook := range guardCopies(t) {
		t.Run(name, func(t *testing.T) {
			r := newGuardRepo(t, hook, "")
			r.write("note.md", "hello\n")
			r.git("add", "note.md")
			blocked, out := r.commit()
			if blocked {
				t.Fatalf("commit blocked with no corpus\n%s", out)
			}
			if strings.Count(out, "no sources corpus") != 1 {
				t.Fatalf("the guard does not say, once, that the corpus is absent\n%s", out)
			}
		})
	}
}

// TestPreCommitHook_RefreshesTheSourcesBlockBeforeChecking is AC3 at the guard: with
// a corpus present the guard runs `abcd source sync-banlist --refresh` BEFORE it
// reads the store, so an entry the refresh writes is enforced on this very commit.
func TestPreCommitHook_RefreshesTheSourcesBlockBeforeChecking(t *testing.T) {
	for name, hook := range guardCopies(t) {
		t.Run(name, func(t *testing.T) {
			makeCorpusDir(t)
			r := newGuardRepo(t, hook, privateFormatDecl+"\nhand-key zzunrelated\n")
			args := fakeAbcd(t, r, "printf '# abcd-banlist: keyed\\nsources/conf2026a/title widgetworks\\n' > .abcd/.work.local/private-names.txt")
			r.write("note.md", "the widgetworks draft\n")
			r.git("add", "note.md")
			blocked, out := r.commit()
			got, err := os.ReadFile(args)
			if err != nil {
				t.Fatalf("the guard did not run abcd\n%s", out)
			}
			if strings.TrimSpace(string(got)) != "source sync-banlist --refresh" {
				t.Fatalf("the guard ran abcd with %q", got)
			}
			if !blocked || !strings.Contains(out, "sources/conf2026a/title") {
				t.Fatalf("the refreshed entry was not enforced on this commit\n%s", out)
			}
		})
	}
}

// TestPreCommitHook_AFailedRefreshWarnsAndChecksTheStoreAsItStands: a refresh that
// fails is named, never fatal, and never silent; the existing store still guards.
func TestPreCommitHook_AFailedRefreshWarnsAndChecksTheStoreAsItStands(t *testing.T) {
	for name, hook := range guardCopies(t) {
		t.Run(name, func(t *testing.T) {
			makeCorpusDir(t)
			r := newGuardRepo(t, hook, privateFormatDecl+"\nhand-key widgetworks\n")
			fakeAbcd(t, r, "exit 2")
			r.write("note.md", "the widgetworks draft\n")
			r.git("add", "note.md")
			blocked, out := r.commit()
			if !strings.Contains(out, "refresh failed") {
				t.Fatalf("a failed refresh was silent\n%s", out)
			}
			if !blocked || !strings.Contains(out, "hand-key") {
				t.Fatalf("the existing store stopped guarding after a failed refresh\n%s", out)
			}
		})
	}
}
