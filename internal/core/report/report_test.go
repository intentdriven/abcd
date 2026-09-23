package report

import (
	"errors"
	"strings"
	"testing"
)

// filled returns the issued template with its blanks filled, the way a
// reporter hands one in. Each test edits the result to break one field.
func filled(t *testing.T) string {
	t.Helper()
	s := string(Template("v0.9.0"))
	s = strings.Replace(s, `title: ""`, `title: "capture refuses a slug with a digit first"`, 1)
	s = strings.Replace(s, `surface: ""`, `surface: "abcd capture"`, 1)
	s = strings.Replace(s, `remedy: ""`, `remedy: "accept a leading digit"`, 1)
	s = strings.Replace(s, "evidence: []", "evidence:\n  - iss-2609221656361680\n  - https://example.com/run/1", 1)
	s = strings.Replace(s, prosePlaceholder, "Running capture with a slug of 9lives was refused.", 1)
	return s
}

// TestTemplateAsIssuedIsRefused: the skeleton is the shape, not a report. An
// unfilled template must never file, so its blank title is refused by name.
func TestTemplateAsIssuedIsRefused(t *testing.T) {
	_, err := Parse(Template("v0.9.0"))
	var fe *FieldError
	if !errors.As(err, &fe) || fe.Field != "title" {
		t.Fatalf("Parse(template) = %v, want a refusal naming title", err)
	}
	if !errors.Is(err, ErrRefused) {
		t.Fatalf("refusal %v is not classed ErrRefused", err)
	}
}

// TestFilledTemplateParses: criterion 1 — a filled template validates, block
// and prose both read.
func TestFilledTemplateParses(t *testing.T) {
	r, err := Parse([]byte(filled(t)))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if r.Kind != "defect" || r.Severity != "minor" || r.Category != "bug" {
		t.Errorf("enums = %q/%q/%q", r.Kind, r.Severity, r.Category)
	}
	if r.Title != "capture refuses a slug with a digit first" || r.Surface != "abcd capture" || r.AbcdVersion != "v0.9.0" {
		t.Errorf("scalars = %+v", r)
	}
	if len(r.Evidence) != 2 || r.Evidence[0] != "iss-2609221656361680" {
		t.Errorf("evidence = %q", r.Evidence)
	}
	if r.Prose != "Running capture with a slug of 9lives was refused." {
		t.Errorf("prose = %q", r.Prose)
	}
}

// TestMalformedBlocksAreRefusedNamingTheField: each defect names its field.
func TestMalformedBlocksAreRefusedNamingTheField(t *testing.T) {
	cases := []struct {
		name, from, to, field string
	}{
		{"unknown kind", "\nkind: defect\n", "\nkind: wish\n", "kind"},
		{"unknown severity", "\nseverity: minor\n", "\nseverity: urgent\n", "severity"},
		{"unknown category", "\ncategory: bug\n", "\ncategory: gripe\n", "category"},
		{"missing surface", `surface: "abcd capture"` + "\n", "", "surface"},
		{"unknown key", "\nkind: defect\n", "\nkind: defect\nflavour: sour\n", "flavour"},
		{"duplicate key", "\nkind: defect\n", "\nkind: defect\nkind: enhancement\n", "kind"},
		{"envelope key from the reporter", "\nkind: defect\n", "\nkind: defect\nsender_name: x\n", "sender_name"},
		{"absolute path in evidence", "  - iss-2609221656361680", "  - /etc/passwd", "evidence"},
		{"home path in surface", `surface: "abcd capture"`, `surface: "~/.abcd/inbox"`, "surface"},
		{"traversal in evidence", "  - iss-2609221656361680", "  - ../../secret", "evidence"},
		{"windows path in remedy", `remedy: "accept a leading digit"`, `remedy: "edit C:\\x"`, "remedy"},
		{"empty prose", "Running capture with a slug of 9lives was refused.", "", "prose"},
		{"list for a scalar", `title: "capture refuses a slug with a digit first"`, "title:\n  - a", "title"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			in := filled(t)
			if !strings.Contains(in, c.from) {
				t.Fatalf("fixture lacks %q", c.from)
			}
			in = strings.Replace(in, c.from, c.to, 1)
			_, err := Parse([]byte(in))
			var fe *FieldError
			if !errors.As(err, &fe) || fe.Field != c.field {
				t.Fatalf("Parse = %v, want a refusal naming %q", err, c.field)
			}
		})
	}
}

// TestUntrustedBytesAreBounded: the report arrives from another repository.
func TestUntrustedBytesAreBounded(t *testing.T) {
	big := filled(t) + strings.Repeat("x", MaxBytes)
	if _, err := Parse([]byte(big)); !errors.Is(err, ErrRefused) {
		t.Errorf("oversize report: err = %v, want a refusal", err)
	}
	esc := strings.Replace(filled(t), "was refused.", "was refused.\x1b[31m", 1)
	if _, err := Parse([]byte(esc)); !errors.Is(err, ErrRefused) {
		t.Errorf("control byte: err = %v, want a refusal", err)
	}
	long := strings.Replace(filled(t), `title: "capture`, `title: "`+strings.Repeat("y", 300), 1)
	if _, err := Parse([]byte(long)); !errors.Is(err, ErrRefused) {
		t.Errorf("long title: err = %v, want a refusal", err)
	}
}

