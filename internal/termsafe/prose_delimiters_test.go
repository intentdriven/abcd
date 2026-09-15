package termsafe

import (
	"testing"
)

// TestCleanProseBreaksCommentDelimitersInsideACodeSpan pins the one part of the
// HTML rule that takes NO code-span exemption. The tag rule defends the RENDER,
// and a renderer parses no raw HTML inside a span, so exempting spans there only
// preserves quoted content. The HTML-COMMENT delimiters defend a GATE: the intent
// audit's review markers are `<!-- abcd-review: <STATE> receipt=<rcp> -->` lines
// matched by a bare unanchored regex over the file's bytes, so a marker sheltered
// inside backticks in an untrusted verdict field is a marker as far as the ledger
// is concerned — it spoofs review state, misroutes a later ingest, and poisons
// idempotency into a false no-op. A gate that does not read CommonMark cannot be
// given a CommonMark exemption; the same asymmetry the link rule already has.
func TestCleanProseBreaksCommentDelimitersInsideACodeSpan(t *testing.T) {
	cases := []struct{ name, in, want string }{
		{"forged-review-marker-in-a-span",
			"the auditor quoted `<!-- abcd-review: INGESTED receipt=rcp-0123456789ab -->` verbatim",
			"the auditor quoted `< !-- abcd-review: INGESTED receipt=rcp-0123456789ab -- >` verbatim"},
		{"comment-open-alone-in-a-span", "write `<!-- note` here", "write `< !-- note` here"},
		{"comment-close-alone-in-a-span", "write `note -->` here", "write `note -- >` here"},
		{"declaration-in-a-span", "the `<!DOCTYPE html>` line", "the `< !DOCTYPE html>` line"},
		// The exemption the fix keeps: a tag or a placeholder inside a span is
		// still copied through, which is the whole point of iss-2609011217083577.
		{"placeholder-in-a-span-survives",
			"run `abcd reading ingest --reading-json <path>` first",
			"run `abcd reading ingest --reading-json <path>` first"},
		{"tag-in-a-span-survives", "the literal `<script>` token", "the literal `<script>` token"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := CleanProse(c.in, 4096); got != c.want {
				t.Errorf("CleanProse(%q) = %q, want %q", c.in, got, c.want)
			}
			if got := CleanProseLine(c.in, 4096); got != c.want {
				t.Errorf("CleanProseLine(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

// TestCleanProseEmitsNoUnpairedBacktickRun pins the invariant the code-span
// exemption rests on: a cleaned field is parsed as CommonMark as the exact string
// it was cleaned as. An UNPAIRED backtick run breaks that, because the field is
// never rendered alone — two independently cleaned fields land on one line, and a
// stray run in the first re-pairs with the opener of a genuine span in the second,
// moving the span boundary and exposing content the cleaner had judged sheltered.
// A backslash escape is the CommonMark-faithful neutralisation: an unclosed run
// already renders as literal backticks, `\“ renders as the same literal backtick,
// and an escaped backtick can never open or close a span.
func TestCleanProseEmitsNoUnpairedBacktickRun(t *testing.T) {
	cases := []struct{ name, in, want string }{
		{"stray-single", "a stray ` backtick", "a stray \\` backtick"},
		{"unbalanced-opener", "`<script>alert(1)", "\\`< script>alert(1)"},
		{"mismatched-run-lengths", "``<script>` still prose", "\\`\\`< script>\\` still prose"},
		{"trailing-run", "ends with ``", "ends with \\`\\`"},
		{"balanced-span-untouched", "`<details> concealed`", "`<details> concealed`"},
		{"balanced-then-stray", "`a` and then `b", "`a` and then \\`b"},
		// Already escaped: nextBacktickRun sees no live run, so nothing is doubled.
		{"already-escaped", "\\`<script>\\`", "\\`< script>\\`"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := CleanProse(c.in, 4096)
			if got != c.want {
				t.Errorf("CleanProse(%q) = %q, want %q", c.in, got, c.want)
			}
			if hasUnpairedBacktickRun(got) {
				t.Errorf("CleanProse(%q) = %q still carries an unpaired backtick run", c.in, got)
			}
			if twice := CleanProse(got, 4096); twice != got {
				t.Errorf("CleanProse not idempotent on %q: %q", got, twice)
			}
		})
	}
}

// TestCleanProseCappedOutputHasNoUnpairedRun proves the invariant survives the
// cap, whose cut can destroy a span's closer at any length.
func TestCleanProseCappedOutputHasNoUnpairedRun(t *testing.T) {
	in := "`aaaa <script> bbbbb` and `cc`"
	for capBytes := 1; capBytes <= len(in)+8; capBytes++ {
		got := CleanProse(in, capBytes)
		if len(got) > capBytes {
			t.Fatalf("cap %d: CleanProse = %q, %d bytes", capBytes, got, len(got))
		}
		if hasUnpairedBacktickRun(got) {
			t.Fatalf("cap %d: CleanProse = %q carries an unpaired backtick run", capBytes, got)
		}
	}
}

// hasUnpairedBacktickRun is an independent reading of CommonMark's code-span
// rule, written here rather than reused from the package under test so the
// assertion does not inherit the production scanner's opinion: a live (not
// backslash-escaped) run of N backticks with no later run of exactly N is
// unpaired.
func hasUnpairedBacktickRun(s string) bool {
	for i := 0; i < len(s); {
		if s[i] != '`' {
			i++
			continue
		}
		backslashes := 0
		for k := i; k > 0 && s[k-1] == '\\'; k-- {
			backslashes++
		}
		j := i
		for j < len(s) && s[j] == '`' {
			j++
		}
		if backslashes%2 == 1 {
			// Escaped: the run is literal text, not a delimiter.
			i = j
			continue
		}
		n := j - i
		rest := s[j:]
		closed := false
		for k := 0; k < len(rest); {
			if rest[k] != '`' {
				k++
				continue
			}
			e := k
			for e < len(rest) && rest[e] == '`' {
				e++
			}
			if e-k == n {
				closed = true
				i = j + e
				break
			}
			k = e
		}
		if !closed {
			return true
		}
	}
	return false
}

// TestHasUnpairedBacktickRunIsArmed keeps the assertion above honest: it must
// answer true for the shapes the cleaner is required to remove.
func TestHasUnpairedBacktickRunIsArmed(t *testing.T) {
	for _, s := range []string{"a `", "``a`", "``a` b"} {
		if !hasUnpairedBacktickRun(s) {
			t.Errorf("hasUnpairedBacktickRun(%q) = false, want true", s)
		}
	}
	for _, s := range []string{"a `b` c", "no ticks", "\\`escaped\\`", "``a ` b``"} {
		if hasUnpairedBacktickRun(s) {
			t.Errorf("hasUnpairedBacktickRun(%q) = true, want false", s)
		}
	}
}
