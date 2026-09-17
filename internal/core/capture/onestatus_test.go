package capture

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeLedgerRecord lays a minimal well-formed record into one status directory,
// creating the directory. It writes bytes rather than calling Capture, because
// every case here is a ledger state the write paths refuse to produce: the
// allocator will not mint a duplicate and a transition will not split one record
// across two folders. The states arise from MERGES, so they are built as merges
// leave them (iss-2609100507430423).
func writeLedgerRecord(t *testing.T, issuesRoot, status, name, id string) {
	t.Helper()
	dir := filepath.Join(issuesRoot, status)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	slug := strings.TrimPrefix(strings.TrimSuffix(name, ".md"), id+"-")
	body := "---\nschema_version: 1\nid: \"" + id + "\"\nslug: \"" + slug + "\"\n" +
		"severity: \"major\"\ncategory: \"bug\"\nsource: \"agent-finding\"\n" +
		"found_during: \"a merge\"\n"
	// The status folder decides which terminal property the record must carry, so
	// the fixture writes the one its folder requires: a record missing it is
	// skipped by the reader, which would make a clean-ledger control pass for the
	// wrong reason.
	switch status {
	case "resolved":
		body += "resolution: \"fixed in the change that found it\"\nimpact: fix\n"
	case "wontfix":
		body += "wontfix_reason: \"the cost outruns the defect\"\n"
	}
	body += "---\n\nthe finding.\n"
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// One id in two status folders has no status, so every read of the ledger
// refuses and names both files (iss-2609100507430423).
//
// The state is what landing a batch of worker branches produced: a record
// committed to the default branch in open/ after the branches were cut, and the
// same record resolved on one of them. Rename detection paired neither side, the
// integration branch carried both copies, and folder-membership-as-status then
// said one id was open and resolved at once. Nothing looked for it; it was found
// by reading open/ by hand.
func TestARecordIDInTwoStatusFoldersRefusesEveryRead(t *testing.T) {
	repo, ir := ledger(t)
	writeLedgerRecord(t, ir, "open", "iss-42-a-finding.md", "iss-42")
	writeLedgerRecord(t, ir, "resolved", "iss-42-a-finding.md", "iss-42")

	// Every read: the board, the filtered list, and the unfiltered one. The
	// FILTERED list matters most — the duplicate is invisible inside one status
	// directory, so a check that scanned only the requested state would pass on
	// the very view an operator uses.
	reads := map[string]func() error{
		"status": func() error {
			_, err := Status(StatusRequest{RepoRoot: repo, IssuesRoot: ir})
			return err
		},
		"list --all": func() error {
			_, err := List(ListRequest{RepoRoot: repo, IssuesRoot: ir, State: StateAll})
			return err
		},
		"list --open": func() error {
			_, err := List(ListRequest{RepoRoot: repo, IssuesRoot: ir, State: StateOpen})
			return err
		},
	}
	for name, read := range reads {
		t.Run(name, func(t *testing.T) {
			err := read()
			if err == nil {
				t.Fatalf("%s rendered a ledger holding iss-42 in two status folders; the id is open "+
					"and resolved at once, which is not a status", name)
			}
			if !errors.Is(err, ErrDuplicateIssueID) {
				t.Fatalf("error = %v, want ErrDuplicateIssueID", err)
			}
			// The refusal has to carry both locations and say what is wrong: an
			// operator can only fix this by moving or removing one of the two files.
			for _, want := range []string{
				"iss-42",
				".abcd/work/issues/open/iss-42-a-finding.md",
				".abcd/work/issues/resolved/iss-42-a-finding.md",
				"no defined status",
			} {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("the refusal does not name %q: %v", want, err)
				}
			}
			// A repo-relative locator, never an absolute developer path (iss-81).
			if strings.Contains(err.Error(), repo) {
				t.Errorf("the refusal leaks the absolute repo path: %v", err)
			}
		})
	}
}

// The padded twin is the same id, so it collides with it.
//
// A detector keyed on the filename's raw digits would file `iss-007` beside
// `iss-7` rather than on top of it, and fail open on exactly the spelling a
// hand-written or hand-merged file is most likely to carry.
func TestAZeroPaddedTwinIsTheSameRecordID(t *testing.T) {
	repo, ir := ledger(t)
	writeLedgerRecord(t, ir, "open", "iss-7-a-finding.md", "iss-7")
	writeLedgerRecord(t, ir, "resolved", "iss-007-a-finding.md", "iss-7")

	_, err := List(ListRequest{RepoRoot: repo, IssuesRoot: ir, State: StateAll})
	if !errors.Is(err, ErrDuplicateIssueID) {
		t.Fatalf("error = %v, want ErrDuplicateIssueID — iss-007 and iss-7 are one id", err)
	}
}

