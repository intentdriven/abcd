package ahoy

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/banlist"
	"github.com/intentdriven/abcd/internal/core/interview"
	"github.com/intentdriven/abcd/internal/core/launch"
	"github.com/intentdriven/abcd/internal/core/releasedigest"
)

// standInWaiting replaces the digest's range and the default-change table
// for one test; the range records the versions it was asked for.
func standInWaiting(t *testing.T, changes []DefaultChange, releases []releasedigest.Release) *[2]string {
	t.Helper()
	asked := &[2]string{}
	prevRange, prevChanges := waitingRange, waitingDefaultChanges
	waitingRange = func(from, to string) (releasedigest.Summary, error) {
		asked[0], asked[1] = from, to
		return releasedigest.Summary{From: from, To: to, Releases: releases}, nil
	}
	waitingDefaultChanges = func() ([]DefaultChange, error) { return changes, nil }
	t.Cleanup(func() { waitingRange, waitingDefaultChanges = prevRange, prevChanges })
	return asked
}

// cites is a digest release whose one entry cites ids.
func cites(tag string, ids ...string) releasedigest.Release {
	e := releasedigest.Entry{Line: "a change"}
	for _, id := range ids {
		e.Records = append(e.Records, releasedigest.Record{ID: id, Impact: releasedigest.ImpactAdditive})
	}
	return releasedigest.Release{Tag: tag, Entries: []releasedigest.Entry{e}}
}

// recordSetup writes a setup answers record giving each id its value, to the
// repository or, with machine, to the home.
func recordSetup(t *testing.T, place interview.Place, values map[string]string) {
	t.Helper()
	rec := interview.Record{Interview: SetupInterview, Target: interview.TargetRepository}
	if place.Home != "" {
		rec.Target = interview.TargetMachine
	}
	for id, v := range values {
		rec.Answers = append(rec.Answers, interview.Answer{ID: id, Value: v, AnsweredIn: interview.Terminal})
	}
	if _, err := interview.Write(place, rec, time.Now()); err != nil {
		t.Fatal(err)
	}
}

// writeConfigJSON writes the repository's .abcd/config.json and its local tier.
func writeConfigJSON(t *testing.T, repo string, cfg map[string]any) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(repo, ".abcd", ".work.local"), 0o700); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, ".abcd", "config.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func keysOf(items []WaitingItem) []string {
	var out []string
	for _, it := range items {
		out = append(out, it.Key)
	}
	return out
}

func detWith(ids ...string) DetectionResult {
	var det DetectionResult
	for _, id := range ids {
		det.Gaps = append(det.Gaps, Gap{ID: id, Title: "title of " + id})
	}
	return det
}

// TestWaitingListsQuestionsNeverAnswered: the questions the detection's gaps
// arm wait unless this repository's records answer them, with a value or with
// "decide later"; a machine-wide question is answered by the home's records,
// never the repository's; a gap that arms no question, the oracle question an
// install records without asking, and a category approval wait on nothing.
func TestWaitingListsQuestionsNeverAnswered(t *testing.T) {
	home, _ := setupHermetic(t)
	standInWaiting(t, nil, nil)
	repo := t.TempDir()
	writeConfigJSON(t, repo, map[string]any{"meta": map[string]any{"setup_version": "v0.15.0"}})
	det := detWith("config.visibility_missing", "config.docs_target_missing", "config.oracle_backend_missing",
		StatusLineOfferGapID, OracleRoutingRepoGapID, "deps.gitleaks_missing", "rules.missing")

	got, err := waitingIn(repo, det, "v0.15.0", "v0.16.0")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"visibility", "docs_target", StatusLineOfferGapID, OracleRoutingRepoGapID}
	if oracleBackendAsked() {
		t.Fatal("the fixture assumes one reviewer ships, so the oracle question is not asked")
	}
	if !slices.Equal(keysOf(got.Questions), want) {
		t.Fatalf("waiting questions %v, want %v", keysOf(got.Questions), want)
	}
	if got.Questions[1].Title != "title of config.docs_target_missing" || got.Questions[1].Kind != WaitingQuestion {
		t.Fatalf("a question is titled by the gap that arms it: %+v", got.Questions[1])
	}

	recordSetup(t, interview.Place{Repo: repo}, map[string]string{"visibility": "private", "docs_target": SetupLaterValue, StatusLineOfferGapID: "no"})
	recordSetup(t, interview.Place{Home: home}, map[string]string{OracleRoutingRepoGapID: "yes"})
	got, err = waitingIn(repo, det, "v0.15.0", "v0.16.0")
	if err != nil {
		t.Fatal(err)
	}
	// The status line is answered on the machine, so the repository's record
	// of it answers nothing; the repository's routing is answered here, so
	// the home's record of it answers nothing either.
	if want := []string{StatusLineOfferGapID, OracleRoutingRepoGapID}; !slices.Equal(keysOf(got.Questions), want) {
		t.Fatalf("after the records: %v, want %v", keysOf(got.Questions), want)
	}
	recordSetup(t, interview.Place{Home: home}, map[string]string{StatusLineOfferGapID: SetupLaterValue})
	recordSetup(t, interview.Place{Repo: repo}, map[string]string{OracleRoutingRepoGapID: "no"})
	if got, err = waitingIn(repo, det, "v0.15.0", "v0.16.0"); err != nil || len(got.Questions) != 0 {
		t.Fatalf("every question answered: %v, %v", keysOf(got.Questions), err)
	}
}

