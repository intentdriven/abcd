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

// TestBuildNextMaxOneOnALiveSeriesIsRefused (iss-2610120925509505): --max 1
// is a count other than a live --max 3 series' own, so it is refused at the
// pick, exit 2, starting no run; a call with no count still continues the
// series.
func TestBuildNextMaxOneOnALiveSeriesIsRefused(t *testing.T) {
	repo := buildRepo(t)
	secondBuildIntent(t, repo)
	mustImplement(t, "build", "next", "--max", "3", "--json")

	ref := refusalDocs(t, 2, "build", "next", "--max", "1", "--json")
	reason, _ := ref["reason"].(string)
	if ref["stage"] != "pick" || !strings.Contains(reason, "--max 3") || !strings.Contains(reason, "--max 1") {
		t.Fatalf("--max 1 on a live --max 3 series is refused at the pick: %v", ref)
	}
	if code, _, errOut := implementCLI(t, "build", "next", "--max", "1"); code != 2 || !strings.Contains(errOut, "--max 3") {
		t.Fatalf("the text refusal exits 2 naming the series' count: %d %q", code, errOut)
	}
	var waiting map[string]any
	out := mustImplement(t, "build", "next", "--json")
	if err := json.Unmarshal([]byte(out), &waiting); err != nil {
		t.Fatalf("%v: %s", err, out)
	}
	if waiting["pick"] != nil || waiting["series"] == nil {
		t.Fatalf("with no count the call continues the series: %s", out)
	}
	if runs, _ := loop.Runs(repo.Root()); len(runs) != 1 {
		t.Fatalf("the refused count started no run: %d runs", len(runs))
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

// TestBuildNextRefusesAnExplicitMaxZero (iss-2610121210280182): --max 0 as
// typed is no run count, so it is refused at the pick with the negative
// count's words, exit 2, starting no run: with no series, rather than read as
// a single pick, and on a live series, rather than read as no count and
// continuing it. A call with no --max still gives a single pick, and still
// continues a live series.
func TestBuildNextRefusesAnExplicitMaxZero(t *testing.T) {
	refusedZero := func(t *testing.T) {
		t.Helper()
		ref := refusalDocs(t, 2, "build", "next", "--max", "0", "--json")
		reason, _ := ref["reason"].(string)
		remedy, _ := ref["remedy"].(string)
		if ref["stage"] != "pick" || reason != "--max 0 is not a run count" || !strings.Contains(remedy, "1 or more") {
			t.Fatalf("--max 0 is refused at the pick with the negative count's words: %v", ref)
		}
		if code, _, errOut := implementCLI(t, "build", "next", "--max", "0"); code != 2 || !strings.Contains(errOut, "--max 0 is not a run count") {
			t.Fatalf("the text refusal exits 2 naming --max 0: %d %q", code, errOut)
		}
	}

	t.Run("no series", func(t *testing.T) {
		repo := buildRepo(t)
		refusedZero(t)
		runDirAbsent(t, repo.Root())
		var single struct {
			Pick   *struct{} `json:"pick"`
			Series *struct{} `json:"series"`
		}
		if err := json.Unmarshal([]byte(mustImplement(t, "build", "next", "--json")), &single); err != nil {
			t.Fatal(err)
		}
		if single.Pick == nil || single.Series != nil {
			t.Fatalf("with no --max the call is a single pick: %+v", single)
		}
	})

	t.Run("live series", func(t *testing.T) {
		repo := buildRepo(t)
		secondBuildIntent(t, repo)
		mustImplement(t, "build", "next", "--max", "2", "--json")
		refusedZero(t)
		var waiting map[string]any
		if err := json.Unmarshal([]byte(mustImplement(t, "build", "next", "--json")), &waiting); err != nil {
			t.Fatal(err)
		}
		if waiting["pick"] != nil || waiting["series"] == nil {
			t.Fatalf("with no --max the call continues the series: %v", waiting)
		}
		if runs, _ := loop.Runs(repo.Root()); len(runs) != 1 {
			t.Fatalf("the refused count started no run: %d runs", len(runs))
		}
	})
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