// TestUnknownVersionIsAVersionError: criterion 6 at filing — the version is
// named, and a later template's unknown keys do not mask it.
func TestUnknownVersionIsAVersionError(t *testing.T) {
	in := strings.Replace(filled(t), "schema_version: 1", "schema_version: 2\nmood: hopeful", 1)
	_, err := Parse([]byte(in))
	var ve *VersionError
	if !errors.As(err, &ve) || ve.Version != "2" {
		t.Fatalf("Parse = %v, want a VersionError naming 2", err)
	}
	if !strings.Contains(err.Error(), "2") {
		t.Errorf("message %q does not name the version", err)
	}
}

// TestSerializeRoundTrips: what abcd stores reads back as what it parsed.
func TestSerializeRoundTrips(t *testing.T) {
	r, err := Parse([]byte(filled(t)))
	if err != nil {
		t.Fatal(err)
	}
	r.ReceivedAt = "2026-09-23T10:00:00Z"
	r.SenderKey = strings.Repeat("a", 40)
	r.SenderName = "sender-repo"
	back, err := parseFiled(serialize(r))
	if err != nil {
		t.Fatalf("parseFiled(serialize): %v", err)
	}
	if back.Title != r.Title || back.SenderName != r.SenderName || back.Prose != r.Prose || len(back.Evidence) != 2 {
		t.Errorf("round trip = %+v, want %+v", back, r)
	}
}

// TestPathRefusalLeavesOrdinaryWordsAlone: the path refusal is about
// locations, so a slash between words, a tilde before a number and a URL
// all pass.
func TestPathRefusalLeavesOrdinaryWordsAlone(t *testing.T) {
	for _, v := range []string{"defect / enhancement", "took ~5 minutes", "https://example.com/a/b", "and/or", "internal/core/report", "abcd hook session-start"} {
		if err := refusePath("title", v); err != nil {
			t.Errorf("refusePath(%q) = %v, want it accepted", v, err)
		}
	}
	for _, v := range []string{"/etc/passwd", "see (/tmp/x)", "~/notes", "~alice/notes", `\\\\server\\share`, "C:/Users", "file:///x", "../up", `a\\..\\b`, "x/../y"} {
		if err := refusePath("title", v); err == nil {
			t.Errorf("refusePath(%q) accepted a location", v)
		}
	}
}

// TestHiddenRunesAreRefusedNamingTheField: a bidi override or a zero-width rune
// makes a value display differently from its bytes, so a report carrying one in
// any field is refused, naming that field. The runes are written numerically so
// this file carries none of them.
func TestHiddenRunesAreRefusedNamingTheField(t *testing.T) {
	hidden := []rune{0x202A, 0x202E, 0x2066, 0x2069, 0x200B, 0x200E, 0x200F, 0xFEFF, 0x061C}
	fields := []struct{ field, from string }{
		{"title", "capture refuses a slug"},
		{"surface", `surface: "abcd capture"`},
		{"remedy", "accept a leading digit"},
		{"evidence", "https://example.com/run/1"},
		{"prose", "Running capture with"},
	}
	for _, r := range hidden {
		for _, f := range fields {
			in := filled(t)
			if !strings.Contains(in, f.from) {
				t.Fatalf("fixture lacks %q", f.from)
			}
			at := len(f.from) - 3
			in = strings.Replace(in, f.from, f.from[:at]+string(r)+f.from[at:], 1)
			_, err := Parse([]byte(in))
			var fe *FieldError
			if !errors.As(err, &fe) || fe.Field != f.field {
				t.Errorf("U+%04X in %s: Parse = %v, want a refusal naming %q", r, f.field, err, f.field)
			}
		}
	}
	// A leading byte-order mark is an encoding marker, not text, and still reads.
	if _, err := Parse([]byte(string(rune(0xFEFF)) + filled(t))); err != nil {
		t.Errorf("a leading BOM: Parse = %v, want it accepted", err)
	}
	// A C1 control is a control byte like ESC: U+009B acts as ESC[ on an 8-bit terminal.
	c1 := strings.Replace(filled(t), "was refused.", "was refused."+string(rune(0x9B))+"31m", 1)
	if _, err := Parse([]byte(c1)); !errors.Is(err, ErrRefused) {
		t.Errorf("C1 control: Parse = %v, want a refusal", err)
	}
}

// TestPathRefusalCatchesTheCheapForms: the environment's home, a UNC path in
// its forward-slash spelling and a path after a colon are locations too, while
// a URL's scheme separator and a clock time are not.
func TestPathRefusalCatchesTheCheapForms(t *testing.T) {
	for _, v := range []string{"$HOME/notes", "${HOME}/notes", "see $HOME", "//host/share", "path:/etc/passwd", "found at:/tmp/x", "%USERPROFILE%\\x"} {
		if err := refusePath("title", v); err == nil {
			t.Errorf("refusePath(%q) accepted a location", v)
		}
	}
	for _, v := range []string{"https://example.com/a", "at 10:30 today", "iss-1: done", "HOMEWORK", "$HOMER"} {
		if err := refusePath("title", v); err != nil {
			t.Errorf("refusePath(%q) = %v, want it accepted", v, err)
		}
	}
}
