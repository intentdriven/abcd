package ahoy

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core"
	"github.com/intentdriven/abcd/internal/core/banlist"
	"github.com/intentdriven/abcd/internal/core/interview"
	"github.com/intentdriven/abcd/internal/core/launch"
)

// onlyWaitingRepo is a repository set up with v0.15.0 from which the docs
// target, the artefact kind and the docs-check config were taken, so an
// install would put the docs target, the artefact kind and the house-style
// question; the running binary is v0.16.0.
func onlyWaitingRepo(t *testing.T) string {
	t.Helper()
	setupHermetic(t)
	standInWaiting(t, nil, nil)
	repo := t.TempDir()
	idMustGit(t, repo, "init")
	yes := true
	if _, err := Install(repo, InstallOptions{Adopt: &yes, Yes: true,
		ValueOverrides: map[string]string{"visibility": "private", "docs_target": "agents_md"}}, RefusingPrompter{}); err != nil {
		t.Fatal(err)
	}
	cfg := readConfigT(t, repo)
	delete(cfg, "docs")
	cfg["meta"].(map[string]any)["setup_version"] = "v0.15.0"
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, ".abcd", "config.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{launch.ArtefactRelPath, banlist.PublicConfigRelPath} {
		if err := os.Remove(filepath.Join(repo, filepath.FromSlash(rel))); err != nil {
			t.Fatal(err)
		}
	}
	prev := core.Version
	core.Version = "v0.16.0"
	t.Cleanup(func() { core.Version = prev })
	return repo
}

// waitingKeys is every key Waiting returns for repo, v0.15.0 to v0.16.0.
func waitingKeys(t *testing.T, repo string) []string {
	t.Helper()
	det, err := Detect(repo)
	if err != nil {
		t.Fatal(err)
	}
	w, err := Waiting(repo, det, "v0.15.0", "v0.16.0")
	if err != nil {
		t.Fatal(err)
	}
	return append(keysOf(w.Questions), keysOf(w.Defaults)...)
}

// TestOnlyWaitingAsksOnlyWhatWaits is the yes to the after-update offer: the
// install puts only the questions Waiting returns, never one this repository
// deferred with "decide later" (the docs target here), puts no approval,
// since the yes approved every kind of change, and records the running
// release as the setup version.
func TestOnlyWaitingAsksOnlyWhatWaits(t *testing.T) {
	repo := onlyWaitingRepo(t)
	recordSetup(t, interview.Place{Repo: repo}, map[string]string{"docs_target": SetupLaterValue})
	can := waitingKeys(t, repo)
	if slices.Contains(can, "docs_target") || !slices.Contains(can, artefactKindKey) || !slices.Contains(can, emDashPromptKey) {
		t.Fatalf("the fixture no longer defers the docs target and leaves the kind and house style waiting: %v", can)
	}

	p := &keyRecordingPrompter{}
	res, err := Install(repo, InstallOptions{OnlyWaiting: true}, p)
	if err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{artefactKindKey, emDashPromptKey} {
		if !slices.Contains(p.keys, k) {
			t.Errorf("the only-waiting install did not put %s, which waits (put %v)", k, p.keys)
		}
	}
	for _, k := range p.keys {
		if strings.HasPrefix(k, approvePrefix) || k == "" {
			t.Errorf("the only-waiting install put an approval (%q); the yes approved every kind of change", k)
			continue
		}
		if !slices.Contains(can, k) {
			t.Errorf("the only-waiting install put %q, which does not wait (waiting: %v)", k, can)
		}
	}
	if got := readConfigT(t, repo)["meta"].(map[string]any)["setup_version"]; got != "v0.16.0" {
		t.Errorf("setup_version = %v after the only-waiting install (status %s, notes %v), want the running v0.16.0", got, res.Status, res.Notes)
	}
	if _, ok := readConfigT(t, repo)["docs"]; ok {
		t.Error("the docs target deferred with decide later was set without being asked")
	}
}

