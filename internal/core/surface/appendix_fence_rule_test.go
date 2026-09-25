package surface

import (
	"errors"
	"testing"
)

// markerLines reads fences by mdrecord's rule. A backtick line inside a tilde
// fence is content, so the marker after that fence is live and the chapter
// splits; a marker inside a tilde fence is still refused.
func TestMarkerLinesReadFencesByTheCommonMarkRule(t *testing.T) {
	live := "# C\n\n~~~\n```\n~~~\n\n" + AppendixBegin + "\n" + AppendixEnd + "\n"
	if _, err := SplitChapter(live); err != nil {
		t.Fatalf("a marker after a tilde fence holding a backtick line was refused: %v", err)
	}
	fenced := "# C\n\n~~~\n" + AppendixBegin + "\n~~~\n\n" + AppendixBegin + "\n" + AppendixEnd + "\n"
	if _, err := SplitChapter(fenced); !errors.Is(err, ErrMarkerInFence) {
		t.Fatalf("a marker inside a tilde fence: err = %v, want %v", err, ErrMarkerInFence)
	}
}

// A sub-verb path written inside a fence is an invocation, whatever the fence is
// spelled with: codeRegions paired backtick runs only, so a tilde-fenced example
// named no claim and a stale sub-verb in one passed the drift check.
func TestProseShapeClaimsReadATildeFenceAsCode(t *testing.T) {
	prose := "# Capture\n\n~~~\ncapture list\n~~~\n"
	got := ProseShapeClaims(prose, []string{"abcd capture"}, fixtureTree())
	if len(got) != 1 || got[0].Spelling != "capture list" || got[0].Line != 4 {
		t.Fatalf("claims = %+v, want the fenced `capture list` on line 4", got)
	}
}
