package memory

import (
	"strings"
	"testing"
)

// TestRenderCitedMatchesSanitizesCitationFields is the gh-250 detector: the
// per-citation fields printed by RenderCitedMatches are page-derived content
// from the same untrusted ingest boundary as Summary/Filename (which ARE
// sanitised), so a citation carrying an ESC/C1/bidi/zero-width rune must reach
// the terminal defanged, not raw. encoding/json escapes only C0 and a couple of
// runes, passing C1 (U+009B), bidi overrides (U+202E) and zero-width (U+200B)
// through untouched — so the render site itself must route the compacted JSON
// through termsafe.Sanitize. Attack runes are built numerically so this source
// file carries none of the invisible characters it defends against.
func TestRenderCitedMatchesSanitizesCitationFields(t *testing.T) {
	attacks := map[string]rune{
		"ESC":             0x1b,
		"C1 CSI":          0x9b,
		"RLO override":    0x202e,
		"zero-width ZWSP": 0x200b,
	}

	poisoned := "title-a"
	for _, r := range attacks {
		poisoned += string(r) + "b"
	}

	matches := []MatchedPage{{
		Filename: "topic_auth_tokens.md",
		Score:    3,
		Summary:  "safe summary",
		Citations: []AskCitation{{
			SourceClass: "knowledge",
			SourceHash:  strings.Repeat("a", 64),
			Citation: map[string]any{
				"title": poisoned,
			},
		}},
	}}

	out := RenderCitedMatches("what tokens", matches)

	for name, r := range attacks {
		if strings.ContainsRune(out, r) {
			t.Errorf("gh-250: rendered ask output carries a raw %s (U+%04X) attack rune from the citation field; it must be defanged like the sibling Summary/Filename fields\noutput: %q", name, r, out)
		}
	}
	// Legitimate content must survive — only the control/escape/bidi/zero-width
	// runes are defanged, the surrounding letters stay intact and byte-faithful.
	if !strings.Contains(out, "title") {
		t.Fatalf("gh-250: sanitising must not drop the legitimate citation content:\n%s", out)
	}
}

// TestRenderCitedMatchesMarksATruncatedCitation: the citation JSON is capped at
// the page-value cap like every other field, and a cap cuts it mid-JSON. The
// cut must be visible — an explicit marker at the end, the whole field still
// inside the cap — so a reader never takes a cut citation for the whole one.
// A citation under the cap carries no marker.
func TestRenderCitedMatchesMarksATruncatedCitation(t *testing.T) {
	render := func(title string) string {
		t.Helper()
		out := RenderCitedMatches("what tokens", []MatchedPage{{
			Filename: "topic_auth_tokens.md",
			Score:    1,
			Summary:  "summary",
			Citations: []AskCitation{{
				SourceClass: "knowledge",
				SourceHash:  strings.Repeat("a", 64),
				Citation:    map[string]any{"title": title, "type": "knowledge"},
			}},
		}})
		for _, line := range strings.Split(out, "\n") {
			if _, cj, ok := strings.Cut(line, "| citation="); ok {
				return cj
			}
		}
		t.Fatalf("no citation line in:\n%s", out)
		return ""
	}

	long := render(strings.Repeat("x", maxPageValueBytes+500))
	if !strings.HasSuffix(long, citationTruncatedMarker) {
		t.Fatalf("an over-cap citation renders cut with no marker; it ends %q", long[len(long)-40:])
	}
	if len(long) > maxPageValueBytes {
		t.Fatalf("the marked citation is %d bytes, over the %d-byte cap", len(long), maxPageValueBytes)
	}

	short := render("Token rotation")
	if strings.Contains(short, citationTruncatedMarker) {
		t.Fatalf("a citation under the cap carries the truncation marker: %q", short)
	}
	if !strings.HasSuffix(short, "}") {
		t.Fatalf("a citation under the cap must render whole: %q", short)
	}
}
