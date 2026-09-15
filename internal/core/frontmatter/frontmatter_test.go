package frontmatter

import (
	"strings"
	"testing"
)

func TestFieldsReadsLeadingBlock(t *testing.T) {
	lines := []string{
		"---",
		"id: itd-9",
		"slug: my-thing",
		"spec_id: null",
		"---",
		"# Title",
		"key: not-frontmatter",
	}
	fields := Fields(lines)
	if got := fields["id"]; got.Value != "itd-9" || got.Line != 2 {
		t.Fatalf("id = %+v, want {itd-9 2}", got)
	}
	if got := fields["slug"]; got.Value != "my-thing" || got.Line != 3 {
		t.Fatalf("slug = %+v, want {my-thing 3}", got)
	}
	if _, ok := fields["key"]; ok {
		t.Fatal("a key past the closing --- must not be read")
	}
}

func TestFieldsFirstKeyWins(t *testing.T) {
	lines := []string{"---", "kind: standalone", "kind: discipline", "---"}
	if got := Fields(lines)["kind"]; got.Value != "standalone" || got.Line != 2 {
		t.Fatalf("kind = %+v, want first-key-wins {standalone 2}", got)
	}
}

func TestFieldsNoFrontmatter(t *testing.T) {
	if got := Fields([]string{"# Title", "id: itd-9"}); len(got) != 0 {
		t.Fatalf("no leading --- must yield no fields, got %+v", got)
	}
	if got := Fields(nil); len(got) != 0 {
		t.Fatalf("empty input must yield no fields, got %+v", got)
	}
}

func TestFieldsIgnoresNested(t *testing.T) {
	lines := []string{"---", "top: v", "  nested: v", "- item", "---"}
	fields := Fields(lines)
	if _, ok := fields["nested"]; ok {
		t.Fatal("indented key must be ignored (top-level only)")
	}
	if got := fields["top"]; got.Value != "v" {
		t.Fatalf("top = %+v, want v", got)
	}
}

func TestFieldsTrimsCarriageReturn(t *testing.T) {
	lines := []string{"---\r", "id: itd-9\r", "---\r"}
	if got := Fields(lines)["id"]; got.Value != "itd-9" {
		t.Fatalf("CRLF id = %+v, want itd-9", got)
	}
}

func TestIsNull(t *testing.T) {
	// The four YAML nulls (YAML 1.1 !!null / YAML 1.2 core schema §10.2.1.1:
	// null | Null | NULL | ~) plus the empty scalar all read as null. The
	// uppercase spellings were previously missed (iss #290) even though the
	// repo's own YAML scalar parser already holds the full set.
	for _, v := range []string{"", "null", "Null", "NULL", "~"} {
		if !IsNull(v) {
			t.Errorf("IsNull(%q) = false, want true", v)
		}
	}
	// Only the exact YAML null spellings count: near-misses and record handles
	// must stay non-null, so a fix can never widen this to a case-fold or a
	// substring match. Fields does not strip quotes, so a *quoted* null reaches
	// IsNull with its quote characters intact and must stay a string — an
	// explicit quoted scalar is not a YAML null.
	for _, v := range []string{"itd-9", "spc-1", "standalone", "None", "nil", "NUL", "nullish", `"null"`, `"NULL"`, "'~'"} {
		if IsNull(v) {
			t.Errorf("IsNull(%q) = true, want false", v)
		}
	}
}

// TestFieldsToleratesDelimiterTrailingSpace (iss-69 C5) proves a delimiter line
// with trailing whitespace ("--- ") is still recognised, at both ends. Otherwise
// a trailing-space closing delimiter is not seen as the close and body lines
// after it leak in as frontmatter fields.
func TestFieldsToleratesDelimiterTrailingSpace(t *testing.T) {
	lines := []string{
		"--- ", // opening delimiter with a trailing space
		"id: itd-9",
		"---\t",         // closing delimiter with a trailing tab
		"status: draft", // body — must NOT be read as a field
	}
	fields := Fields(lines)
	if got := fields["id"]; got.Value != "itd-9" {
		t.Fatalf("id must parse under a trailing-space opening delimiter, got %+v", got)
	}
	if _, ok := fields["status"]; ok {
		t.Fatal("a body line after a trailing-whitespace closing delimiter must not leak in as a field")
	}
}

// TestFieldsUnclosedBlockYieldsNoFields (B42) proves that a leading `---` with no
// closing `---` is treated as no frontmatter: the block is the region between the
// FIRST TWO delimiters, so without a close there is no block, and column-0 body
// prose (e.g. a thematic-break document, or a "----" fat-fingered close) must not
// be harvested as top-level fields.
func TestFieldsUnclosedBlockYieldsNoFields(t *testing.T) {
	lines := []string{"---", "id: x", "body prose", "status: retired"}
	if got := Fields(lines); len(got) != 0 {
		t.Fatalf("unclosed block must yield no fields, got %+v", got)
	}
	// A "----" typo does not close the block (it does not trim to "---"), so the
	// same rule applies.
	typo := []string{"---", "id: x", "----", "status: retired"}
	if got := Fields(typo); len(got) != 0 {
		t.Fatalf(`a "----" close is not a delimiter; block stays unclosed, got %+v`, got)
	}
}

