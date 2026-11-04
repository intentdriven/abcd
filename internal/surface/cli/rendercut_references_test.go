package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/lint"
	"github.com/intentdriven/abcd/internal/core/release"
)

// The cut reports each forward-looking marker whose records have all reached
// the tree, for removal (iss-2611031731385043). The JSON carries the list; the
// terminal render must name each too, or the operator who reads the report
// never learns which markers to remove.
func TestRenderCutNamesEveryStaleMarker(t *testing.T) {
	cut := release.Cut{BaseTag: "v0.16.0", Ready: true, NextTag: "v0.16.1", References: lint.ReferenceAudit{
		StaleMarkers: []lint.StaleMarker{
			{File: ".abcd/work/decisions/dcn-2611031720012399-x.md", Line: 6, IDs: []string{"dcn-2611031615094092"}},
			{File: ".abcd/work/issues/open/iss-2611031731385043-y.md", Line: 15, IDs: []string{"iss-42", "itd-73"}},
		},
	}}
	var buf bytes.Buffer
	renderCut(&buf, "abcd launch ship", cut)
	out := buf.String()
	for _, want := range []string{
		"  stale marker: .abcd/work/decisions/dcn-2611031720012399-x.md:6 — dcn-2611031615094092 now in the tree; remove the forward-looking marker",
		"  stale marker: .abcd/work/issues/open/iss-2611031731385043-y.md:15 — iss-42, itd-73 now in the tree; remove the forward-looking marker",
	} {
		if !strings.Contains(out, want+"\n") {
			t.Errorf("the render does not carry %q:\n%s", want, out)
		}
	}
}

// A stale marker's path comes out of the record tree, which is author-supplied
// text reaching a terminal, so it is sanitised like every other value there.
func TestRenderCutSanitisesAStaleMarkersPath(t *testing.T) {
	cut := release.Cut{BaseTag: "v0.16.0", References: lint.ReferenceAudit{
		StaleMarkers: []lint.StaleMarker{{File: "a\x1b[31mb.md", Line: 1, IDs: []string{"iss-42"}}},
	}}
	var buf bytes.Buffer
	renderCut(&buf, "abcd launch ship", cut)
	if strings.Contains(buf.String(), "\x1b") {
		t.Errorf("a control byte from a record path reached the render: %q", buf.String())
	}
}
