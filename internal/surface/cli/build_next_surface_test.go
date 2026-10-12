package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/implement/loop"
	"github.com/intentdriven/abcd/internal/core/intent"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gittest"
)

// TestBuildNextPicksAndCarriesTheReasonInJSON is itd-2609211116005482's
// criterion 10 over a pick: the payload carries the candidate set with each
// score, the pick with its rule and runner-up, the reason's text, and the run
// the pick started; the text form names the pick and the reason.
func TestBuildNextPicksAndCarriesTheReasonInJSON(t *testing.T) {
	repo := buildRepo(t)
	repo.Write(".abcd/development/intents/planned/itd-11-beta.md",
		strings.NewReplacer("itd-10", "itd-11", "alpha", "beta", "spc-1", "spc-2", "- Given x, when y, then z.", "- The page states the flags.").Replace(buildIntent()))
	repo.Write(".abcd/development/specs/open/spc-2-beta.md", "---\nid: spc-2\nslug: beta\nintent: itd-11\n---\n# beta\n\n## Summary\n\nWritten.\n")
	repo.Commit("a second intent")

	out := mustImplement(t, "build", "next", "--json")
	var res struct {
		Candidates []struct {
			ID    string `json:"id"`
			Score struct {
				Total       int  `json:"total"`
				NoFootprint bool `json:"no_footprint"`
			} `json:"score"`
		} `json:"candidates"`
		Excluded []any `json:"excluded"`
		Pick     struct {
			Chosen struct {
				ID string `json:"id"`
			} `json:"chosen"`
			RunnerUp *struct {
				ID string `json:"id"`
			} `json:"runner_up"`
			TieBrokenByAge bool   `json:"tie_broken_by_age"`
			Rule           string `json:"rule"`
			Falsifier      string `json:"falsifier"`
		} `json:"pick"`
		Entry string `json:"entry"`
		Start struct {
			RunID string `json:"run_id"`
			Lane  struct {
				ID    string `json:"id"`
				Stage string `json:"stage"`
			} `json:"lane"`
		} `json:"start"`
	}
	if err := json.Unmarshal([]byte(out), &res); err != nil {
		t.Fatalf("%v: %s", err, out)
	}
	if len(res.Candidates) != 2 || res.Candidates[0].ID != "itd-10" || res.Candidates[0].Score.Total != 100 ||
		res.Candidates[1].Score.Total != 0 || !res.Candidates[0].Score.NoFootprint {
		t.Fatalf("the candidates carry their scores in the pick order: %s", out)
	}
	if res.Excluded == nil || res.Pick.Chosen.ID != "itd-10" || res.Pick.RunnerUp == nil || res.Pick.RunnerUp.ID != "itd-11" ||
		res.Pick.Rule == "" || res.Pick.Falsifier == "" {
		t.Fatalf("the pick carries the chosen, the runner-up, the rule and the falsifier: %s", out)
	}
	if !strings.HasPrefix(res.Entry, "picked by run "+res.Start.RunID+" on ") || res.Start.Lane.ID != "lane-1" || res.Start.Lane.Stage != "worktree" {
		t.Fatalf("the entry is the run's, and the run is the build's: %s", out)
	}
}

