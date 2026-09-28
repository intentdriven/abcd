package repolint_test

import (
	"strings"
	"testing"
)

// slashes spells every '#' of p with sep, so a specimen names a home path in
// an escaped spelling while this file carries no home path at all.
func slashes(p, sep string) string { return strings.ReplaceAll(p, "#", sep) }

// uSolidus is the JSON unicode escape of '/', assembled so no tool reading the
// source folds the six bytes back into a '/'.
var uSolidus = `\` + "u002f"

// privacy-hygiene reads each committed line in the spellings the scanner reads
// (iss-2609261658553101): a JSON fixture, export or transcript that writes a
// third-party home after a \n escape, with the solidus or unicode escape, with
// doubled Windows separators or percent-encoded, a session URL straight after
// a \n escape, or a footer on a line of its own inside the string, carries the
// same leak as the plain spelling and is refused like it. Read raw, the escape
// letter before the value is a word or path byte, so every anchor declined it.
func TestAC_PrivacyReadsEscapedSpellings(t *testing.T) {
	const name = "zqother" // not a registry persona, so the home is a leak
	cases := []struct{ name, line string }{
		{"home after a newline escape", `{"out":"cwd\n` + slashes("#home#"+name+"#x", "/") + `"}`},
		{"solidus-escaped home", `{"cwd":"` + slashes("#home#"+name, `\/`) + `"}`},
		{"unicode-escaped home", `{"cwd":"` + slashes("#Users#"+name, uSolidus) + `"}`},
		{"doubled Windows separators", `{"cwd":"C:` + slashes("#Users#"+name, `\\`) + `"}`},
		{"percent-encoded home", "see file:" + slashes("#home#"+name, "%2F") + "\n"},
		{"session URL after a newline escape", `{"body":"done\n` + synthSessionURL("agent-host.dev", 31) + `"}`},
		{"footer on a line of its own inside a JSON string", `{"body":"Shipped.\n\n` + harnessFooter + `"}`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			res := newFixtureRepo(t).conforming().
				file("reference/export.json", c.line+"\n").
				commit().run()
			f := findingFor(res, "privacy-hygiene")
			if f == nil {
				t.Fatalf("no privacy-hygiene finding for %q", c.line)
			}
			if f.File != "reference/export.json" || f.Line != 1 {
				t.Errorf("citation = %s:%d, want reference/export.json:1", f.File, f.Line)
			}
		})
	}
}

// The decoded views add no finding of their own on a line whose escapes hide
// nothing, and they keep every exemption the plain spelling has: a persona
// home, a system root and the line waiver.
func TestAC_PrivacyEscapedSpellingsKeepTheExemptions(t *testing.T) {
	body := strings.Join([]string{
		`{"out":"line one\nline two\t%41"}`,
		`{"cwd":"` + slashes("#Users#alice#x", `\/`) + `"}`,
		`{"cwd":"C:` + slashes("#Users#Public#x", `\\`) + `"}`,
		`{"cwd":"` + slashes("#home#zqother", `\/`) + `"} abcd-lint:allow`,
	}, "\n") + "\n"
	res := newFixtureRepo(t).conforming().
		file("reference/export.json", body).
		commit().run()
	if f := findingFor(res, "privacy-hygiene"); f != nil {
		t.Fatalf("unexpected privacy-hygiene finding: %s:%d %s", f.File, f.Line, f.Message)
	}
}
