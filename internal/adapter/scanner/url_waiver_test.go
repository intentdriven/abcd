package scanner

import "testing"

// url_waiver_test.go — iss-2608292005445725: inside a URL span the leading
// half of the home anchor was waived unconditionally, so under HOME=/root (a
// container, CI running as root) a URL whose path merely CONTAINS a /root
// segment was rewritten and hard-failed as the caller's home. The waiver now
// holds only where the home is the URL's path root — the first '/' after the
// authority — and a match buried deeper in the path is judged by the ordinary
// anchor.
func TestURLHomeWaiverHoldsOnlyAtThePathRoot(t *testing.T) {
	const home = "/root"
	id := Identity{HomePath: home, HomeUser: "root"}
	kept := []string{
		"git@github.com:acme/root/tool.git",
		"https://pkg.go.dev/example.com/mod/root",
		"https://docs.example.com/guide/root/index.html",
	}
	for _, line := range kept {
		if got := SweepCallerHome(line, home); got != line {
			t.Errorf("a /root segment deep in a URL path was swept: %q -> %q", line, got)
		}
		if f := ScanText(line, id, DefaultPatterns(), DefaultIdentitySeverities(), "f"); hasKind(f, kindHomeSelf) {
			t.Errorf("a /root segment deep in a URL path is a home_path_self finding: %q %+v", line, f)
		}
	}
	swept := map[string]string{
		"https://ci.example.com/root/build.log": "https://ci.example.com~/build.log",
		"file:///root/notes.md":                 "file://~/notes.md",
		"git@host.example.com:/root/repo.git":   "git@host.example.com:~/repo.git",
	}
	for line, want := range swept {
		if got := SweepCallerHome(line, home); got != want {
			t.Errorf("a home at the URL's path root was not swept: %q -> %q, want %q", line, got, want)
		}
		if f := ScanText(line, id, DefaultPatterns(), DefaultIdentitySeverities(), "f"); !hasKind(f, kindHomeSelf) {
			t.Errorf("a home at the URL's path root is not a home_path_self finding: %q %+v", line, f)
		}
	}
}
