package ahoy

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// planPrompter is a person at a terminal who answers the plan screen with
// choice and every other question through confirm (no, when nil), recording
// each question in the order it was put. A value question takes its default.
type planPrompter struct {
	choice  PlanChoice
	confirm func(q string) bool
	plans   []SetupPlan
	asked   []string
}

func (p *planPrompter) Plan(sp SetupPlan) (PlanChoice, bool) {
	p.plans = append(p.plans, sp)
	p.asked = append(p.asked, PlanQuestionID)
	return p.choice, true
}
func (p *planPrompter) Confirm(q string) bool {
	p.asked = append(p.asked, q)
	return p.confirm != nil && p.confirm(q)
}
func (p *planPrompter) Prompt(key string, _ []string, def string) string {
	p.asked = append(p.asked, key)
	return def
}
func (p *planPrompter) AtTerminal() bool { return true }

// gitOnlyDir is a directory holding only a link to the machine's git, for a
// PATH narrowed to what the run under test may find.
func gitOnlyDir(t *testing.T) string {
	t.Helper()
	git, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git is not on PATH")
	}
	dir := t.TempDir()
	if err := os.Symlink(git, filepath.Join(dir, "git")); err != nil {
		t.Fatal(err)
	}
	return dir
}

// planOpts are the options of a person's run at a terminal: nothing answered
// in advance, the accepting provenance stand-in.
func planOpts() InstallOptions {
	return InstallOptions{VerifyProvenance: acceptProvenance}
}

// TestPlanProceedAsksOnceAndTakesTheDefaults is criteria 2 and 5: a repository
// abcd has never seen, at a terminal, with the suggested answer: one plan
// screen (which adopts), no approval of a kind of change, no value question,
// and every default the plan lists is what lands. The live brief session
// question, which only the owner answers, is the one question that follows,
// and the plan says so rather than listing it as a change.
func TestPlanProceedAsksOnceAndTakesTheDefaults(t *testing.T) {
	setupHermetic(t)
	repo := gitRepo(t)
	p := &planPrompter{choice: PlanProceed}

	res, err := Install(repo, planOpts(), p)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(p.asked, []string{PlanQuestionID, SessionsPromptKey}) {
		t.Fatalf("asked %q, want the plan screen and then the live brief session question", p.asked)
	}
	if res.Status == "aborted" || res.Status == "refused" {
		t.Fatalf("status %q, notes %q", res.Status, res.Notes)
	}
	plan := p.plans[0]
	if len(plan.AsksYou) == 0 || !strings.HasPrefix(plan.AsksYou[0], "the live brief session question") {
		t.Errorf("the plan does not list the live brief session question among its questions: %q", plan.AsksYou)
	}
	for _, g := range plan.Changes {
		for _, it := range g.Items {
			if strings.Contains(it, "live brief session") {
				t.Errorf("the plan lists the live brief session question as a change: %q", it)
			}
		}
	}
	if len(plan.Changes) == 0 || plan.Changes[0].Items[0] != "adopt this repository into abcd" {
		t.Errorf("the plan does not lead with the adoption: %+v", plan.Changes)
	}
	var keys []string
	for _, d := range plan.Defaults {
		keys = append(keys, d.Key)
		if d.ChangeWith == "" || d.Value == "" {
			t.Errorf("default %q does not say its value and where to change it: %+v", d.Key, d)
		}
	}
	for _, want := range []string{"visibility", "docs_target"} {
		if !slices.Contains(keys, want) {
			t.Errorf("the plan lists no default for %s: %v", want, keys)
		}
	}
	cfg, err := readConfig(repo)
	if err != nil {
		t.Fatal(err)
	}
	if repoSec, _ := cfg["repo"].(map[string]any); repoSec["visibility"] != "private" {
		t.Errorf("visibility = %v, want the plan's default private", repoSec["visibility"])
	}
}

// TestPlanCancelWritesNothing: cancel ends setup before its first write.
func TestPlanCancelWritesNothing(t *testing.T) {
	setupHermetic(t)
	repo := gitRepo(t)
	p := &planPrompter{choice: PlanCancel}
	res, err := Install(repo, planOpts(), p)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != "aborted" || len(res.Writes) != 0 {
		t.Fatalf("status %q writes %v, want aborted with nothing written", res.Status, res.Writes)
	}
	if _, err := os.Stat(filepath.Join(repo, ".abcd")); err == nil {
		t.Error("cancel created .abcd/")
	}
}

