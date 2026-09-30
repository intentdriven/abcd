package statusblock

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/intent"
	"github.com/intentdriven/abcd/internal/core/spec"
)

// readyIntent is a planned intent the readiness gate reports READY: criteria,
// the mechanism and conditions answered, linked to specID.
func readyIntent(id, title, specID, extra string) string {
	return "---\nid: " + id + "\nslug: s\nspec_id: " + specID + "\nkind: standalone\n" + extra + "---\n# " + title + "\n\n" +
		"## Mechanism\n\nWe expect it to work because it is small; shown wrong if it is not.\n\n" +
		"## Scope Conditions\n\nNone stated.\n\n## Acceptance Criteria\n\n- Given x, when y, then z.\n\n" +
		"## Grounds\n\n- pursued: we expect the block to read true; shown wrong if it names a stale intent\n"
}

// writtenSpec is an open spec linked back to intentID, its body written.
func writtenSpec(id, intentID string) string {
	return "---\nid: " + id + "\nslug: s\nintent: " + intentID + "\n---\n# s\n\n## Summary\n\nA written design record.\n"
}

// scoredSpec is writtenSpec carrying a `## Footprint` that names its tests and
// one package, so the pick's score gives it full marks on both parts.
func scoredSpec(id, intentID string) string {
	return writtenSpec(id, intentID) + "\n## Footprint\n\n- packages: internal/core/intent\n- tests: the score over fixtures\n"
}

func draft(id, title string) string {
	return "---\nid: " + id + "\nslug: s\nspec_id: null\nkind: standalone\n---\n# " + title + "\n\n## Acceptance Criteria\n\n- Given x, when y, then z.\n"
}