// TestBuildNextRefusalCarriesEveryExclusion is criterion 1's refusal on the
// surface and criterion 10's refusal half: with no candidate, --json renders
// the refusal with every excluded intent and its check, exits 2 and writes
// nothing; the text form names them.
func TestBuildNextRefusalCarriesEveryExclusion(t *testing.T) {
	repo := buildRepo(t)
	repo.Write(buildPlanned, strings.Replace(buildIntent(), "kind: standalone\n", "kind: standalone\nheld: \"not yet\"\n", 1))
	repo.Commit("hold it")

	ref := refusalDocs(t, 2, "build", "next", "--json")
	ex, _ := ref["excluded"].([]any)
	if ref["stage"] != "pick" || len(ex) != 1 {
		t.Fatalf("the refusal is the pick's, with every exclusion: %v", ref)
	}
	if e, _ := ex[0].(map[string]any); e["id"] != "itd-10" || e["check"] != "hold" || e["reason"] == "" {
		t.Fatalf("each exclusion names the intent, the check and the reason: %v", ex[0])
	}
	code, _, errOut := implementCLI(t, "build", "next")
	if code != 2 || !strings.Contains(errOut, "itd-10 (hold") {
		t.Fatalf("the text refusal names each exclusion: %d %q", code, errOut)
	}
	code, _, errOut = implementCLI(t, "build", "next", "--until-empty")
	if code != 2 || !strings.Contains(errOut, "itd-10 (hold") {
		t.Fatalf("a series' first pick with no candidate is refused the same way: %d %q", code, errOut)
	}
	runDirAbsent(t, repo.Root())
}

// secondBuildIntent adds itd-11, a candidate scored below itd-10.
func secondBuildIntent(t *testing.T, repo *gittest.Repo) {
	t.Helper()
	repo.Write(".abcd/development/intents/planned/itd-11-beta.md",
		strings.NewReplacer("itd-10", "itd-11", "alpha", "beta", "spc-1", "spc-2", "- Given x, when y, then z.", "- The page states the flags.").Replace(buildIntent()))
	repo.Write(".abcd/development/specs/open/spc-2-beta.md", "---\nid: spc-2\nslug: beta\nintent: itd-11\n---\n# beta\n\n## Summary\n\nWritten.\n")
	repo.Commit("a second intent")
}

// TestBuildNextMaxCarriesTheSeriesInJSON is criteria 5 and 10 at the front
// door: --max begins a series whose state, count and picks are in the
// payload; a call while the pick's lane is in progress picks nothing, carries
// a null pick and start, and names the run to drive; the text form says the
// same.
func TestBuildNextMaxCarriesTheSeriesInJSON(t *testing.T) {
	repo := buildRepo(t)
	secondBuildIntent(t, repo)

	type series struct {
		State   string `json:"state"`
		Started bool   `json:"started"`
		Max     int    `json:"max"`
		Picks   []struct {
			Intent  string `json:"intent"`
			RunID   string `json:"run_id"`
			Outcome string `json:"outcome"`
		} `json:"picks"`
		Complete bool `json:"complete"`
	}
	var first struct {
		Pick *struct {
			Chosen struct {
				ID string `json:"id"`
			} `json:"chosen"`
		} `json:"pick"`
		Start *struct {
			RunID string `json:"run_id"`
		} `json:"start"`
		Series *series `json:"series"`
		Next   string  `json:"next"`
	}
	out := mustImplement(t, "build", "next", "--max", "2", "--json")
	if err := json.Unmarshal([]byte(out), &first); err != nil {
		t.Fatalf("%v: %s", err, out)
	}
	if first.Pick == nil || first.Pick.Chosen.ID != "itd-10" || first.Start == nil || first.Series == nil ||
		first.Series.State != loop.NextSeriesRel || !first.Series.Started || first.Series.Max != 2 || len(first.Series.Picks) != 1 ||
		first.Series.Picks[0].RunID != first.Start.RunID || first.Series.Picks[0].Outcome != "in-progress" ||
		!strings.Contains(first.Next, "the series picks again") {
		t.Fatalf("--max begins a series carried in the payload: %s", out)
	}

	out = mustImplement(t, "build", "next", "--json")
	var waiting map[string]any
	if err := json.Unmarshal([]byte(out), &waiting); err != nil {
		t.Fatalf("%v: %s", err, out)
	}
	if waiting["pick"] != nil || waiting["start"] != nil || waiting["series"] == nil ||
		!strings.Contains(waiting["next"].(string), "implement step --run "+first.Start.RunID) {
		t.Fatalf("a call while the pick is in progress picks nothing and names the run: %s", out)
	}
	text := mustImplement(t, "build", "next")
	for _, want := range []string{"build next: no pick", "series:", "--max 2", "itd-10 " + first.Start.RunID + " in-progress", "next: drive itd-10"} {
		if !strings.Contains(text, want) {
			t.Fatalf("the text form names %q:\n%s", want, text)
		}
	}

	ref := refusalDocs(t, 2, "build", "next", "--until-empty", "--json")
	if ref["stage"] != "pick" || !strings.Contains(ref["reason"].(string), "--max 2") {
		t.Fatalf("a live series keeps its count, refused in the payload: %v", ref)
	}
}

