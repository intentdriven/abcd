package loop

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// The drain state's stored text (iss-2610121505153157): every free-text field
// of a lane, a hand-back and a judgement is read back from drain.json and
// echoed on `drain --json`; a file carrying a control, bidi or zero-width rune
// in any of them is refused when it is read, never echoing the value.

// writtenDrain is a drain state with every text field filled the way the
// drain fills it: a promoted lane hand-back, a decision-record flag from a
// judgement, a home flag, an unachievable lane, an applied judgement and one
// that decided nothing, and a judgement awaited.
func writtenDrain() DrainState {
	at := time.Date(2026, 10, 12, 9, 0, 0, 0, time.UTC)
	return DrainState{SchemaVersion: DrainSchemaVersion, StartedAt: at, UpdatedAt: at,
		Rule: ".abcd/development/decisions/adrs/2610011200000000-drain-rule.md", Max: 0, WindowStartedAt: &at,
		Lanes: []DrainLane{
			{Issue: "iss-2610010000000001", RunID: "run-2610010000000001", OpenedAt: at, Outcome: DrainLaneHandedBack},
			{Issue: "iss-2610010000000002", RunID: "run-2610010000000002", OpenedAt: at, Outcome: DrainLanePullRequest, PR: 7},
		},
		HandBacks: []DrainRoute{
			{Issue: "iss-2610010000000001", From: "lane", Kind: HandBackUserVisible, Route: RoutePromoted,
				Draft: "itd-2610010000000003", Reason: "the person decides how the page reads",
				Wrote: "intent draft itd-2610010000000003, and iss-2610010000000001's related_intents names it", At: &at},
			{Issue: "iss-2610010000000004", From: DrainFromJudgement, Kind: HandBackTrustRule, Route: RouteDecisionRecord,
				Question: "should the guard refuse it?", Reason: "should the guard refuse it?", Wrote: "nothing: a flag in this summary", At: &at},
			{Issue: "iss-2610010000000008", From: "lane", Kind: HandBackDesignFinding, Route: RouteHome,
				Home: "the parser's ADR", Reason: "the fix changes the record's shape", Wrote: "nothing: a flag in this summary", At: &at},
			{Issue: "iss-2610010000000005", From: "lane", Kind: "unachievable", Route: RouteHome,
				Home:   "the issue stays open for a person, with the lane's last findings",
				Reason: "iss-2610010000000005 is handed back as UNACHIEVABLE: round 2 did not pass after 1 fix round, the run's cap (ruthless-reviewer FIX FIRST); the last findings: lane-1/ruthless-2.md",
				Wrote:  "nothing: a flag in this summary", At: &at},
		},
		Judging: &DrainJudging{Issue: "iss-2610010000000006", RemedySHA256: strings.Repeat("ab", 32),
			Request: DrainJudgementRequestRel, Answer: DrainJudgementAnswerRel, RequestedAt: at},
		Judgements: []DrainJudgement{
			{Issue: "iss-2610010000000004", RemedySHA256: strings.Repeat("cd", 32), Answer: JudgementYes, Kind: HandBackTrustRule,
				Reason: "a rule about trust", Applied: true, At: at},
			{Issue: "iss-2610010000000007", RemedySHA256: strings.Repeat("ef", 32), Answer: JudgementNo, Reason: "an internal fix",
				Note: "iss-2610010000000007 is no longer eligible, so the judgement decides nothing: unreadable (unreadable): the reader refuses the record at its frontmatter stage: malformed frontmatter: invalid severity \"x\"",
				At:   at},
		},
	}
}

// TestADrainStateRefusesTextItNeverWrites: a drain state written the way the
// drain writes one round-trips; one whose lane, hand-back or judgement text,
// field by field, carries U+202E or U+009B is refused at the drain stage
// without echoing it.
func TestADrainStateRefusesTextItNeverWrites(t *testing.T) {
	repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
	root := repo.Root()
	if err := fsutil.EnsureRealDirAll(root, RunRelDir, dirPerm); err != nil {
		t.Fatal(err)
	}
	want := writtenDrain()
	if err := writeDrain(root, want); err != nil {
		t.Fatal(err)
	}
	got, live, err := readDrain(root)
	if err != nil || !live || !reflect.DeepEqual(got, want) {
		t.Fatalf("a drain state the drain writes round-trips: %v %v\n got %+v\nwant %+v", live, err, got, want)
	}
	path := filepath.Join(root, filepath.FromSlash(DrainStateRel))
	good, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	// Every string field of every hand-back, judgement and the judgement
	// awaited, whatever it is named, and the lane's outcome, the drain's rule
	// and why it stopped: a field added later is forged too.
	type target struct {
		where string
		at    func(doc map[string]any) map[string]any
		keys  []string
	}
	elem := func(list string, i int) func(map[string]any) map[string]any {
		return func(doc map[string]any) map[string]any { return doc[list].([]any)[i].(map[string]any) }
	}
	stringKeys := func(at func(map[string]any) map[string]any, skip ...string) []string {
		var doc map[string]any
		if err := json.Unmarshal(good, &doc); err != nil {
			t.Fatal(err)
		}
		var keys []string
		for k, v := range at(doc) {
			if _, ok := v.(string); ok && !slices.Contains(skip, k) {
				keys = append(keys, k)
			}
		}
		slices.Sort(keys)
		return keys
	}
	var targets []target
	for i := range want.HandBacks {
		at := elem("hand_backs", i)
		keys := stringKeys(at, "at")
		if !slices.Contains(keys, "rule") {
			// A field route's rule: the drain stores none, so it is forged in.
			keys = append(keys, "rule")
		}
		targets = append(targets, target{"hand_backs", at, keys})
	}
	for i := range want.Judgements {
		at := elem("judgements", i)
		targets = append(targets, target{"judgements", at, stringKeys(at, "at")})
	}
	judging := func(doc map[string]any) map[string]any { return doc["judging"].(map[string]any) }
	targets = append(targets,
		target{"judging", judging, stringKeys(judging, "requested_at")},
		target{"lanes", elem("lanes", 0), []string{"outcome"}},
		target{"the drain", func(doc map[string]any) map[string]any { return doc }, []string{"rule", "stopped"}})
	forged := 0
	for _, tg := range targets {
		for _, key := range tg.keys {
			for _, bad := range []string{"‮", "\u009b"} {
				var doc map[string]any
				if err := json.Unmarshal(good, &doc); err != nil {
					t.Fatal(err)
				}
				obj := tg.at(doc)
				s, _ := obj[key].(string)
				obj[key] = s + bad
				data, err := json.Marshal(doc)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, data, 0o600); err != nil {
					t.Fatal(err)
				}
				_, _, err = readDrain(root)
				r := mustRefusal(t, err)
				noRawRune(t, tg.where+"."+key, r)
				if r.Stage != StageDrain || !strings.Contains(r.Reason, "text the drain never writes") ||
					r.Remedy != "the drain is the file's only writer; restore it or remove it" {
					t.Fatalf("%s.%s with %q: refused at the drain: %+v", tg.where, key, bad, r)
				}
				forged++
			}
		}
	}
	for _, key := range []string{"reason", "home", "question", "rule", "draft", "wrote", "kind", "route", "from", "issue", "note", "answer", "request", "remedy_sha256"} {
		found := false
		for _, tg := range targets {
			found = found || slices.Contains(tg.keys, key)
		}
		if !found {
			t.Fatalf("the forgeries reach %s", key)
		}
	}
	t.Logf("%d forged fields refused", forged)
}