// TestOnlyWaitingWithYesAnswersNoValueQuestion: --yes is never the person's
// yes to a value question, so an only-waiting --yes install puts no value
// question and leaves every waiting one unset (the docs target, the artefact
// kind, the house-style rule's severity), each still waiting afterwards.
func TestOnlyWaitingWithYesAnswersNoValueQuestion(t *testing.T) {
	repo := onlyWaitingRepo(t)
	before := waitingKeys(t, repo)
	for _, k := range []string{"docs_target", artefactKindKey, emDashPromptKey} {
		if !slices.Contains(before, k) {
			t.Fatalf("the fixture no longer leaves %s waiting: %v", k, before)
		}
	}

	p := &keyRecordingPrompter{}
	res, err := Install(repo, InstallOptions{OnlyWaiting: true, Yes: true}, p)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.keys) != 0 {
		t.Errorf("an only-waiting --yes install put %v; it answers no question", p.keys)
	}
	if _, ok := readConfigT(t, repo)["docs"]; ok {
		t.Error("the docs target was set by --yes")
	}
	for _, rel := range []string{launch.ArtefactRelPath, banlist.PublicConfigRelPath} {
		if _, err := os.Lstat(filepath.Join(repo, filepath.FromSlash(rel))); !os.IsNotExist(err) {
			t.Errorf("%s was written by --yes, answering its question (err %v)", rel, err)
		}
	}
	after := waitingKeys(t, repo)
	for _, k := range []string{"docs_target", artefactKindKey, emDashPromptKey} {
		if !slices.Contains(after, k) {
			t.Errorf("%s no longer waits after an only-waiting --yes install: %v", k, after)
		}
	}
	if !slices.ContainsFunc(res.Notes, func(n string) bool { return strings.Contains(n, "abcd ahoy install --only-waiting") }) {
		t.Errorf("no note names the run that asks what was left: %v", res.Notes)
	}
}

// TestWalkConfigValueQuestionsOnlyWaiting: the answers file's pre-check walks
// only the value questions an only-waiting install puts, so a question
// deferred with "decide later" needs no answer in the file, and no approval
// is asked of it.
func TestWalkConfigValueQuestionsOnlyWaiting(t *testing.T) {
	repo := onlyWaitingRepo(t)
	recordSetup(t, interview.Place{Repo: repo}, map[string]string{"docs_target": SetupLaterValue})
	var put []string
	err := WalkConfigValueQuestions(repo, InstallOptions{OnlyWaiting: true}, func(id string) bool {
		put = append(put, id)
		return false
	}, func(key string, _ []string, _ string) string {
		put = append(put, key)
		return ""
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(put) != 0 {
		t.Fatalf("the walk put %v; nothing it would ask waits", put)
	}
}

// TestWaitingCountsNoChangedDefaultInADevBuild: a running build that is not
// a release ("" here) is in no release, so no default it carries counts and
// the digest is not asked.
func TestWaitingCountsNoChangedDefaultInADevBuild(t *testing.T) {
	setupHermetic(t)
	repo := t.TempDir()
	writeConfigJSON(t, repo, map[string]any{"meta": map[string]any{"setup_version": "v0.15.0"}})
	asked := standInWaiting(t, []DefaultChange{{Key: "visibility", Record: "itd-7"}}, nil)
	w, err := Waiting(repo, detWith(), "v0.15.0", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(w.Defaults) != 0 || *asked != [2]string{} {
		t.Fatalf("defaults %v, range asked %v; a dev build counts in no release", w.Defaults, *asked)
	}
}

// TestOnlyWaitingDeclinesAnOfferThatDoesNotWait: of two offers an
// only-waiting install meets, the one that waits (the drain rule offer here)
// reaches the person and the one that does not (the commit identity) is
// declined without being put; the adoption is consent and always reaches
// them.
func TestOnlyWaitingDeclinesAnOfferThatDoesNotWait(t *testing.T) {
	identity := "Commit to this repository as Alice <alice@example.com>?"
	if setupConfirmID(identity) != "git_identity.establish" || setupConfirmID(drainRuleQuestion()) != DrainRuleOfferGapID {
		t.Fatal("the fixture's texts no longer carry the identity and drain rule offer ids")
	}
	f := &waitingFilter{keys: map[string]bool{DrainRuleOfferGapID: true}}
	inner := &keyRecordingPrompter{}
	p := f.prompter(inner)
	if p.Confirm(identity) {
		t.Error("the identity offer, which does not wait, was accepted")
	}
	p.Confirm(drainRuleQuestion())
	p.Confirm("Adopt this unmanaged repo into abcd?")
	if want := []string{DrainRuleOfferGapID, "adopt"}; !slices.Equal(inner.keys, want) {
		t.Fatalf("the person was put %v, want %v", inner.keys, want)
	}
}
