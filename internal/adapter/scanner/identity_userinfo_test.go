package scanner

import (
	"strings"
	"testing"
)

// A URL's userinfo is the one part of a URL that names an account rather than
// a resource, so the URL-span suppression of the username matcher stops at it
// (iss-2609251549447970): a clone URL or a proxy setting quoted in a
// transcript carries the caller's login there, with or without a password.
func TestLocalUsernameReportedInAURLUserinfo(t *testing.T) {
	pats := DefaultPatterns()
	sev := DefaultIdentitySeverities()
	for _, tc := range []struct {
		user, line string
	}{
		{"zq8home", "clone https://zq8home@git.example.com/acme/tool.git"},
		{"zq8home", "proxy=http://zq8home:s3cret@proxy.example.com:3128"},
		{"zq8home", "fetch ftp://ZQ8HOME@files.example.com/pub"},
		{"dev", "clone https://dev@git.example.com/acme/tool.git"},
		{"dev", "proxy=http://dev:s3cret@proxy.example.com:3128"},
	} {
		got := ScanText(tc.line, Identity{HomeUser: tc.user}, pats, sev, "f")
		if !hasKind(got, kindLocalUser) {
			t.Errorf("login %q in the userinfo of %q was not flagged: %+v", tc.user, tc.line, got)
			continue
		}
		if red, _ := Redact(tc.line, got); strings.Contains(strings.ToLower(red), "//"+tc.user) {
			t.Errorf("the userinfo login survived redaction: %q -> %q", tc.line, red)
		}
	}
	// The rest of a URL stays suppressed: a path segment is a resource name,
	// and the forge's own service account in an ssh remote is not the caller.
	for _, tc := range []struct {
		user, line string
	}{
		{"zq8home", "see https://docs.example.com/zq8home/guide"},
		{"dev", "see https://docs.example.com/dev/guide"},
		{"git", "remote ssh://git@github.com/acme/tool.git"},
	} {
		if got := ScanText(tc.line, Identity{HomeUser: tc.user}, pats, sev, "f"); hasKind(got, kindLocalUser) {
			t.Errorf("login %q outside a userinfo in %q was flagged: %+v", tc.user, tc.line, got)
		}
	}
}
