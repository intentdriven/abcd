package lint

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// The local tier is each checkout's own: a file there that a commit carries
// reaches every clone, where abcd would read it as that clone's state. So the
// change that tracks one is refused at its record gate, naming the file.
func TestATrackedLocalTierFileIsRefused(t *testing.T) {
	rule := map[string]RuleConfig{ruleLocalTierUntracked: {Enabled: true, Severity: severityBlocker}}
	repo := gittest.NewRepo(t)
	repo.Write(".gitignore", ".abcd/.work.local/\n")
	repo.Write(".abcd/.work.local/scratch/note.md", "mine alone\n")
	repo.Commit("ignore the local tier")
	if fs, err := Lint(Config{Rules: rule}, repo.Root()); err != nil || len(fs) != 0 {
		t.Fatalf("an untracked local tier is clean: %+v %v", fs, err)
	}

	repo.Write(".abcd/.work.local/implement/drain.json", "{}\n")
	repo.Git("add", "-f", ".abcd/.work.local/implement/drain.json")
	repo.Commit("an unrelated change")
	fs, err := Lint(Config{Rules: rule}, repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	if len(fs) != 1 || fs[0].RuleID != ruleLocalTierUntracked || fs[0].File != filepath.FromSlash(".abcd/.work.local/implement/drain.json") {
		t.Fatalf("want one finding naming the tracked file, got %+v", fs)
	}
	if !strings.Contains(fs[0].Message, "git rm --cached") {
		t.Errorf("the finding says how to untrack it: %s", fs[0].Message)
	}
}

// Outside a repository nothing is tracked, so there is nothing to refuse; a
// repository git cannot answer for is an error, never a pass.
func TestTheLocalTierRuleFailsClosed(t *testing.T) {
	rule := map[string]RuleConfig{ruleLocalTierUntracked: {Enabled: true, Severity: severityBlocker}}
	if fs, err := Lint(Config{Rules: rule}, t.TempDir()); err != nil || len(fs) != 0 {
		t.Fatalf("outside a repository the rule has nothing to read: %+v %v", fs, err)
	}
	repo := gittest.NewRepo(t)
	repo.Write("README.md", "x\n")
	repo.Commit("seed")
	if err := os.WriteFile(filepath.Join(repo.Root(), ".git", "index"), []byte("not an index"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Lint(Config{Rules: rule}, repo.Root()); err == nil {
		t.Fatal("a repository git cannot list is an error, never a clean pass")
	}
}

// The shipped config arms the rule as a blocker.
func TestLocalTierUntrackedArmedInRealConfig(t *testing.T) {
	cfg, err := LoadConfig(filepath.Join("..", "..", "..", ".abcd", "record-lint.json"))
	if err != nil {
		t.Fatal(err)
	}
	if rc := cfg.Rules[ruleLocalTierUntracked]; !rc.Enabled || rc.Severity != severityBlocker {
		t.Fatalf("record-lint.json must arm %s as a blocker, got %+v", ruleLocalTierUntracked, rc)
	}
}

// A spelling a case-folding filesystem opens as the tier is refused on every
// filesystem, the one CI runs on included: the clone that folds it may be
// another one.
func TestATrackedSpellingThatFoldsOntoTheLocalTierIsRefused(t *testing.T) {
	rule := map[string]RuleConfig{ruleLocalTierUntracked: {Enabled: true, Severity: severityBlocker}}
	for _, spelled := range []string{".abcd/.Work.Local/run/drain.json", ".ABCD/.wor\u212a.local/run/drain.json"} {
		repo := gittest.NewRepo(t)
		repo.Write("seed.md", "x\n")
		repo.Commit("seed")
		blob := repo.Git("rev-parse", "HEAD:seed.md")
		repo.Git("update-index", "--add", "--cacheinfo", "100644,"+blob+","+spelled)
		fs, err := Lint(Config{Rules: rule}, repo.Root())
		if err != nil {
			t.Fatal(err)
		}
		if len(fs) != 1 || fs[0].File != filepath.FromSlash(spelled) {
			t.Fatalf("%s tracked: want one finding naming it, got %+v", spelled, fs)
		}
	}
}
