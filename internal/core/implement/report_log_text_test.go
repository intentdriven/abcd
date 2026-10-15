package implement

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestAReportsLogTextIsMasked (iss-2610121653432051): beyond the outage spans,
// a mode tally's mode, a session's id and role, the leader and an
// intervention's kind come from the run log the sessions append to, and an
// unparsed line names the log it came from; each reaches `implement report
// --json`, so a hand-edited or repository-shipped log whose fields carry
// U+202E, U+009B or U+034F is compared with the rune masked, every figure
// kept.
func TestAReportsLogTextIsMasked(t *testing.T) {
	const runes = "\u202e\u009b\u034f"
	for _, bad := range []string{"\u202e", "\u009b", "\u034f"} {
		q := func(s string) string { b, _ := json.Marshal(s); return string(b) }
		sess := q("s1" + bad)
		log := `{"ts":"2026-10-12T10:00:00Z","session":` + sess + `,"event":"session_open","role":` + q("lead"+bad) + "}\n" +
			`{"ts":"2026-10-12T10:00:00Z","session":` + sess + `,"event":"window_mode","mode":` + q("solo"+bad) + "}\n" +
			`{"ts":"2026-10-12T10:10:00Z","session":` + sess + `,"event":"lane_open"}` + "\n" +
			`{"ts":"2026-10-12T10:20:00Z","session":` + sess + `,"event":"context","used_pct":40}` + "\n" +
			`{"ts":"2026-10-12T10:30:00Z","session":` + sess + `,"event":"intervention","kind":` + q("nudge"+bad) + "}\n" +
			`{"ts":"2026-10-12T11:00:00Z","session":` + sess + `,"event":"lane_close","outcome":"merged"}` + "\n" +
			"not json\n"
		events, unparsed := ParseLog("run"+bad+".jsonl", []byte(log))
		rep := Compare(events, unparsed)
		b, err := json.Marshal(rep)
		if err != nil {
			t.Fatal(err)
		}
		if strings.ContainsAny(string(b), runes) {
			t.Errorf("%q: a raw rune reaches the report: %s", bad, b)
		}
		if len(rep.Modes) != 1 || rep.Modes[0].Mode != "solo?" || rep.Modes[0].LanesLanded != 1 || rep.Leader != "solo?" {
			t.Fatalf("%q: the mode keeps its figures, masked: %+v leader %q", bad, rep.Modes, rep.Leader)
		}
		if s := rep.Modes[0].Sessions; len(s) != 1 || s[0].Session != "s1?" || s[0].Role != "lead?" || s[0].LanesOpened != 1 {
			t.Errorf("%q: the session keeps its share, masked: %+v", bad, s)
		}
		if c := rep.Context; len(c) != 1 || c[0].Session != "s1?" || c[0].Role != "lead?" || c[0].Events != 1 {
			t.Errorf("%q: the session's context, masked: %+v", bad, c)
		}
		if rep.Evidence.InterventionsByKind["nudge?"] != 1 {
			t.Errorf("%q: the intervention's kind, masked: %+v", bad, rep.Evidence.InterventionsByKind)
		}
		if len(rep.Unparsed) != 1 || rep.Unparsed[0].File != "run?.jsonl" {
			t.Errorf("%q: the unparsed line names its log, masked: %+v", bad, rep.Unparsed)
		}
	}
}