// TestBuildNextRefusesControlRunesInTheSeriesFlags (iss-2610120925598775): a
// series state whose stored pace flags carry U+202E or U+009B is refused at
// the pick, exit 2, and no raw rune reaches the --json stream; the flags a
// series was begun with still round-trip.
func TestBuildNextRefusesControlRunesInTheSeriesFlags(t *testing.T) {
	repo := buildRepo(t)
	secondBuildIntent(t, repo)
	mustImplement(t, "build", "next", "--max", "2", "--pace", "90/0", "--sub-agents", "2", "--fix-rounds", "1", "--json")
	if out := mustImplement(t, "build", "next", "--json"); !strings.Contains(out, `"pace": "90/0"`) {
		t.Fatalf("the series' flags round-trip: %s", out)
	}
	path := filepath.Join(repo.Root(), filepath.FromSlash(loop.NextSeriesRel))
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	forged := strings.NewReplacer(`"pace": "90/0"`, `"pace": "\u202e90/0"`, `"sub_agents": "2"`, `"sub_agents": "\u009b2"`).Replace(string(raw))
	if forged == string(raw) {
		t.Fatal("the series' state carries its flags")
	}
	if err := os.WriteFile(path, []byte(forged), 0o600); err != nil {
		t.Fatal(err)
	}
	code, out, _ := implementCLI(t, "build", "next", "--json")
	if strings.ContainsAny(out, "\u202e\u009b") {
		t.Fatalf("a raw control rune reaches the --json stream: %q", out)
	}
	ref := refusalDocs(t, 2, "build", "next", "--json")
	if code != 2 || ref["stage"] != "pick" || !strings.Contains(ref["reason"].(string), "pace flags") {
		t.Fatalf("the forged flags are refused at the pick: %d %v", code, ref)
	}
}

// forgeStoredPace writes good to path with one stored pace setting's field
// replaced by bad.
func forgeStoredPace(t *testing.T, path string, good []byte, setting, field, bad string) {
	t.Helper()
	var doc map[string]any
	if err := json.Unmarshal(good, &doc); err != nil {
		t.Fatal(err)
	}
	pace, _ := doc["pace"].(map[string]any)
	v, ok := pace[setting].(map[string]any)
	if !ok {
		t.Fatalf("the file carries pace.%s: %s", setting, good)
	}
	v[field] = bad
	data, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
}

// storedPaceForgeries plant U+202E and U+009B in a pace setting's layer and
// origin.
var storedPaceForgeries = []struct{ setting, field, bad string }{
	{"work_minutes", "layer", "‮flag"},
	{"sub_agents", "layer", "bundled\u009b"},
	{"work_minutes", "origin", "--pace 90/0‮"},
	{"sub_agents", "origin", "\u009bbundled"},
}