// TestFieldsReadsPastBOM pins that a UTF-8 byte-order mark ahead of the opening
// `---` does not hide the frontmatter. The BOM is not Unicode White_Space, so an
// untrimmed mark makes the first line compare unequal to "---" and the whole
// block reads as body — silently emptying every frontmatter-keyed gate.
func TestFieldsReadsPastBOM(t *testing.T) {
	lines := []string{"\ufeff---", "id: itd-7", "---", "# body"}
	fields := Fields(lines)
	if got := fields["id"]; got.Value != "itd-7" {
		t.Fatalf("id must parse under a leading BOM, got %+v (fields=%v)", got, fields)
	}
	if got := fields["id"]; got.Line != 2 {
		t.Errorf("id line under a leading BOM = %d, want 2", got.Line)
	}
}

func TestTrimBOM(t *testing.T) {
	if got := TrimBOM("\ufeff---"); got != "---" {
		t.Errorf("TrimBOM stripped wrong: %q", got)
	}
	if got := TrimBOM("---"); got != "---" {
		t.Errorf("TrimBOM must leave a BOM-free line unchanged, got %q", got)
	}
	// Only a leading mark is stripped; an interior U+FEFF is left alone.
	if got := TrimBOM("a\ufeffb"); got != "a\ufeffb" {
		t.Errorf("TrimBOM must not touch an interior mark, got %q", got)
	}
}

// GitHub #357: `key : value` (whitespace before the colon) must read as the key
// it is, the way the strict ledger parser (internal/core/capture) reads it \u2014
// otherwise the lenient scanner sees NO key and a gate armed on that key
// (record_schema's filename\u2194id blocker) is silently disarmed by a stray space.
func TestFieldsToleratesSpaceBeforeColon(t *testing.T) {
	lines := []string{"---", "id : itd-999", "kind:\tstandalone", "---"}
	fields := Fields(lines)
	if got := fields["id"]; got.Value != "itd-999" {
		t.Fatalf("id = %+v, want the value read past a space-before-colon {itd-999 2}", got)
	}
	if got := fields["kind"]; got.Value != "standalone" {
		t.Fatalf("kind = %+v, want a tab-before-colon key read too", got)
	}
}

// GitHub #357: Fields keeps the FIRST occurrence of a duplicated key and drops
// the rest silently. Duplicates surfaces them so a gate can refuse what its
// strict consumer refuses.
func TestDuplicates(t *testing.T) {
	lines := []string{"---", "id: adr-12", "impact: fix", "impact: banana", "---", "# body"}
	dups := Duplicates(lines)
	if len(dups) != 1 || dups[0].Key != "impact" || dups[0].Line != 4 {
		t.Fatalf("Duplicates = %+v, want one impact duplicate at line 4", dups)
	}
	// A clean block has none.
	if got := Duplicates([]string{"---", "id: adr-12", "impact: fix", "---"}); len(got) != 0 {
		t.Fatalf("a block with no duplicates must yield none, got %+v", got)
	}
	// No frontmatter -> no duplicates (never harvest body lines).
	if got := Duplicates([]string{"# Title", "id: a", "id: b"}); len(got) != 0 {
		t.Fatalf("no leading --- must yield no duplicates, got %+v", got)
	}
}

// GitHub #338: IsDelimiter is the ONE delimiter rule every reader of a record's
// bytes shares. It is tolerant of trailing whitespace and of the end-of-line
// bytes a keep-ends splitter leaves on; it is intolerant of everything else, so
// a fat-fingered `----`, a delimiter carrying content, an indented `---`, and a
// U+FEFF-prefixed line all stay non-delimiters. BOM tolerance is the caller's,
// applied to line 0 only (iss-2608270926036966).
func TestIsDelimiter(t *testing.T) {
	for _, ln := range []string{
		"---",
		"--- ",
		"---\t",
		"---  \t ",
		"---\n",
		"---\r\n",
		"--- \n",
		"--- \r\n",
	} {
		if !IsDelimiter(ln) {
			t.Fatalf("IsDelimiter(%q) = false, want true", ln)
		}
	}
	for _, ln := range []string{
		"",
		"----",
		"--",
		"--- yaml",
		"---x",
		"  ---",
		"\t---",
		" --- ",
		"# ---",
		// U+FEFF is a BOM only at byte 0 of a file, which is a property of the
		// POSITION, not of the character. The predicate sees a bare line and
		// cannot know the position, so callers trim it at line 0 themselves.
		"\ufeff---",
		"\ufeff--- \n",
	} {
		if IsDelimiter(ln) {
			t.Fatalf("IsDelimiter(%q) = true, want false", ln)
		}
	}
}

