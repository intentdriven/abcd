package statusblock

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
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

func draft(id, title string) string {
	return "---\nid: " + id + "\nslug: s\nspec_id: null\nkind: standalone\n---\n# " + title + "\n\n## Acceptance Criteria\n\n- Given x, when y, then z.\n"
}

// store lays a record with three READY planned intents (one of them held), one
// planned intent the gate refuses, and two drafts, in an order that is not
// their id order.
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
	w(sp+"spc-2609010000000011-late.md", writtenSpec("spc-2609010000000011", "itd-2609010000000001"))
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
// Next every READY planned intent in order; Later the planned intents the gate
// refuses, each naming its failing checks, then the drafts; every row carries
// its id and title.
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
		t.Errorf("Now[1] = %+v, want itd-7 marked next up: the oldest READY intent that is not held", b.Now[1])
	}

	if got, want := ids(b.Next), []string{"itd-5", "itd-7", "itd-2609010000000001"}; !reflect.DeepEqual(got, want) {
		t.Errorf("Next = %v, want %v: every READY planned intent, oldest id first", got, want)
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
	if b.Order != OrderRecordID {
		t.Errorf("Order = %q, want %q until the pick order exists", b.Order, OrderRecordID)
	}
}

// TestBlockWithoutAStateFileKeepsOnlyTheHead is criterion 3: with no lanes, Now
// holds only the head, and Next and Later are exactly what they were.
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
		if got := ids(without.Now); !reflect.DeepEqual(got, []string{"itd-7"}) || !without.Now[0].NextUp {
			t.Errorf("%s: Now = %v, want only the head itd-7 marked next up", name, without.Now)
		}
		if !reflect.DeepEqual(with.Next, without.Next) || !reflect.DeepEqual(with.Later, without.Later) {
			t.Errorf("%s: Next or Later changed with the state file:\nwith    %+v %+v\nwithout %+v %+v", name, with.Next, with.Later, without.Next, without.Later)
		}
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
		`"now":[`, `"next":[`, `"later":[`, `"order":"record-id"`,
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
	if want := `{"now":[],"next":[],"later":[],"order":"record-id"}`; string(data) != want {
		t.Errorf("empty block = %s, want %s", data, want)
	}
}

func TestIDLessOrdersOrdinalsBeforeStamps(t *testing.T) {
	for _, c := range []struct {
		a, b string
		want bool
	}{
		{"itd-7", "itd-50", true},
		{"itd-50", "itd-7", false},
		{"itd-100", "itd-2609010000000001", true},
		{"itd-2609010000000001", "itd-2609020000000002", true},
		{"itd-007", "itd-8", true},
	} {
		if got := idLess(c.a, c.b); got != c.want {
			t.Errorf("idLess(%s, %s) = %v, want %v", c.a, c.b, got, c.want)
		}
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
