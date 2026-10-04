package site

// No block of text widens the page, whatever token it carries.
//
// A record's title and its source path are the record's own words, and some of
// them hold a long run with nowhere to break: a timestamp-id ADR's path is about
// fifty characters to its first hyphen, and a title can quote a home-relative
// path. Given nowhere to break, the token sets the width of its block and the
// page scrolls sideways on a phone (iss-2610040732240935). The net is one rule on
// `body`: `overflow-wrap:break-word` is inherited by every block, breaks a token
// only where the line would otherwise overflow, and — unlike `anywhere` — leaves
// min-content sizing alone, so tables, grids and flex rows lay out as before.

import "testing"

// TestTheStylesheetBreaksAnOverflowingTokenInEveryBlock holds the net in abcd's
// own stylesheet and in the copy setup seeds a managed repository with, at the
// top level so it applies at every width.
func TestTheStylesheetBreaksAnOverflowingTokenInEveryBlock(t *testing.T) {
	for name, src := range bothStylesheets(t) {
		if got := cascadeOf(src).top["body"]["overflow-wrap"].value; got != "break-word" {
			t.Errorf("%s gives body overflow-wrap:%q, want break-word: a title or path with nowhere to break widens the page",
				name, got)
		}
	}
}
