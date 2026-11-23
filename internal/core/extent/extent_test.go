package extent

import (
	"bufio"
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"
)

// looked is a fixture helper: a looked-at entry with a label derived from its
// identifier.
func looked(id string, count int) Looked {
	return Looked{ID: id, Label: strings.ReplaceAll(id, "_", " "), Count: count}
}

func leftOut(id string, cause Cause) NotLooked {
	return NotLooked{ID: id, Label: strings.ReplaceAll(id, "_", " "), Source: SourceLeftOut, Cause: cause}
}

func gap(check, kind string, reason Reason) Gap {
	return Gap{Check: check, Kind: kind, Reason: reason, Text: check + " did not answer"}
}

// encode marshals s and decodes the result into a generic map, failing the
// test on an encoding error.
func encode(t *testing.T, s Statement) map[string]any {
	t.Helper()
	b, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("decode %s: %v", b, err)
	}
	return m
}

// TestVerdictFollowsItsOrder holds the four verdicts to their order: found
// whenever something was found; otherwise none when no registry kind was
// looked at, whatever the reason; otherwise clear_except_gaps when a check
// could not be made; otherwise clear.
func TestVerdictFollowsItsOrder(t *testing.T) {
	cases := []struct {
		name string
		s    Statement
		want Verdict
	}{
		{"a finding beside a gap and a left-out kind is found", Statement{
			LookedAt:      []Looked{looked("worktrees", 2), looked("branches", 0)},
			NotLookedAt:   []NotLooked{leftOut("reservations", CauseNothingToRead)},
			CouldNotCheck: []Gap{gap("process check", "worktrees", ReasonToolMissing)},
		}, VerdictFound},
		{"no check armed is none", Statement{
			NotLookedAt: []NotLooked{leftOut("rule_a", CauseNotArmed), leftOut("rule_b", CauseNotArmed)},
		}, VerdictNone},
		{"nothing to read is none", Statement{
			NotLookedAt: []NotLooked{leftOut("rule_a", CauseNothingToRead)},
		}, VerdictNone},
		{"no kind readable is none, ahead of clear_except_gaps", Statement{
			CouldNotCheck: []Gap{gap("store", "reservations", ReasonUnreadable), gap("forge", "branches", ReasonUnreachable)},
		}, VerdictNone},
		{"an empty registry is none", Statement{}, VerdictNone},
		{"nothing found with a gap is clear_except_gaps", Statement{
			LookedAt:      []Looked{looked("worktrees", 0)},
			CouldNotCheck: []Gap{gap("process check", "worktrees", ReasonTimedOut)},
		}, VerdictClearExceptGaps},
		{"a left-out kind alone does not make a gap", Statement{
			LookedAt:    []Looked{looked("worktrees", 0)},
			NotLookedAt: []NotLooked{leftOut("reservations", CauseNotArmed)},
		}, VerdictClear},
		{"nothing found and no gap is clear", Statement{
			LookedAt: []Looked{looked("worktrees", 0), looked("branches", 0)},
		}, VerdictClear},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := tc.s.Verdict()
			if !ok || got != tc.want {
				t.Fatalf("Verdict() = %q, %v; want %q, true", got, ok, tc.want)
			}
			if m := encode(t, tc.s); m["verdict"] != string(tc.want) {
				t.Fatalf("recorded verdict = %v, want %q", m["verdict"], tc.want)
			}
		})
	}
}

// TestNoClearStateRecordsNoVerdict: a command with no clear state (the drive
// of a drain) carries no verdict member at all, and the lists stand in every
// recorded result, empty or not.
func TestNoClearStateRecordsNoVerdict(t *testing.T) {
	s := Statement{NoClearState: true, LookedAt: []Looked{looked("open_issues", 3)}}
	if v, ok := s.Verdict(); ok {
		t.Fatalf("Verdict() = %q, true; want no verdict", v)
	}
	m := encode(t, s)
	if _, has := m["verdict"]; has {
		t.Fatalf("recorded a verdict member for a command with no clear state: %v", m)
	}
	for _, member := range []string{"looked_at", "not_looked_at", "could_not_check"} {
		if _, ok := m[member].([]any); !ok {
			t.Errorf("member %q = %#v, want a list, empty or not", member, m[member])
		}
	}
	if _, has := m["depth"]; has {
		t.Errorf("recorded a depth member where no model judged: %v", m)
	}
}

