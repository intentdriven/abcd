package sweep

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// TestTheDeleteProofReadsAncestryAsStoredUnderAGraft: a graft that gives the
// default branch's tip an unmerged branch's tip as a parent makes that branch
// look kept by the default branch for a read that honours grafts. The delete
// proof reads ancestry as stored, so the branch is not proven kept.
func TestTheDeleteProofReadsAncestryAsStoredUnderAGraft(t *testing.T) {
	r := gittest.NewRepo(t)
	r.Write("README.md", "a project\n")
	r.Commit("start")
	start := r.Git("rev-parse", "HEAD")
	r.Git("switch", "-q", "-c", "feat")
	r.Write("f.txt", "the feature\n")
	r.Commit("the feature")
	feat := r.Git("rev-parse", "HEAD")
	r.Git("switch", "-q", "main")
	r.Write("m.txt", "the main line\n")
	r.Commit("the main line")
	main := r.Git("rev-parse", "HEAD")
	info := filepath.Join(r.Root(), ".git", "info")
	if err := os.MkdirAll(info, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(info, "grafts"), []byte(main+" "+start+" "+feat+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	r.Git("-c", "advice.graftFileDeprecated=false", "merge-base", "--is-ancestor", feat, "main")

	if keptBy(r.Root(), "refs/heads/main", "refs/heads/feat", feat, nil) {
		t.Fatal("the proof finds feat kept by main; its commit is on no stored line of main")
	}
}