// TestWaitingListsChangedDefaultsTheRepositoryTakes: a default changed by a
// record a release in the range cites waits where the repository's config
// leaves the key unset and no waiting question covers it; a key the config
// sets, a record no release in the range cites, and a key a waiting question
// already covers wait on nothing. The range is asked for from and to as given.
func TestWaitingListsChangedDefaultsTheRepositoryTakes(t *testing.T) {
	setupHermetic(t)
	asked := standInWaiting(t,
		[]DefaultChange{
			{Key: "oracle_backend", Record: "itd-9"},
			{Key: "docs_target", Record: "itd-9"},
			{Key: "scan_deep", Record: "itd-7"},
			{Key: "visibility", Record: "iss-3"},
		},
		[]releasedigest.Release{cites("v0.16.0", "itd-9"), cites("v0.15.1", "iss-3")})
	repo := t.TempDir()
	writeConfigJSON(t, repo, map[string]any{"repo": map[string]any{"visibility": "private"}})
	det := detWith("config.docs_target_missing", "config.oracle_backend_missing")

	got, err := waitingIn(repo, det, "v0.15.0", "v0.16.0")
	if err != nil {
		t.Fatal(err)
	}
	if *asked != [2]string{"v0.15.0", "v0.16.0"} {
		t.Fatalf("the range was asked for %v", *asked)
	}
	if !slices.Equal(keysOf(got.Questions), []string{"docs_target"}) {
		t.Fatalf("questions %v", keysOf(got.Questions))
	}
	if len(got.Defaults) != 1 {
		t.Fatalf("defaults %+v, want oracle_backend alone", got.Defaults)
	}
	d := got.Defaults[0]
	if d.Kind != WaitingDefault || d.Key != "oracle_backend" || d.Record != "itd-9" || d.Release != "v0.16.0" ||
		d.Title != "the default of oracle.backend changed in v0.16.0 (itd-9)" {
		t.Fatalf("default %+v", d)
	}

	// Deferred with "decide later", docs_target no longer waits as a
	// question, so its changed default waits in its place.
	recordSetup(t, interview.Place{Repo: repo}, map[string]string{"docs_target": SetupLaterValue})
	if got, err = waitingIn(repo, det, "v0.15.0", "v0.16.0"); err != nil {
		t.Fatal(err)
	}
	if len(got.Questions) != 0 || !slices.Equal(keysOf(got.Defaults), []string{"oracle_backend", "docs_target"}) {
		t.Fatalf("after deferring docs_target: questions %v, defaults %v", keysOf(got.Questions), keysOf(got.Defaults))
	}

	// Set in the repository's layer, neither key takes its default.
	writeConfigJSON(t, repo, map[string]any{
		"repo": map[string]any{"visibility": "private"}, "docs": map[string]any{"target": "skip"},
		"oracle": map[string]any{"backend": "host-delegated"},
	})
	if got, err = waitingIn(repo, det, "v0.15.0", "v0.16.0"); err != nil || len(got.Defaults) != 0 {
		t.Fatalf("every key set: defaults %v, %v", keysOf(got.Defaults), err)
	}
}