// Two records claiming one id inside ONE folder is the sibling shape. It refuses
// too, with the other diagnosis: the status is legible here, and what is wrong is
// that two files claim one identity.
func TestTwoRecordsInOneFolderClaimingOneIDRefuse(t *testing.T) {
	repo, ir := ledger(t)
	writeLedgerRecord(t, ir, "open", "iss-42-a-finding.md", "iss-42")
	writeLedgerRecord(t, ir, "open", "iss-42-the-same-finding-again.md", "iss-42")

	_, err := List(ListRequest{RepoRoot: repo, IssuesRoot: ir, State: StateOpen})
	if !errors.Is(err, ErrDuplicateIssueID) {
		t.Fatalf("error = %v, want ErrDuplicateIssueID", err)
	}
	if strings.Contains(err.Error(), "no defined status") {
		t.Errorf("the one-folder case borrows the two-folder diagnosis: %v", err)
	}
	if !strings.Contains(err.Error(), "must name one record") {
		t.Errorf("the refusal does not say what is wrong: %v", err)
	}
}

// The direction that keeps the check usable: an ordinary ledger, and the same
// record after a legitimate transition, read clean.
//
// Every status move deletes the source and writes the destination, which is the
// byte-level shape the duplicate check reads — so a check keyed on the wrong
// signal refuses every resolved record in the tree.
func TestAnOrdinaryLedgerReadsClean(t *testing.T) {
	repo, ir := ledger(t)
	writeLedgerRecord(t, ir, "open", "iss-1-still-open.md", "iss-1")
	writeLedgerRecord(t, ir, "resolved", "iss-2-answered.md", "iss-2")
	writeLedgerRecord(t, ir, "wontfix", "iss-3-declined.md", "iss-3")
	// A neighbour whose ordinal shares a prefix with another: iss-11 must not
	// answer for iss-1.
	writeLedgerRecord(t, ir, "open", "iss-11-a-different-finding.md", "iss-11")
	// Files that claim no id at all are none of this check's business.
	if err := os.WriteFile(filepath.Join(ir, "open", "README.md"), []byte("# open\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := List(ListRequest{RepoRoot: repo, IssuesRoot: ir, State: StateAll})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(res.Issues) != 4 {
		t.Fatalf("listed %d issues, want 4: %+v", len(res.Issues), res.Issues)
	}
	if _, err := Status(StatusRequest{RepoRoot: repo, IssuesRoot: ir}); err != nil {
		t.Fatalf("Status: %v", err)
	}
}

// A virgin ledger — no directories at all — is not a duplicate, and the read
// stays tolerant of it, as it was before this check existed.
func TestAVirginLedgerIsNotADuplicate(t *testing.T) {
	repo, ir := ledger(t)
	if _, err := List(ListRequest{RepoRoot: repo, IssuesRoot: ir, State: StateAll}); err != nil {
		t.Fatalf("List on a virgin ledger: %v", err)
	}
	if _, err := Status(StatusRequest{RepoRoot: repo, IssuesRoot: ir}); err != nil {
		t.Fatalf("Status on a virgin ledger: %v", err)
	}
}

// The refusal names EVERY duplicated id, not the first.
//
// An operator fixing a merge artefact should see the whole list in one pass; a
// refusal that named one would send them round the loop once per duplicate, which
// is how a gate gets worked around rather than satisfied.
func TestTheRefusalNamesEveryDuplicatedID(t *testing.T) {
	repo, ir := ledger(t)
	writeLedgerRecord(t, ir, "open", "iss-42-a-finding.md", "iss-42")
	writeLedgerRecord(t, ir, "resolved", "iss-42-a-finding.md", "iss-42")
	writeLedgerRecord(t, ir, "open", "iss-43-another-finding.md", "iss-43")
	writeLedgerRecord(t, ir, "wontfix", "iss-43-another-finding.md", "iss-43")
	writeLedgerRecord(t, ir, "open", "iss-44-an-untouched-finding.md", "iss-44")

	_, err := List(ListRequest{RepoRoot: repo, IssuesRoot: ir, State: StateAll})
	if err == nil {
		t.Fatal("List rendered a ledger holding two duplicated ids")
	}
	for _, want := range []string{"iss-42", "iss-43"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal does not name %q: %v", want, err)
		}
	}
	if strings.Contains(err.Error(), "iss-44") {
		t.Errorf("the refusal names iss-44, which is claimed by one record: %v", err)
	}
}
