package loop

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A lane receipt's stored text (iss-2610121532247004): the model a receipt reports and
// each resolution's note and grounds are stored on the lane and echoed on
// `implement record --json` and `implement status --json`. A receipt carrying
// a control, bidi or zero-width rune there is refused when it is handed in,
// and a run state carrying one is refused when it is read; prose keeps its
// line breaks and tabs.

// TestAReceiptsStoredTextIsHeldToWhatTheLoopStores: a receipt whose model, or
// a resolution's note or grounds, carries U+202E or U+009B is refused at the
// receipt without echoing it, and the lane does not move; one with a
// multi-line, tabbed note verifies and round-trips; a run state forged in any
// of those fields, or in a resolution's issue, commit or impact, is refused
// as the run's state by the read and the record.
func TestAReceiptsStoredTextIsHeldToWhatTheLoopStores(t *testing.T) {
	repo, runID, l, dir := issueAwaiting(t)
	sha := laneCommit(t, repo, l, "fix.txt")
	good := func() LaneReceipt {
		rc := goodReceipt(t, runID, l, dir, sha)
		rc.Model = "Claude:claude-opus-5-5"
		rc.Resolves = []Resolution{{Issue: eligibleIssue, Commit: sha, Note: "guarded the map\nand the slice,\twith one check", Impact: "fix",
			Grounds: "pursued: the guard holds;\nshown wrong if it panics"}}
		return rc
	}
	for _, c := range []struct {
		what  string
		forge func(*LaneReceipt, string)
	}{
		{"model", func(rc *LaneReceipt, bad string) { rc.Model += bad }},
		{"note", func(rc *LaneReceipt, bad string) { rc.Resolves[0].Note += bad }},
		{"grounds", func(rc *LaneReceipt, bad string) { rc.Resolves[0].Grounds = bad + rc.Resolves[0].Grounds }},
	} {
		for _, bad := range []string{"‮", "\u009b"} {
			rc := good()
			c.forge(&rc, bad)
			path := writeReceipt(t, dir, rc)
			_, err := Receipt(repo.Root(), runID, path, DefaultStages(), Options{})
			r := mustRefusal(t, err)
			noRawRune(t, c.what, r)
			if r.Stage != "receipt" || !strings.Contains(r.Reason, c.what) || !strings.Contains(r.Reason, "control, bidi or zero-width") {
				t.Fatalf("a receipt whose %s carries %q is refused at the receipt naming it: %+v", c.what, bad, r)
			}
			if st, _ := ReadState(repo.Root(), runID); st.Lanes[0].Stage != StageImplement || len(st.Lanes[0].Receipts) != 0 {
				t.Fatalf("a refused receipt moves nothing: %+v", st.Lanes[0])
			}
		}
	}
	path := writeReceipt(t, dir, good())
	if _, err := Receipt(repo.Root(), runID, path, DefaultStages(), Options{}); err != nil {
		t.Fatalf("a receipt with a multi-line, tabbed note verifies: %v", err)
	}
	st, err := ReadState(repo.Root(), runID)
	if err != nil || len(st.Lanes[0].Receipts) != 1 || st.Lanes[0].Receipts[0].Model != good().Model ||
		len(st.Lanes[0].Resolves) != 1 || st.Lanes[0].Resolves[0] != good().Resolves[0] {
		t.Fatalf("the receipt's model and resolution round-trip: %+v %v", st.Lanes[0], err)
	}

	statePath := filepath.Join(repo.Root(), filepath.FromSlash(StateRelPath(runID)))
	stored := stateBytes(t, repo.Root(), runID)
	forged := 0
	for _, f := range []struct{ list, field string }{
		{"receipts", "model"},
		{"resolves", "note"}, {"resolves", "grounds"}, {"resolves", "issue"}, {"resolves", "commit"}, {"resolves", "impact"},
	} {
		for _, version := range []int{SchemaVersion, schemaVersionUnkept} {
			for _, bad := range []string{"‮", "\u009b"} {
				var doc map[string]any
				if err := json.Unmarshal(stored, &doc); err != nil {
					t.Fatal(err)
				}
				entry := doc["lanes"].([]any)[0].(map[string]any)[f.list].([]any)[0].(map[string]any)
				entry[f.field] = entry[f.field].(string) + bad
				doc["schema_version"] = version
				data, err := json.Marshal(doc)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(statePath, data, 0o600); err != nil {
					t.Fatal(err)
				}
				what := fmt.Sprintf("%s.%s at version %d with %q", f.list, f.field, version, bad)
				for how, read := range map[string]func() error{
					"the read":   func() error { _, err := ReadState(repo.Root(), runID); return err },
					"the record": func() error { _, err := ReadRecord(repo.Root(), runID); return err },
				} {
					r := mustRefusal(t, read())
					noRawRune(t, what+", "+how, r)
					if r.Stage != "state" || !strings.Contains(r.Reason, StateRelPath(runID)+" carries a receipt's model or resolution the loop never writes") ||
						r.Remedy != "the loop is the file's only writer; restore it or remove the run directory "+runRel(runID) {
						t.Fatalf("%s, %s: refused as the run's state: %+v", what, how, r)
					}
				}
				forged++
			}
		}
	}
	if err := os.WriteFile(statePath, stored, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Logf("%d forged receipt texts refused", forged)
}
