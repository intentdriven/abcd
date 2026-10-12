package dogfood

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
)

// TestABuildUnderAGroupWritableUmaskIsReady: go build writes the binary at
// 0777 less the umask, and the store runs only a binary no one else can
// write. Under umask 002 (a user-private-group account's default) the build
// landed group-writable, was refused as not ready, and every later build
// failed to move into the directory the refused one occupied, so the dogfood
// build never ran. The build is the store's own, so it is made 0700, and a
// refused directory for the same commit is replaced.
func TestABuildUnderAGroupWritableUmaskIsReady(t *testing.T) {
	old := syscall.Umask(0o002)
	t.Cleanup(func() { syscall.Umask(old) })

	f := newFixture(t, "github.com/intentdriven/abcd")
	f.declare(t)
	p := Resolve(f.home, f.checkout)
	if p.Checkout == nil {
		t.Fatalf("premise: a listed checkout resolves a source commit: %+v", p)
	}
	// A directory an earlier build left for this commit, its binary refused
	// as group-writable.
	tip := git(t, f.upstream, "rev-parse", "HEAD")
	stale := abcdhome.Path(f.home, "dogfood", p.Checkout.RootSHA, tip)
	if err := os.MkdirAll(stale, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stale, BinaryName), []byte("#!/bin/sh\n"), 0o775); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(filepath.Join(stale, BinaryName), 0o775); err != nil {
		t.Fatal(err)
	}

	bin, err := Build(f.home, *p.Checkout)
	if err != nil {
		t.Fatalf("a build over a refused directory for the same commit succeeds: %v", err)
	}
	fi, err := os.Stat(bin)
	if err != nil || fi.Mode().Perm() != 0o700 {
		t.Fatalf("the built binary is the store's alone (0700): %v %v", fi.Mode(), err)
	}
	if got := Resolve(f.home, f.checkout); got.Binary != bin || got.StartBuild {
		t.Fatalf("Resolve names the build under umask 002: %+v", got)
	}
}

// TestPruneRemovesLogsOfCommitsThatNeverBuilt: a commit whose build failed
// leaves a log and a failed marker and no build directory, and prune only
// removed those files beside a directory, so every failed tip's pair stayed
// in the store for good. A pair for any commit other than the one kept goes
// once a build succeeds.
func TestPruneRemovesLogsOfCommitsThatNeverBuilt(t *testing.T) {
	f := newFixture(t, "github.com/intentdriven/abcd")
	f.declare(t)
	p := Resolve(f.home, f.checkout)
	if p.Checkout == nil {
		t.Fatalf("premise: a listed checkout resolves a source commit: %+v", p)
	}
	store := abcdhome.Path(f.home, "dogfood", p.Checkout.RootSHA)
	if err := os.MkdirAll(store, 0o700); err != nil {
		t.Fatal(err)
	}
	gone := strings.Repeat("e", 40)
	for _, suffix := range []string{logSuffix, failedSuffix} {
		if err := os.WriteFile(filepath.Join(store, gone+suffix), []byte("x\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := Build(f.home, *p.Checkout); err != nil {
		t.Fatal(err)
	}
	for _, suffix := range []string{logSuffix, failedSuffix} {
		if _, err := os.Lstat(filepath.Join(store, gone+suffix)); !os.IsNotExist(err) {
			t.Fatalf("the %s of a commit that never built is pruned: %v", suffix, err)
		}
	}
	if _, err := os.Lstat(filepath.Join(store, git(t, f.upstream, "rev-parse", "HEAD")+logSuffix)); err != nil {
		t.Fatalf("the kept build's log stays: %v", err)
	}
}
