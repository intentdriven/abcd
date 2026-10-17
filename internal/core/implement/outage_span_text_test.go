package implement

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestAnOutageSpansLogTextIsMasked (iss-2610121542263820): an outage span's
// services, kinds and retried work come from the run log the sessions append
// to, and reach `implement record --json` and `implement report --json`, so a
// hand-edited or repository-shipped log line carrying U+202E or U+009B is
// read with the rune masked, the span kept.
func TestAnOutageSpansLogTextIsMasked(t *testing.T) {
	for _, bad := range []string{"‮", "\u009b"} {
		q := func(s string) string { b, _ := json.Marshal(s); return string(b) }
		log := `{"ts":"2026-10-12T10:00:00Z","session":"s1","event":"outage_start","service":` + q("network"+bad) +
			`,"kind":` + q("tool"+bad) + `,"what":` + q("git push"+bad+" lane-1") + "}\n" +
			`{"ts":"2026-10-12T10:03:00Z","session":"s1","event":"outage_end","started_at":"2026-10-12T10:00:00Z","minutes":3,` +
			`"retried":[` + q("git fetch"+bad) + `]}` + "\n"
		events, unparsed := ParseLog("2026-10-12.jsonl", []byte(log))
		rep := Compare(events, unparsed)
		if len(rep.Outages) != 1 {
			t.Fatalf("%q: the log's one outage is a span: %+v %+v", bad, rep.Outages, unparsed)
		}
		b, err := json.Marshal(rep.Outages)
		if err != nil {
			t.Fatal(err)
		}
		if strings.ContainsAny(string(b), "‮\u009b") {
			t.Fatalf("%q: a raw rune reaches the span: %s", bad, b)
		}
		s := rep.Outages[0]
		if strings.Join(s.Services, ",") != "network?" || strings.Join(s.Kinds, ",") != "tool?" || strings.Join(s.Retried, ",") != "git fetch?" {
			t.Fatalf("%q: the span keeps its text, masked: %+v", bad, s)
		}
	}
}
