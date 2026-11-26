package gitutil_test

import (
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gitutil"
)

// TestARefListingReadsTheRefsOwnLinesAlone: of a listing, only the lines
// whose ref is exactly the one asked for are read, however many lines end in
// its name and wherever they stand; a line as long as the line bound, which
// may have been cut, names nothing.
func TestARefListingReadsTheRefsOwnLinesAlone(t *testing.T) {
	sha, other := "816b5286bb3bfd133378aa7f880b0affc8903823", "0123456789abcdef0123456789abcdef01234567"
	l := gitutil.RefListing{Ref: "refs/heads/main"}
	l.Read(other + "\trefs/heads/a/refs/heads/main\n" +
		"ref: refs/heads/x\trefs/heads/b/refs/heads/main\n" +
		other + "\trefs/heads/main^{}\n" +
		sha + "\trefs/heads/main\r\n")
	if l.Tips != 1 || l.Tip != sha || l.Symrefs != 0 {
		t.Fatalf("listing = %+v; want main's own line alone, at %s", l, sha)
	}
	h := gitutil.RefListing{Ref: "HEAD"}
	h.Read("ref: refs/heads/evil\trefs/heads/HEAD/HEAD\nref: refs/heads/main\tHEAD\n" + other + "\trefs/heads/x/HEAD\n" + sha + "\tHEAD\n")
	if h.Symrefs != 1 || h.Target != "refs/heads/main" || h.Tips != 1 || h.Tip != sha {
		t.Fatalf("listing = %+v; want HEAD's two lines alone", h)
	}
	long := gitutil.RefListing{Ref: "HEAD"}
	long.Line(sha + "\tHEAD" + strings.Repeat(" ", gitutil.RefLineMax))
	long.Line((sha + "\tHEAD" + strings.Repeat(" ", gitutil.RefLineMax))[:gitutil.RefLineMax])
	if long.Tips != 0 {
		t.Fatalf("a line at the line bound was read: %+v", long)
	}
}
