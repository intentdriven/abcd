package ahoy

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/intentdriven/abcd/internal/core/interview"
)

// TestInstallSavesAnAnswerBesideADeferredValue is iss-2610170930394303: a
// config value deferred with "decide later" leaves only itself unset, so an
// answer given to another value in the same run reaches .abcd/config.json,
// in either order the two are asked, and the run still reports the deferred
// one as remaining.
func TestInstallSavesAnAnswerBesideADeferredValue(t *testing.T) {
	adopt := true
	for name, c := range map[string]struct {
		answers                  map[string]string
		wantSaved, wantUnset     [2]string // section, key
		wantValue, deferredGapID string
	}{
		"visibility deferred, docs target answered": {
			answers:   map[string]string{"visibility": "", "docs_target": "agents_md"},
			wantSaved: [2]string{"docs", "target"}, wantValue: "agents_md",
			wantUnset: [2]string{"repo", "visibility"}, deferredGapID: "config.visibility_missing",
		},
		"visibility answered, docs target deferred": {
			answers:   map[string]string{"visibility": "public", "docs_target": ""},
			wantSaved: [2]string{"repo", "visibility"}, wantValue: "public",
			wantUnset: [2]string{"docs", "target"}, deferredGapID: "config.docs_target_missing",
		},
	} {
		t.Run(name, func(t *testing.T) {
			setupHermetic(t)
			trufflehogResolves(t, false)
			repo := t.TempDir()
			idMustGit(t, repo, "init")
			p := &valueKeyPrompter{answers: c.answers}
			res, err := Install(repo, InstallOptions{Adopt: &adopt, Yes: true}, p)
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(p.keys, []string{"visibility", "docs_target"}) {
				t.Errorf("the install asked %v; a deferred value must not stop the questions after it", p.keys)
			}
			cfg := readConfigT(t, repo)
			if got, _ := stringVal(subMap(cfg, c.wantSaved[0]), c.wantSaved[1]); got != c.wantValue {
				t.Errorf("%s.%s = %q after the install (status %s, notes %v), want the answered %q",
					c.wantSaved[0], c.wantSaved[1], got, res.Status, res.Notes, c.wantValue)
			}
			if _, ok := subMap(cfg, c.wantUnset[0])[c.wantUnset[1]]; ok {
				t.Errorf("%s.%s was set, though it was deferred", c.wantUnset[0], c.wantUnset[1])
			}
			if res.Status != "partial" || !slices.Contains(res.Remaining, c.deferredGapID) {
				t.Errorf("status %s, remaining %v; want partial with %s remaining", res.Status, res.Remaining, c.deferredGapID)
			}
		})
	}
}

// TestOnlyWaitingSavesAnAnswerBesideADeferredValue is iss-2610170930394303
// after an update: the visibility was deferred with "decide later" at the
// last setup, so the only-waiting install does not put it, and the docs
// target, which waits, is answered; the answer reaches .abcd/config.json and
// the visibility stays unset.
func TestOnlyWaitingSavesAnAnswerBesideADeferredValue(t *testing.T) {
	repo := onlyWaitingRepo(t)
	cfg := readConfigT(t, repo)
	delete(cfg["repo"].(map[string]any), "visibility")
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, ".abcd", "config.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	recordSetup(t, interview.Place{Repo: repo}, map[string]string{"visibility": SetupLaterValue})
	can := waitingKeys(t, repo)
	if slices.Contains(can, "visibility") || !slices.Contains(can, "docs_target") {
		t.Fatalf("the fixture no longer defers the visibility and leaves the docs target waiting: %v", can)
	}

	p := &valueKeyPrompter{answers: map[string]string{"docs_target": "agents_md"}}
	res, err := Install(repo, InstallOptions{OnlyWaiting: true}, p)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(p.keys, []string{"docs_target"}) {
		t.Errorf("the only-waiting install asked %v, want [docs_target]", p.keys)
	}
	after := readConfigT(t, repo)
	if got, _ := stringVal(subMap(after, "docs"), "target"); got != "agents_md" {
		t.Errorf("docs.target = %q after the only-waiting install (status %s, notes %v), want the answered agents_md", got, res.Status, res.Notes)
	}
	if _, ok := subMap(after, "repo")["visibility"]; ok {
		t.Error("repo.visibility was set, though it was deferred with decide later")
	}
}