// TestBuildNextRefusesControlRunesInTheSeriesPace (iss-2610121206163414): a
// series state whose stored pace names a layer or an origin carrying U+202E
// or U+009B is refused at the pick, exit 2, on a plain move and on one typed
// with another pace, and no raw rune reaches the --json stream; the pace a
// series was begun with still round-trips.
func TestBuildNextRefusesControlRunesInTheSeriesPace(t *testing.T) {
	repo := buildRepo(t)
	secondBuildIntent(t, repo)
	mustImplement(t, "build", "next", "--max", "2", "--pace", "90/0", "--json")
	if out := mustImplement(t, "build", "next", "--json"); !strings.Contains(out, `"origin": "--pace 90/0"`) ||
		!strings.Contains(out, `"layer": "flag"`) || !strings.Contains(out, `"layer": "bundled"`) {
		t.Fatalf("the series' pace round-trips: %s", out)
	}
	path := filepath.Join(repo.Root(), filepath.FromSlash(loop.NextSeriesRel))
	good, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range storedPaceForgeries {
		forgeStoredPace(t, path, good, c.setting, c.field, c.bad)
		for _, args := range [][]string{{"build", "next", "--json"}, {"build", "next", "--pace", "60/0", "--json"}} {
			if _, out, _ := implementCLI(t, args...); strings.ContainsAny(out, "‮\u009b") {
				t.Fatalf("%s.%s, %v: a raw control rune reaches the --json stream: %q", c.setting, c.field, args, out)
			}
			ref := refusalDocs(t, 2, args...)
			if ref["stage"] != "pick" || !strings.Contains(ref["reason"].(string), "a pace the series never writes") {
				t.Fatalf("%s.%s, %v: the forged pace is refused at the pick: %v", c.setting, c.field, args, ref)
			}
		}
	}
}

// TestImplementRecordRefusesControlRunesInTheRunPace (iss-2610121206163414):
// a run state whose stored pace names a layer or an origin carrying U+202E or
// U+009B is refused as the run's state, exit 2, by `implement record` and by a
// resume typed with another pace, and no raw rune reaches the --json stream;
// the run's pace still round-trips into the record.
func TestImplementRecordRefusesControlRunesInTheRunPace(t *testing.T) {
	repo := buildRepo(t)
	secondBuildIntent(t, repo)
	var res struct {
		Start struct {
			RunID string `json:"run_id"`
		} `json:"start"`
	}
	if err := json.Unmarshal([]byte(mustImplement(t, "build", "next", "--pace", "90/0", "--json")), &res); err != nil {
		t.Fatal(err)
	}
	id := res.Start.RunID
	if out := mustImplement(t, "implement", "record", "--run", id, "--json"); !strings.Contains(out, `"origin": "--pace 90/0"`) ||
		!strings.Contains(out, `"layer": "bundled"`) {
		t.Fatalf("the run's pace round-trips into the record: %s", out)
	}
	path := filepath.Join(repo.Root(), filepath.FromSlash(loop.StateRelPath(id)))
	good, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range storedPaceForgeries {
		forgeStoredPace(t, path, good, c.setting, c.field, c.bad)
		for _, args := range [][]string{{"implement", "record", "--run", id, "--json"}, {"build", "itd-10", "--pace", "60/0", "--json"}} {
			if _, out, _ := implementCLI(t, args...); strings.ContainsAny(out, "‮\u009b") {
				t.Fatalf("%s.%s, %v: a raw control rune reaches the --json stream: %q", c.setting, c.field, args, out)
			}
			ref := refusalDocs(t, 2, args...)
			if ref["stage"] != "state" || !strings.Contains(ref["reason"].(string), "a pace the loop never writes") ||
				!strings.Contains(ref["remedy"].(string), "restore it or remove the run directory") {
				t.Fatalf("%s.%s, %v: the forged pace is refused as the run's state: %v", c.setting, c.field, args, ref)
			}
		}
	}
}

// TestBuildNextRefusesTwoCounts: --max and --until-empty together are refused
// at the pick, exit 2, writing nothing.
func TestBuildNextRefusesTwoCounts(t *testing.T) {
	repo := buildRepo(t)
	ref := refusalDocs(t, 2, "build", "next", "--max", "3", "--until-empty", "--json")
	if ref["stage"] != "pick" || !strings.Contains(ref["reason"].(string), "two run counts") {
		t.Fatalf("two counts are refused at the pick: %v", ref)
	}
	runDirAbsent(t, repo.Root())
}

