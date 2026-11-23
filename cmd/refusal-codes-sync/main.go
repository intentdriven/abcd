// Command refusal-codes-sync writes the committed ledger of refusal codes,
// .abcd/development/release/refusal-codes.json, from the registry in
// internal/core/refusal (itd-2610301428378830, spc-2610301431108175 step 1).
// The release guard reads the ledger at the last release tag and refuses a cut
// in which a code that ledger lists is missing (adr-2610301638558260), so the
// ledger is the record of every code a release has carried. Codes are added,
// and retired, in the registry, never in the file.
//
// It mirrors command-facts-sync, run by hand via `make refusal-codes-sync`.
// Nothing in CI invokes it: the drift is gated by
// TestCommittedLedgerMatchesTheRegistry in internal/core/refusal, which fails
// when the committed ledger differs from the registry's.
//
// Exit codes: 0 nothing to do (or, with no -check, the ledger was written);
// 1 a fault; 3 with -check, the ledger is out of date and nothing was written.
package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/intentdriven/abcd/internal/core/refusal"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
)

func main() {
	check := flag.Bool("check", false, "report drift and write nothing; exit 3 when the ledger is out of date")
	rootPath := flag.String("root", "", "repo root to sync (default: git toplevel, or cwd)")
	flag.Parse()

	root := *rootPath
	if root == "" {
		root = resolveRoot()
	}
	wrote, err := run(root, refusal.BuildLedger(), *check)
	switch {
	case errors.Is(err, errOutOfDate):
		fmt.Printf("refusal-codes-sync: %s %s\n", refusal.LedgerPath, err)
		fmt.Fprintln(os.Stderr, "refusal-codes-sync: run `make refusal-codes-sync` to write the registry into the ledger.")
		os.Exit(3)
	case err != nil:
		fmt.Fprintln(os.Stderr, "refusal-codes-sync:", err)
		os.Exit(1)
	case wrote:
		fmt.Printf("refusal-codes-sync: wrote %s\n", refusal.LedgerPath)
	default:
		fmt.Println("refusal-codes-sync: the ledger already matches the registry; nothing to do.")
	}
}

// errOutOfDate is -check finding the committed ledger differs from the
// registry's.
var errOutOfDate = errors.New("is out of date with the refusal registry")

// run writes the ledger l under root, or with check only compares it. It
// reports whether it wrote the file.
func run(root string, l refusal.Ledger, check bool) (bool, error) {
	want, err := refusal.EncodeLedger(l)
	if err != nil {
		return false, err
	}
	path := filepath.Join(root, filepath.FromSlash(refusal.LedgerPath))
	got, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return false, fmt.Errorf("read %s: %w", refusal.LedgerPath, err)
	}
	if err == nil && bytes.Equal(got, want) {
		return false, nil
	}
	if check {
		return false, errOutOfDate
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return false, fmt.Errorf("create the ledger's folder: %w", err)
	}
	if err := fsutil.WriteFileAtomicPreserveMode(path, want); err != nil {
		return false, fmt.Errorf("write %s: %w", refusal.LedgerPath, err)
	}
	return true, nil
}

// resolveRoot returns the git toplevel, falling back to the working directory,
// as command-facts-sync does.
func resolveRoot() string {
	if wd, err := os.Getwd(); err == nil {
		if top, err := gitutil.Toplevel(wd); err == nil {
			return top
		}
		return wd
	}
	return "."
}
