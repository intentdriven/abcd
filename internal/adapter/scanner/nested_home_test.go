package scanner

import (
	"strings"
	"testing"
)

// nested_home_test.go — iss-324 and iss-2608291915432717: a third party's home
// path that follows a path byte was reported by no detector. leadingBoundaryOK
// refuses a /Users or /home match whose preceding byte continues a path, which
// is right for a relative path ("docs/Users/guide.md") and wrong for an
// absolute one: a file:// URL puts a slash before the home, and a backup
// volume or a mount puts a longer root before it. home_path_other now reads
// the whole path token the match sits in, and a token that is absolute is a
// home path wherever its /Users or /home segment falls.
func TestHomePathOtherUnderANestedAbsoluteRoot(t *testing.T) {
	id := Identity{HomePath: "/Users/zq8home", HomeUser: "zq8home"}
	caught := map[string]string{
		"open file:///home/alice/notes.md":       "/home/alice",  // abcd-audit:allow
		"see file:///Users/bob/secret.txt":       "/Users/bob",   // abcd-audit:allow
		"restored /Volumes/Backup/Users/alice/x": "/Users/alice", // abcd-audit:allow
		"mounted at /mnt/data/home/alice/x":      "/home/alice",  // abcd-audit:allow
	}
	for line, want := range caught {
		f := ScanText(line, id, DefaultPatterns(), DefaultIdentitySeverities(), "f")
		got := ""
		for _, x := range f {
			if x.Kind == kindHomeOther {
				got = x.Matched
			}
		}
		if got != want {
			t.Errorf("nested third-party home in %q: home_path_other matched %q, want %q (%+v)", line, got, want, f)
		}
		if red, _ := Redact(line, f); strings.Contains(red, want) {
			t.Errorf("the nested home survived redaction: %q", red)
		}
	}
	// A RELATIVE path whose segment happens to be named Users or home, and a
	// web URL whose path begins with /home, name nobody's home directory.
	for _, line := range []string{
		"see docs/Users/guide.md",
		"src/components/home/Header.tsx",
		"https://docs.example.com/home/getting-started",
		"https://fossil-scm.org/home/doc/trunk/www/index.wiki",
	} {
		if f := ScanText(line, id, DefaultPatterns(), DefaultIdentitySeverities(), "f"); hasKind(f, kindHomeOther) {
			t.Errorf("a path that is not a home was reported: %q %+v", line, f)
		}
	}
}
