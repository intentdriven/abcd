package tools

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

// ownCopy records a copy of gitleaks at the bin folder as abcd's, at version.
func ownCopy(t *testing.T, r *rig, version string) []byte {
	t.Helper()
	body := []byte("#!/bin/sh\necho " + version + "\n")
	if err := os.MkdirAll(filepath.Dir(r.dest()), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(r.dest(), body, 0o755); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(body)
	if err := writeRecord(r.home, Record{Tools: map[string]Installed{"gitleaks": {Path: r.dest(),
		SHA256: hex.EncodeToString(sum[:]), Version: version}}}); err != nil {
		t.Fatal(err)
	}
	return body
}

// TestAnOwnedCopyOutsideTheRangeIsReplaced: abcd's own copy is used only while
// its version lies in the range abcd works with; outside it, abcd's pinned
// copy is offered over it, and on a yes replaces it.
func TestAnOwnedCopyOutsideTheRangeIsReplaced(t *testing.T) {
	r := newRig(t)
	ownCopy(t, r, "8.10.0")
	in := r.installer()
	if p := in.Plan("gitleaks"); p.Action != InstallOwn || p.Owned == nil || p.Owned.Compatible {
		t.Fatalf("plan = %+v, want abcd's copy offered over its incompatible one", p)
	}
	withHome(t, r.home, func(string) (string, error) { return "", errNotFound })
	if got, err := Resolve("gitleaks"); err == nil {
		t.Fatalf("Resolve ran abcd's incompatible copy %q", got)
	}
	res := in.Install("gitleaks", TranscriptScan, yes)
	if !res.Ready || !res.Installed {
		t.Fatalf("result = %+v", res)
	}
	rec, err := readRecord(r.home)
	if err != nil || rec.Tools["gitleaks"].Version != "8.30.1" {
		t.Fatalf("record = %+v, %v", rec, err)
	}
}

// TestAnOwnedCopyAtAnotherPinIsAnUpgrade: a compatible copy of abcd's at a
// version other than the pin is still run, and the pin is offered as an
// upgrade; a no keeps the copy that works.
func TestAnOwnedCopyAtAnotherPinIsAnUpgrade(t *testing.T) {
	r := newRig(t)
	body := ownCopy(t, r, "8.24.3")
	in := r.installer()
	if p := in.Plan("gitleaks"); p.Action != InstallOwn || p.Owned == nil || !p.Owned.Compatible {
		t.Fatalf("plan = %+v, want the pin offered as an upgrade", p)
	}
	withHome(t, r.home, func(string) (string, error) { return "", errNotFound })
	if got, _ := Resolve("gitleaks"); got != r.dest() {
		t.Fatalf("Resolve = %q, want abcd's compatible copy", got)
	}
	res := in.Install("gitleaks", TranscriptScan, no)
	if !res.Ready || res.Ran {
		t.Fatalf("a no to the upgrade left no copy ready: %+v", res)
	}
	if b, _ := os.ReadFile(r.dest()); string(b) != string(body) {
		t.Fatal("a no changed the copy")
	}
}

// TestAForeignFileAtAnOwnedPathIsNeverReplaced: a record whose file is no
// longer abcd's bytes never licenses writing over it.
func TestAForeignFileAtAnOwnedPathIsNeverReplaced(t *testing.T) {
	r := newRig(t)
	ownCopy(t, r, "8.10.0")
	if err := os.WriteFile(r.dest(), []byte("theirs now"), 0o755); err != nil {
		t.Fatal(err)
	}
	res := r.installer().Install("gitleaks", TranscriptScan, yes)
	if res.Installed || r.hits != 0 {
		t.Fatalf("a changed file was replaced: %+v", res)
	}
	if b, _ := os.ReadFile(r.dest()); string(b) != "theirs now" {
		t.Fatal("the file was changed")
	}
}
