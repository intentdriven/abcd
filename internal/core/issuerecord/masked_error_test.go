package issuerecord

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// hiddenForms are the runes the validator's echo tests plant: a bidi
// override, the C1 control sequence introducer, and the combining grapheme
// joiner, which %q leaves raw because Go counts a mark as printable.
var hiddenForms = []string{"‮", "\u009b", "͏"}

// TestTheValidatorNeverEchoesAHiddenRune (iss-2610121542258412): the issue
// record validator's errors reach --json payloads and refusal reasons on
// several verbs, so a stored value carrying a control, bidi or zero-width
// rune is echoed masked: schema_version (once printed with %v), a slug and a
// severity (printed with %q), an unparsable frontmatter line, and a filename
// slug that disagrees with the record's. The error still names its class.
func TestTheValidatorNeverEchoesAHiddenRune(t *testing.T) {
	for _, bad := range hiddenForms {
		base := func() map[string]any {
			return map[string]any{"schema_version": 1, "id": "iss-1", "slug": "a-b", "severity": "minor", "category": "bug",
				"source": "agent-finding", "found_during": "x"}
		}
		cases := map[string]struct {
			err  error
			want error
		}{}
		fm := base()
		fm["schema_version"] = "1" + bad
		cases["schema_version"] = struct{ err, want error }{ValidateStrict(fm), ErrMissingRequiredField}
		fm = base()
		fm["slug"] = "a" + bad + "b"
		cases["slug"] = struct{ err, want error }{ValidateStrict(fm), ErrMalformedFrontmatter}
		fm = base()
		fm["severity"] = "minor" + bad
		cases["severity"] = struct{ err, want error }{ValidateStrict(fm), ErrMalformedFrontmatter}
		_, _, perr := Parse("---\nschema_version: 1\n not" + bad + " a line\n---\nbody\n")
		cases["frontmatter line"] = struct{ err, want error }{perr, ErrMalformedFrontmatter}
		cases["filename slug"] = struct{ err, want error }{ValidateInvariants(base(), "open", "/x/open/iss-1-a"+bad+"b.md"), ErrInvariantViolation}
		for what, c := range cases {
			if c.err == nil {
				t.Fatalf("%q in %s: refused", bad, what)
			}
			if strings.Contains(c.err.Error(), bad) {
				t.Errorf("%q in %s: the error echoes the rune raw: %q", bad, what, c.err.Error())
			}
			if !errors.Is(c.err, c.want) {
				t.Errorf("%q in %s: the error still wraps its class %v: %v", bad, what, c.want, c.err)
			}
		}
	}
}

// TestJudgeNeverEchoesAHiddenRune: the reader's refusal of a record whose
// stored schema_version carries the rune names the schema stage, masked.
func TestJudgeNeverEchoesAHiddenRune(t *testing.T) {
	for _, bad := range hiddenForms {
		dir := filepath.Join(t.TempDir(), "open")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(dir, "iss-1-a-b.md")
		rec := "---\nschema_version: \"1" + bad + "\"\nid: \"iss-1\"\nslug: \"a-b\"\n---\nbody\n"
		if err := os.WriteFile(path, []byte(rec), 0o644); err != nil {
			t.Fatal(err)
		}
		_, _, ref, claims := Judge(path, "open")
		if !claims || ref == nil || ref.Layer != LayerSchema {
			t.Fatalf("%q: the record is refused at the schema stage: %+v", bad, ref)
		}
		if strings.Contains(ref.Error(), bad) {
			t.Errorf("%q: the refusal echoes the rune raw: %q", bad, ref.Error())
		}
	}
}