// TestDecodeKeepsTheVerdictMember: a recorded statement reads back with every
// entry it carried, its absent verdict reads as no clear state, and a verdict
// the entries do not support is refused.
func TestDecodeKeepsTheVerdictMember(t *testing.T) {
	full := Statement{
		LookedAt: []Looked{looked("worktrees", 0), looked("local_branches", 0)},
		NotLookedAt: []NotLooked{
			leftOut("reservations", CauseNothingToRead),
			{ID: "embargo", Label: "embargo store", Source: SourceExclusion},
			{ID: "scratch", Label: "scratch folders", Source: SourceListed},
		},
		CouldNotCheck: []Gap{gap("process check", "worktrees", ReasonToolMissing), gap("forge", "", ReasonUnreachable)},
		Depth: []Depth{
			{Role: "validator", Lane: "itd-1",
				Class:  &Told{Values: []string{"strong", "standard"}, From: OriginRunState},
				Runs:   Runs{Count: 2, From: OriginRunState},
				Effort: &Told{Values: []string{"high"}, From: OriginReceipt}},
			{Role: "drain judge", Issue: "iss-1", Class: &Told{Mark: MarkNoneAsked}, Runs: Runs{Count: 1, From: OriginRunState}, Effort: &Told{Mark: MarkNoneAsked}},
			{Role: "receipts", Class: &Told{Mark: MarkNotTold}, Runs: Runs{Mark: MarkNotTold}, Effort: &Told{Mark: MarkNotTold}},
			{Role: "validator", Lane: "itd-2", Runs: Runs{Count: 0, From: OriginRunState}},
		},
	}
	drive := full
	drive.NoClearState = true
	drive.LookedAt = []Looked{looked("open_issues", 1)}
	for _, s := range []Statement{full, drive} {
		b, err := json.Marshal(s)
		if err != nil {
			t.Fatal(err)
		}
		var back Statement
		if err := json.Unmarshal(b, &back); err != nil {
			t.Fatalf("decode %s: %v", b, err)
		}
		if !reflect.DeepEqual(back, s) {
			t.Fatalf("decode %s:\n got %+v\nwant %+v", b, back, s)
		}
	}
	var s Statement
	err := json.Unmarshal([]byte(`{"looked_at":[{"id":"worktrees","label":"worktrees","count":0}],"not_looked_at":[],"could_not_check":[],"verdict":"found"}`), &s)
	if err == nil {
		t.Fatal("decoded a verdict of found over a statement that found nothing")
	}
}

// TestIdentifiersOutsideTheirSetFailToEncode: a reason, a cause or a source
// outside its closed set fails to encode, and so does a cause on an entry that
// is not left out in this run, or a left-out entry with none.
func TestIdentifiersOutsideTheirSetFailToEncode(t *testing.T) {
	base := func() Statement { return Statement{LookedAt: []Looked{looked("worktrees", 0)}} }
	cases := map[string]func(*Statement){
		"unknown reason": func(s *Statement) { s.CouldNotCheck = []Gap{gap("forge", "", Reason("offline"))} },
		"unknown cause":  func(s *Statement) { s.NotLookedAt = []NotLooked{leftOut("reservations", Cause("not_implemented"))} },
		"unknown source": func(s *Statement) {
			s.NotLookedAt = []NotLooked{{ID: "stashes", Label: "stashes", Source: Source("forgotten")}}
		},
		"left out with no cause": func(s *Statement) { s.NotLookedAt = []NotLooked{leftOut("reservations", "")} },
		"cause on an exclusion": func(s *Statement) {
			s.NotLookedAt = []NotLooked{{ID: "embargo", Label: "embargo store", Source: SourceExclusion, Cause: CauseNotArmed}}
		},
		"gap with no text": func(s *Statement) {
			s.CouldNotCheck = []Gap{{Check: "forge", Reason: ReasonUnreachable}}
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			s := base()
			mutate(&s)
			if b, err := json.Marshal(s); err == nil {
				t.Fatalf("encoded %s", b)
			}
		})
	}
}

