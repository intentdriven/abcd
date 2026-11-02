package gitutil_test

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// remoteWorld is a checkout whose origin is a local bare repository holding
// main and develop, with origin's HEAD on head.
func remoteWorld(t *testing.T, head string) (*gittest.Repo, string) {
	t.Helper()
	r := gittest.NewRepo(t)
	r.Commit("c0")
	bare := filepath.Join(t.TempDir(), "origin.git")
	gittest.InitBare(t, bare)
	r.Git("remote", "add", "origin", bare)
	r.Git("push", "-q", "origin", "main", "main:develop")
	if _, err := gitutil.Run(bare, "symbolic-ref", "HEAD", "refs/heads/"+head); err != nil {
		t.Fatal(err)
	}
	return r, bare
}

func TestRemoteHeadBranchNamesOriginsHEADBranch(t *testing.T) {
	for _, head := range []string{"main", "develop"} {
		r, _ := remoteWorld(t, head)
		before := r.Git("for-each-ref")
		got, err := gitutil.RemoteHeadBranch(nil, r.Root(), "origin")
		if err != nil || got != head {
			t.Fatalf("RemoteHeadBranch = %q, %v; want %q", got, err, head)
		}
		// It asks; it never fetches and never writes a ref here.
		if after := r.Git("for-each-ref"); after != before {
			t.Fatalf("refs changed:\nbefore %s\nafter %s", before, after)
		}
	}
}

func TestRemoteHeadBranchErrsOnAnUnreachableRemote(t *testing.T) {
	r, bare := remoteWorld(t, "main")
	if err := os.RemoveAll(bare); err != nil {
		t.Fatal(err)
	}
	if got, err := gitutil.RemoteHeadBranch(nil, r.Root(), "origin"); err == nil {
		t.Fatalf("RemoteHeadBranch = %q, nil; want an error", got)
	}
}

func TestRemoteHeadBranchErrsOnADetachedHEAD(t *testing.T) {
	r, bare := remoteWorld(t, "main")
	sha := r.Git("rev-parse", "HEAD")
	if _, err := gitutil.Run(bare, "update-ref", "--no-deref", "HEAD", sha); err != nil {
		t.Fatal(err)
	}
	if got, err := gitutil.RemoteHeadBranch(nil, r.Root(), "origin"); err == nil {
		t.Fatalf("RemoteHeadBranch = %q, nil; want an error for a detached HEAD", got)
	}
}

// A HEAD naming a branch whose name is not a valid ref is not advertised as
// a symref at all, so the live answer carries no symref line.
func TestRemoteHeadBranchErrsOnAnInvalidName(t *testing.T) {
	r, bare := remoteWorld(t, "main")
	sha := r.Git("rev-parse", "HEAD")
	if err := os.WriteFile(filepath.Join(bare, "refs", "heads", "a..b"), []byte(sha+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bare, "HEAD"), []byte("ref: refs/heads/a..b\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got, err := gitutil.RemoteHeadBranch(nil, r.Root(), "origin"); err == nil {
		t.Fatalf("RemoteHeadBranch = %q, nil; want an error for an invalid name", got)
	}
}

// liveAbcdAnswer is `git ls-remote --symref origin HEAD` as github.com
// answered it for intentdriven/abcd on 2026-10-16, byte for byte.
const liveAbcdAnswer = "ref: refs/heads/main\tHEAD\n816b5286bb3bfd133378aa7f880b0affc8903823\tHEAD\n"

func TestRemoteHeadFromParsesEachAnswer(t *testing.T) {
	root := gittest.NewRepo(t).Root()
	sha := "816b5286bb3bfd133378aa7f880b0affc8903823"
	cases := []struct {
		name, out, want string
	}{
		{"the forge's live answer", liveAbcdAnswer, "main"},
		{"a slashed branch", "ref: refs/heads/release/v1\tHEAD\n" + sha + "\tHEAD\n", "release/v1"},
		{"no symref line", sha + "\tHEAD\n", ""},
		{"nothing at all", "", ""},
		{"a target outside refs/heads", "ref: refs/tags/v1\tHEAD\n" + sha + "\tHEAD\n", ""},
		{"an empty name", "ref: refs/heads/\tHEAD\n", ""},
		{"a branch named HEAD", "ref: refs/heads/HEAD\tHEAD\n" + sha + "\tHEAD\n", ""},
		{"an invalid name", "ref: refs/heads/a..b\tHEAD\n" + sha + "\tHEAD\n", ""},
		{"a name with a space", "ref: refs/heads/a b\tHEAD\n", ""},
		{"a lock name", "ref: refs/heads/x.lock\tHEAD\n", ""},
		{"a symref for another ref", "ref: refs/heads/main\trefs/remotes/x\n", ""},
	}
	for _, c := range cases {
		got, err := gitutil.RemoteHeadFrom(root, "origin", c.out)
		if c.want == "" {
			if err == nil {
				t.Errorf("%s: got %q, nil; want an error", c.name, got)
			}
			continue
		}
		if err != nil || got != c.want {
			t.Errorf("%s: got %q, %v; want %q", c.name, got, err, c.want)
		}
	}
}

// The caller's runner asks the remote (iss-2610161849042422): it gets the one
// ls-remote call, from root, and its answer is the one read; an option-shaped
// remote never reaches it.
func TestRemoteHeadBranchAsksThroughTheCallersRunner(t *testing.T) {
	root := gittest.NewRepo(t).Root()
	var calls [][]string
	git := func(dir string, args ...string) (string, error) {
		if dir != root {
			t.Errorf("runner called from %q, want %q", dir, root)
		}
		calls = append(calls, args)
		return "ref: refs/heads/trunk\tHEAD\n", nil
	}
	got, err := gitutil.RemoteHeadBranch(git, root, "origin")
	if err != nil || got != "trunk" {
		t.Fatalf("RemoteHeadBranch = %q, %v; want trunk, the runner's answer", got, err)
	}
	if want := []string{"ls-remote", "--symref", "origin", "HEAD"}; len(calls) != 1 || !slices.Equal(calls[0], want) {
		t.Fatalf("runner calls %v, want one %v", calls, want)
	}
	if _, err := gitutil.RemoteHeadBranch(git, root, "--upload-pack=x"); err == nil || len(calls) != 1 {
		t.Fatalf("an option-shaped remote: err %v, runner calls %d; want an error and no call", err, len(calls))
	}
}
