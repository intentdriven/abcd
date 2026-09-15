package update

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/ahoy"
)

// --- NextStep: one primitive renders the way forward for every shape ---

func TestNextStepNamesTheVerbForASwappableFile(t *testing.T) {
	got := NextStep(ahoy.UpdateTarget{Path: "/x/abcd", ResolvedPath: "/x/abcd", Kind: ahoy.UpdateTargetFile})
	if !strings.Contains(got, "`abcd update`") {
		t.Fatalf("a swappable file must be pointed at the update verb, got %q", got)
	}
}

// TestNextStepRelaysThePlanRemedyForEveryOtherShape pins that a non-swappable
// shape gets exactly the remedy `abcd update` itself would print — the text
// lives once, in Plan, and NextStep never paraphrases it.
func TestNextStepRelaysThePlanRemedyForEveryOtherShape(t *testing.T) {
	cases := []ahoy.UpdateTarget{
		{Path: "/x/abcd", Kind: ahoy.UpdateTargetPluginRoot},
		{Path: "/x/abcd", Kind: ahoy.UpdateTargetDevShim},
		{Path: "/x/abcd", Kind: ahoy.UpdateTargetDangling},
		{Path: "/x/abcd", Kind: ahoy.UpdateTargetForeign},
		{Kind: ahoy.UpdateTargetAbsent},
		{Path: "/opt/homebrew/bin/abcd", ResolvedPath: "/opt/homebrew/Cellar/abcd/0.6.1/bin/abcd", Kind: ahoy.UpdateTargetForeign},
	}
	for _, tgt := range cases {
		r := Plan(tgt)
		if r == nil {
			t.Fatalf("%+v must be a refusal shape", tgt)
		}
		if got := NextStep(tgt); got != r.Remedy {
			t.Errorf("NextStep(%s) = %q, want Plan's own remedy %q", tgt.Kind, got, r.Remedy)
		}
	}
}

// --- TooNew: the canonical schema-too-new refusal names the next step ---

func TestTooNewNamesTheNextStepForTheInstallShape(t *testing.T) {
	orig := resolveTarget
	t.Cleanup(func() { resolveTarget = orig })

	resolveTarget = func() ahoy.UpdateTarget {
		return ahoy.UpdateTarget{Path: "/x/abcd", ResolvedPath: "/x/abcd", Kind: ahoy.UpdateTargetFile}
	}
	err := TooNew("lifeboat", 3, 2)
	if err == nil {
		t.Fatal("TooNew must return an error")
	}
	msg := err.Error()
	for _, want := range []string{"lifeboat schema v3", "this abcd knows up to v2", "`abcd update`"} {
		if !strings.Contains(msg, want) {
			t.Errorf("TooNew on a swappable install = %q, want it to carry %q", msg, want)
		}
	}
	if strings.Contains(msg, "upgrade"+" abcd") {
		t.Errorf("TooNew must name the verb, not the bare phrase: %q", msg)
	}

	resolveTarget = func() ahoy.UpdateTarget {
		return ahoy.UpdateTarget{Path: "/x/abcd", Kind: ahoy.UpdateTargetPluginRoot}
	}
	if msg := TooNew("verdict", 9, 1).Error(); !strings.Contains(msg, "plugin update") {
		t.Errorf("TooNew on a plugin-root install must relay the plugin remedy, got %q", msg)
	}
}

// --- Guard: the bare phrase never regrows outside the one remedy that owns it ---

// TestNoBareUpgradeAbcdOutsideTheRemedy walks the shipped source and prose and
// refuses the verbless phrase wherever it is not Homebrew's own `brew upgrade
// abcd` — the one remedy where the words are a real command. Every schema-too-new
// site and every next-step line routes through NextStep/TooNew instead, so a
// new site written by hand cannot silently bring the verbless phrase back
// (iss-2609012111168872).
func TestNoBareUpgradeAbcdOutsideTheRemedy(t *testing.T) {
	root := repoRoot(t)
	// Spelled in two halves so this file does not trip its own guard.
	phrase := "upgrade" + " abcd"
	var findings []string
	for _, dir := range []string{"internal", "cmd", "commands", "docs", "hooks"} {
		base := filepath.Join(root, dir)
		if _, err := os.Stat(base); err != nil {
			continue
		}
		err := filepath.WalkDir(base, func(p string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if d.Name() == "testdata" {
					return filepath.SkipDir
				}
				return nil
			}
			switch filepath.Ext(p) {
			case ".go", ".md", ".sh":
			default:
				return nil
			}
			raw, err := os.ReadFile(p)
			if err != nil {
				return err
			}
			text := string(raw)
			for off := 0; ; {
				i := strings.Index(text[off:], phrase)
				if i < 0 {
					break
				}
				at := off + i
				if !strings.HasSuffix(text[:at], "brew ") {
					line := 1 + strings.Count(text[:at], "\n")
					rel, _ := filepath.Rel(root, p)
					findings = append(findings, filepath.ToSlash(rel)+":"+strconv.Itoa(line))
				}
				off = at + len(phrase)
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", dir, err)
		}
	}
	if len(findings) > 0 {
		t.Fatalf("bare %q outside the brew remedy — route the site through update.TooNew or update.NextStep:\n  %s",
			phrase, strings.Join(findings, "\n  "))
	}
}

// repoRoot walks up from the test's working directory to the module root.
func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("no go.mod above the test directory")
		}
		dir = parent
	}
}
