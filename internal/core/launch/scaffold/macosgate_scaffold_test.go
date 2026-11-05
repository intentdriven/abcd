package scaffold

import (
	"strings"
	"testing"
)

// TestOnlyAbcdsRenderingRunsAMacOSSuite holds the release path's macOS gate
// (iss-2611042224527148) to abcd's own rendering. A scaffolded repository did not
// ask for macOS jobs, may not build on macOS at all, and on a private repository
// pays for each runner minute, and it receives no macos-suite.yml for the jobs to
// call: a call to a workflow it does not have fails its whole release run before
// any job starts. The non-Abcd profiles therefore render no trace of it, in the
// workflows or the runbook.
func TestOnlyAbcdsRenderingRunsAMacOSSuite(t *testing.T) {
	semantic := BareSubstitutions("main")
	semantic.SemanticGates = []string{"docs-currency-reviewer"}
	for name, subs := range map[string]Substitutions{
		"bare":     BareSubstitutions("main"),
		"semantic": semantic,
		"gate":     GateSubstitutions("main", ""),
	} {
		t.Run(name, func(t *testing.T) {
			r, err := Render(subs)
			if err != nil {
				t.Fatal(err)
			}
			for file, body := range map[string][]byte{
				"release workflow": r.ReleaseYML,
				"auto-release.yml": r.AutoReleaseYML,
				"runbook":          r.Runbook,
			} {
				lower := strings.ToLower(string(body))
				for _, word := range []string{"macos", "macos-latest", "macos-suite", "macos-gate"} {
					if strings.Contains(lower, word) {
						t.Errorf("the %s profile's %s names %q; a scaffolded repository asked for no macOS jobs", name, file, word)
					}
				}
			}
		})
	}

	r, err := Render(AbcdSubstitutions())
	if err != nil {
		t.Fatal(err)
	}
	rel := string(r.ReleaseYML)
	if !strings.Contains(rel, "uses: ./.github/workflows/macos-suite.yml") {
		t.Error("abcd's own rendering runs no macOS suite; the gate belongs to its release path")
	}
	for _, job := range []string{"tag", "release", "site"} {
		block := jobSection(t, rel, job)
		if !strings.Contains(block, "macos-gate") {
			t.Errorf("abcd's rendering: the %s job does not need macos-gate:\n%s", job, block)
		}
	}
}
