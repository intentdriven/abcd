package scanner

import (
	"strings"
	"testing"
)

// TestScanTextFindsAWordGluedToken — iss-2609290541525428. ScanText is the one
// scan behind the launch scan, the memory writer's page-name bar and every
// store-before-commit redactor, and its patterns open on a leading \b, so a
// token right behind a letter, a digit or an underscore was not a finding at
// all. The glued sweep reports it, with the byte span Redact seals.
func TestScanTextFindsAWordGluedToken(t *testing.T) {
	pat, _, akia, _ := gluedTokens()
	for _, tc := range []struct{ name, line, kind, token string }{
		{"pat behind a letter", "see x" + pat + " here", "token:github_pat", pat},
		{"pat behind an underscore", "notes_" + pat, "token:github_pat", pat},
		{"access key between two letters", "x" + akia + "y", "token:aws_access_key", akia},
		{"access key behind a digit", "v2" + akia, "token:aws_access_key", akia},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var hit *Finding
			fs := ScanText(tc.line, Identity{}, DefaultPatterns(), nil, "f")
			for i := range fs {
				if fs[i].Kind == tc.kind {
					hit = &fs[i]
				}
			}
			if hit == nil {
				t.Fatalf("ScanText did not find the glued %s in %q: %+v", tc.kind, tc.line, fs)
			}
			if want := strings.Index(tc.line, tc.token) + 1; hit.Column != want || hit.Matched != tc.token {
				t.Errorf("the finding's span is column %d %q, want column %d %q", hit.Column, hit.Matched, want, tc.token)
			}
			if out, _ := Redact(tc.line, fs); strings.Contains(out, tc.token) {
				t.Errorf("Redact left the glued token raw: %q", out)
			}
		})
	}
}

// TestScanTextGluedSweepKeepsTheDocumentationKey: the sweep applies each
// pattern's own Skip, so the documentation example key stays accepted wherever
// it is glued, and a bounded token is reported once, not twice.
func TestScanTextGluedSweepKeepsTheDocumentationKey(t *testing.T) {
	if fs := ScanText("x"+awsExample+"y notes_"+awsExample, Identity{}, DefaultPatterns(), nil, "f"); len(fs) != 0 {
		t.Errorf("the glued sweep flagged the documentation example key: %+v", fs)
	}
	pat, _, _, _ := gluedTokens()
	n := 0
	for _, f := range ScanText("a "+pat+" b", Identity{}, DefaultPatterns(), nil, "f") {
		if f.Kind == "token:github_pat" {
			n++
		}
	}
	if n != 1 {
		t.Errorf("a bounded token was reported %d times, want once", n)
	}
}

// escapedGluedLines builds the lines of iss-2609290743362554: a glued token whose own
// bytes are percent- or JSON-escaped. The raw line carries no token (the escape
// breaks it) and the decoded view carries it glued behind a word byte, where
// the bounded patterns' leading \b cannot see it. Every token and escape is
// built at runtime.
func escapedGluedLines() []struct{ name, line, kind, token string } {
	pat, _, akia, _ := gluedTokens()
	bs := string(rune(0x5c))
	patTail := pat[len("gh"+"p_"):]
	akiaTail := akia[len("AK"+"IA"):]
	return []struct{ name, line, kind, token string }{
		{"pat with its first byte percent-encoded, behind an underscore", "notes_%67" + pat[1:], "token:github_pat", "%67" + pat[1:]},
		{"pat with its prefix's last letter percent-encoded, behind an underscore", "notes_gh%70_" + patTail, "token:github_pat", "gh%70_" + patTail},
		{"pat with its first byte JSON-escaped, behind an underscore", `{"k":"notes_` + bs + "u0067" + pat[1:] + `"}`, "token:github_pat", bs + "u0067" + pat[1:]},
		{"access key with its first byte JSON-escaped, behind a letter", `{"k":"x` + bs + "u0041" + "KIA" + akiaTail + `"}`, "token:aws_access_key", bs + "u0041" + "KIA" + akiaTail},
	}
}

// TestScanTextFindsAnEscapedGluedToken — the decoded layers (percent.go) ran
// the bounded patterns alone, so a glued token spelled with escaped bytes
// survived ScanText and Redact raw. Each is found on the decoded view and
// reported at the raw bytes it sits in, which Redact seals.
func TestScanTextFindsAnEscapedGluedToken(t *testing.T) {
	for _, tc := range escapedGluedLines() {
		t.Run(tc.name, func(t *testing.T) {
			var hit *Finding
			fs := ScanText(tc.line, Identity{}, DefaultPatterns(), nil, "f")
			for i := range fs {
				if fs[i].Kind == tc.kind {
					hit = &fs[i]
				}
			}
			if hit == nil {
				t.Fatalf("ScanText did not find the escaped glued %s: %+v", tc.kind, fs)
			}
			if want := strings.Index(tc.line, tc.token) + 1; hit.Column != want || hit.Matched != tc.token {
				t.Errorf("the finding's span is column %d %q, want column %d %q", hit.Column, hit.Matched, want, tc.token)
			}
			if out, _ := Redact(tc.line, fs); strings.Contains(out, tc.token[len(tc.token)-12:]) {
				t.Errorf("Redact left the escaped glued token raw: %q", out)
			}
		})
	}
}
