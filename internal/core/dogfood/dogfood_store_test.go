package dogfood

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

// TestASecondRepositoryGetsItsFirstBuild: the check that refuses a store
// another user can write took a store whose top level existed and whose
// repository level did not as not this account's alone, so once any
// repository had been built, the next one declared was refused forever and
// never built. A missing level is made by the first build and refuses
// nothing.
func TestASecondRepositoryGetsItsFirstBuild(t *testing.T) {
	f := newFixture(t, "github.com/intentdriven/abcd")
	f.declare(t)
	if err := os.MkdirAll(abcdhome.Path(f.home, "dogfood"), 0o700); err != nil {
		t.Fatal(err)
	}
	p := Resolve(f.home, f.checkout)
	if !p.StartBuild || strings.Contains(notes(p), "not this account's alone") {
		t.Fatalf("Resolve with only the store's top level present = %+v, want its first build asked for", p)
	}
}

// TestPruneKeepsEveryBuildAVerifiedTipNames: each declaration keeps its own
// verified tip, so four declarations of one repository at four tips named
// four builds, and pruning to the three newest evicted one another clone
// still ran, whose next check rebuilt it and evicted another, for good. A
// build a record names is never pruned; keepBuilds caps only the builds no
// record names.
func TestPruneKeepsEveryBuildAVerifiedTipNames(t *testing.T) {
	store := t.TempDir()
	sha := func(c byte) string { return strings.Repeat(string(c), 40) }
	base := time.Now().Add(-time.Hour)
	// Oldest first: four named builds, then three no record names, then the
	// build just made.
	order := []string{sha('a'), sha('b'), sha('c'), sha('d'), sha('5'), sha('6'), sha('7'), sha('f')}
	for i, b := range order {
		dir := filepath.Join(store, b)
		if err := os.Mkdir(dir, 0o700); err != nil {
			t.Fatal(err)
		}
		at := base.Add(time.Duration(i) * time.Minute)
		if err := os.Chtimes(dir, at, at); err != nil {
			t.Fatal(err)
		}
	}
	for i, b := range order[:4] {
		if err := writeTip(store, tipRecord{Commit: b, URL: "https://forge.example.invalid/clone" + string(rune('0'+i)) + ".git", Branch: "main"}); err != nil {
			t.Fatal(err)
		}
	}
	// A fifth declaration's tip failed to build: its log and marker, and no
	// directory, stay for its retry window.
	failed := sha('e')
	if err := writeTip(store, tipRecord{Commit: failed, URL: "https://forge.example.invalid/clone4.git", Branch: "main"}); err != nil {
		t.Fatal(err)
	}
	for _, suffix := range []string{logSuffix, failedSuffix} {
		if err := os.WriteFile(filepath.Join(store, failed+suffix), []byte("x\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	prune(store, sha('f'))
	for _, suffix := range []string{logSuffix, failedSuffix} {
		if _, err := os.Lstat(filepath.Join(store, failed+suffix)); err != nil {
			t.Errorf("the %s of a named tip that failed to build is pruned: %v", suffix, err)
		}
	}
	for _, b := range append(order[:4], sha('f'), sha('7'), sha('6')) {
		if _, err := os.Lstat(filepath.Join(store, b)); err != nil {
			t.Errorf("build %s is pruned: %v", b[:7], err)
		}
	}
	if _, err := os.Lstat(filepath.Join(store, sha('5'))); !os.IsNotExist(err) {
		t.Errorf("the oldest build no record names stays: %v", err)
	}
}