// TestAKindInBothListsFailsToEncode is criterion 12: a kind is never both
// looked at and not looked at.
func TestAKindInBothListsFailsToEncode(t *testing.T) {
	s := Statement{
		LookedAt:    []Looked{looked("reservations", 0)},
		NotLookedAt: []NotLooked{leftOut("reservations", CauseNothingToRead)},
	}
	b, err := json.Marshal(s)
	if err == nil {
		t.Fatalf("encoded a kind in both lists: %s", b)
	}
	if !strings.Contains(err.Error(), "reservations") {
		t.Fatalf("error %q does not name the kind", err)
	}
	// A gap beside a kind the run did read is not the kind in both lists.
	s = Statement{
		LookedAt:      []Looked{looked("worktrees", 0)},
		CouldNotCheck: []Gap{gap("process check", "worktrees", ReasonToolMissing)},
	}
	if _, err := json.Marshal(s); err != nil {
		t.Fatalf("refused a gap beside a looked-at kind: %v", err)
	}
}

// TestLeaveOutTakesTheFirstCause: where more than one cause applies, the entry
// carries the first of host_run, other_run, not_armed and nothing_to_read.
func TestLeaveOutTakesTheFirstCause(t *testing.T) {
	cases := []struct {
		causes []Cause
		want   Cause
	}{
		{[]Cause{CauseNothingToRead, CauseNotArmed}, CauseNotArmed},
		{[]Cause{CauseNotArmed, CauseOtherRun}, CauseOtherRun},
		{[]Cause{CauseNothingToRead, CauseOtherRun, CauseHostRun}, CauseHostRun},
		{[]Cause{CauseNothingToRead}, CauseNothingToRead},
	}
	for _, tc := range cases {
		var s Statement
		s.LeaveOut("deep_smoke", "deep smoke", tc.causes...)
		if len(s.NotLookedAt) != 1 || s.NotLookedAt[0].Cause != tc.want || s.NotLookedAt[0].Source != SourceLeftOut {
			t.Errorf("LeaveOut(%v) = %+v, want one left-out entry with cause %q", tc.causes, s.NotLookedAt, tc.want)
		}
	}
}

