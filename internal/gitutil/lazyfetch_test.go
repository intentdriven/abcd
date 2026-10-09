package gitutil_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// TestIsolatedGitDoesNotLazyFetchAMissingObject is iss-2610090821527948. In a
// partial clone (a promisor remote), git answers a read of a MISSING object by
// fetching it, and the fetch runs the transport the repository configures:
// remote.<name>.uploadpack for a local or file:// URL, core.sshCommand for an
// ssh:// one. A copied checkout carries its own .git/config, so an object read
// abcd makes (`show rev:path`, `cat-file blob`, a `tag --list` whose repo
// tag.sort reads the tagged object) ran a program the checkout named. The
// isolated environment sets GIT_NO_LAZY_FETCH=1, so a missing object is an error
// and no transport starts; a PRESENT object still reads.
func TestIsolatedGitDoesNotLazyFetchAMissingObject(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX mark script")
	}
	requireGitAtLeast(t, 2, 44) // GIT_NO_LAZY_FETCH arrived in git 2.44.

	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("GIT_TERMINAL_PROMPT", "0")

	for _, transport := range []string{"uploadpack", "sshCommand"} {
		t.Run(transport, func(t *testing.T) {
			repo := t.TempDir()
			mark := filepath.Join(t.TempDir(), "transport-ran")
			script := filepath.Join(t.TempDir(), "transport.sh")
			if err := os.WriteFile(script, []byte("#!/bin/sh\ntouch '"+mark+"'\nexit 1\n"), 0o755); err != nil {
				t.Fatal(err)
			}
			git := func(args ...string) string {
				t.Helper()
				cmd := exec.Command("git", append([]string{"-C", repo,
					"-c", "user.email=t@t.invalid", "-c", "user.name=t", "-c", "commit.gpgsign=false",
					"-c", "tag.gpgsign=false"}, args...)...)
				cmd.Env = gittest.Env(t)
				out, err := cmd.CombinedOutput()
				if err != nil {
					t.Fatalf("git %v: %v: %s", args, err, out)
				}
				return strings.TrimSpace(string(out))
			}
			git("init", "-q")
			for name, body := range map[string]string{"gone.txt": "gone\n", "kept.txt": "kept\n"} {
				if err := os.WriteFile(filepath.Join(repo, name), []byte(body), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			git("add", ".")
			git("commit", "-q", "-m", "c0")
			gone := git("rev-parse", "HEAD:gone.txt")
			// A commit only a tag reaches, so deleting it leaves HEAD intact.
			tagged := git("commit-tree", "-m", "tagged", git("rev-parse", "HEAD^{tree}"))
			git("tag", "v1.1.0", tagged)
			for _, sha := range []string{gone, tagged} {
				if err := os.Remove(filepath.Join(repo, ".git", "objects", sha[:2], sha[2:])); err != nil {
					t.Fatalf("delete loose object %s: %v", sha, err)
				}
			}
			git("config", "core.repositoryformatversion", "1")
			git("config", "extensions.partialClone", "origin")
			git("config", "remote.origin.promisor", "true")
			git("config", "remote.origin.partialclonefilter", "blob:none")
			git("config", "tag.sort", "taggerdate")
			if transport == "uploadpack" {
				git("config", "remote.origin.url", t.TempDir())
				git("config", "remote.origin.uploadpack", script)
			} else {
				git("config", "remote.origin.url", "ssh://example.invalid/x.git")
				git("config", "core.sshCommand", script)
			}

			// The control: a present blob still reads.
			if out, err := gitutil.Run(repo, "cat-file", "blob", "HEAD:kept.txt"); err != nil || out != "kept" {
				t.Fatalf("a present blob no longer reads: %q, %v", out, err)
			}
			fired := func(sink string) {
				t.Helper()
				if _, err := os.Stat(mark); err == nil {
					t.Errorf("%s of a missing object ran the repository's %s transport (lazy fetch); the isolated environment must set GIT_NO_LAZY_FETCH=1", sink, transport)
					_ = os.Remove(mark)
				}
			}
			if _, err := gitutil.Run(repo, "show", "HEAD:gone.txt"); err == nil {
				t.Error("show of a missing blob succeeded")
			}
			fired("show rev:path")
			if _, err := gitutil.RunLimited(repo, 1<<20, "cat-file", "blob", gone); err == nil {
				t.Error("cat-file of a missing blob succeeded")
			}
			fired("cat-file blob")
			list := exec.Command("git", "-C", repo, "tag", "--list", "v*")
			list.Env = gitutil.IsolatedEnv()
			_ = list.Run()
			fired("tag --list under tag.sort=taggerdate")
		})
	}
}

// requireGitAtLeast skips when the git on PATH is older than major.minor.
func requireGitAtLeast(t *testing.T, major, minor int) {
	t.Helper()
	cmd := exec.Command("git", "version")
	cmd.Env = gittest.Env(t)
	out, err := cmd.Output()
	if err != nil {
		t.Skip("git not on PATH")
	}
	m := regexp.MustCompile(`(\d+)\.(\d+)`).FindStringSubmatch(string(out))
	if m == nil {
		t.Skipf("unreadable git version %q", out)
	}
	ma, _ := strconv.Atoi(m[1])
	mi, _ := strconv.Atoi(m[2])
	if ma < major || (ma == major && mi < minor) {
		t.Skipf("git %s.%s is older than %d.%d", m[1], m[2], major, minor)
	}
}
