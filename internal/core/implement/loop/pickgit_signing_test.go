package loop

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
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

// withoutPersonalConfig keeps the developer's global and system git config out
// of the commands a test runs through commitGit, which reads them as the
// person's own git does, so a machine whose global config names a hooks
// directory or a signing key cannot change what the test proves.
func withoutPersonalConfig(t *testing.T) {
	t.Helper()
	empty := filepath.Join(t.TempDir(), "gitconfig")
	if err := os.WriteFile(empty, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_CONFIG_GLOBAL", empty)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
}

// stubSigner writes a gpg.program that leaves a mark for each use: signing
// (git's `-bsau`) answers with a signature git accepts, and verifying
// (`--verify`) answers that the signature is bad. It returns the program and
// the two marks.
func stubSigner(t *testing.T) (program, signed, verified string) {
	t.Helper()
	dir := t.TempDir()
	signed, verified = filepath.Join(dir, "signed"), filepath.Join(dir, "verified")
	program = filepath.Join(dir, "stub-gpg.sh")
	body := "#!/bin/sh\n" +
		"for a in \"$@\"; do [ \"$a\" = --verify ] && { touch " + verified + "; cat >/dev/null; echo '[GNUPG:] BADSIG 0000000000000000 Fixture'; exit 1; }; done\n" +
		"touch " + signed + "\n" +
		"cat >/dev/null\n" +
		"echo '[GNUPG:] BEGIN_SIGNING' >&2\n" +
		"echo '[GNUPG:] SIG_CREATED D 1 8 00 1700000000 0000000000000000000000000000000000000000' >&2\n" +
		"printf '%s\\n' '-----BEGIN PGP SIGNATURE-----' '' 'c3R1Yg==' '-----END PGP SIGNATURE-----'\n"
	if err := os.WriteFile(program, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	return program, signed, verified
}

// TestThePickCommitAndTheSyncMergeAreSignedAsTheRepositorySays is
// iss-2610111729399920. A repository that sets commit.gpgsign=true signs every
// commit the person's own git makes, so the pick's record commit and a sync's
// merge commit are signed with the program it names, and a repository that
// sets merge.verifySignatures=true has the sync's merge refuse a tip whose
// signature does not verify, as the person's own `git merge` does: abcd
// switches neither off (principle abcd-fixes-what-it-adds).
func TestThePickCommitAndTheSyncMergeAreSignedAsTheRepositorySays(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX stub script")
	}
	withoutPersonalConfig(t)
	r := gittest.NewRepo(t)
	r.Write("a.md", "a\n")
	r.Commit("seed")
	program, signed, verified := stubSigner(t)
	for k, v := range map[string]string{
		"user.name": "Fixture", "user.email": "fixture@example.invalid",
		"commit.gpgsign": "true", "gpg.program": program,
	} {
		r.Git("config", k, v)
	}

	r.Write("a.md", "picked\n")
	if _, err := commitGit(r.Root(), "commit", "-q", "-m", "pick", "--", "a.md"); err != nil {
		t.Fatalf("the pick commit must be made: %v", err)
	}
	if _, err := os.Stat(signed); err != nil {
		t.Fatal("the pick commit did not start the repository's gpg.program to sign it")
	}
	if obj := r.Git("cat-file", "commit", "HEAD"); !strings.Contains(obj, "gpgsig -----BEGIN PGP SIGNATURE-----") {
		t.Fatalf("the pick commit is not signed:\n%s", obj)
	}
	if err := os.Remove(signed); err != nil {
		t.Fatal(err)
	}

	r.Git("branch", "side", "HEAD~1")
	r.Git("checkout", "-q", "side")
	r.Write("b.txt", "b\n")
	r.Git("add", "b.txt")
	r.Git("-c", "commit.gpgsign=false", "commit", "-q", "-m", "side")
	r.Git("checkout", "-q", "main")
	if err := syncMerge(r.Root(), "sync", "side"); err != nil {
		t.Fatalf("the sync merge must be made: %v", err)
	}
	if _, err := os.Stat(signed); err != nil {
		t.Fatal("the sync's merge commit did not start the repository's gpg.program to sign it")
	}
	if obj := r.Git("cat-file", "commit", "HEAD"); !strings.Contains(obj, "gpgsig -----BEGIN PGP SIGNATURE-----") {
		t.Fatalf("the sync's merge commit is not signed:\n%s", obj)
	}

	r.Git("config", "merge.verifySignatures", "true")
	head := r.Git("rev-parse", "HEAD")
	signedTip := signedCommit(t, r, "signed tip")
	if err := syncMerge(r.Root(), "sync", signedTip); err == nil {
		t.Fatal("the sync merged a tip whose signature does not verify, where the repository's merge.verifySignatures refuses it")
	}
	if _, err := os.Stat(verified); err != nil {
		t.Fatal("the sync merge did not verify the merged tip's signature with the repository's gpg.program")
	}
	if got := r.Git("rev-parse", "HEAD"); got != head {
		t.Fatalf("the refused sync merge moved HEAD to %s", got)
	}
}

// TestThePickCommitRunsTheRepositorysHooks is iss-2610111729399920: the
// repository's own hooks run on the pick's record commit and on a sync's
// merge commit, as they do on the person's own commit and merge, and a hook
// that refuses the commit refuses it.
func TestThePickCommitRunsTheRepositorysHooks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX hook script")
	}
	withoutPersonalConfig(t)
	r := gittest.NewRepo(t)
	r.Git("config", "user.name", "Fixture")
	r.Git("config", "user.email", "fixture@example.invalid")
	r.Write("a.md", "a\n")
	r.Commit("seed")
	marks := t.TempDir()
	for _, h := range []string{"pre-commit", "commit-msg", "pre-merge-commit"} {
		hook := filepath.Join(r.Root(), ".git", "hooks", h)
		if err := os.WriteFile(hook, []byte("#!/bin/sh\ntouch "+filepath.Join(marks, h)+"\n[ ! -e "+filepath.Join(marks, "refuse")+" ]\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	ran := func(h string) bool {
		_, err := os.Stat(filepath.Join(marks, h))
		return err == nil
	}

	r.Write("a.md", "picked\n")
	if _, err := commitGit(r.Root(), "commit", "-q", "-m", "pick", "--", "a.md"); err != nil {
		t.Fatalf("the pick commit must be made: %v", err)
	}
	if !ran("pre-commit") || !ran("commit-msg") {
		t.Fatal("the pick commit did not run the repository's pre-commit and commit-msg hooks")
	}

	r.Git("branch", "side", "HEAD~1")
	r.Git("checkout", "-q", "side")
	r.Write("b.txt", "b\n")
	r.Git("add", "b.txt")
	r.Git("commit", "-q", "--no-verify", "-m", "side")
	r.Git("checkout", "-q", "main")
	if err := syncMerge(r.Root(), "sync", "side"); err != nil {
		t.Fatalf("the sync merge must be made: %v", err)
	}
	if !ran("pre-merge-commit") {
		t.Fatal("the sync's merge commit did not run the repository's pre-merge-commit hook")
	}

	if err := os.WriteFile(filepath.Join(marks, "refuse"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	head := r.Git("rev-parse", "HEAD")
	r.Write("a.md", "refused\n")
	if _, err := commitGit(r.Root(), "commit", "-q", "-m", "pick", "--", "a.md"); err == nil {
		t.Fatal("a pre-commit hook that refuses did not refuse the pick commit")
	}
	if got := r.Git("rev-parse", "HEAD"); got != head {
		t.Fatalf("the refused pick commit moved HEAD to %s", got)
	}
}
