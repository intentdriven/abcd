package capture

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/changelog"
	"github.com/intentdriven/abcd/internal/gittest"
)

// What capture WRITES, the derivation must be able to READ (iss-2608241612087533).
//
// This is the test whose absence let two separate bugs ship in one change, and
// both were invisible to a reader-only suite:
//
//  1. `shipped_in` was missing from validate.go's knownFields allow-list, so
//     every `capture resolve --shipped-in` invocation was refused outright with
//     `unknown property "shipped_in"` and wrote nothing. The flag, the core
//     field, the surface baseline entry and the docs line all existed for a code
//     path that could never execute.
//  2. With that fixed, the writer passed a plain string, which yamlScalar
//     double-quotes, so the record landed as `shipped_in: "v0.1.0"` — while the
//     derivation reads the RAW scalar and rejects the quotes. The field parsed as
//     malformed and excluded nothing.
//
// Each half passed its own side's tests. Only crossing the boundary catches
// either, which is the whole point: a claim with two authorities has to be
// checked against both.
func TestResolveWritesAShippedInTheDerivationHonours(t *testing.T) {
	r := gittest.NewRepo(t)
	r.Write(".abcd/work/issues/open/iss-1-old.md",
		"---\nschema_version: 1\nid: \"iss-1\"\nslug: \"old\"\nseverity: \"minor\"\n"+
			"category: \"bug\"\nsource: \"user-observation\"\nfound_during: \"t\"\n---\n\nan old fix\n")
	r.Commit("an open record")
	r.Git("tag", "v0.1.0")

	res, err := Resolve(ResolveRequest{
		Grounds:  testGrounds,
		RepoRoot: r.Root(), ID: "iss-1", Resolution: "released long ago",
		Impact: "fix", ShippedIn: "v0.1.0",
	})
	if err != nil {
		t.Fatalf("Resolve with --shipped-in was refused; the flag cannot execute: %v", err)
	}

	// The value must land BARE. A quoted scalar is a different string to the
	// derivation, which reads the raw text — the same hazard rawScalar exists for
	// on `impact`.
	blob, err := os.ReadFile(filepath.Join(r.Root(), res.Path))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(blob), "shipped_in: v0.1.0\n") {
		t.Errorf("shipped_in was not written bare; the derivation reads the raw scalar.\n%s", blob)
	}

	// The crossing. Commit the move, then ask the derivation what it sees.
	r.Commit("resolve with shipped_in")
	set, err := changelog.ShippedSince(r.Root(), "v0.1.0")
	if err != nil {
		t.Fatalf("ShippedSince: %v", err)
	}
	for _, rec := range set.Added {
		if rec.ID != "iss-1" {
			continue
		}
		t.Fatalf("the record capture just wrote is still in the cut (shipped_in=%q, err=%q); "+
			"writer and reader disagree about the value", rec.ShippedIn, rec.ShippedInErr)
	}
}

// The refusal path, asserted separately: a malformed value must be rejected
// BEFORE anything moves, so a bad flag leaves the ledger exactly as it was.
//
// This direction matters more than it looks. The field's job is to take records
// out of a release, so a value that reached the ledger malformed would sit there
// looking like an exclusion and never be one.
func TestResolveRefusesABadShippedInBeforeWriting(t *testing.T) {
	for _, bad := range []string{"0.1.0", "v0.1", "yesterday", "v0.1.0; rm -rf /", "v0.1.0\nimpact: breaking"} {
		t.Run(bad, func(t *testing.T) {
			r := gittest.NewRepo(t)
			openPath := ".abcd/work/issues/open/iss-1-old.md"
			r.Write(openPath,
				"---\nschema_version: 1\nid: \"iss-1\"\nslug: \"old\"\nseverity: \"minor\"\n"+
					"category: \"bug\"\nsource: \"user-observation\"\nfound_during: \"t\"\n---\n\nan old fix\n")
			r.Commit("an open record")

			if _, err := Resolve(ResolveRequest{
				Grounds:  testGrounds,
				RepoRoot: r.Root(), ID: "iss-1", Resolution: "note",
				Impact: "fix", ShippedIn: bad,
			}); err == nil {
				t.Fatalf("shipped_in %q was accepted; a value the derivation cannot read "+
					"must never reach the ledger", bad)
			}
			if _, err := os.Stat(filepath.Join(r.Root(), openPath)); err != nil {
				t.Errorf("the record left open/ despite the refusal: %v", err)
			}
		})
	}
}
