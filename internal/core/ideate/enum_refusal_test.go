package ideate

import (
	"strings"
	"testing"
)

// enumLeak is what a composer pasted into a closed-set field: a home path and a
// distinctive marker, the shape the recorder redacts field by field in prose.
const enumLeak = "zzleak-7f3a /Users/zzotherperson/notes"

// shortLeak is the same under the record-id length cap, so the id refusal that
// quotes rather than the one that counts bytes is the one reached.
const shortLeak = "zzleak-7f3a/Users/zzotherperson"

// TestEnumRefusalsDoNotEchoTheValue — iss-2609090951295881. The recorder
// redacts every free-text field and its stage-two refusal names finding kinds
// only, because a refusal that quoted the span it refused would publish it into
// a terminal, a log and the session transcript. The closed-set refusals did the
// opposite: the verdict, the leg kind, a claim status, a grill relation, a kill
// outcome, and a grill hit's record id were interpolated into the error after
// only the terminal-escape cleaner, which strips control sequences and redacts
// nothing. Each is now described (its length, or that it is empty), never
// quoted, and the refusal still says which field and which item it was.
func TestEnumRefusalsDoNotEchoTheValue(t *testing.T) {
	root := seedRepo(t)
	legs := func(p map[string]any, i int) map[string]any { return p["legs"].([]any)[i].(map[string]any) }
	cases := map[string]struct {
		mutate func(p map[string]any)
		names  string
	}{
		"verdict": {func(p map[string]any) { p["verdict"] = enumLeak }, "verdict"},
		"leg kind": {func(p map[string]any) { legs(p, 0)["kind"] = enumLeak }, "leg 1"},
		"claim status": {func(p map[string]any) {
			legs(p, 0)["claims"].([]any)[0].(map[string]any)["status"] = enumLeak
		}, "claim 1"},
		"grill record id": {func(p map[string]any) {
			legs(p, 1)["hits"].([]any)[0].(map[string]any)["record"] = shortLeak
		}, "grill hit 1"},
		"grill relation": {func(p map[string]any) {
			legs(p, 1)["hits"].([]any)[0].(map[string]any)["relation"] = enumLeak
		}, "grill hit 1"},
		"kill outcome": {func(p map[string]any) {
			legs(p, 2)["kill_attempts"].([]any)[0].(map[string]any)["outcome"] = enumLeak
		}, "kill attempt 1"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			p := validPayload()
			tc.mutate(p)
			_, err := Record(root, "the-ideate-gate", encode(t, p), at)
			if err == nil {
				t.Fatal("the out-of-set value was accepted")
			}
			msg := err.Error()
			for _, part := range []string{"zzleak-7f3a", "zzotherperson", "/Users/"} {
				if strings.Contains(msg, part) {
					t.Errorf("the refusal echoes the refused value (%q): %s", part, msg)
				}
			}
			if !strings.Contains(msg, tc.names) {
				t.Errorf("the refusal no longer says which field it was (want %q): %s", tc.names, msg)
			}
		})
	}
	assertNothingWritten(t, root)
}
