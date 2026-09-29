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
