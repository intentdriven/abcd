package guard

import "testing"

// TestGuardReadsGoRunAsTheProgramItsPackageNames: `go run <package>` builds
// and runs the program its package path names, as npx runs the one its
// package names, so a hazard naming that program fires behind it. A source
// checkout of abcd runs every abcd invocation as `go run ./cmd/abcd`, so the
// person-only ledger flip was one go run away from passing unread.
func TestGuardReadsGoRunAsTheProgramItsPackageNames(t *testing.T) {
	r := Defaults()
	const id = "abcd-source-ledger-flip"
	for _, line := range []string{
		`go run ./cmd/abcd library ledger --flip 3`,
		`go run ../abcd/cmd/abcd library ledger --flip 3`,
		`go run -tags dev ./cmd/abcd library ledger --flip 3`,
		`go run -ldflags=-s ./cmd/abcd library ledger --flip=3`,
		`go run -C /src/abcd ./cmd/abcd library ledger --flip 3`,
		`go run github.com/intentdriven/abcd/cmd/abcd@v0.15.1 library ledger --flip 3`,
		`env GOFLAGS=-mod=mod go run ./cmd/abcd library ledger --flip 3`,
	} {
		d, err := r.Check(line)
		if err != nil {
			t.Fatalf("%q: %v", line, err)
		}
		if d.Verdict != VerdictBlock || d.EntryID != id {
			t.Errorf("%q: verdict %q under %q, want a block under %q", line, d.Verdict, d.EntryID, id)
		}
	}
	for _, line := range []string{
		`go run ./cmd/abcd library ledger --list`,
		`go run ./cmd/abcd capture "an agent ran abcd library ledger --flip 3"`,
		`go run ./cmd/other library ledger --flip 3`,
		`go test ./... -run TestLedger`,
		`go build -o bin/abcd ./cmd/abcd`,
	} {
		if d, err := r.Check(line); err != nil || d.Verdict == VerdictBlock {
			t.Errorf("%q: blocked under %q (%v)", line, d.EntryID, err)
		}
	}
}