// TestPlanCustomiseAsksEachKindOfChange: customise asks each kind of change
// and each value as setup asked before the plan, and the plan adopts.
func TestPlanCustomiseAsksEachKindOfChange(t *testing.T) {
	setupHermetic(t)
	repo := gitRepo(t)
	p := &planPrompter{choice: PlanCustomise, confirm: func(string) bool { return true }}
	if _, err := Install(repo, planOpts(), p); err != nil {
		t.Fatal(err)
	}
	if slices.Contains(p.asked, "Adopt this unmanaged repo into abcd?") {
		t.Error("customise asked the adoption again after the plan")
	}
	approvals := 0
	for _, q := range p.asked {
		if askedCategory(q) != "" {
			approvals++
		}
	}
	if approvals == 0 {
		t.Errorf("customise asked no kind of change: %q", p.asked)
	}
	if !slices.Contains(p.asked, "visibility") {
		t.Errorf("customise did not ask the visibility: %q", p.asked)
	}
}

// TestDownloadComesBeforeTheStatusLine is criterion 4: with abcd not on the
// PATH, setup offers the download before the status line, and a yes wires the
// status line in the same run; declined, the status line is never asked.
func TestDownloadComesBeforeTheStatusLine(t *testing.T) {
	release := []byte("#!/bin/sh\necho abcd v9.9.9\n")
	sum := sha256.Sum256(release)
	fetch := func(tag string) (DownloadedRelease, error) {
		if tag != "v9.9.9" {
			return DownloadedRelease{}, errors.New("unexpected tag " + tag)
		}
		return DownloadedRelease{Bytes: release, SHA256: hex.EncodeToString(sum[:])}, nil
	}
	defer swapDownloadTag(func(string) string { return "v9.9.9" })()

	for _, tc := range []struct {
		name     string
		yes      bool
		wantLine bool
	}{{"yes", true, true}, {"no", false, false}} {
		t.Run(tc.name, func(t *testing.T) {
			home, _ := setupUserScope(t, gitOnlyDir(t))
			t.Setenv("ABCD_PLUGIN_ROOT", "")
			harnessFixture(t, harnessSettingsWith(""))
			repo := gitRepo(t)
			opts := planOpts()
			opts.DownloadAbcd = fetch
			p := &planPrompter{choice: PlanProceed, confirm: func(q string) bool {
				if isDownloadQuestion(q) {
					return tc.yes
				}
				return isStatusLineOfferQuestion(q)
			}}
			res, err := Install(repo, opts, p)
			if err != nil {
				t.Fatal(err)
			}
			dl := slices.IndexFunc(p.asked, isDownloadQuestion)
			sl := slices.IndexFunc(p.asked, isStatusLineOfferQuestion)
			if dl < 0 {
				t.Fatalf("the download was not offered: %q (notes %q)", p.asked, res.Notes)
			}
			_, ask := splitConfirm(p.asked[dl])
			if !strings.Contains(ask, "v9.9.9") || !strings.Contains(ask, "~/.local/bin/abcd") {
				t.Errorf("the download's one line does not name the release and where it goes: %q", ask)
			}
			if !strings.Contains(strings.Join(p.plans[0].AsksYou, "\n"), "abcd download") {
				t.Errorf("the plan does not list the download among its questions: %q", p.plans[0].AsksYou)
			}
			target := filepath.Join(home, ".local", "bin", "abcd")
			if tc.yes {
				if got, _ := os.ReadFile(target); string(got) != string(release) {
					t.Fatalf("the verified release was not written to %s (notes %q)", target, res.Notes)
				}
				if sl < 0 || sl < dl {
					t.Fatalf("the status line was not offered after the download: %q (notes %q)", p.asked, res.Notes)
				}
				if !slices.Contains(res.writeKinds, writeStatusLine) {
					t.Errorf("the status line was not wired in the same run (notes %q)", res.Notes)
				}
				return
			}
			if sl >= 0 {
				t.Errorf("the status line was asked about with abcd not on the PATH: %q", p.asked)
			}
			if _, err := os.Stat(target); err == nil {
				t.Error("a declined download wrote the binary")
			}
		})
	}
}
