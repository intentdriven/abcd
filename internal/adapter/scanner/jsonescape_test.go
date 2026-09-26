package scanner

import (
	"strings"
	"testing"
)

// The transcript store scans raw JSONL, so every line it hands the scanner is
// a JSON document and every string in it is JSON-escaped. A spelling that
// decodes to a secret or an identity is the same leak as the decoded text
// (iss-2609261647358395, iss-2609251639263391): these tests hand the scanner
// the escaped spelling and require the finding the decoded one gets, and a
// redaction that leaves none of the value's bytes behind.

// jsonEscapeSpecimen builds the escaped line at run time, so no literal of a
// credential shape enters the history (network_test.go states the
// discipline).
func jsonEscapeToken() string { return "ghp_" + "0123456789abcdefABCDEF0123456789abcd" }

// jsonEscapeIdentity is a synthetic caller: a POSIX home, a generic login in
// the local part of none of the lines, and a real name with a non-ASCII rune
// that an ASCII-only JSON encoder writes as a \u escape.
func jsonEscapeIdentity() Identity {
	return Identity{
		HomePath:    "/Users/" + "zqjsonme",
		HomeUser:    "zqjsonme",
		GitUserName: "Zoë Quenby",
	}
}

func findingOf(fs []Finding, kind string) (Finding, bool) {
	for _, f := range fs {
		if f.Kind == kind {
			return f, true
		}
	}
	return Finding{}, false
}

func TestJSONEscapeBesideASecretOrHomeIsStillAFinding(t *testing.T) {
	tok := jsonEscapeToken()
	other := "zqjsonother"
	cases := []struct {
		name  string
		line  string
		kind  string
		gone  string // what the redacted line must no longer carry
		keeps string // what the redacted line must still carry
	}{
		{"token after a newline escape", `{"t":"pasted:\n` + tok + `\nend"}`, "token:github_pat", tok, "pasted:"},
		{"token after a tab escape", `{"t":"key\t` + tok + `"}`, "token:github_pat", tok, "key"},
		{"token after a doubly escaped newline", `{"t":"{\"out\":\"a\\n` + tok + `\"}"}`, "token:github_pat", tok, "out"},
		{"home after a newline escape", `{"t":"ls\n/home/` + other + `/src"}`, kindHomeOther, other, "src"},
		{"home after a tab escape", `{"t":"ls\t/home/` + other + `/src"}`, kindHomeOther, other, "src"},
		{"home before a newline escape", `{"t":"cd /home/` + other + `\nok"}`, kindHomeOther, other, "ok"},
		{"home inside escaped quotes", `{"t":"cd \"/home/` + other + `\" ok"}`, kindHomeOther, other, "ok"},
		{"real name written with a unicode escape", `{"t":"signed Zo\u00eb Quenby here"}`, kindRealName, `Zo\u00eb`, "here"},
		{"real name written with an upper-case unicode escape", `{"t":"signed Zo\u00EB Quenby here"}`, kindRealName, `Zo\u00EB`, "here"},
	}
	id := jsonEscapeIdentity()
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fs := ScanText(c.line, id, DefaultPatterns(), nil, "transcript")
			if !hasKind(fs, c.kind) {
				t.Fatalf("%s: no %s finding for %s; got %+v", c.name, c.kind, c.line, fs)
			}
			red, _ := Redact(c.line, fs)
			if strings.Contains(red, c.gone) {
				t.Errorf("%s: the value survived redaction:\n%s", c.name, red)
			}
			if !strings.Contains(red, c.keeps) {
				t.Errorf("%s: redaction took the surrounding text too:\n%s", c.name, red)
			}
		})
	}
}

// TestJSONSolidusEscapeReadsAsASeparator is iss-2609251639263391's detector:
// a home path written with escaped forward slashes — the JSON solidus escape
// PHP's json_encode and org.json write, and its \u002f spelling — raises the
// finding the unescaped path raises, for the caller's own home, another
// user's home, and a generic login standing in a home.
func TestJSONSolidusEscapeReadsAsASeparator(t *testing.T) {
	id := jsonEscapeIdentity()
	other := "zqjsonother"
	cases := []struct {
		name, line, kind, gone string
	}{
		{"own home, solidus escape", `{"p":"\/Users\/zqjsonme\/Desktop\/a.txt"}`, kindHomeSelf, "zqjsonme"},       // abcd-lint:allow
		{"own home, unicode solidus", `{"p":"\u002fUsers\u002fzqjsonme\u002fDesktop"}`, kindHomeSelf, "zqjsonme"}, // abcd-lint:allow
		{"other home, solidus escape", `{"p":"\/home\/` + other + `\/x"}`, kindHomeOther, other},
		{"other home, unicode solidus", `{"p":"\u002Fhome\u002F` + other + `\u002Fx"}`, kindHomeOther, other},
		{"other home, solidus at line end", `{"p":"\/home\/` + other + `"}`, kindHomeOther, other},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fs := ScanText(c.line, id, DefaultPatterns(), nil, "transcript")
			if !hasKind(fs, c.kind) {
				t.Fatalf("%s: no %s finding for %s; got %+v", c.name, c.kind, c.line, fs)
			}
			red, _ := Redact(c.line, fs)
			if strings.Contains(red, c.gone) {
				t.Errorf("%s: the name survived redaction:\n%s", c.name, red)
			}
		})
	}

	// A generic login is reported where it stands as an account, and the
	// solidus-escaped home root is such a position.
	generic := Identity{HomePath: "/home/" + "dev", HomeUser: "dev"}
	line := `{"p":"\/home\/dev\/project"}` // abcd-lint:allow
	if f, ok := findingOf(ScanText(line, generic, DefaultPatterns(), nil, "transcript"), kindHomeSelf); !ok {
		t.Errorf("a generic login's own home behind solidus escapes raised no %s", kindHomeSelf)
	} else if !strings.Contains(f.Matched, "dev") {
		t.Errorf("the %s span does not cover the login: %q", kindHomeSelf, f.Matched)
	}
}

// TestJSONEscapeLayerAddsNothingToPlainLines is the false-positive side: an
// ordinary transcript line full of escapes and carrying no secret and no
// identity stays clean, a Windows path whose escaped separators decode into
// control escapes on a further layer ("\\b", "\\n") invents nothing, and a
// finding the raw line already carries is reported once, not once per layer.
func TestJSONEscapeLayerAddsNothingToPlainLines(t *testing.T) {
	id := jsonEscapeIdentity()
	clean := []string{
		`{"type":"assistant","text":"Line one.\nLine two with \"quotes\" and a tab\there.\n\u2014 done"}`,
		`{"cmd":"dir C:\\\\build\\\\bin\\\\new\\\\tmp","out":"ok\r\n"}`,
		`{"re":"^\\/api\\/v1\\/items$","path":"\/srv\/app\/bin"}`,
	}
	for _, line := range clean {
		if fs := ScanText(line, id, DefaultPatterns(), nil, "transcript"); len(fs) != 0 {
			t.Errorf("a clean line raised findings: %s\n%+v", line, fs)
		}
	}
	tok := jsonEscapeToken()
	line := `{"t":"a \"` + tok + `\" b\n"}`
	n := 0
	for _, f := range ScanText(line, id, DefaultPatterns(), nil, "transcript") {
		if f.Kind == "token:github_pat" {
			n++
		}
	}
	if n != 1 {
		t.Errorf("a token the raw line already finds was reported %d times, want 1", n)
	}
}