// TestImplementRecordCarriesThePick is criterion 6 at the front door: the run
// record of a picked run carries the pick's score table and whether its
// falsifier was met, in --json and in the text.
func TestImplementRecordCarriesThePick(t *testing.T) {
	repo := buildRepo(t)
	secondBuildIntent(t, repo)
	var res struct {
		Start struct {
			RunID string `json:"run_id"`
		} `json:"start"`
	}
	if err := json.Unmarshal([]byte(mustImplement(t, "build", "next", "--json")), &res); err != nil {
		t.Fatal(err)
	}
	out := mustImplement(t, "implement", "record", "--run", res.Start.RunID, "--json")
	var rec struct {
		Pick *struct {
			Candidates []struct {
				ID    string `json:"id"`
				Score struct {
					Total int `json:"total"`
				} `json:"score"`
			} `json:"candidates"`
			Falsifier string `json:"falsifier"`
			Falsified *bool  `json:"falsified"`
			Entry     string `json:"entry"`
		} `json:"pick"`
	}
	if err := json.Unmarshal([]byte(out), &rec); err != nil {
		t.Fatalf("%v: %s", err, out)
	}
	if rec.Pick == nil || len(rec.Pick.Candidates) != 2 || rec.Pick.Candidates[0].ID != "itd-10" || rec.Pick.Falsifier == "" ||
		rec.Pick.Falsified == nil || *rec.Pick.Falsified || !strings.HasPrefix(rec.Pick.Entry, "picked by run "+res.Start.RunID) {
		t.Fatalf("the record carries the pick's score table and its falsifier: %s", out)
	}
	text := mustImplement(t, "implement", "record", "--run", res.Start.RunID)
	for _, want := range []string{"pick:    itd-10 from 2 candidate(s); falsifier not met", "itd-10  100", "itd-11  0"} {
		if !strings.Contains(text, want) {
			t.Fatalf("the text record names %q:\n%s", want, text)
		}
	}
}

// TestBuildNextRendersMaskTheStateFilesText: the series' picks and a run
// record's pick are read back from state files a hostile checkout can ship,
// so their text form masks terminal control runes as every sibling line does.
func TestBuildNextRendersMaskTheStateFilesText(t *testing.T) {
	const esc = "\x1b[2J\u202e"
	var b bytes.Buffer
	renderSeries(&b, &loop.SeriesResult{State: loop.NextSeriesRel, NextSeries: loop.NextSeries{Max: 2,
		Picks: []loop.SeriesPick{{Intent: "itd-10" + esc, RunID: "run-1" + esc, Outcome: "delivered" + esc}}}})
	renderRecordPick(&b, &loop.RecordPick{Pick: intent.Pick{Chosen: intent.PickCandidate{ID: "itd-10" + esc}}})
	out := b.String()
	if strings.ContainsAny(out, "\x1b\u202e") {
		t.Fatalf("a control rune read from a state file reaches the terminal: %q", out)
	}
	for _, want := range []string{"itd-10?[2J? run-1?[2J? delivered?[2J?", "pick:    itd-10?[2J? from 0 candidate(s)"} {
		if !strings.Contains(out, want) {
			t.Fatalf("the masked line still names %q:\n%s", want, out)
		}
	}
}

// TestBuildNextBacksOffWhileTheSeriesIsLocked: a move that cannot take the
// series' lock exits 3, contention, naming it.
func TestBuildNextBacksOffWhileTheSeriesIsLocked(t *testing.T) {
	repo := buildRepo(t)
	secondBuildIntent(t, repo)
	mustImplement(t, "build", "next", "--max", "2", "--json")
	held, release, done := make(chan struct{}), make(chan struct{}), make(chan error, 1)
	go func() {
		done <- fsutil.WithFileLock(filepath.Join(repo.Root(), ".abcd", ".work.local", "run", "next.lock"), time.Second, func() error {
			close(held)
			<-release
			return nil
		})
	}()
	<-held
	ref := refusalDocs(t, 3, "build", "next", "--json")
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if ref["stage"] != "pick" || ref["contention"] != true || !strings.Contains(ref["reason"].(string), "another `abcd build next` is moving") {
		t.Fatalf("the contention names the series' move: %v", ref)
	}
}
