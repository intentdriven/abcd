package runner

import (
	"errors"
	"strings"
	"testing"
)

// TestClaudeRefusesAReportedModelCarryingAHiddenRune (iss-2610121542385455):
// the model the harness's init event reports reaches the route record, and
// through it `implement record --json`, so one carrying a bidi override, a C1
// control, a zero-width or a tag character is refused as unparsable, never an
// answer, and the refusal does not echo it.
func TestClaudeRefusesAReportedModelCarryingAHiddenRune(t *testing.T) {
	for _, bad := range []string{"‮", "\u009b", "​", "\U000E0041"} {
		stream := `{"type":"system","subtype":"init","model":"claude-opus` + bad + `","session_id":"s1"}` + "\n" +
			`{"type":"result","subtype":"success","is_error":false,"result":"done","session_id":"s1"}` + "\n"
		ans, err := parseClaude([]byte(stream))
		var fl *Failure
		if !errors.As(err, &fl) || fl.Reason != ReasonUnparsable {
			t.Fatalf("%q: a model carrying the rune is refused as unparsable: %+v %v", bad, ans, err)
		}
		if ans.Model != "" || strings.Contains(fl.Error(), bad) {
			t.Fatalf("%q: the model travelled on: %q %q", bad, ans.Model, fl.Error())
		}
	}
	clean := `{"type":"system","subtype":"init","model":"claude-opus-5-5","session_id":"s1"}` + "\n" +
		`{"type":"result","subtype":"success","is_error":false,"result":"done","session_id":"s1"}` + "\n"
	if ans, err := parseClaude([]byte(clean)); err != nil || ans.Model != "claude-opus-5-5" {
		t.Fatalf("a plain model id is an answer: %+v %v", ans, err)
	}
}
