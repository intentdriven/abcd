package lifeboat

import (
	"strings"
	"testing"
)

// TestReviewLifeboatRefusalsDoNotEchoTheValue — iss-2609290033521472. The
// delegated review payload is host-composed, and its closed-set fields — the
// mode, the prompt_version and the verdict — were quoted into the refusal with
// a bare %q, so a token or a home path pasted into one reached the terminal and
// the transcript verbatim. Each is described now, never quoted.
func TestReviewLifeboatRefusalsDoNotEchoTheValue(t *testing.T) {
	const leak = "zzleak-7f3a /Users/zzotherperson/notes" // abcd-lint:allow — a planted home path the refusal must not echo
	good := string(reviewPayloadJSON("SHIP", `{"id":"fnd-ok","finding":"fine","evidence":["coverage.json"]}`))
	cases := map[string]string{
		"mode":           strings.Replace(good, `"mode":"delegated"`, `"mode":"`+leak+`"`, 1),
		"prompt_version": strings.Replace(good, `"prompt_version":"0.1.0"`, `"prompt_version":"`+leak+`"`, 1),
		"verdict":        strings.Replace(good, `"verdict":"SHIP"`, `"verdict":"`+leak+`"`, 1),
	}
	for field, payload := range cases {
		t.Run(field, func(t *testing.T) {
			if payload == good {
				t.Fatalf("fixture: the %s field was not replaced", field)
			}
			dir := reviewFixture(t, "abc", &Summary{Grounded: 7, Blank: 3})
			_, err := ReviewLifeboat(dir, realSourceDir(t), []byte(payload))
			if err == nil {
				t.Fatalf("an out-of-set %s was accepted", field)
			}
			for _, part := range []string{"zzleak-7f3a", "zzotherperson"} {
				if strings.Contains(err.Error(), part) {
					t.Errorf("the refusal echoes the refused %s: %v", field, err)
				}
			}
			if !strings.Contains(err.Error(), field) {
				t.Errorf("the refusal no longer names the field %s: %v", field, err)
			}
		})
	}
}
