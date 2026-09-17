package scanner

import (
	"strings"
	"testing"
)

// iss-2609100505145554, the Stage-1 half. A system root under /Users is not a
// home root, so an entry reached directly beneath it is a shared-folder entry,
// not somebody's home directory.
//
// This matters in the redactor and not only in the lint gate: home_path_other is
// an identity kind, so BlockingResidual refuses a write on it whatever its
// severity. The product creates a directory under the shared root and names it in
// its own comments, tests and install docs, and flagging that made the product's
// own committed text refuse the write.
//
// The traversal shapes still flag: "/Users/Shared/../<user>" leaves the shared
// root, so the name after it is a home segment again.
func TestSharedRootSubtreeIsNotAHomePath(t *testing.T) {
	user := strings.Join([]string{"j", "doe"}, "")
	cases := []struct {
		name string
		line string
		want bool // want a home_path_other finding
	}{
		{"product data dir", "data at /Users/Shared/abcd-data/x", false},
		{"plain file", "report at /Users/Shared/report.txt", false},
		{"deep subtree", "cache at /Users/Shared/abcd/cache/v2/blob", false},
		{"guest subtree", "state at /Users/Guest/abcd/state", false},
		{"name reached directly", "keys at /Users/Shared/" + user + "/keys.txt", false},
		{"bare system directory", "installs to /Users/Shared", false},
		{"prose ellipsis", "flags /Users/Shared/... in files", false},
		// Traversal out of the shared root: still a home path.
		{"parent marker then name", "keys at /Users/Shared/../" + user + "/keys.txt", true},
		{"relative marker then name", "keys at /Users/Shared/./" + user + "/keys.txt", true},
		{"doubled separator then name", "keys at /Users/Shared//" + user + "/keys.txt", true},
		// Not a system root at all.
		{"segment beginning with a system name", "notes at /Users/sharedstuff/notes.md", true},
		// An ordinary home path is untouched by any of this.
		{"ordinary home path", "notes at /Users/" + user + "/notes.md", true},
	}
	sc, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := false
			for _, f := range sc.ScanText(c.line, "probe.md") {
				if f.Kind == kindHomeOther {
					got = true
				}
			}
			if got != c.want {
				t.Fatalf("home_path_other = %v, want %v for %q", got, c.want, c.line)
			}
		})
	}
}
