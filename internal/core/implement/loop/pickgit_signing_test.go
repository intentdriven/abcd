package loop

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// signedCommit writes a commit object over HEAD's tree, with HEAD as its
// parent, whose header carries a (bogus) PGP signature, and returns its name.
// No signing program is involved in making it.
func signedCommit(t *testing.T, r *gittest.Repo, subject string) string {
	t.Helper()
	tree := strings.TrimSpace(r.Git("rev-parse", "HEAD^{tree}"))
	parent := strings.TrimSpace(r.Git("rev-parse", "HEAD"))
	who := "Fixture <fixture@example.invalid> 1700000000 +0000"
	obj := "tree " + tree + "\nparent " + parent + "\nauthor " + who + "\ncommitter " + who + "\n" +
		"gpgsig -----BEGIN PGP SIGNATURE-----\n \n iQEzBAABCAAdFiEEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=\n =AAAA\n -----END PGP SIGNATURE-----\n" +
		"\n" + subject + "\n"
	path := filepath.Join(t.TempDir(), "commit-object")
	if err := os.WriteFile(path, []byte(obj), 0o644); err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(r.Git("hash-object", "-t", "commit", "-w", path))
}

// TestPickGitStartsNoRepoSigningProgram is iss-2610090821520843. The pick's
// record commit and a sync's merge commit go through pickGit, which keeps the
// repository's config; a repository that sets commit.gpgsign=true makes git
// start gpg.program, gpg.ssh.program or gpg.ssh.defaultKeyCommand to sign the
// commit. pickGit must start none of them, the commit must still be made, and
// a content filter the repository configures still runs (signing is pinned
// off, filters are not).
func TestPickGitStartsNoRepoSigningProgram(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX sentinel script")
	}
	for _, tc := range []struct {
		name string
		cfg  map[string]string
		key  string
	}{
		{"gpg.program", map[string]string{}, "gpg.program"},
		{"gpg.ssh.program", map[string]string{"gpg.format": "ssh", "user.signingkey": "key::ssh-ed25519 AAAA"}, "gpg.ssh.program"},
		{"gpg.ssh.defaultKeyCommand", map[string]string{"gpg.format": "ssh"}, "gpg.ssh.defaultKeyCommand"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := gittest.NewRepo(t)
			r.Write("a.md", "a\n")
			r.Commit("seed")
			mark := filepath.Join(t.TempDir(), "signer-ran")
			signer := filepath.Join(t.TempDir(), "evil-signer.sh")
			if err := os.WriteFile(signer, []byte("#!/bin/sh\ntouch "+mark+"\nexit 1\n"), 0o755); err != nil {
				t.Fatal(err)
			}
			filterMark := filepath.Join(t.TempDir(), "filter-ran")
			filter := filepath.Join(t.TempDir(), "keep-filter.sh")
			if err := os.WriteFile(filter, []byte("#!/bin/sh\ntouch "+filterMark+"\ncat\n"), 0o755); err != nil {
				t.Fatal(err)
			}
			cfg := map[string]string{
				"user.name": "Fixture", "user.email": "fixture@example.invalid",
				"commit.gpgsign": "true", tc.key: signer,
				"filter.keep.clean": filter,
			}
			for k, v := range tc.cfg {
				cfg[k] = v
			}
			for k, v := range cfg {
				r.Git("config", k, v)
			}
			if err := os.WriteFile(filepath.Join(r.Root(), ".git", "info", "attributes"), []byte("*.md filter=keep\n"), 0o644); err != nil {
				t.Fatal(err)
			}

			// The fixture is live: a plain commit under the same environment
			// starts the signer.
			r.Write("a.md", "fixture\n")
			plain := exec.Command("git", "-C", r.Root(), "commit", "-q", "-m", "plain", "--", "a.md")
			plain.Env = gitutil.ScrubbedEnv()
			_ = plain.Run()
			if _, err := os.Stat(mark); err != nil {
				t.Fatalf("fixture: a plain commit did not start %s, so this test proves nothing", tc.key)
			}
			if err := os.Remove(mark); err != nil {
				t.Fatal(err)
			}

			r.Write("a.md", "picked\n")
			if _, err := pickGit(r.Root(), "commit", "-q", "-m", "pick", "--", "a.md"); err != nil {
				t.Fatalf("the pick commit must still be made: %v", err)
			}
			if _, err := os.Stat(mark); err == nil {
				t.Fatalf("pickGit's commit started the repository's %s", tc.key)
			}
			if _, err := os.Stat(filterMark); err != nil {
				t.Errorf("a configured clean filter must still run on the pick commit: %v", err)
			}

			// A sync's merge commit goes through the same pickGit.
			r.Git("branch", "side", "HEAD~1")
			r.Git("checkout", "-q", "side")
			r.Write("b.txt", "b\n")
			r.Git("add", "b.txt")
			r.Git("commit", "-q", "-m", "side")
			r.Git("checkout", "-q", "main")
			if _, err := pickGit(r.Root(), "merge", "--no-ff", "--no-edit", "-m", "sync", "side"); err != nil {
				t.Fatalf("the sync merge must still be made: %v", err)
			}
			if _, err := os.Stat(mark); err == nil {
				t.Fatalf("pickGit's merge commit started the repository's %s", tc.key)
			}

			// merge.verifySignatures=true makes the merge verify the merged
			// tip's signature, which starts gpg.program for a tip whose commit
			// object carries a gpgsig header.
			r.Git("config", "merge.verifySignatures", "true")
			r.Git("config", "gpg.program", signer)
			signedTip := signedCommit(t, r, "signed tip")
			plainMerge := exec.Command("git", "-C", r.Root(), "merge", "--no-ff", "--no-edit", "-m", "plain", signedTip)
			plainMerge.Env = gitutil.ScrubbedEnv()
			_ = plainMerge.Run()
			if _, err := os.Stat(mark); err != nil {
				t.Fatal("fixture: a plain merge of the signed tip did not verify it, so this test proves nothing")
			}
			if err := os.Remove(mark); err != nil {
				t.Fatal(err)
			}
			_, mergeErr := pickGit(r.Root(), "merge", "--no-ff", "--no-edit", "-m", "sync", signedTip)
			if _, err := os.Stat(mark); err == nil {
				t.Fatal("pickGit's merge verified the merged tip's signature, starting the repository's gpg.program")
			}
			if mergeErr != nil {
				t.Fatalf("the sync merge of a signed tip must still be made: %v", mergeErr)
			}
		})
	}
}
