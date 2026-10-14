package loop

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// A run's stored hand-back text (iss-2610121516559163): a lane's hand-back is
// read back from state.json and echoed on `implement record --json`,
// `implement status --json` and, by the call that routes it, `drain --json`.
// A state file whose hand-back carries a control, bidi or zero-width rune in
// any of its text is refused as the run's state, never echoing the value.

// capHandBack is a hand-back shaped as the fix-round cap writes one.
var capHandBack = map[string]any{
	"at": "2026-10-12T09:00:00Z", "verdict": VerdictUnachievable, "round": 2, "fix_rounds": 1,
	"verdicts": "ruthless-reviewer FIX FIRST, security-reviewer SHIP, intent-auditor NOT_MET",
	"findings": []any{"lane-1/ruthless-2.md"}, "not_met": []any{"c1"}, "undecided": []any{"c2"},
}

// handBackForgeries lists every text of a stored hand-back, each string field
// and each element of a list, by the path forgeHandBack takes.
func handBackForgeries(hb map[string]any) []string {
	var out []string
	for k, v := range hb {
		switch v := v.(type) {
		case string:
			if k != "at" {
				out = append(out, k)
			}
		case []any:
			for i := range v {
				out = append(out, fmt.Sprintf("%s.%d", k, i))
			}
		}
	}
	slices.Sort(out)
	return out
}

// withHandBack returns state with lane 0's hand-back replaced by hb, one field
// (a key, or key.index into a list) suffixed with bad when field is set, and
// the version set.
func withHandBack(t *testing.T, state []byte, hb map[string]any, field, bad string, version int) []byte {
	t.Helper()
	var doc map[string]any
	if err := json.Unmarshal(state, &doc); err != nil {
		t.Fatal(err)
	}
	cp := map[string]any{}
	for k, v := range hb {
		if l, ok := v.([]any); ok {
			v = slices.Clone(l)
		}
		cp[k] = v
	}
	if field != "" {
		key, idx, isList := strings.Cut(field, ".")
		if isList {
			var i int
			fmt.Sscan(idx, &i)
			l := cp[key].([]any)
			l[i] = l[i].(string) + bad
		} else {
			cp[key] = cp[key].(string) + bad
		}
	}
	lane := doc["lanes"].([]any)[0].(map[string]any)
	lane["hand_back"], lane["stage"] = cp, string(StageHandedBack)
	doc["schema_version"] = version
	data, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// TestARunStateRefusesHandBackTextItNeverWrites: a hand-back a lane's
// receipt wrote, and one shaped as the fix-round cap writes it, read back at
// the current version and as version 9; each with one text forged with U+202E
// or U+009B is refused as the run's state by the read, the record and the
// drain's call that routes the lane's hand-back, none echoing the rune.
func TestARunStateRefusesHandBackTextItNeverWrites(t *testing.T) {
	repo := drainRepo(t)
	first := drainToLane(t, repo, Options{}, DrainOptions{})
	id := first.Start.RunID
	handBackLaneOf(t, repo, id, Options{}, LaneHandBack{Kind: HandBackDesignFinding, Reason: "the fix changes the record's shape", Home: "the parser's ADR"})
	st, err := ReadState(repo.Root(), id)
	if err != nil || st.Lanes[0].HandBack == nil || st.Lanes[0].HandBack.Home != "the parser's ADR" || st.Lanes[0].HandBack.Discarded == "" {
		t.Fatalf("a receipt's hand-back round-trips: %+v %v", st.Lanes[0].HandBack, err)
	}
	good := stateBytes(t, repo.Root(), id)
	var doc map[string]any
	if err := json.Unmarshal(good, &doc); err != nil {
		t.Fatal(err)
	}
	receiptHandBack := doc["lanes"].([]any)[0].(map[string]any)["hand_back"].(map[string]any)
	path := filepath.Join(repo.Root(), filepath.FromSlash(StateRelPath(id)))
	defer func() {
		if err := os.WriteFile(path, good, 0o600); err != nil {
			t.Fatal(err)
		}
	}()
	forged := 0
	for name, hb := range map[string]map[string]any{"the receipt's": receiptHandBack, "the cap's": capHandBack} {
		for _, version := range []int{SchemaVersion, schemaVersionUnkept} {
			if err := os.WriteFile(path, withHandBack(t, good, hb, "", "", version), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := ReadState(repo.Root(), id); err != nil {
				t.Fatalf("%s hand-back at version %d round-trips: %v", name, version, err)
			}
			if _, err := ReadRecord(repo.Root(), id); err != nil {
				t.Fatalf("%s hand-back at version %d reaches the record: %v", name, version, err)
			}
			for _, field := range handBackForgeries(hb) {
				for _, bad := range []string{"‮", "\u009b"} {
					what := fmt.Sprintf("%s hand-back, version %d, %s with %q", name, version, field, bad)
					if err := os.WriteFile(path, withHandBack(t, good, hb, field, bad, version), 0o600); err != nil {
						t.Fatal(err)
					}
					reads := map[string]func() error{
						"the read":   func() error { _, err := ReadState(repo.Root(), id); return err },
						"the record": func() error { _, err := ReadRecord(repo.Root(), id); return err },
						"the drain":  func() error { _, err := Drain(repo.Root(), Options{}, DrainOptions{}); return err },
					}
					for how, read := range reads {
						r := mustRefusal(t, read())
						noRawRune(t, what+", "+how, r)
						if r.Stage != "state" || !strings.Contains(r.Reason, StateRelPath(id)+" carries a hand-back the loop never writes") ||
							r.Remedy != "the loop is the file's only writer; restore it or remove the run directory "+runRel(id) {
							t.Fatalf("%s, %s: refused as the run's state: %+v", what, how, r)
						}
					}
					forged++
				}
			}
		}
	}
	t.Logf("%d forged hand-back texts refused", forged)
}