// iss-2608270926036966: U+FEFF is a byte-order mark ONLY at the very start of a
// file. Anywhere else it is ZERO WIDTH NO-BREAK SPACE, an ordinary character, and
// a line spelled "\ufeff---" is not a delimiter to any other reader of the file.
//
// Trimming it on every line made Fields close the block early at such a line
// while intent's writer (setFrontmatterFields, which trims only " \t\r") read on
// to the NEXT bare `---` — the body's thematic break — and inserted keys there.
// The record stayed lint-green, `abcd intent plan` reported success, and the
// value it wrote was invisible on reload: two readers, one file, different block
// boundaries. The tolerance belongs to line 0's position, not to the character.
func TestMidFileZWNBSPIsNotADelimiter(t *testing.T) {
	lines := []string{
		"---",
		"id: itd-9",
		"\ufeff---", // ZWNBSP, not a BOM: an ordinary body-ish line
		"kind: intent",
		"impact: fix",
		"impact: banana",
		"---",
		"# Body",
	}

	fields := Fields(lines)
	// The block does NOT end at the ZWNBSP line, so every key up to the real
	// closing `---` is in scope — exactly what the writer sees.
	for _, key := range []string{"id", "kind", "impact"} {
		if _, ok := fields[key]; !ok {
			t.Fatalf("Fields stopped early at a mid-file ZWNBSP line: %q missing from %v", key, fields)
		}
	}
	if got := fields["kind"].Value; got != "intent" {
		t.Fatalf("kind = %q, want %q", got, "intent")
	}

	// Duplicates must scan the same span, or a gate armed on the second `impact:`
	// is silently disarmed by one invisible character.
	dups := Duplicates(lines)
	if len(dups) != 1 || dups[0].Key != "impact" || dups[0].Line != 6 {
		t.Fatalf("Duplicates = %+v, want one impact duplicate at line 6", dups)
	}

	// And the delimiter predicate itself says so.
	if IsDelimiter("\ufeff---") {
		t.Fatal(`IsDelimiter("\ufeff---") = true; U+FEFF is only a BOM at file position 0`)
	}
}

// A BOM ahead of the OPENING delimiter is still tolerated — that is the position
// where U+FEFF really is a byte-order mark. Callers trim it at line 0 themselves;
// Fields and Duplicates do it before their scan.
func TestBOMBeforeOpeningDelimiter(t *testing.T) {
	lines := []string{"\ufeff---", "id: itd-9", "---", "# Body"}
	if got := Fields(lines); got["id"].Value != "itd-9" {
		t.Fatalf("a BOM before the opening delimiter must not hide the block: %v", got)
	}
	dupLines := []string{"\ufeff---", "id: a", "id: b", "---"}
	if got := Duplicates(dupLines); len(got) != 1 {
		t.Fatalf("Duplicates = %+v, want one duplicate past a leading BOM", got)
	}
}

// TestSplitIsLossless: the two halves must concatenate back to the input byte
// for byte, or a writer that splices an edited body onto the head returns a file
// different from the one it read in ways nobody asked for.
func TestSplitIsLossless(t *testing.T) {
	for name, text := range map[string]string{
		"record":              "---\nid: iss-1\n---\n\nbody\n",
		"crlf":                "---\r\nid: iss-1\r\n---\r\n\r\nbody\r\n",
		"no trailing newline": "---\nid: iss-1\n---\nbody",
		"padded delimiter":    "---\nid: iss-1\n--- \n\nbody\n",
		"bom":                 "\ufeff---\nid: iss-1\n---\n\nbody\n",
		"no frontmatter":      "# a plain document\n\nbody\n",
		"unterminated":        "---\nid: iss-1\n\nbody\n",
		"empty":               "",
	} {
		head, body := Split(text)
		if head+body != text {
			t.Fatalf("%s: Split is lossy: head+body = %q, want %q", name, head+body, text)
		}
	}
}

// TestSplitHoldsBackOnlyTheFrontmatter: what the body must NOT contain is the
// block, and what it must contain is everything after it. A text with no
// frontmatter, and a block nothing closes, are both all body — holding back
// prose no reader treats as frontmatter would hide it from the caller that asked
// for the body.
func TestSplitHoldsBackOnlyTheFrontmatter(t *testing.T) {
	head, body := Split("---\nid: iss-1\n# Grounds\n---\n\n## Grounds\n\n- pursued: a conjecture\n")
	if !strings.Contains(head, "\n# Grounds\n") {
		t.Fatalf("the frontmatter comment is not in the head: %q", head)
	}
	if strings.Contains(body, "\n# Grounds\n") {
		t.Fatalf("the frontmatter comment leaked into the body: %q", body)
	}
	if !strings.Contains(body, "\n## Grounds\n") {
		t.Fatalf("the body section is missing: %q", body)
	}

	// An indented `---` is not a close, exactly as the strict ledger parser reads
	// it, so the block runs to the next un-indented one.
	head, body = Split("---\nfoo: |\n  ---\n---\nbody\n")
	if !strings.Contains(head, "\n  ---\n") || body != "body\n" {
		t.Fatalf("an indented --- closed the block early: head=%q body=%q", head, body)
	}

	for name, text := range map[string]string{
		"no frontmatter": "# a plain document\n\nbody\n",
		"unterminated":   "---\nid: iss-1\n\nbody\n",
	} {
		if head, body := Split(text); head != "" || body != text {
			t.Fatalf("%s: Split = (%q, %q), want the whole text as body", name, head, body)
		}
	}
}
