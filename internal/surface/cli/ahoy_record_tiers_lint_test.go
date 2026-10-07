package cli

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/intentdriven/abcd/internal/core/repolint"
	"github.com/intentdriven/abcd/internal/gittest"
)

// TestAhoyInstallLeavesTheLayoutLintClean is iss-2610071538028804 end to end:
// a repository `abcd ahoy install` reports set up passes the repository lint's
// three-tier-layout rule, and its decision-durability rule, with nothing done
// by hand in between.
func TestAhoyInstallLeavesTheLayoutLintClean(t *testing.T) {
	repo := hermeticRepo(t)
	// The lint reads git (tracked files, ignore rules), so the stand-in .git
	// the hermetic repo carries becomes a real, empty repository.
	if err := os.RemoveAll(filepath.Join(repo, ".git")); err != nil {
		t.Fatal(err)
	}
	git := exec.Command("git", "init", "-q", repo)
	git.Env = gittest.Env(t)
	if b, err := git.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, b)
	}
	out := runCLI(t, "ahoy", "install", "--yes", "--adopt",
		"--visibility", "private", "--docs-target", "agents_md",
		"--oracle-backend", "host-delegated", "--scan-deep", "false", "--json")
	var res struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(out, &res); err != nil {
		t.Fatalf("install output not JSON: %v\n%s", err, out)
	}
	if res.Status != "clean" {
		t.Fatalf("install status = %q, want clean\n%s", res.Status, out)
	}
	result, err := repolint.Evaluate(repolint.DefaultRules(), repolint.Context{RepoRoot: repo})
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range result.Findings {
		if f.RuleID == "three-tier-layout" || f.RuleID == "decision-durability" {
			t.Errorf("%s after a clean install: %s (%s)", f.RuleID, f.Message, f.File)
		}
	}
}
