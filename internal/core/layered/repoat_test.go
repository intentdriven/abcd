package layered

import (
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// TestTheRepositoryLayerReadsAtACommit is spc-2610161609350616 step 4's
// reader (dcn-2610230058333177): with RepoAt set, the repository layer is the
// file as that commit holds it, never the working tree's, so a branch that
// edits or deletes .abcd/config.json does not change what is read at the
// default branch's tip. A commit with no file reads as absent, and a file the
// commit holds as a symlink is refused, as the working-tree read refuses one.
func TestTheRepositoryLayerReadsAtACommit(t *testing.T) {
	repo := gittest.NewRepo(t)
	repo.Write(".abcd/config.json", `{"facilitator":{"mode":"human"}}`+"\n")
	repo.Commit("the default branch's config")
	base := repo.Git("rev-parse", "HEAD")
	repo.Write(".abcd/config.json", `{"facilitator":{"mode":"auto"}}`+"\n")
	repo.Commit("a branch that says auto")
	repo.Remove(".abcd/config.json")
	repo.Commit("a branch that deletes the file")
	empty := repo.Git("rev-parse", "HEAD")
	home := t.TempDir()

	s, err := Load(Config, Roots{Repo: repo.Root(), Home: home, RepoAt: base, RepoAtName: "origin/main"})
	if err != nil {
		t.Fatal(err)
	}
	found, err := s.Lookup("facilitator.mode")
	if err != nil || len(found) != 1 || string(found[0].Raw) != `"human"` {
		t.Fatalf("the layer at the commit says human: %+v %v", found, err)
	}
	if o := found[0].Origin; !strings.Contains(o, ".abcd/config.json") || !strings.Contains(o, "origin/main") {
		t.Fatalf("the origin names the file and the commit it was read at: %q", o)
	}

	s, err = Load(Config, Roots{Repo: repo.Root(), Home: home, RepoAt: empty, RepoAtName: "origin/main"})
	if err != nil {
		t.Fatal(err)
	}
	if s.Present(Repo) {
		t.Fatal("a commit with no file reads as an absent layer")
	}

	repo.Git("checkout", "-q", base)
	repo.Remove(".abcd/config.json")
	repo.Git("-c", "core.symlinks=true", "update-index", "--add", "--cacheinfo", "120000,"+strings.TrimSpace(hashObject(t, repo, "/etc/passwd"))+",.abcd/config.json")
	repo.Git("commit", "-q", "-m", "a symlink in its place")
	link := repo.Git("rev-parse", "HEAD")
	if _, err := Load(Config, Roots{Repo: repo.Root(), Home: home, RepoAt: link, RepoAtName: "origin/main"}); err == nil || !strings.Contains(err.Error(), "not a regular file") {
		t.Fatalf("a symlink at the commit is refused: %v", err)
	}
}

func hashObject(t *testing.T, repo *gittest.Repo, content string) string {
	t.Helper()
	repo.Write(".tmp-blob", content)
	defer repo.Remove(".tmp-blob")
	return repo.Git("hash-object", "-w", ".tmp-blob")
}