// TestWaitingWithNothingToWait: a repository whose gaps arm no question and
// whose update changed no default waits on nothing, as two empty lists.
func TestWaitingWithNothingToWait(t *testing.T) {
	setupHermetic(t)
	standInWaiting(t, []DefaultChange{{Key: "docs_target", Record: "itd-9"}}, []releasedigest.Release{cites("v0.16.0", "itd-1")})
	repo := t.TempDir()
	writeConfigJSON(t, repo, map[string]any{})
	got, err := waitingIn(repo, detWith("rules.missing"), "v0.15.0", "v0.16.0")
	if err != nil || got.Questions == nil || got.Defaults == nil || len(got.Questions)+len(got.Defaults) != 0 {
		t.Fatalf("got %+v, %v", got, err)
	}
}

// TestWaitingRefusesARecordItCannotRead: a setup record that does not parse
// is an error, never read as "nothing answered".
func TestWaitingRefusesARecordItCannotRead(t *testing.T) {
	setupHermetic(t)
	standInWaiting(t, nil, nil)
	repo := t.TempDir()
	writeConfigJSON(t, repo, map[string]any{})
	dir := filepath.Join(repo, filepath.FromSlash(interview.RecordsRel))
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "setup-20261001T090000Z.json"), []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := waitingIn(repo, detWith("config.docs_target_missing"), "v0.15.0", "v0.16.0"); err == nil {
		t.Fatal("an unreadable record was read as nothing answered")
	}
}

// keyRecordingPrompter answers every value question with its default and
// declines every approval, recording the id of each question it is put: the
// prompt's key, or the approval's setup id. It is at a terminal, so the
// offers put only to a person are put too.
type keyRecordingPrompter struct{ keys []string }

func (p *keyRecordingPrompter) Confirm(text string) bool {
	p.keys = append(p.keys, setupConfirmID(text))
	return false
}

func (p *keyRecordingPrompter) Prompt(key string, _ []string, def string) string {
	p.keys = append(p.keys, key)
	return def
}

func (p *keyRecordingPrompter) AtTerminal() bool { return true }

// TestEveryQuestionTheInstallAsksCanWait drives an install over a set-up
// repository from which three answers were taken (docs.target, the artefact
// kind and the docs-check config) and holds every question it is put to be
// one waiting returned for that repository beforehand. A question added to
// the install that waiting cannot return shows here. The approvals are given
// up front, as the yes to the offer gives them, so none is put.
func TestEveryQuestionTheInstallAsksCanWait(t *testing.T) {
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

	det, err := Detect(repo)
	if err != nil {
		t.Fatal(err)
	}
	w, err := waitingIn(repo, det, "v0.15.0", "v0.16.0")
	if err != nil {
		t.Fatal(err)
	}
	can := keysOf(w.Questions)

	approve := map[GapCategory]bool{Readme: true}
	for _, c := range categoryPromptOrder {
		approve[c] = true
	}
	p := &keyRecordingPrompter{}
	if _, err := Install(repo, InstallOptions{Adopt: &yes, ApprovedCategories: approve}, p); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"docs_target", artefactKindKey, emDashPromptKey} {
		if !slices.Contains(p.keys, k) {
			t.Fatalf("the install was not put %s (put %v): the fixture no longer exercises it", k, p.keys)
		}
	}
	for _, k := range p.keys {
		if strings.HasPrefix(k, readmePartKeyPrefix) {
			k = ReadmePromptKey // a part follows the README question it belongs to
		}
		if !slices.Contains(can, k) {
			t.Errorf("the install put %q, which waiting cannot return (it returned %v)", k, can)
		}
	}
}
