package capture

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/intentdriven/abcd/internal/core/recordid"
)

// TestLedgerDetectionMatchesResolverGrammar pins capture's ledger record-detection
// to the ONE shared recordid grammar (recordid.FilenameNumRe) the read-side
// resolver and record-lint already use.
//
// A hand-crafted iss-7--x.md (a double hyphen in the slug tail) matches
// FilenameNumRe, so the resolver resolves it to iss-7 and record-lint reads it as
// a record — but capture's reader detected records with the STRICTER
// SplitRecordFilename, so findIssue could not locate iss-7 at all: the
// writer/reader/resolver/gate sat on two detection grammars
// (iss-2608280739112123). After the fix the detection sites share the resolver's
// grammar, so capture and the resolver agree on which filenames name a record.
//
// The strict filename<->frontmatter slug agreement (validateInvariants /
// record-lint's checkRecordFilenameSlug) stays on SplitRecordFilename: that is a
// separate, already-shared invariant, not the detection question this pins.
func TestLedgerDetectionMatchesResolverGrammar(t *testing.T) {
	repo := t.TempDir()
	ir := filepath.Join(repo, ".abcd", "work", "issues")
	open := filepath.Join(ir, "open")
	if err := os.MkdirAll(open, 0o755); err != nil {
		t.Fatal(err)
	}
	// Record-shaped for the resolver/lint (FilenameNumRe) but rejected by the
	// strict splitter: a double hyphen in the slug tail.
	const name = "iss-7--x.md"
	if err := os.WriteFile(filepath.Join(open, name), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	// The invariant we are matching TO: the resolver resolves iss-7 from this file.
	res, err := recordid.NewResolver(repo)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Has("iss-7") {
		t.Fatalf("resolver did not resolve iss-7 from %q; the grammar assumption changed", name)
	}

	// Capture's ledger lookup must AGREE: locate the same file the resolver
	// resolves, rather than reporting the id as unknown.
	path, status, err := findIssue(ir, "iss-7")
	if err != nil {
		t.Fatalf("findIssue could not locate iss-7 that the resolver resolves (grammars still diverge): %v", err)
	}
	if status != StateOpen || filepath.Base(path) != name {
		t.Fatalf("findIssue located %q (%s); want %q (open)", path, status, name)
	}
}
