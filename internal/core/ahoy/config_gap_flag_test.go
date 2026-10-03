package ahoy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestConfigValueGapsNameTheFlagThatAnswersThem is iss-2609120447486547: the
// required config gaps said "ahoy install prompts for the value", but in a piped
// run the flag is the only reliable answer, so each gap's fix hint names the
// flag, and the flag it names is the one the value question's help carries.
func TestConfigValueGapsNameTheFlagThatAnswersThem(t *testing.T) {
	setupHermetic(t)
	harnessFixture(t, "")
	repo := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(repo, ".abcd"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, ".abcd", "config.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	det, err := Detect(repo)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"config.visibility_missing":     "--visibility",
		"config.docs_target_missing":    "--docs-target",
		"config.oracle_backend_missing": "--oracle-backend",
	}
	for id, flag := range want {
		var g *Gap
		for i := range det.Gaps {
			if det.Gaps[i].ID == id {
				g = &det.Gaps[i]
			}
		}
		if g == nil {
			t.Errorf("%s not raised on an empty config: %+v", id, det.Gaps)
			continue
		}
		if !strings.Contains(g.FixHint, flag) {
			t.Errorf("%s fix hint does not name %s: %q", id, flag, g.FixHint)
		}
	}
	// scan.deep is raised only beside trufflehog on PATH, so its gap is built
	// directly: the same constructor, the same contract.
	if g := configValueGap("config.scan_deep_missing", "scan_deep", "t", "d"); !strings.Contains(g.FixHint, "--scan-deep") {
		t.Errorf("scan.deep fix hint does not name --scan-deep: %q", g.FixHint)
	}
	for key, flag := range map[string]string{"visibility": "--visibility", "docs_target": "--docs-target",
		"oracle_backend": "--oracle-backend", "scan_deep": "--scan-deep"} {
		h, ok := helpFor(key)
		if !ok || h.Flag != flag {
			t.Errorf("helpFor(%q).Flag = %q, want %q", key, h.Flag, flag)
		}
	}
}

// TestGitignoreDriftGapDoesNotClaimAPrompt: the drift gap is closed by
// rewriting the block for the visibility already persisted, so no question is
// asked and its fix hint must not say one is.
func TestGitignoreDriftGapDoesNotClaimAPrompt(t *testing.T) {
	g := gitignoreDriftGap("private")
	if strings.Contains(g.FixHint, "prompt") {
		t.Errorf("the drift gap claims a prompt it never asks: %q", g.FixHint)
	}
	if g.ID != "visibility.gitignore_drift" || !g.Required || !g.Resolvable || g.Category != ConfigChange {
		t.Errorf("the drift gap's contract changed: %+v", g)
	}
}
