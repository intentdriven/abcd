package capture

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

// The filename and the frontmatter slug are ONE value written twice. Capture
// derives it once (deriveSlug, which is where the cap lives),
// normalises it once, and hands that single string to both writers:
// reservePath builds issID+"-"+slug+".md" and commitCapture stores the same
// variable as fm["slug"]. Nothing downstream re-truncates, so the filename can
// never be a shortened form of the frontmatter value — the cap is applied
// BEFORE the fork, not after it. Exact equality is therefore the whole rule,
// and a disagreement is drift a human introduced by renaming one side.
//
// validateInvariants already pins the id half of that agreement. Without the
// slug half a record can carry a stale foreign handle in its name while its
// frontmatter says something else, and every reader that locates a record by
// its filename and every reader that trusts the field disagree in silence.
func TestFilenameSlugMustMatchFrontmatterSlug(t *testing.T) {
	// A 60-char slug sat exactly on deriveSlug's former cap, and the ledger holds
	// slugs well past recordid.MaxSlugLen, minted before the cap
	// (iss-2610100626320367), so both sides of the boundary must pass whole
	// rather than being compared as prefixes: the reader never caps.
	at60 := strings.Repeat("ab-", 20)[:59] + "c" // 60 chars, kebab-case
	past60 := at60 + "-and-then-some-more-words"

	for _, c := range []struct {
		name     string
		filename string
		slug     string
		wantErr  bool
	}{
		{"agrees", "iss-1-broken-thing.md", "broken-thing", false},
		{"disagrees", "iss-1-broken-thing.md", "some-other-handle", true},
		{"stale foreign handle in the name", "iss-1-oracle-gates.md", "spc-12-oracle-gates", true},
		{"at the derive cap, carried whole", "iss-1-" + at60 + ".md", at60, false},
		{"past the derive cap, carried whole", "iss-1-" + past60 + ".md", past60, false},
		// The store never writes a truncated filename against a longer field, so a
		// prefix is drift, not an accepted shortening. Pinning it stops a future
		// "be lenient about length" patch from reopening the hole.
		{"filename truncated to the cap", "iss-1-" + at60 + ".md", past60, true},
		{"filename is a bare prefix", "iss-1-broken.md", "broken-thing", true},
	} {
		t.Run(c.name, func(t *testing.T) {
			fm := map[string]any{
				"schema_version": 1,
				"id":             "iss-1",
				"slug":           c.slug,
				"severity":       "minor",
				"category":       "bug",
				"source":         "agent-finding",
				"found_during":   "review",
			}
			err := validateInvariants(fm, StateOpen, filepath.Join("open", c.filename))
			if c.wantErr {
				if err == nil {
					t.Fatalf("filename %q with slug %q accepted; the name and the field name "+
						"different records and no gate says so", c.filename, c.slug)
				}
				if !errors.Is(err, ErrInvariantViolation) {
					t.Errorf("got %v, want an ErrInvariantViolation", err)
				}
				// The message must name BOTH values: a reader who only learns that
				// "the slug disagrees" still has to open the file to find out which
				// half to fix.
				if !strings.Contains(err.Error(), c.slug) {
					t.Errorf("message %q does not name the frontmatter slug %q", err, c.slug)
				}
				fnSlug := strings.TrimSuffix(strings.TrimPrefix(c.filename, "iss-1-"), ".md")
				if !strings.Contains(err.Error(), fnSlug) {
					t.Errorf("message %q does not name the filename slug %q", err, fnSlug)
				}
				return
			}
			if err != nil {
				t.Fatalf("filename %q with slug %q refused: %v", c.filename, c.slug, err)
			}
		})
	}
}