// TestDepthEncodesWhereEachValueCameFrom: a depth entry encodes its class, its
// number of judging runs and its effort, each with where it came from or
// marked not_told or none_asked, and an entry with no run gives the number,
// zero, and neither of the others.
func TestDepthEncodesWhereEachValueCameFrom(t *testing.T) {
	s := Statement{
		NoClearState: true,
		LookedAt:     []Looked{looked("open_issues", 2)},
		Depth: []Depth{
			{Role: "validator", Lane: "itd-1",
				Class:  &Told{Values: []string{"strong"}, From: OriginRunState},
				Runs:   Runs{Count: 3, From: OriginRunState},
				Effort: &Told{Values: []string{"high"}, From: OriginRunState}},
			{Role: "receipts", Class: &Told{Mark: MarkNotTold}, Runs: Runs{Mark: MarkNotTold}, Effort: &Told{Mark: MarkNotTold}},
			{Role: "drain judge", Issue: "iss-1",
				Class:  &Told{Mark: MarkNoneAsked},
				Runs:   Runs{Count: 1, From: OriginRunState},
				Effort: &Told{Mark: MarkNoneAsked}},
			{Role: "validator", Lane: "itd-2", Runs: Runs{Count: 0, From: OriginRunState}},
			{Role: "host review", Class: &Told{Values: []string{"frontier"}, From: OriginReceipt}, Runs: Runs{Mark: MarkNotTold}, Effort: &Told{Mark: MarkNotTold}},
		},
	}
	b, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	var got struct {
		Depth []map[string]any `json:"depth"`
	}
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Depth) != 5 {
		t.Fatalf("depth = %v, want five entries", got.Depth)
	}
	wants := []string{
		`{"class":{"from":"run_state","values":["strong"]},"effort":{"from":"run_state","values":["high"]},"lane":"itd-1","role":"validator","runs":{"count":3,"from":"run_state"}}`,
		`{"class":{"mark":"not_told"},"effort":{"mark":"not_told"},"role":"receipts","runs":{"mark":"not_told"}}`,
		`{"class":{"mark":"none_asked"},"effort":{"mark":"none_asked"},"issue":"iss-1","role":"drain judge","runs":{"count":1,"from":"run_state"}}`,
		`{"lane":"itd-2","role":"validator","runs":{"count":0,"from":"run_state"}}`,
		`{"class":{"from":"receipt","values":["frontier"]},"effort":{"mark":"not_told"},"role":"host review","runs":{"mark":"not_told"}}`,
	}
	for i, want := range wants {
		entry, _ := json.Marshal(got.Depth[i])
		if string(entry) != want {
			t.Errorf("depth[%d] = %s\nwant      %s", i, entry, want)
		}
	}

	bad := map[string]Depth{
		"unknown origin":              {Role: "r", Runs: Runs{Count: 1, From: Origin("guess")}},
		"unknown mark":                {Role: "r", Runs: Runs{Mark: Mark("unknown")}},
		"value and mark both":         {Role: "r", Runs: Runs{Count: 1, From: OriginRunState, Mark: MarkNotTold}},
		"neither value nor mark":      {Role: "r", Runs: Runs{Count: 1}},
		"class with no run":           {Role: "r", Runs: Runs{Count: 0, From: OriginRunState}, Class: &Told{Values: []string{"strong"}, From: OriginRunState}},
		"class values with no from":   {Role: "r", Runs: Runs{Count: 1, From: OriginRunState}, Class: &Told{Values: []string{"strong"}}},
		"lane and issue both":         {Role: "r", Lane: "itd-1", Issue: "iss-1", Runs: Runs{Count: 1, From: OriginRunState}},
		"runs and no class":           {Role: "r", Runs: Runs{Count: 2, From: OriginRunState}, Effort: &Told{Values: []string{"high"}, From: OriginRunState}},
		"runs not told and no effort": {Role: "r", Runs: Runs{Mark: MarkNotTold}, Class: &Told{Mark: MarkNotTold}},
	}
	for name, d := range bad {
		s := Statement{LookedAt: []Looked{looked("k", 0)}, Depth: []Depth{d}}
		if b, err := json.Marshal(s); err == nil {
			t.Errorf("%s: encoded %s", name, b)
		}
	}
}

// identifiersGolden holds the reasons and causes a recorded result may carry.
// A release may add to it and never rename or remove a line (criterion 24).
const identifiersGolden = "testdata/identifiers.golden"

