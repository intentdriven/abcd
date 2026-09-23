package ahoy

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/banlist"
	"github.com/intentdriven/abcd/internal/core/lint"
	"github.com/intentdriven/abcd/internal/gittest"
)

// The em-dash-in-list-item house-style rule is offered at install (ruling G1,
// 2026-09-23): the adopter chooses whether it blocks or warns, the choice is
// written into the seeded docs-lint config, and an unattended --yes install
// seeds it as a warning. These tests hold each branch of that choice at the
// install seam, and prove the seeded severity is the one the lint then enforces.

// interactiveInstallOpts is installOpts without --yes: every category is asked,
// so the em-dash question is asked too.
func interactiveInstallOpts() InstallOptions {
	o := installOpts()
	o.Yes = false
	return o
}

// seededEmDashSeverity reads the severity the scaffolded docs-lint config gives
// the em-dash token, failing when the token is absent.
func seededEmDashSeverity(t *testing.T, repo string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(banlist.PublicConfigRelPath)))
	if err != nil {
		t.Fatalf("no seeded docs-lint config: %v", err)
	}
	var cfg struct {
		BannedTokens []lint.BannedToken `json:"banned_tokens"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("seeded docs-lint config is not JSON: %v", err)
	}
	for _, tok := range cfg.BannedTokens {
		if tok.ID == emDashTokenID {
			return tok.Severity
		}
	}
	t.Fatalf("the seeded docs-lint config carries no %s token", emDashTokenID)
	return ""
}

func askedKey(asked []string, key string) bool {
	for _, a := range asked {
		if a == key {
			return true
		}
	}
	return false
}

func notesMention(notes []string, sub string) bool {
	for _, n := range notes {
		if strings.Contains(n, sub) {
			return true
		}
	}
	return false
}

// TestEmDashRuleSeverityIsTheAdoptersChoice asks the question interactively and
// seeds what was answered. "No answer" is the adopter declining to choose (a
// bare Enter, or end of input): it takes the displayed default, warning, the
// same severity an unattended install seeds, and the seeded config records it.
// An answer that names neither choice (the "y" a `yes |` pipe sends) is not
// guessed into a blocker: it seeds the warning and says so in a note.
func TestEmDashRuleSeverityIsTheAdoptersChoice(t *testing.T) {
	cases := []struct {
		name     string
		answers  map[string]string
		want     string
		wantNote bool
	}{
		{"blocking", map[string]string{emDashPromptKey: "blocking"}, "blocker", false},
		{"warning", map[string]string{emDashPromptKey: "warning"}, "warn", false},
		{"blocker spelled as the config spells it", map[string]string{emDashPromptKey: "Blocker"}, "blocker", false},
		{"no answer", nil, "warn", false},
		{"an answer naming neither choice", map[string]string{emDashPromptKey: "y"}, "warn", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			setupHermetic(t)
			repo := gittest.NewRepo(t).Root()
			p := offerPrompter(false, tc.answers)
			res, err := Install(repo, interactiveInstallOpts(), p)
			if err != nil {
				t.Fatal(err)
			}
			if !askedKey(p.asked, emDashPromptKey) {
				t.Fatalf("the install never asked %s; asked %v", emDashPromptKey, p.asked)
			}
			if got := seededEmDashSeverity(t, repo); got != tc.want {
				t.Errorf("seeded severity = %q, want %q", got, tc.want)
			}
			if got := notesMention(res.Notes, emDashTokenID); got != tc.wantNote {
				t.Errorf("note naming %s present = %v, want %v; notes %v", emDashTokenID, got, tc.wantNote, res.Notes)
			}
		})
	}
}

// TestUnattendedInstallSeedsEmDashAsWarning is the --yes half of the ruling: an
// unattended install is not asked, seeds the rule as a warning, and says that it
// chose on the adopter's behalf. The prompter would answer "blocking" if asked,
// so a --yes run that consulted it would seed a blocker and fail here.
func TestUnattendedInstallSeedsEmDashAsWarning(t *testing.T) {
	setupHermetic(t)
	repo := gittest.NewRepo(t).Root()
	p := offerPrompter(false, map[string]string{emDashPromptKey: "blocking"})
	res, err := Install(repo, installOpts(), p)
	if err != nil {
		t.Fatal(err)
	}
	if askedKey(p.asked, emDashPromptKey) {
		t.Errorf("an unattended install asked %s", emDashPromptKey)
	}
	if got := seededEmDashSeverity(t, repo); got != "warn" {
		t.Errorf("seeded severity under --yes = %q, want warn", got)
	}
	if !notesMention(res.Notes, emDashTokenID) {
		t.Errorf("an unattended install must say it seeded %s as a warning without asking; notes %v", emDashTokenID, res.Notes)
	}
}

// TestEmDashQuestionIsNotAskedOverAnExistingConfig: the seed is create-if-absent,
// so a repository that already carries a docs-lint config keeps its own choice
// and is not asked a question whose answer would go nowhere.
func TestEmDashQuestionIsNotAskedOverAnExistingConfig(t *testing.T) {
	setupHermetic(t)
	repo := gittest.NewRepo(t).Root()
	if _, err := Install(repo, installOpts(), RefusingPrompter{}); err != nil {
		t.Fatal(err)
	}
	p := offerPrompter(false, map[string]string{emDashPromptKey: "blocking"})
	if _, err := Install(repo, interactiveInstallOpts(), p); err != nil {
		t.Fatal(err)
	}
	if askedKey(p.asked, emDashPromptKey) {
		t.Errorf("a re-install over an existing docs-lint config asked %s", emDashPromptKey)
	}
	if got := seededEmDashSeverity(t, repo); got != "warn" {
		t.Errorf("the existing config's severity changed to %q", got)
	}
}

// TestSeededEmDashSeverityIsTheOneTheLintEnforces proves the choice is live
// (guards-prove-themselves): an em dash in a list item inside a configured root
// is a blocker when the adopter chose blocking and a warning when they chose
// warning, and a list item without one raises nothing.
func TestSeededEmDashSeverityIsTheOneTheLintEnforces(t *testing.T) {
	for _, answer := range []string{"blocking", "warning"} {
		t.Run(answer, func(t *testing.T) {
			setupHermetic(t)
			repo := gittest.NewRepo(t).Root()
			if err := os.MkdirAll(filepath.Join(repo, "docs"), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("# Example\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			guide := filepath.Join(repo, "docs", "guide.md")
			if err := os.WriteFile(guide, []byte("# Guide\n\n- Run it: the check reads the tree.\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			p := offerPrompter(false, map[string]string{emDashPromptKey: answer})
			if _, err := Install(repo, interactiveInstallOpts(), p); err != nil {
				t.Fatal(err)
			}
			cfg, err := lint.LoadConfig(filepath.Join(repo, filepath.FromSlash(banlist.PublicConfigRelPath)))
			if err != nil {
				t.Fatalf("seeded config does not load: %v", err)
			}
			emDashFindings := func() []lint.Finding {
				t.Helper()
				fs, err := lint.Lint(cfg, repo)
				if err != nil {
					t.Fatal(err)
				}
				var out []lint.Finding
				for _, f := range fs {
					if f.RuleID == emDashTokenID {
						out = append(out, f)
					}
				}
				return out
			}
			if fs := emDashFindings(); len(fs) != 0 {
				t.Fatalf("a list item with no em dash drew %v", fs)
			}
			if err := os.WriteFile(guide, []byte("# Guide\n\n- Run it — the check reads the tree.\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			want := map[string]string{"blocking": "blocker", "warning": "warn"}[answer]
			fs := emDashFindings()
			if len(fs) != 1 || fs[0].Severity != want {
				t.Fatalf("an em dash in a list item drew %+v, want one %s finding", fs, want)
			}
		})
	}
}