// store lays a record with three READY planned intents (one of them held), one
// planned intent the gate refuses, and two drafts, in an order that is not
// their id order. The youngest READY intent's spec carries a footprint, so the
// pick order puts it first: the readiest, not the oldest.
func store(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	w := func(rel, body string) {
		t.Helper()
		p := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	const in = ".abcd/development/intents/"
	const sp = ".abcd/development/specs/open/"
	w(in+"planned/itd-2609010000000001-late.md", readyIntent("itd-2609010000000001", "The stamped one", "spc-2609010000000011", ""))
	w(sp+"spc-2609010000000011-late.md", scoredSpec("spc-2609010000000011", "itd-2609010000000001"))
	w(in+"planned/itd-7-seven.md", readyIntent("itd-7", "The seventh", "spc-17", ""))
	w(sp+"spc-17-seven.md", writtenSpec("spc-17", "itd-7"))
	w(in+"planned/itd-5-held.md", readyIntent("itd-5", "The held one", "spc-15", "held: \"awaiting a ruling\"\n"))
	w(sp+"spc-15-held.md", writtenSpec("spc-15", "itd-5"))
	w(in+"planned/itd-8-unlinked.md", readyIntent("itd-8", "The unlinked one", "null", ""))
	w(in+"drafts/itd-2609020000000002-idea.md", draft("itd-2609020000000002", "A later idea"))
	w(in+"drafts/itd-3-old.md", draft("itd-3", "An old idea"))
	return root
}

func ids(rows []Row) []string {
	out := []string{}
	for _, r := range rows {
		out = append(out, r.ID)
	}
	return out
}

func lanesOf(started ...Started) LaneReader {
	return func(string) ([]Started, error) { return started, nil }
}

// TestBlockPlacesEveryIntent is criterion 1: Now holds the intents the state
// file shows in a lane with their lane state, then the head marked next up;
// Next every READY planned intent in pick order that is in no lane; Later the
// planned intents the gate refuses, each naming its failing checks, then the
// drafts; every row carries its id and title.
func TestBlockPlacesEveryIntent(t *testing.T) {
	root := store(t)
	lane := Lane{Run: "run-2609290000000001", Lane: "lane-1", Step: "implement", Awaiting: "implementer"}
	b, err := Read(root, lanesOf(Started{Intent: "itd-2609010000000001", Lane: lane}))
	if err != nil {
		t.Fatal(err)
	}

	if got, want := ids(b.Now), []string{"itd-2609010000000001", "itd-7"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Now = %v, want %v (the lane, then the head)", got, want)
	}
	if b.Now[0].Lane == nil || *b.Now[0].Lane != lane || b.Now[0].NextUp {
		t.Errorf("Now[0] = %+v, want the lane state %+v and no next-up mark", b.Now[0], lane)
	}
	if b.Now[0].Title != "The stamped one" {
		t.Errorf("Now[0].Title = %q, want the intent's title", b.Now[0].Title)
	}
	if !b.Now[1].NextUp || b.Now[1].Lane != nil || b.Now[1].Title != "The seventh" {
		t.Errorf("Now[1] = %+v, want itd-7 marked next up: the first READY intent in pick order neither in a lane nor held", b.Now[1])
	}

	if got, want := ids(b.Next), []string{"itd-5", "itd-7"}; !reflect.DeepEqual(got, want) {
		t.Errorf("Next = %v, want %v: every READY planned intent in no lane, in pick order, the readiest first, the oldest among equals", got, want)
	}
	if got, want := ids(b.Later), []string{"itd-8", "itd-3", "itd-2609020000000002"}; !reflect.DeepEqual(got, want) {
		t.Errorf("Later = %v, want %v: the planned intents not READY, then the drafts", got, want)
	}
	if f := b.Later[0].Failing; len(f) == 0 || f[0] != "spec_link" {
		t.Errorf("Later[0].Failing = %v, want the failing gate check spec_link named", f)
	}
	if b.Later[1].Title != "An old idea" || b.Later[1].Failing != nil {
		t.Errorf("Later[1] = %+v, want the draft with its title and no failing check", b.Later[1])
	}
	for _, r := range append(append(append([]Row{}, b.Now...), b.Next...), b.Later...) {
		if r.ID == "" || r.Title == "" {
			t.Errorf("row %+v lacks its id or title", r)
		}
	}
	if b.Order != OrderPick || OrderPick != "pick" {
		t.Errorf("Order = %q, want %q: Next and the head are read in the pick order", b.Order, "pick")
	}
}

// TestBlockWithoutAStateFileKeepsOnlyTheHead is criterion 3 as ruling BV2 of
// 2026-09-29 amends it: with no lanes, Now holds only the head, the intent the
// state file showed in a lane returns to the list the gate places it in, and
// Next and Later are otherwise exactly what they were.
func TestBlockWithoutAStateFileKeepsOnlyTheHead(t *testing.T) {
	root := store(t)
	with, err := Read(root, lanesOf(Started{Intent: "itd-7", Lane: Lane{Run: "run-1", Lane: "lane-1", Step: "brief"}}))
	if err != nil {
		t.Fatal(err)
	}
	for name, reader := range map[string]LaneReader{"nil reader": nil, "no lanes": lanesOf()} {
		without, err := Read(root, reader)
		if err != nil {
			t.Fatal(err)
		}
		if got := ids(without.Now); !reflect.DeepEqual(got, []string{"itd-2609010000000001"}) || !without.Now[0].NextUp {
			t.Errorf("%s: Now = %v, want only the head itd-2609010000000001 marked next up", name, without.Now)
		}
		if got, want := ids(without.Next), []string{"itd-2609010000000001", "itd-5", "itd-7"}; !reflect.DeepEqual(got, want) {
			t.Errorf("%s: Next = %v, want %v: itd-7 back in its pick-order place", name, got, want)
		}
		var kept []Row
		for _, r := range without.Next {
			if r.ID != "itd-7" {
				kept = append(kept, r)
			}
		}
		if !reflect.DeepEqual(with.Next, kept) || !reflect.DeepEqual(with.Later, without.Later) {
			t.Errorf("%s: Next or Later changed beyond the lane's intent:\nwith    %+v %+v\nwithout %+v %+v", name, with.Next, with.Later, without.Next, without.Later)
		}
	}
}

// TestAnIntentInALaneIsOnlyUnderNow is ruling BV2 of 2026-09-29: an intent the
// state file shows in a lane is listed under Now alone, never also under Next
// (a READY planned intent) or Later (a planned intent the gate refuses, or a
// draft). The head is still the first READY intent in pick order that is in no
// lane, so it stays in Next beside its mark on Now.
func TestAnIntentInALaneIsOnlyUnderNow(t *testing.T) {
	root := store(t)
	lane := Lane{Run: "run-1", Lane: "lane-1", Step: "implement"}
	inLane := []string{"itd-7", "itd-8", "itd-3"}
	var started []Started
	for _, id := range inLane {
		started = append(started, Started{Intent: id, Lane: lane})
	}
	b, err := Read(root, lanesOf(started...))
	if err != nil {
		t.Fatal(err)
	}
	if got, want := ids(b.Now), []string{"itd-7", "itd-8", "itd-3", "itd-2609010000000001"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Now = %v, want %v (the three lanes, then the head)", got, want)
	}
	if got, want := ids(b.Next), []string{"itd-2609010000000001", "itd-5"}; !reflect.DeepEqual(got, want) {
		t.Errorf("Next = %v, want %v: the READY intents in no lane, the head among them", got, want)
	}
	if got, want := ids(b.Later), []string{"itd-2609020000000002"}; !reflect.DeepEqual(got, want) {
		t.Errorf("Later = %v, want %v: the refused intent and the draft in a lane are under Now only", got, want)
	}
}

// TestBlockJSONCarriesTheThreeLists is criterion 4's shape: the three lists by
// name, each row with its id and title, the lane state and the failing checks.
func TestBlockJSONCarriesTheThreeLists(t *testing.T) {
	root := store(t)
	b, err := Read(root, lanesOf(Started{Intent: "itd-2609010000000001", Lane: Lane{Run: "run-1", Lane: "lane-2", Step: "validate", Awaiting: "validator"}}))
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(b)
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	for _, want := range []string{
		`"now":[`, `"next":[`, `"later":[`, `"order":"pick"`,
		`"lane":{"run":"run-1","lane":"lane-2","step":"validate","awaiting":"validator"}`,
		`"next_up":true`, `"failing_checks":["spec_link"`, `"title":"An old idea"`,
	} {
		if !strings.Contains(s, want) {
			t.Errorf("the JSON lacks %s:\n%s", want, s)
		}
	}
}

// TestBlockOnAnEmptyStoreHasEmptyLists: a record with no intents renders three
// empty lists, never null ones, and no head.
func TestBlockOnAnEmptyStoreHasEmptyLists(t *testing.T) {
	b, err := Read(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(b)
	if want := `{"now":[],"next":[],"later":[],"order":"pick"}`; string(data) != want {
		t.Errorf("empty block = %s, want %s", data, want)
	}
}

// TestTheHeadSkipsAHeldIntent is the hold rule on the head: a READY intent held
// by `abcd intent hold` stays in Next, but the head passes over it, and so does
// one whose `held:` key is in a shape no verb writes (the loader marks it
// malformed, and a malformed hold fails closed as the build's own check does).
// With every READY intent held there is no head, and Now is empty.
func TestTheHeadSkipsAHeldIntent(t *testing.T) {
	root := t.TempDir()
	w := func(rel, body string) {
		t.Helper()
		p := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	const in = ".abcd/development/intents/planned/"
	const sp = ".abcd/development/specs/open/"
	w(in+"itd-2-malformed.md", readyIntent("itd-2", "The malformed hold", "spc-12", "held: [a, b]\n"))
	w(sp+"spc-12-malformed.md", writtenSpec("spc-12", "itd-2"))
	w(in+"itd-3-held.md", readyIntent("itd-3", "The held one", "spc-13", "held: \"awaiting a ruling\"\n"))
	w(sp+"spc-13-held.md", writtenSpec("spc-13", "itd-3"))

	b, err := Read(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := ids(b.Next), []string{"itd-2", "itd-3"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Next = %v, want %v: a held intent is still READY", got, want)
	}
	if len(b.Now) != 0 {
		t.Errorf("Now = %+v, want no head: every READY intent is held, one of them malformed", b.Now)
	}

	w(in+"itd-4-free.md", readyIntent("itd-4", "The free one", "spc-14", ""))
	w(sp+"spc-14-free.md", writtenSpec("spc-14", "itd-4"))
	b, err = Read(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := ids(b.Now); !reflect.DeepEqual(got, []string{"itd-4"}) || !b.Now[0].NextUp {
		t.Errorf("Now = %+v, want only itd-4 marked next up: the oldest READY intent no hold covers", b.Now)
	}
}

// TestTheHeadIsThePicksChoice: the head and Next are read in the pick order
// `build next` takes, scored by the same function from the same records —
// the readiest first, the oldest among equals — so the head is the intent the
// pick would choose, not the oldest id. An intent the state file shows in a
// lane is one the pick would not start, so the head passes over it.
func TestTheHeadIsThePicksChoice(t *testing.T) {
	root := t.TempDir()
	type rec struct{ id, spec, specBody string }
	recs := []rec{
		{"itd-3", "spc-13", writtenSpec("spc-13", "itd-3")}, // oldest, least ready
		{"itd-4", "spc-14", scoredSpec("spc-14", "itd-4")},  // tied with itd-9, older
		{"itd-9", "spc-19", scoredSpec("spc-19", "itd-9")},  // tied with itd-4, younger
		{"itd-2609010000000001", "spc-2609010000000011", writtenSpec("spc-2609010000000011", "itd-2609010000000001") +
			"\n## Footprint\n\n- packages: internal/core/intent, internal/core/spec\n- tests: the score\n"}, // youngest, between
	}
	for _, r := range recs {
		ic := readyIntent(r.id, "Title of "+r.id, r.spec, "")
		p := filepath.Join(root, ".abcd/development/intents/planned", r.id+"-s.md")
		q := filepath.Join(root, ".abcd/development/specs/open", r.spec+"-s.md")
		for path, body := range map[string]string{p: ic, q: r.specBody} {
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}
	corpus, err := intent.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	store, err := spec.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	var cands []intent.PickCandidate
	for _, r := range recs {
		it, ok := corpus.Lookup(r.id)
		if !ok {
			t.Fatalf("precondition: %s is in the corpus", r.id)
		}
		score, err := intent.ReadinessIn(root, store, it, r.spec)
		if err != nil {
			t.Fatal(err)
		}
		cands = append(cands, intent.PickCandidate{ID: r.id, Score: score})
	}
	pick, ok := intent.Choose(cands)
	if !ok || pick.Chosen.ID != "itd-4" || !pick.TieBrokenByAge {
		t.Fatalf("precondition: the pick takes itd-4 over its tie with itd-9: %+v", pick)
	}

	b, err := Read(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	var order []string
	for _, c := range pick.Candidates {
		order = append(order, c.ID)
	}
	if got := ids(b.Next); !reflect.DeepEqual(got, order) {
		t.Errorf("Next = %v, want the pick order %v", got, order)
	}
	if got := ids(b.Now); !reflect.DeepEqual(got, []string{"itd-4"}) || !b.Now[0].NextUp {
		t.Errorf("Now = %+v, want only itd-4 marked next up: the pick's choice, not the oldest id itd-3", b.Now)
	}
	if b.Order != OrderPick {
		t.Errorf("Order = %q, want %q", b.Order, OrderPick)
	}

	// itd-4 in a lane: the pick would not start it again, so the head is the
	// runner-up.
	b, err = Read(root, lanesOf(Started{Intent: "itd-4", Lane: Lane{Run: "run-1", Lane: "lane-1", Step: "implement"}}))
	if err != nil {
		t.Fatal(err)
	}
	if got := ids(b.Now); !reflect.DeepEqual(got, []string{"itd-4", "itd-9"}) || b.Now[0].NextUp || !b.Now[1].NextUp {
		t.Errorf("Now = %+v, want itd-4's lane row then itd-9 marked next up", b.Now)
	}
}

// TestTheHeadTakesAnIntentWhoseBlockerASettledRecordReplaced is rulings CF1
// and CF2 of 2026-09-30 on the board: the "next up" reads the build's own
// blocked check (intent.StartChecksIn), so an intent whose blocker was
// superseded by an accepted decision, or reclassified as a discipline, heads
// the board, and one whose blocker a proposed decision replaced does not.
func TestTheHeadTakesAnIntentWhoseBlockerASettledRecordReplaced(t *testing.T) {
	const superseded = "---\nid: itd-27\nslug: s\nkind: standalone\nsuperseded_by: adr-37\nkind_at_supersession: standalone\n---\n# The replaced one\n"
	cases := []struct {
		name  string
		files map[string]string
		heads bool
	}{
		{"superseded by an accepted decision", map[string]string{
			".abcd/development/intents/superseded/itd-27-s.md": superseded,
			".abcd/development/decisions/adrs/0037-d.md":       "---\nid: adr-37\nstatus: accepted\n---\n# d\n",
		}, true},
		{"reclassified as a discipline", map[string]string{
			".abcd/development/intents/disciplines/itd-27-s.md": "---\nid: itd-27\nslug: s\nkind: discipline\n---\n# A rule\n",
		}, true},
		{"superseded by a proposed decision", map[string]string{
			".abcd/development/intents/superseded/itd-27-s.md": superseded,
			".abcd/development/decisions/adrs/0037-d.md":       "---\nid: adr-37\nstatus: proposed\n---\n# d\n",
		}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			w := func(rel, body string) {
				t.Helper()
				p := filepath.Join(root, filepath.FromSlash(rel))
				if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			w(".abcd/development/intents/planned/itd-4-blocked.md", readyIntent("itd-4", "The blocked one", "spc-14", "blocked_by: [itd-27]\n"))
			w(".abcd/development/specs/open/spc-14-blocked.md", scoredSpec("spc-14", "itd-4"))
			for rel, body := range tc.files {
				w(rel, body)
			}
			b, err := Read(root, nil)
			if err != nil {
				t.Fatal(err)
			}
			if got := ids(b.Next); !reflect.DeepEqual(got, []string{"itd-4"}) {
				t.Fatalf("Next = %v, want [itd-4]", got)
			}
			switch {
			case tc.heads && (len(b.Now) != 1 || b.Now[0].ID != "itd-4" || !b.Now[0].NextUp):
				t.Errorf("Now = %+v, want itd-4 marked next up: its blocker is settled", b.Now)
			case !tc.heads && len(b.Now) != 0:
				t.Errorf("Now = %+v, want no head: a proposed decision does not settle the blocker", b.Now)
			}
		})
	}
}

// TestTheHeadPassesOverWhatTheBuildRefusesFromTheRecord is the head under the
// build's record-only pre-start checks (iss-2609291803334904): a READY intent
// with an open question, an unanswered claim section, an unshipped blocker, or
// a spec that leaves no step to build is one `abcd build next` excludes, so it
// stays in Next but is never the head, however ready it scores. The free,
// less ready intent beside it is the head; alone, it leaves no head at all.
func TestTheHeadPassesOverWhatTheBuildRefusesFromTheRecord(t *testing.T) {
	const answered = "We expect it to work because it is small; shown wrong if it is not."
	cases := []struct {
		name, check    string
		extra, specAdd string
		edit           func(string) string
	}{
		{name: "an open question", check: intent.StartCheckOpenQuestions,
			edit: func(s string) string {
				return strings.Replace(s, "## Grounds", "## Open Questions\n\n- Which runner?\n\n## Grounds", 1)
			}},
		{name: "an unanswered mechanism prompt", check: intent.StartCheckClaimSections,
			edit: func(s string) string { return strings.Replace(s, answered, intent.MechanismPrompt, 1) }},
		{name: "an unrecorded scope condition", check: intent.StartCheckClaimSections,
			edit: func(s string) string { return strings.Replace(s, "## Scope Conditions\n\nNone stated.\n\n", "", 1) }},
		{name: "an unshipped blocker", check: intent.StartCheckBlocked, extra: "blocked_by: [itd-99]\n"},
		{name: "every step landed", check: intent.StartCheckSteps, specAdd: "\n## Steps\n\n1. The parser\n   - landed: #1\n"},
		{name: "an unreadable steps section", check: intent.StartCheckSteps, specAdd: "\n## Steps\n\nthe parser, then the loop\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			w := func(rel, body string) {
				t.Helper()
				p := filepath.Join(root, filepath.FromSlash(rel))
				if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			const in = ".abcd/development/intents/planned/"
			const sp = ".abcd/development/specs/open/"
			refused := readyIntent("itd-4", "The refused one", "spc-14", tc.extra)
			if tc.edit != nil {
				refused = tc.edit(refused)
			}
			// itd-4 is the readiest and the oldest: without the check it heads.
			w(in+"itd-4-refused.md", refused)
			w(sp+"spc-14-refused.md", scoredSpec("spc-14", "itd-4")+tc.specAdd)

			b, err := Read(root, nil)
			if err != nil {
				t.Fatal(err)
			}
			if got := ids(b.Next); !reflect.DeepEqual(got, []string{"itd-4"}) {
				t.Fatalf("Next = %v, want [itd-4]: the intent is READY, so it stays in Next", got)
			}
			if len(b.Now) != 0 {
				t.Errorf("Now = %+v, want no head: the build refuses itd-4 on %s", b.Now, tc.check)
			}

			w(in+"itd-9-free.md", readyIntent("itd-9", "The free one", "spc-19", ""))
			w(sp+"spc-19-free.md", writtenSpec("spc-19", "itd-9"))
			if b, err = Read(root, nil); err != nil {
				t.Fatal(err)
			}
			if got := ids(b.Next); !reflect.DeepEqual(got, []string{"itd-4", "itd-9"}) {
				t.Errorf("Next = %v, want [itd-4 itd-9] in pick order", got)
			}
			if got := ids(b.Now); !reflect.DeepEqual(got, []string{"itd-9"}) || !b.Now[0].NextUp {
				t.Errorf("Now = %+v, want only itd-9 marked next up: itd-4 fails %s", b.Now, tc.check)
			}
		})
	}
}