func readGolden(t *testing.T, path string) []string {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	var out []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		out = append(out, line)
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

// TestIdentifiersOnlyGrow is criterion 24 for the statement's own sets: every
// reason and cause in the golden file is still one the package encodes.
func TestIdentifiersOnlyGrow(t *testing.T) {
	golden := readGolden(t, identifiersGolden)
	if len(golden) == 0 {
		t.Fatalf("%s holds no identifier", identifiersGolden)
	}
	removed, unrecorded := identifierDrift(golden, identifiers())
	if len(removed) > 0 {
		t.Errorf("identifiers renamed or removed (recorded results carry them, so they may not change; add a new one instead): %v", removed)
	}
	if len(unrecorded) > 0 {
		t.Errorf("identifiers the package encodes and %s does not hold; append each so it may not change later: %v", identifiersGolden, unrecorded)
	}
}

// TestTheGoldenCheckRefusesARenameAndAdmitsAnAddition proves the comparison
// itself: a renamed identifier is reported, an added one is not.
func TestTheGoldenCheckRefusesARenameAndAdmitsAnAddition(t *testing.T) {
	golden := []string{"reason unreadable", "cause not_armed"}
	removed, unrecorded := identifierDrift(golden, []string{"reason unreadable", "cause not_configured"})
	if len(removed) != 1 || removed[0] != "cause not_armed" {
		t.Fatalf("a rename gave removed %v, want [cause not_armed]", removed)
	}
	if len(unrecorded) != 1 || unrecorded[0] != "cause not_configured" {
		t.Fatalf("a rename gave unrecorded %v, want [cause not_configured]", unrecorded)
	}
	added := []string{"reason unreadable", "cause not_armed", "reason rate_limited"}
	if removed, unrecorded := identifierDrift(golden, added); len(removed) != 0 || len(unrecorded) != 1 || unrecorded[0] != "reason rate_limited" {
		t.Fatalf("an addition the golden lacks gave removed %v, unrecorded %v; want only [reason rate_limited] unrecorded", removed, unrecorded)
	}
	if removed, unrecorded := identifierDrift(append(golden, "reason rate_limited"), added); len(removed) != 0 || len(unrecorded) != 0 {
		t.Fatalf("an addition written to both gave removed %v, unrecorded %v; want nothing", removed, unrecorded)
	}
	// Every identifier the package encodes is accepted by its own encoder.
	for _, id := range identifiers() {
		set, value, _ := strings.Cut(id, " ")
		var err error
		switch set {
		case "reason":
			_, err = Reason(value).MarshalText()
		case "cause":
			_, err = Cause(value).MarshalText()
		default:
			t.Fatalf("identifier %q names no set", id)
		}
		if err != nil {
			t.Errorf("%s: %v", id, err)
		}
	}
}

// TestDecodeRefusesUnknownFields: a recorded statement is read strictly, so a
// member the type does not name is refused at every level, the depth entry's
// number of runs included.
func TestDecodeRefusesUnknownFields(t *testing.T) {
	cases := map[string]string{
		"top level":     `{"looked_at":[],"not_looked_at":[],"could_not_check":[],"verdict":"none","extra":1}`,
		"looked-at":     `{"looked_at":[{"id":"k","label":"k","count":0,"extra":1}],"not_looked_at":[],"could_not_check":[],"verdict":"clear"}`,
		"not-looked-at": `{"looked_at":[],"not_looked_at":[{"id":"k","label":"k","source":"listed","extra":1}],"could_not_check":[],"verdict":"none"}`,
		"gap":           `{"looked_at":[{"id":"k","label":"k","count":0}],"not_looked_at":[],"could_not_check":[{"check":"c","reason":"unreadable","text":"t","extra":1}],"verdict":"clear_except_gaps"}`,
		"depth":         `{"looked_at":[{"id":"k","label":"k","count":0}],"not_looked_at":[],"could_not_check":[],"verdict":"clear","depth":[{"role":"r","runs":{"count":1,"from":"run_state"},"class":{"mark":"none_asked"},"effort":{"mark":"none_asked"},"extra":1}]}`,
		"told":          `{"looked_at":[{"id":"k","label":"k","count":0}],"not_looked_at":[],"could_not_check":[],"verdict":"clear","depth":[{"role":"r","runs":{"count":1,"from":"run_state"},"class":{"mark":"not_told","extra":1},"effort":{"mark":"not_told"}}]}`,
		"runs":          `{"looked_at":[{"id":"k","label":"k","count":0}],"not_looked_at":[],"could_not_check":[],"verdict":"clear","depth":[{"role":"r","runs":{"count":1,"from":"run_state","extra":1},"class":{"mark":"none_asked"},"effort":{"mark":"none_asked"}}]}`,
	}
	for name, doc := range cases {
		var s Statement
		if err := json.Unmarshal([]byte(doc), &s); err == nil {
			t.Errorf("%s: decoded a statement with an unknown field", name)
		} else if !strings.Contains(err.Error(), "extra") {
			t.Errorf("%s: error %q does not name the field", name, err)
		}
	}
	// The same documents without the unknown field read.
	for name, doc := range cases {
		var s Statement
		clean := strings.ReplaceAll(strings.ReplaceAll(doc, `,"extra":1`, ""), `"extra":1,`, "")
		if err := json.Unmarshal([]byte(clean), &s); err != nil {
			t.Errorf("%s: refused the document without its unknown field: %v", name, err)
		}
	}
}
