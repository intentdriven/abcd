package dogfood

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
)

// TestTwoDeclarationsOfOneRepositoryBothSettle: the store is keyed on the
// repository's root commit, and it held one record of the last check, for one
// remote and branch. Two clones of one repository declared with different
// branches overwrote each other's record, so neither ever resolved and every
// hook in either asked the remote again. Each declaration now keeps its own
// record, and both settle on their builds.
func TestTwoDeclarationsOfOneRepositoryBothSettle(t *testing.T) {
	f := newFixture(t, "github.com/intentdriven/abcd")
	git(t, f.upstream, "push", "-q", f.remote, "main:release")
	parent := realDir(t)
	git(t, parent, "clone", "-q", f.remote, "second")
	second := filepath.Join(parent, "second")
	f.list(t, f.checkout+"\t"+f.url+"\tmain", second+"\t"+f.url+"\trelease")

	a := buildFor(t, f.home, f.checkout)
	b := buildFor(t, f.home, second)
	for i := 0; i < 2; i++ {
		for _, tc := range []struct{ cwd, bin string }{{f.checkout, a}, {second, b}} {
			p := Resolve(f.home, tc.cwd)
			if p.Binary != tc.bin || p.StartBuild || len(p.Notes) != 0 {
				t.Fatalf("round %d, %s: Resolve = %+v, want its build named and silence", i, tc.cwd, p)
			}
		}
	}
}

// TestAStoreNotTheAccountsAloneIsRefusedOutLoud: a build store another user
// can write was passed over silently, so Resolve said a build was starting in
// the background, and the build it asked for refused at once into a log no
// one reads, on every hook, for good. Resolve now refuses the store out loud
// and asks for no build.
func TestAStoreNotTheAccountsAloneIsRefusedOutLoud(t *testing.T) {
	f := newFixture(t, "github.com/intentdriven/abcd")
	f.declare(t)
	buildFor(t, f.home, f.checkout)
	store := abcdhome.Path(f.home, "dogfood")
	if err := os.Chmod(store, 0o770); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(store, 0o700) })
	for i := 0; i < 2; i++ {
		p := Resolve(f.home, f.checkout)
		if p.Binary != "" || p.StartBuild || !strings.Contains(notes(p), "not this account's alone") {
			t.Fatalf("hook %d: Resolve = %+v, want the store refused out loud and no build asked for", i, p)
		}
	}
}
